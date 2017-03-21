package ansible

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gamunu/rmq"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/exec/misc"
	"github.com/pearsonappeng/tensor/exec/sync"
	"github.com/pearsonappeng/tensor/exec/types"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	"github.com/adjust/uniuri"
	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/ssh"
	"github.com/pearsonappeng/tensor/util"
	"path/filepath"
)

// Consumer is implementation of rmq.Consumer interface
type Consumer struct {
	name   string
	count  int
	before time.Time
}

// NewConsumer is the entrypoint to runners.Consumer
func NewConsumer(tag int) *Consumer {
	return &Consumer{
		name:   fmt.Sprintf("consumer%d", tag),
		count:  0,
		before: time.Now(),
	}
}

// Consume delegates jobs to appropriate runners
func (consumer *Consumer) Consume(delivery rmq.Delivery) {
	jb := types.AnsibleJob{}
	if err := json.Unmarshal([]byte(delivery.Payload()), &jb); err != nil {
		// handle error
		logrus.Warningln("Job delivery rejected")
		delivery.Reject()
		jobFail(&jb)
		return
	}

	// perform task
	delivery.Ack()
	logrus.WithFields(logrus.Fields{
		"Job ID": jb.Job.ID.Hex(),
		"Name":   jb.Job.Name,
	}).Infoln("Job successfuly received")

	status(&jb, "pending")

	logrus.WithFields(logrus.Fields{
		"Job ID": jb.Job.ID.Hex(),
		"Name":   jb.Job.Name,
	}).Infoln("Job changed status to pending")

	if jb.Job.JobType == ansible.JOBTYPE_UPDATE_JOB {
		sync.Sync(types.SyncJob{
			Job:           jb.Job,
			JobTemplateID: jb.Template.ID,
			ProjectID:     jb.Project.ID,
			SCM:           jb.SCM,
			Token:         jb.Token,
			User:          jb.User,
		})
		return
	}
	ansibleRun(&jb)
}

// Run starts consuming jobs into a channel of size prefetchLimit
func Run() {
	q := queue.OpenAnsibleQueue()

	q.StartConsuming(1, 500 * time.Millisecond)
	q.AddConsumer(util.UniqueNew(), NewConsumer(1))
}

func ansibleRun(j *types.AnsibleJob) {
	logrus.WithFields(logrus.Fields{
		"Job ID": j.Job.ID.Hex(),
		"Name":   j.Job.Name,
	}).Infoln("Job starting")

	// update if requested
	if j.PreviousJob != nil {
		// wait for scm update
		status(j, "waiting")

		logrus.WithFields(logrus.Fields{
			"Job ID": j.Job.ID.Hex(),
			"Name":   j.Job.Name,
		}).Infoln("Job changed status to waiting")

		ticker := time.NewTicker(time.Second * 2)

		for range ticker.C {
			if err := db.Jobs().FindId(j.PreviousJob.Job.ID).One(&j.PreviousJob.Job); err != nil {
				logrus.Warningln("Could not find Previous Job", err)
				continue
			}

			if j.PreviousJob.Job.Status == "failed" || j.PreviousJob.Job.Status == "error" {
				e := "Previous Task Failed: {\"job_type\": \"project_update\", \"job_name\": \"" + j.Job.Name + "\", \"job_id\": \"" + j.PreviousJob.Job.ID.Hex() + "\"}"
				logrus.Errorln(e)
				j.Job.JobExplanation = e
				j.Job.ResultStdout = "stdout capture is missing"
				jobError(j)
				return
			}
			if j.PreviousJob.Job.Status == "successful" {
				// stop the ticker and break the loop
				logrus.WithFields(logrus.Fields{
					"Job ID": j.PreviousJob.Job.ID.Hex(),
					"Name":   j.PreviousJob.Job.Name,
				}).Infoln("Update job successful")
				ticker.Stop()
				break
			}
		}
	}

	start(j)

	logrus.WithFields(logrus.Fields{
		"Job ID": j.Job.ID.Hex(),
		"Name":   j.Job.Name,
	}).Infoln("Job started")

	// Start SSH agent
	client, socket, pid, sshcleanup := ssh.StartAgent()

	if len(j.Machine.SSHKeyData) > 0 {
		if len(j.Machine.SSHKeyUnlock) > 0 {
			key, err := ssh.GetKey(util.Decipher(j.Machine.SSHKeyData), util.Decipher(j.Machine.SSHKeyUnlock))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while decyrpting Machine Credential")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
			if client.Add(key); err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
		}

		key, err := ssh.GetKey(util.Decipher(j.Machine.SSHKeyData), nil)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while decyrpting Machine Credential")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

		if client.Add(key); err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

	}

	if len(j.Network.SSHKeyData) > 0 {
		if len(j.Network.SSHKeyUnlock) > 0 {
			key, err := ssh.GetKey(util.Decipher(j.Machine.SSHKeyData), util.Decipher(j.Network.SSHKeyUnlock))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while decyrpting Machine Credential")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
			if client.Add(key); err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
		}

		key, err := ssh.GetKey(util.Decipher(j.Machine.SSHKeyData), nil)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while decyrpting Machine Credential")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

		if client.Add(key); err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

	}

	cmd, cleanup, err := getCmd(j, socket, pid)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Running playbook failed")
		j.Job.ResultStdout = "stdout capture is missing"
		j.Job.JobExplanation = err.Error()
		jobFail(j)
		return
	}

	// cleanup credential files
	defer func() {
		logrus.WithFields(logrus.Fields{
			"Job ID": j.Job.ID.Hex(),
			"Name":   j.Job.Name,
			"Status": j.Job.Status,
		}).Infoln("Stopped running Job")
		sshcleanup()
		cleanup()
	}()

	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	// Set setsid to create a new session, The new process group has no controlling
	// terminal which disables the stdin & will skip prompts
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Running ansible job failed")
		j.Job.JobExplanation = err.Error()
		j.Job.ResultStdout = string(b.Bytes())
		jobFail(j)
		return
	}

	var timer *time.Timer
	timer = time.AfterFunc(time.Duration(util.Config.AnsibleJobTimeOut) * time.Second, func() {
		logrus.Println("Killing the process. Execution exceeded threashold value")
		cmd.Process.Kill()
	})

	if err := cmd.Wait(); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Running playbook failed")
		j.Job.JobExplanation = err.Error()
		j.Job.ResultStdout = string(b.Bytes())
		jobFail(j)
		return
	}

	timer.Stop()
	// set stdout
	j.Job.ResultStdout = string(b.Bytes())
	//success
	jobSuccess(j)
}

// runPlaybook runs a Job using ansible-playbook command
func getCmd(j *types.AnsibleJob, socket string, pid int) (cmd *exec.Cmd, cleanup func(), err error) {
	// Generate directory paths and create directories
	tmp := "/tmp/tensor_proot_" + uniuri.New() + "/"
	j.Paths = types.JobPaths{
		Etc:             filepath.Join(tmp, uniuri.New()),
		Tmp:             filepath.Join(tmp, uniuri.New()),
		VarLib:          filepath.Join(tmp, uniuri.New()),
		VarLibJobStatus: filepath.Join(tmp, uniuri.New()),
		VarLibProjects:  filepath.Join(tmp, uniuri.New()),
		VarLog:          filepath.Join(tmp, uniuri.New()),
		TmpRand:         "/tmp/tensor__" + uniuri.New(),
		ProjectRoot:     filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		CredentialPath:  "/tmp/tensor_" + uniuri.New(),
	}
	// create job directories
	createTmpDirs(j)
	// ansible-playbook parameters
	pPlaybook := []string{
		"ansible-playbook", "-i", "/var/lib/tensor/plugins/inventory/tensorrest.py",
	}
	pPlaybook = buildParams(*j, pPlaybook)
	// parameters that are hidden from output
	pSecure := []string{}
	// check whether the username not empty
	if len(j.Machine.Username) > 0 {
		uname := j.Machine.Username
		// append domain if exist
		if len(j.Machine.Domain) > 0 {
			uname = j.Machine.Username + "@" + j.Machine.Domain
		}
		pPlaybook = append(pPlaybook, "-u", uname)
		if len(j.Machine.Password) > 0 && j.Machine.Kind == common.CredentialKindSSH {
			pSecure = append(pSecure, "-e", "ansible_ssh_pass=" + string(util.Decipher(j.Machine.Password)) + "")
		}
		// if credential type is windows the issue a kinit to acquire a kerberos ticket
		if len(j.Machine.Password) > 0 && j.Machine.Kind == common.CredentialKindWIN {
			kinit(*j)
		}
	}

	if j.Job.BecomeEnabled {
		pPlaybook = append(pPlaybook, "-b")
		// default become method is sudo
		if len(j.Machine.BecomeMethod) > 0 {
			pPlaybook = append(pPlaybook, "--become-method=" + j.Machine.BecomeMethod)
		}
		// default become user is root
		if len(j.Machine.BecomeUsername) > 0 {
			pPlaybook = append(pPlaybook, "--become-user=" + j.Machine.BecomeUsername)
		}
		// for now this is more convenient than --ask-become-pass with sshpass
		if len(j.Machine.BecomePassword) > 0 {
			pSecure = append(pSecure, "-e", "'ansible_become_pass=" + string(util.Decipher(j.Machine.BecomePassword)) + "'")
		}
	}
	// add proot and ansible parameters
	pargs := []string{"-v", "0", "-r", "/",
		"-b", j.Paths.Etc + ":/etc/tensor",
		"-b", j.Paths.Tmp + ":/tmp",
		"-b", j.Paths.VarLib + ":/var/lib/tensor",
		"-b", j.Paths.VarLibJobStatus + ":/var/lib/tensor/job_status",
		"-b", j.Paths.VarLibProjects + ":" + util.Config.ProjectsHome,
		"-b", j.Paths.VarLog + ":/var/log",
		"-b", j.Paths.TmpRand + ":" + j.Paths.TmpRand,
		"-b", filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()) + ":" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"-b", "/var/lib/tensor:/var/lib/tensor",
		"-w", filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
	}
	pargs = append(pargs, pPlaybook...)
	j.Job.JobARGS = pargs
	// should not included in any output
	pargs = append(pargs, pSecure...)
	// set job arguments, exclude unencrypted passwords etc.
	j.Job.JobARGS = []string{strings.Join(j.Job.JobARGS, " ") + " " + j.Job.Playbook + "'"}
	pargs = append(pargs, j.Job.Playbook)
	logrus.Infoln("Job Arguments", append([]string{}, j.Job.JobARGS...))
	cmd = exec.Command("proot", pargs...)
	cmd.Dir = filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex())
	cmd.Env = []string{
		"TERM=xterm",
		"PROJECT_PATH=" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"HOME_PATH=" + util.Config.ProjectsHome,
		"PWD=" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"SHLVL=0",
		"HOME=" + os.Getenv("HOME"),
		"_=/usr/bin/tensord",
		"PROOT_NO_SECCOMP=1",
		"PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"REST_API_TOKEN=" + j.Token,
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"ANSIBLE_CALLBACK_PLUGINS=/var/lib/tensor/plugins/callback",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + j.Job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
		"REST_API_URL=http://localhost" + util.Config.Port,
		"INVENTORY_HOSTVARS=True",
		"INVENTORY_ID=" + j.Inventory.ID.Hex(),
		"SSH_AUTH_SOCK=" + socket,
		"SSH_AGENT_PID=" + strconv.Itoa(pid),
	}
	// Assign job env here to ensure that sensitive information will
	// not be exposed
	j.Job.JobENV = []string{
		"TERM=xterm",
		"PROJECT_PATH=" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"HOME_PATH=" + util.Config.ProjectsHome,
		"PWD=" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"SHLVL=0",
		"HOME=" + os.Getenv("HOME"),
		"_=/usr/bin/tensord",
		"PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"REST_API_TOKEN=" + strings.Repeat("*", len(j.Token)),
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"ANSIBLE_CALLBACK_PLUGINS=/var/lib/tensor/plugins/callback",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + j.Job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
		"REST_API_URL=http://localhost" + util.Config.Port,
		"INVENTORY_HOSTVARS=True",
		"INVENTORY_ID=" + j.Inventory.ID.Hex(),
		"SSH_AUTH_SOCK=" + socket,
		"SSH_AGENT_PID=" + strconv.Itoa(pid),
	}
	var f *os.File
	if j.Cloud.Cloud {
		cmd.Env, f, err = misc.GetCloudCredential(cmd.Env, j.Cloud)
		if err != nil {
			return nil, nil, err
		}
	}
	logrus.WithFields(logrus.Fields{
		"Dir":         cmd.Dir,
		"Environment": append([]string{}, cmd.Env...),
	}).Infoln("Job Directory and Environment")
	return cmd, func() {
		if f != nil {
			if err := os.RemoveAll(f.Name()); err != nil {
				logrus.Errorln("Unable to remove cloud credential")
			}
		}

		if err := os.RemoveAll(tmp); err != nil {
			logrus.Errorln("Unable to remove tmp directories")
		}

		if err := os.RemoveAll(j.Paths.TmpRand); err != nil {
			logrus.Errorln("Unable to remove tmp random tmp dir")
		}
		if err := os.RemoveAll(j.Paths.CredentialPath); err != nil {
			logrus.Errorln("Unable to remove credential directories")
		}

		if j.Machine.Kind == common.CredentialKindWIN {
			if err := exec.Command("kdestroy").Run(); err != nil {
				logrus.Errorln("kdestroy failed")
			}
		}
	}, nil
}

func buildParams(j types.AnsibleJob, params []string) []string {
	if j.Job.JobType == "check" {
		params = append(params, "--check")
	}
	// forks -f NUM, --forks=NUM
	if j.Job.Forks != 0 {
		params = append(params, "-f", string(j.Job.Forks))
	}
	// limit  -l SUBSET, --limit=SUBSET
	if j.Job.Limit != "" {
		params = append(params, "-l", j.Job.Limit)
	}
	// verbosity  -v, --verbose
	switch j.Job.Verbosity {
	case 1:
		params = append(params, "-v")
		break
	case 2:
		params = append(params, "-vv")
		break
	case 3:
		params = append(params, "-vvv")
		break
	case 4:
		params = append(params, "-vvvv")
		break
	case 5:
		params = append(params, "-vvvv")
	}
	// extra variables -e EXTRA_VARS, --extra-vars=EXTRA_VARS
	if len(j.Job.ExtraVars) > 0 {
		vars, err := json.Marshal(j.Job.ExtraVars)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err,
			}).Errorln("Could not marshal extra vars")
		}
		params = append(params, "-e", string(vars))
	}
	// -t, TAGS, --tags=TAGS
	if len(j.Job.JobTags) > 0 {
		params = append(params, "-t", j.Job.JobTags)
	}
	// --skip-tags=SKIP_TAGS
	if len(j.Job.SkipTags) > 0 {
		params = append(params, "--skip-tags=" + j.Job.SkipTags)
	}
	// --force-handlers
	if j.Job.ForceHandlers {
		params = append(params, "--force-handlers")
	}
	if len(j.Job.StartAtTask) > 0 {
		params = append(params, "--start-at-task=" + j.Job.StartAtTask)
	}
	extras := map[string]interface{}{
		"tensor_job_template_name": j.Template.Name,
		"tensor_job_id":            j.Job.ID.Hex(),
		"tensor_user_id":           j.Job.CreatedByID.Hex(),
		"tensor_job_template_id":   j.Template.ID.Hex(),
		"tensor_user_name":         "admin",
		"tensor_job_launch_type":   j.Job.LaunchType,
	}
	// Parameters required by the system
	rp, err := json.Marshal(extras)
	if err != nil {
		logrus.Errorln("Error while marshalling parameters")
	}
	params = append(params, "-e", string(rp))
	return params
}

func kinit(j types.AnsibleJob) error {
	uname := j.Machine.Username
	// if credential domain specified
	if len(j.Machine.Domain) > 0 {
		uname = j.Machine.Username + "@" + j.Machine.Domain
	}
	kinit := exec.Command("kinit", uname)
	kinit.Env = os.Environ()
	stdin, err := kinit.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, string(util.Decipher(j.Machine.Password)))
	}()

	if err := kinit.Start(); err != nil {
		return err
	}

	return nil
}

func createTmpDirs(j *types.AnsibleJob) (err error) {
	// create credential paths
	if err = os.MkdirAll(j.Paths.Etc, 0770); err != nil {
		logrus.Errorln("Unable to create directory: ", j.Paths.Etc)
	}
	if err = os.MkdirAll(j.Paths.CredentialPath, 0770); err != nil {
		logrus.Errorln("Unable to create directory: ", j.Paths.CredentialPath)
	}
	if err = os.MkdirAll(j.Paths.Tmp, 0770); err != nil {
		logrus.Errorln("Unable to create directory: ", j.Paths.Tmp)
	}
	if err = os.MkdirAll(j.Paths.TmpRand, 0770); err != nil {
		logrus.Errorln("Unable to create directory: ", j.Paths.TmpRand)
	}
	if err = os.MkdirAll(j.Paths.VarLib, 0770); err != nil {
		logrus.Errorln("Unable to create directory: ", j.Paths.VarLib)
	}
	if err = os.MkdirAll(j.Paths.VarLibJobStatus, 0770); err != nil {
		logrus.Errorln("Unable to create directory: ", j.Paths.VarLibJobStatus)
	}
	if err = os.MkdirAll(j.Paths.VarLibProjects, 0770); err != nil {
		logrus.Errorln("Unable to create directory: ", j.Paths.VarLibProjects)
	}
	if err = os.MkdirAll(j.Paths.VarLog, 0770); err != nil {
		logrus.Errorln("Unable to create directory: ", j.Paths.VarLog)
	}
	return
}
