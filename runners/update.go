package runners

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/queue"
	"bitbucket.pearson.com/apseng/tensor/ssh"
	"bitbucket.pearson.com/apseng/tensor/util"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

func systemRun(j QueueJob) {
	j.start()
	// create job directories
	j.createJobDirs()

	log.Infoln("Started system job", "["+j.Job.ID.Hex()+"]", j.Job.Name)

	// Start SSH agent
	client, socket, pid, cleanup := ssh.StartAgent()

	if len(j.MachineCred.SshKeyData) > 0 {
		if len(j.MachineCred.SshKeyUnlock) > 0 {
			key, err := ssh.GetEncryptedKey([]byte(util.CipherDecrypt(j.MachineCred.SshKeyData)), util.CipherDecrypt(j.MachineCred.SshKeyUnlock))
			if err != nil {
				log.Errorln("Error while decrypting Credential", err)
				j.Job.JobExplanation = err.Error()
				j.jobFail()
				return
			}
			if client.Add(key); err != nil {
				log.Errorln("Error while adding decrypted Key", err)
				j.Job.JobExplanation = err.Error()
				j.jobFail()
				return
			}
		}

		key, err := ssh.GetKey([]byte(util.CipherDecrypt(j.MachineCred.SshKeyData)))

		if err != nil {
			log.Errorln("Error while decrypting Credential", err)
			j.Job.JobExplanation = err.Error()
			j.jobFail()
			return
		}

		if client.Add(key); err != nil {
			log.Errorln("Error while adding decrypted Key to SSH Agent", err)
			j.Job.JobExplanation = err.Error()
			j.jobFail()
			return
		}

	}

	defer func() {
		log.Infoln("Stopped running update system jobs", j.Job.Name, "["+j.Job.ID.Hex()+"]")
		// cleanup the mess
		cleanup()
	}()

	cmd, err := j.getSystemCmd(socket, pid)

	if err != nil {
		log.Errorln("Running Project update task failed", err)
		j.Job.JobExplanation = err.Error()
		j.jobFail()
		return
	}

	output, err := cmd.CombinedOutput()
	j.Job.ResultStdout = string(output)
	if err != nil {
		log.Errorln("Running Project update task failed", err)
		j.Job.JobExplanation = err.Error()
		j.jobFail()
		return
	}
	//success
	j.jobSuccess()
}

func (j *QueueJob) getSystemCmd(socket string, pid int) (*exec.Cmd, error) {

	vars, err := json.Marshal(j.Job.ExtraVars)
	if err != nil {
		log.Errorln("Could not marshal extra vars", err)
	}
	// ansible-playbook parameters
	arguments := []string{"-i", "localhost,", "-v", "-e", string(vars), j.Job.Playbook}

	// set job arguments, exclude unencrypted passwords etc.
	j.Job.JobARGS = []string{"ansible-playbook", strings.Join(arguments, " ")}

	cmd := exec.Command("ansible-playbook", arguments...)
	cmd.Dir = "/var/lib/tensor/projects/"

	cmd.Env = []string{
		"TERM=xterm",
		"PROJECT_PATH=" + util.Config.ProjectsHome + "/" + j.Project.ID.Hex(),
		"HOME_PATH=" + util.Config.ProjectsHome + "/",
		"PWD=" + util.Config.ProjectsHome + "/" + j.Project.ID.Hex(),
		"SHLVL=1",
		"HOME=/root",
		"_=/usr/bin/tensord",
		"PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"ANSIBLE_CALLBACK_PLUGINS=/var/lib/tensor/plugins/callback",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + j.Job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
		"SSH_AUTH_SOCK=" + socket,
		"SSH_AGENT_PID=" + strconv.Itoa(pid),
	}

	j.Job.JobENV = cmd.Env

	return cmd, nil
}

func (j *QueueJob) createJobDirs() {
	if err := os.MkdirAll(util.Config.ProjectsHome+"/"+j.Job.ProjectID.Hex(), 0770); err != nil {
		log.Errorln("Unable to create directory: ", util.Config.ProjectsHome+"/"+j.Job.ProjectID.Hex())
	}
}

// UpdateProject will create and start a update system job
// using ansible playbook project_update.yml
func UpdateProject(p models.Project) (*QueueJob, error) {
	job := models.Job{
		ID:           bson.NewObjectId(),
		Name:         p.Name + " update Job",
		Description:  "Updates " + p.Name + " Project",
		LaunchType:   models.JOB_LAUNCH_TYPE_MANUAL,
		CancelFlag:   false,
		Status:       "pending",
		JobType:      models.JOBTYPE_UPDATE_JOB,
		Playbook:     "project_update.yml",
		Verbosity:    0,
		ProjectID:    p.ID,
		Created:      time.Now(),
		Modified:     time.Now(),
		CreatedByID:  p.CreatedByID,
		ModifiedByID: p.ModifiedByID,
	}

	if p.ScmCredentialID != nil {
		job.SCMCredentialID = p.ScmCredentialID
	}

	extras := map[string]interface{}{
		"scm_branch":           p.ScmBranch,
		"scm_type":             p.ScmType,
		"project_path":         util.Config.ProjectsHome + "/" + p.ID.Hex(),
		"scm_clean":            p.ScmClean,
		"scm_url":              p.ScmUrl,
		"scm_delete_on_update": p.ScmDeleteOnUpdate,
		"scm_accept_hostkey":   true,
	}

	if p.ScmBranch == "" {
		extras["scm_branch"] = "HEAD"
	}

	job.ExtraVars = extras

	// Insert new job into jobs collection
	if err := db.Jobs().Insert(job); err != nil {
		log.Errorln("Error while creating update Job:", err)
		return nil, errors.New("Error while creating update Job")
	}

	// create new background job
	runnerJob := QueueJob{
		Job:     job,
		Project: p,
	}

	if job.SCMCredentialID != nil {
		var credential models.Credential
		if err := db.Credentials().FindId(*job.SCMCredentialID).One(&credential); err != nil {
			log.Errorln("Error while getting SCM Credential", err)
			return nil, errors.New("Error while getting SCM Credential")
		}
		runnerJob.SCMCred = credential
	}

	// Add the job to queue
	jobQueue := queue.OpenJobQueue()
	jobBytes, err := json.Marshal(runnerJob)
	if err != nil {
		return nil, err
	}
	jobQueue.PublishBytes(jobBytes)

	return &runnerJob, nil
}
