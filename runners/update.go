package runners

import (
	"log"
	"os/exec"
	"time"
	"fmt"
	"bitbucket.pearson.com/apseng/tensor/models"
	"strings"
	"os"
	"io/ioutil"
	"bitbucket.pearson.com/apseng/tensor/crypt"
	"bitbucket.pearson.com/apseng/tensor/ssh"
	"gopkg.in/mgo.v2/bson"
	"encoding/json"
	"bitbucket.pearson.com/apseng/tensor/db"
	"errors"
	"bitbucket.pearson.com/apseng/tensor/util/unique"
)

type SystemJobPool struct {
	queue    []*SystemJob
	Register chan *SystemJob
	running  *SystemJob
}

var SystemPool = SystemJobPool{
	queue:    make([]*SystemJob, 0),
	Register: make(chan *SystemJob),
	running:  nil,
}

func (p *SystemJobPool) run() {
	ticker := time.NewTicker(2 * time.Second)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case job := <-p.Register:
			if p.running == nil {
				go job.run()
				continue
			}

			p.queue = append(p.queue, job)
		case <-ticker.C:
			if len(p.queue) == 0 || p.running != nil {
				continue
			}

			fmt.Println("Running a task.")
			go SystemPool.queue[0].run()
			SystemPool.queue = SystemPool.queue[1:]
		}
	}
}

func StartSystemRunner() {
	SystemPool.run()
}

type SystemJob struct {
	Job            models.SystemJob
	Project        models.Project
	Credential     models.Credential
	CredentialPath string
}

func (j *SystemJob) run() {
	SystemPool.running = j

	defer func() {
		log.Println("Stopped running system jobs")
		SystemPool.running = nil
	}()

	j.start()

	j.CredentialPath = "/tmp/tensor_" + uniuri.New()
	// create job directories
	j.createJobDirs()

	log.Println("Started system job: " + j.Job.ID.Hex() + "\n")

	output, err := j.runJob();
	j.Job.ResultStdout = string(output)
	if err != nil {
		log.Println("Running Project update task failed", err)
		j.Job.JobExplanation = err.Error()
		j.fail()
		return
	}
	//success
	j.success()
}

func (j *SystemJob) runJob() ([]byte, error) {

	// ansible-playbook parameters
	arguments := []string{"-i", "localhost,", "-v", "-e", j.Job.ExtraVars, j.Job.Playbook}

	// set job arguments, exclude unencrypted passwords etc.
	j.Job.JobARGS = []string{"ansible-playbook", strings.Join(arguments, " ")}

	// Start SSH agent
	client, socket, cleanup := ssh.StartAgent()

	if j.Credential.SshKeyData != "" {

		if j.Credential.SshKeyUnlock != "" {
			key, err := ssh.GetEncryptedKey([]byte(crypt.Decrypt(j.Credential.SshKeyData)), crypt.Decrypt(j.Credential.SshKeyUnlock))
			if err != nil {
				return []byte("stdout capture is missing"), err
			}
			if client.Add(key); err != nil {
				return []byte("stdout capture is missing"), err
			}
		}

		key, err := ssh.GetKey([]byte(crypt.Decrypt(j.Credential.SshKeyData)))
		if err != nil {
			return []byte("stdout capture is missing"), err
		}

		if client.Add(key); err != nil {
			return []byte("stdout capture is missing"), err
		}

	}

	defer func() {
		// cleanup the mess
		cleanup()
	}()

	cmd := exec.Command("ansible-playbook", arguments...)
	cmd.Dir = "/opt/tensor/system/projects/"

	cmd.Env = append(os.Environ(), []string{
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"PS1=(tensor)",
		"ANSIBLE_CALLBACK_PLUGINS=/opt/tensor/plugins/callback",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + j.Job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
		"SSH_AUTH_SOCK=" + socket,
	}...)

	j.Job.JobENV = cmd.Env

	return cmd.CombinedOutput()
}

func (j *SystemJob) createJobDirs() {
	if err := os.MkdirAll(j.CredentialPath, 0770); err != nil {
		log.Println("Unable to create directory: ", j.CredentialPath)
	}

	if err := os.MkdirAll("/opt/tensor/projects/" + j.Job.ProjectID.Hex(), 0770); err != nil {
		log.Println("Unable to create directory: ", "/opt/tensor/projects/" + j.Job.ProjectID.Hex())
	}
}

func (j *SystemJob) installScmCred() error {
	fmt.Println("SCM Credentials " + j.Credential.Name + " installed")
	err := ioutil.WriteFile(j.CredentialPath + "/scm_crendetial", []byte(crypt.Decrypt(j.Credential.Secret)), 0600)
	return err
}

func UpdateProject(p models.Project) (error, bson.ObjectId) {
	job := models.SystemJob{
		ID: bson.NewObjectId(),
		Name: p.Name + " update Job",
		Description: "Updates " + p.Name + " Project",
		LaunchType: "automated",
		CancelFlag: false,
		Status: "pending",
		JobType: models.JOBTYPE_UPDATE_JOB,
		Playbook: "project_update.yml",
		Verbosity: 5,
		ProjectID: p.ID,
		Created:time.Now(),
		Modified:time.Now(),
	}

	if p.ScmCredentialID != nil {
		job.CredentialID = *p.ScmCredentialID
	}

	extras := map[string]interface{}{
		"scm_branch": p.ScmBranch,
		"scm_type": p.ScmType,
		"project_path": "/opt/tensor/projects/" + p.ID.Hex(),
		"scm_clean": p.ScmClean,
		"scm_url": p.ScmUrl,
		"scm_delete_on_update": p.ScmDeleteOnUpdate,
	}

	if p.ScmBranch == "" {
		extras["scm_branch"] = "HEAD"
	}

	// Parameters required by the system
	rp, err := json.Marshal(extras);

	if err != nil {
		log.Println("Error while marshalling parameters", err)
	}

	job.ExtraVars = string(rp)

	log.Print(string(rp))
	// Insert new job into jobs collection
	if err := db.Jobs().Insert(job); err != nil {
		log.Println("Error while creating update Job:", err)
		return errors.New("Error while creating update Job:"), ""
	}

	// create new background job
	runnerJob := SystemJob{
		Job: job,
		Project:p,
	}

	if len(job.CredentialID) == 12 {
		var credential models.Credential
		if err := db.Credentials().FindId(job.CredentialID).One(&credential); err != nil {
			log.Println("Error while getting SCM Credential", err)
			return errors.New("Error while getting SCM Credential"), ""
		}
		runnerJob.Credential = credential
	}

	SystemPool.Register <- &runnerJob

	return nil, job.ID
}

func getJobStatus(id bson.ObjectId) (string, error) {
	var job models.SystemJob
	if err := db.Jobs().FindId(id).One(&job); err != nil {
		log.Println("Error while getting SCM update Job", err)
		return "", errors.New("Error while getting SCM update Job")
	}
	return job.Status, nil
}