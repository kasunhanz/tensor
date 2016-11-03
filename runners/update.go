package runners

import (
	"log"
	"os/exec"
	"time"
	"fmt"
	"bitbucket.pearson.com/apseng/tensor/models"
	"strings"
	"os"
	"bitbucket.pearson.com/apseng/tensor/ssh"
	"gopkg.in/mgo.v2/bson"
	"encoding/json"
	"bitbucket.pearson.com/apseng/tensor/db"
	"errors"
	"bitbucket.pearson.com/apseng/tensor/util"
	"strconv"
)

type SystemJobPool struct {
	queue    []*SystemJob
	Register chan *SystemJob
	running  []*SystemJob
}

var SystemPool = SystemJobPool{
	queue:    make([]*SystemJob, 0),
	Register: make(chan *SystemJob),
	running:  make([]*SystemJob, 0),
}

// hasRunningJob will loop through the running job queue to
// determine whether the given job is available in the queue
// if the job is exist in the job queue will return true otherwise
// returns false
// Accepts pointer to SystemJob
func (p *SystemJobPool) hasRunningJob(job *SystemJob) bool {
	for _, v := range p.running {
		if v.Job.ID == job.Job.ID {
			return true
		}
	}
	return false
}

// hasJobForProject will loop through the running job queue to
// determine whether a job is running to update the project
// if a job is exist in the job queue will return true otherwise
// returns false
// Accepts pointer to SystemJob
func (p *SystemJobPool) hasJobForProject(job *SystemJob) bool {
	for _, v := range p.running {
		if v.Project.ID == job.Project.ID {
			return true
		}
	}
	return false
}

func (p *SystemJobPool) DetachFromRunning(id bson.ObjectId) bool {
	for k, v := range p.running {
		if v.Job.ID == id {
			p.running = append(p.running[:k], p.running[k + 1:]...)
			return true
		}
	}
	return false
}

// Run
func (p *SystemJobPool) Run() {
	ticker := time.NewTicker(2 * time.Second)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case job := <-p.Register:
		// check whether a existing
		// job running for the project update
			go job.run()
		case <-ticker.C:
			if len(p.queue) == 0 {
				continue
			}

			job := p.queue[0]
		// if has running jobs
			if p.hasRunningJob(job) {
				continue
			}

			fmt.Println("Running a system task.")
			go p.queue[0].run()
			p.queue = p.queue[1:]
		}
	}
}

type SystemJob struct {
	Job            models.SystemJob
	Project        models.Project
	Credential     models.Credential
	CredentialPath string
	SigKill        chan bool
}

func (j *SystemJob) run() {

	//create a boolean channel to send the kill signal
	j.SigKill = make(chan bool)

	SystemPool.running = append(SystemPool.running, j)

	j.start()
	// create job directories
	j.createJobDirs()

	log.Println("Started system job: " + j.Job.ID.Hex() + "\n")

	// Start SSH agent
	client, socket, pid, cleanup := ssh.StartAgent()

	if len(j.Credential.SshKeyData) > 0 {
		if len(j.Credential.SshKeyUnlock) > 0 {
			key, err := ssh.GetEncryptedKey([]byte(util.CipherDecrypt(j.Credential.SshKeyData)), util.CipherDecrypt(j.Credential.SshKeyUnlock))
			if err != nil {
				log.Println("Error while decrypting Credential", err)
				j.Job.JobExplanation = err.Error()
				j.fail()
				return
			}
			if client.Add(key); err != nil {
				log.Println("Error while adding decrypted Key", err)
				j.Job.JobExplanation = err.Error()
				j.fail()
				return
			}
		}

		key, err := ssh.GetKey([]byte(util.CipherDecrypt(j.Credential.SshKeyData)))

		if err != nil {
			log.Println("Error while decrypting Credential", err)
			j.Job.JobExplanation = err.Error()
			j.fail()
			return
		}

		if client.Add(key); err != nil {
			log.Println("Error while adding decrypted Key to SSH Agent", err)
			j.Job.JobExplanation = err.Error()
			j.fail()
			return
		}

	}

	defer func() {
		log.Println("Stopped running system jobs")
		SystemPool.DetachFromRunning(j.Job.ID)
		// cleanup the mess
		cleanup()
	}()

	cmd, err := j.getSystemCmd(socket, pid);

	if err != nil {
		log.Println("Running Project update task failed", err)
		j.Job.JobExplanation = err.Error()
		j.fail()
		return
	}

	// listen to channel
	// if true kill the channel and exit
	go func() {
		for {
			select {
			case kill := <-j.SigKill:
				log.Println("Received update job kill signal:", kill)
			// kill true then kill the job
				if kill {
					if err := cmd.Process.Kill(); err != nil {
						log.Println("Could not cancel the job")
						return // exit from goroutine
					}
					j.jobCancel() // update cancelled status
				}
			}
		}
	}()

	output, err := cmd.CombinedOutput()
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

func (j *SystemJob) getSystemCmd(socket string, pid int) (*exec.Cmd, error) {

	vars, err := json.Marshal(j.Job.ExtraVars)
	if err != nil {
		log.Println("Could not marshal extra vars", err)
	}
	// ansible-playbook parameters
	arguments := []string{"-i", "localhost,", "-v", "-e", string(vars), j.Job.Playbook}

	// set job arguments, exclude unencrypted passwords etc.
	j.Job.JobARGS = []string{"ansible-playbook", strings.Join(arguments, " ")}

	cmd := exec.Command("ansible-playbook", arguments...)
	cmd.Dir = "/opt/tensor/system/projects/"

	cmd.Env = []string{
		"TERM=xterm",
		"PROJECT_PATH=/opt/tensor/projects/" + j.Project.ID.Hex(),
		"HOME_PATH=/opt/tensor/",
		"PWD=/opt/tensor/projects/" + j.Project.ID.Hex(),
		"SHLVL=1",
		"HOME=/opt/tensor/projects/" + j.Project.ID.Hex(),
		"_=/opt/tensor/bin/tensord",
		"PATH=/bin:/usr/local/go/bin:/opt/tensor/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"ANSIBLE_CALLBACK_PLUGINS=/opt/tensor/plugins/callback",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + j.Job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
		"SSH_AUTH_SOCK=" + socket,
		"SSH_AGENT_PID=" + strconv.Itoa(pid),
	}

	j.Job.JobENV = cmd.Env

	return cmd, nil
}

func (j *SystemJob) createJobDirs() {
	if err := os.MkdirAll("/opt/tensor/projects/" + j.Job.ProjectID.Hex(), 0770); err != nil {
		log.Println("Unable to create directory: ", "/opt/tensor/projects/" + j.Job.ProjectID.Hex())
	}
}

func UpdateProject(p models.Project) (*SystemJob, error) {
	job := models.SystemJob{
		ID: bson.NewObjectId(),
		Name: p.Name + " update Job",
		Description: "Updates " + p.Name + " Project",
		LaunchType: models.JOB_LAUNCH_TYPE_MANUAL,
		CancelFlag: false,
		Status: "pending",
		JobType: models.JOBTYPE_UPDATE_JOB,
		Playbook: "project_update.yml",
		Verbosity: 0,
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

	job.ExtraVars = extras

	log.Print(string(rp))
	// Insert new job into jobs collection
	if err := db.Jobs().Insert(job); err != nil {
		log.Println("Error while creating update Job:", err)
		return nil, errors.New("Error while creating update Job:")
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
			return nil, errors.New("Error while getting SCM Credential")
		}
		runnerJob.Credential = credential
	}

	SystemPool.Register <- &runnerJob

	return &runnerJob, nil
}