package sync

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/exec/types"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/ssh"
	"github.com/pearsonappeng/tensor/util"
)

func Sync(j types.SyncJob) {
	start(j)
	// create job directories
	createJobDirs(j)

	logrus.WithFields(logrus.Fields{
		"Job ID": j.Job.ID.Hex(),
		"Name":   j.Job.Name,
	}).Infoln("Started system job")

	// Start SSH agent
	agent, socket, pid, cleanup := ssh.StartAgent()

	if len(j.SCM.SSHKeyData) > 0 {
		if len(j.SCM.SSHKeyUnlock) > 0 {
			key, err := ssh.GetKey(util.Decipher(j.SCM.SSHKeyData), util.Decipher(j.SCM.SSHKeyUnlock))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while decrypting Credential")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
			if agent.Add(key); err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decrypted Key")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
		}

		key, err := ssh.GetKey(util.Decipher(j.SCM.SSHKeyData), nil)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while decrypting Credential")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

		if agent.Add(key); err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while adding decrypted Key to SSH Agent")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

	}

	defer func() {
		logrus.WithFields(logrus.Fields{
			"Job ID": j.Job.ID.Hex(),
			"Name":   j.Job.Name,
		}).Infoln("Stopped running update system jobs")
		// cleanup the mess
		cleanup()
	}()

	cmd, err := getCmd(&j, socket, pid)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Running Project update task failed")
		j.Job.JobExplanation = err.Error()
		jobFail(j)
		return
	}

	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	// Set setsid to create a new session, The new process group has no controlling
	// terminal which disables the stdin & will skip prompts
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Running Project update task failed")
		j.Job.ResultStdout = string(b.Bytes())
		j.Job.JobExplanation = err.Error()
		jobFail(j)
		return
	}

	var timer *time.Timer
	timer = time.AfterFunc(time.Duration(util.Config.SyncJobTimeOut) * time.Second, func() {
		logrus.Println("Killing the process. Execution exceeded threashold value")
		cmd.Process.Kill()
	})

	if err := cmd.Wait(); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Running Project update task failed")
		j.Job.ResultStdout = string(b.Bytes())
		j.Job.JobExplanation = err.Error()
		jobFail(j)
		return
	}

	timer.Stop()

	// set stdout
	j.Job.ResultStdout = string(b.Bytes())
	//success
	jobSuccess(j)
}

func getCmd(j *types.SyncJob, socket string, pid int) (*exec.Cmd, error) {

	vars, err := json.Marshal(j.Job.ExtraVars)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Could not marshal extra vars")
	}
	// ansible-playbook parameters
	arguments := []string{"-i", "localhost,", "-v", "-e", string(vars), j.Job.Playbook}

	// set job arguments, exclude unencrypted passwords etc.
	j.Job.JobARGS = []string{"ansible-playbook", strings.Join(arguments, " ")}

	cmd := exec.Command("ansible-playbook", arguments...)
	cmd.Dir = "/var/lib/tensor/playbooks/"

	cmd.Env = []string{
		"TERM=xterm",
		"PROJECT_PATH=" + util.Config.ProjectsHome + "/" + j.Project.ID.Hex(),
		"HOME_PATH=" + util.Config.ProjectsHome + "/",
		"PWD=" + util.Config.ProjectsHome + "/" + j.Project.ID.Hex(),
		"SHLVL=1",
		"HOME=" + os.Getenv("HOME"),
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

func createJobDirs(j types.SyncJob) {
	if err := os.MkdirAll(util.Config.ProjectsHome + "/" + j.Job.ProjectID.Hex(), 0770); err != nil {
		logrus.WithFields(logrus.Fields{
			"Dir":   util.Config.ProjectsHome + "/" + j.Job.ProjectID.Hex(),
			"Error": err.Error(),
		}).Errorln("Unable to create directory: ")
	}
}

// UpdateProject will create and start a update system job
// using ansible playbook project_update.yml
func UpdateProject(p common.Project) (*types.SyncJob, error) {
	job := ansible.Job{
		ID:           bson.NewObjectId(),
		Name:         p.Name + " update Job",
		Description:  "Updates " + p.Name + " Project",
		LaunchType:   ansible.JOB_LAUNCH_TYPE_MANUAL,
		CancelFlag:   false,
		Status:       "pending",
		JobType:      ansible.JOBTYPE_UPDATE_JOB,
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
		"scm_url":              p.ScmURL,
		"scm_delete_on_update": p.ScmDeleteOnUpdate,
		"scm_accept_hostkey":   true,
	}

	if p.ScmBranch == "" {
		extras["scm_branch"] = "HEAD"
	}

	job.ExtraVars = extras

	// Insert new job into jobs collection
	if err := db.Jobs().Insert(job); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Error while creating update Job")
		return nil, errors.New("Error while creating update Job")
	}

	// create new background job
	runnerJob := types.SyncJob{
		Job:     job,
		Project: p,
	}

	if job.SCMCredentialID != nil {
		var credential common.Credential
		if err := db.Credentials().FindId(*job.SCMCredentialID).One(&credential); err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while getting SCM Credential")
			return nil, errors.New("Error while getting SCM Credential")
		}
		runnerJob.SCM = credential
	}

	// Add the job to queue
	jobQueue := queue.OpenAnsibleQueue()
	jobBytes, err := json.Marshal(runnerJob)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Unable to marshal Job")
		return nil, err
	}
	jobQueue.PublishBytes(jobBytes)

	return &runnerJob, nil
}
