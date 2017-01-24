package runners

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gamunu/rmq"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models"
	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/ssh"
	"github.com/pearsonappeng/tensor/util"
)

// QueueJob contains all the information required to start a job
type QueueJob struct {
	Job            models.Job
	Template       models.JobTemplate
	MachineCred    models.Credential
	NetworkCred    models.Credential
	SCMCred        models.Credential
	CloudCred      models.Credential
	Inventory      models.Inventory
	Project        models.Project
	User           models.User
	PreviousJob    *QueueJob
	Token          string
	CredentialPath string // for system jobs
}

// Consumer is implementation of rmq.Consumer interface
type Consumer struct {
	name   string
	count  int
	before time.Time
}

// NewConsumer is the entrypoint for runners.Consumer
func NewConsumer(tag int) *Consumer {
	return &Consumer{
		name:   fmt.Sprintf("consumer%d", tag),
		count:  0,
		before: time.Now(),
	}
}

// Consume will deligate jobs to appropriate runners
func (consumer *Consumer) Consume(delivery rmq.Delivery) {
	jb := QueueJob{}
	if err := json.Unmarshal([]byte(delivery.Payload()), &jb); err != nil {
		// handle error
		log.Warningln("Job delivery rejected")
		delivery.Reject()
		jb.jobFail()
		return
	}

	// perform task
	delivery.Ack()
	log.WithFields(log.Fields{
		"Job ID": jb.Job.ID.Hex(),
		"Name":   jb.Job.Name,
	}).Infoln("Job successfuly received")

	jb.status("pending")

	log.WithFields(log.Fields{
		"Job ID": jb.Job.ID.Hex(),
		"Name":   jb.Job.Name,
	}).Infoln("Job changed status to pending")

	if jb.Job.JobType == models.JOBTYPE_UPDATE_JOB {
		systemRun(jb)
		return
	}
	ansibleRun(jb)
}

// AnsibleRun starts consuming jobs into a channel of size prefetchLimit
func AnsibleRun() {
	q := queue.OpenJobQueue()

	q.StartConsuming(1, 500*time.Millisecond)
	q.AddConsumer(util.UniqueNew(), NewConsumer(1))
}

func ansibleRun(j QueueJob) {
	log.WithFields(log.Fields{
		"Job ID": j.Job.ID.Hex(),
		"Name":   j.Job.Name,
	}).Infoln("Job starting")

	// update if requested
	if j.PreviousJob != nil {
		// wait for scm update
		j.status("waiting")

		log.WithFields(log.Fields{
			"Job ID": j.Job.ID.Hex(),
			"Name":   j.Job.Name,
		}).Infoln("Job changed status to waiting")

		ticker := time.NewTicker(time.Second * 2)

		for range ticker.C {
			if err := db.Jobs().FindId(j.PreviousJob.Job.ID).One(&j.PreviousJob.Job); err != nil {
				log.Warningln("Could not find Previous Job", err)
				continue
			}

			if j.PreviousJob.Job.Status == "failed" || j.PreviousJob.Job.Status == "error" {
				e := "Previous Task Failed: {\"job_type\": \"project_update\", \"job_name\": \"" + j.Job.Name + "\", \"job_id\": \"" + j.PreviousJob.Job.ID.Hex() + "\"}"
				log.Errorln(e)
				j.Job.JobExplanation = e
				j.Job.ResultStdout = "stdout capture is missing"
				j.jobError()
				return
			}
			if j.PreviousJob.Job.Status == "successful" {
				// stop the ticker and break the loop
				log.WithFields(log.Fields{
					"Job ID": j.PreviousJob.Job.ID.Hex(),
					"Name":   j.PreviousJob.Job.Name,
				}).Infoln("Update job successful")
				ticker.Stop()
				break
			}
		}
	}

	j.start()

	addActivity(j.Job.ID, j.User.ID, "Job "+j.Job.ID.Hex()+" is running", j.Job.JobType)
	log.WithFields(log.Fields{
		"Job ID": j.Job.ID.Hex(),
		"Name":   j.Job.Name,
	}).Infoln("Job started")

	// Start SSH agent
	client, socket, pid, cleanup := ssh.StartAgent()

	defer func() {
		log.WithFields(log.Fields{
			"Job ID": j.Job.ID.Hex(),
			"Name":   j.Job.Name,
			"Status": j.Job.Status,
		}).Infoln("Stopped running Job")
		addActivity(j.Job.ID, j.User.ID, "Job "+j.Job.ID.Hex()+" finished", j.Job.JobType)
		cleanup()
	}()

	if len(j.MachineCred.SshKeyData) > 0 {
		if len(j.MachineCred.SshKeyUnlock) > 0 {
			key, err := ssh.GetEncryptedKey([]byte(util.CipherDecrypt(j.MachineCred.SshKeyData)), util.CipherDecrypt(j.MachineCred.SshKeyUnlock))
			if err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while decyrpting Machine Credential")
				j.Job.JobExplanation = err.Error()
				j.jobFail()
				return
			}
			if client.Add(key); err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
				j.Job.JobExplanation = err.Error()
				j.jobFail()
				return
			}
		}

		key, err := ssh.GetKey([]byte(util.CipherDecrypt(j.MachineCred.SshKeyData)))
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Error while decyrpting Machine Credential")
			j.Job.JobExplanation = err.Error()
			j.jobFail()
			return
		}

		if client.Add(key); err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
			j.Job.JobExplanation = err.Error()
			j.jobFail()
			return
		}

	}

	if len(j.NetworkCred.SshKeyData) > 0 {
		if len(j.NetworkCred.SshKeyUnlock) > 0 {
			key, err := ssh.GetEncryptedKey([]byte(util.CipherDecrypt(j.MachineCred.SshKeyData)), util.CipherDecrypt(j.NetworkCred.SshKeyUnlock))
			if err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while decyrpting Machine Credential")
				j.Job.JobExplanation = err.Error()
				j.jobFail()
				return
			}
			if client.Add(key); err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
				j.Job.JobExplanation = err.Error()
				j.jobFail()
				return
			}
		}

		key, err := ssh.GetKey([]byte(util.CipherDecrypt(j.MachineCred.SshKeyData)))
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Error while decyrpting Machine Credential")
			j.Job.JobExplanation = err.Error()
			j.jobFail()
			return
		}

		if client.Add(key); err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
			j.Job.JobExplanation = err.Error()
			j.jobFail()
			return
		}

	}

	cmd, err := j.getCmd(socket, pid)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Running playbook failed")
		j.Job.ResultStdout = "stdout capture is missing"
		j.Job.JobExplanation = err.Error()
		j.jobFail()
		return
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Running playbook failed")
		j.Job.ResultStdout = "stdout capture is missing"
		j.Job.JobExplanation = err.Error()
		j.jobFail()
		return
	}
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	if err := cmd.Start(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Running playbook failed")
		j.Job.JobExplanation = err.Error()
		j.jobFail()
		return
	}
	var timer *time.Timer
	timer = time.AfterFunc(time.Duration(util.Config.JobTimeOut)*time.Second, func() {
		log.Println("Killing the process. Execution exceeded threashold value")
		cmd.Process.Kill()
	})

	if len(j.MachineCred.Password) > 0 && len(j.MachineCred.SshKeyData) <= 0 {
		log.Println("Using credential instead of SSH key")
		io.WriteString(stdin, util.CipherDecrypt(j.SCMCred.Password)+"\n")
	}

	if err := cmd.Wait(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Running playbook failed")
		j.Job.JobExplanation = err.Error()
		j.jobFail()
		return
	}

	timer.Stop()
	// set stdout
	j.Job.ResultStdout = string(b.Bytes())
	//success
	j.jobSuccess()
}

// runPlaybook runs a Job using ansible-playbook command
func (j *QueueJob) getCmd(socket string, pid int) (*exec.Cmd, error) {

	// ansible-playbook parameters
	pPlaybook := []string{
		"-i", "/var/lib/tensor/plugins/inventory/tensorrest.py",
	}
	pPlaybook = j.buildParams(pPlaybook)

	// parameters that are hidden from output
	pSecure := []string{}

	// check whether the username not empty
	if len(j.MachineCred.Username) > 0 {
		uname := j.MachineCred.Username

		// append domain if exist
		if len(j.MachineCred.Domain) > 0 {
			uname = j.MachineCred.Username + "@" + j.MachineCred.Domain
		}

		pPlaybook = append(pPlaybook, "-u", uname)

		if len(j.MachineCred.Password) > 0 && j.MachineCred.Kind == models.CREDENTIAL_KIND_SSH {
			pSecure = append(pSecure, "-e", "ansible_ssh_pass="+util.CipherDecrypt(j.MachineCred.Password)+"")
		}

		// if credential type is windows the issue a kinit to acquire a kerberos ticket
		if len(j.MachineCred.Password) > 0 && j.MachineCred.Kind == models.CREDENTIAL_KIND_WIN {
			j.kinit()
		}
	}

	if j.Job.BecomeEnabled {
		pPlaybook = append(pPlaybook, "-b")

		// default become method is sudo
		if len(j.MachineCred.BecomeMethod) > 0 {
			pPlaybook = append(pPlaybook, "--become-method="+j.MachineCred.BecomeMethod)
		}

		// default become user is root
		if len(j.MachineCred.BecomeUsername) > 0 {
			pPlaybook = append(pPlaybook, "--become-user="+j.MachineCred.BecomeUsername)
		}

		// for now this is more convenient than --ask-become-pass with sshpass
		if len(j.MachineCred.BecomePassword) > 0 {
			pSecure = append(pSecure, "-e", "'ansible_become_pass="+util.CipherDecrypt(j.MachineCred.BecomePassword)+"'")
		}
	}

	pargs := []string{}
	// add proot and ansible paramters
	pargs = append(pargs, pPlaybook...)
	j.Job.JobARGS = pargs
	// should not included in any output
	pargs = append(pargs, pSecure...)

	// set job arguments, exclude unencrypted passwords etc.
	j.Job.JobARGS = []string{strings.Join(j.Job.JobARGS, " ") + " " + j.Job.Playbook + "'"}

	pargs = append(pargs, j.Job.Playbook)
	log.Infoln("Job Arguments", append([]string{}, j.Job.JobARGS...))

	cmd := exec.Command("ansible-playbook", pargs...)
	cmd.Dir = util.Config.ProjectsHome + "/" + j.Project.ID.Hex()
	cmd.Env = []string{
		"TERM=xterm",
		"PROJECT_PATH=" + util.Config.ProjectsHome + "/" + j.Project.ID.Hex(),
		"HOME_PATH=" + util.Config.ProjectsHome + "/",
		"PWD=" + util.Config.ProjectsHome + "/" + j.Project.ID.Hex(),
		"SHLVL=1",
		"HOME=" + os.Getenv("HOME"),
		"_=/usr/bin/tensord",
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

	j.Job.JobENV = cmd.Env

	log.WithFields(log.Fields{
		"Dir":         cmd.Dir,
		"Environment": append([]string{}, cmd.Env...),
	}).Infoln("Job Directory and Environment")

	return cmd, nil
}

func (j *QueueJob) buildParams(params []string) []string {

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
			log.WithFields(log.Fields{
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
		params = append(params, "--skip-tags="+j.Job.SkipTags)
	}

	// --force-handlers
	if j.Job.ForceHandlers {
		params = append(params, "--force-handlers")
	}

	if len(j.Job.StartAtTask) > 0 {
		params = append(params, "--start-at-task="+j.Job.StartAtTask)
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
		log.Errorln("Error while marshalling parameters")
	}
	params = append(params, "-e", string(rp))

	return params
}

func (j *QueueJob) kinit() error {

	// Create two command structs for echo and kinit
	echo := exec.Command("echo", "-n", util.CipherDecrypt(j.MachineCred.Password))

	uname := j.MachineCred.Username

	// if credential domain specified
	if len(j.MachineCred.Domain) > 0 {
		uname = j.MachineCred.Username + "@" + j.MachineCred.Domain
	}

	kinit := exec.Command("kinit", uname)
	kinit.Env = os.Environ()

	// Create asynchronous in memory pipe
	r, w := io.Pipe()

	// set pipe writer to echo std out
	echo.Stdout = w
	// set pip reader to kinit std in
	kinit.Stdin = r

	// initialize new buffer
	var buffer bytes.Buffer
	kinit.Stdout = &buffer

	// start two commands
	if err := echo.Start(); err != nil {
		log.Errorln(err.Error())
		return err
	}

	if err := kinit.Start(); err != nil {
		log.Errorln(err.Error())
		return err
	}

	if err := echo.Wait(); err != nil {
		log.Errorln(err.Error())
		return err
	}

	if err := w.Close(); err != nil {
		log.Errorln(err.Error())
		return err
	}

	if err := kinit.Wait(); err != nil {
		log.Errorln(err.Error())
		return err
	}

	if _, err := io.Copy(os.Stdout, &buffer); err != nil {
		log.Errorln(err.Error())
		return err
	}

	return nil
}
