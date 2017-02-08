package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gamunu/rmq"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/exec/misc"
	"github.com/pearsonappeng/tensor/exec/types"

	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/ssh"
	"github.com/pearsonappeng/tensor/util"
)

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
	jb := types.TerraformJob{}
	if err := json.Unmarshal([]byte(delivery.Payload()), &jb); err != nil {
		// handle error
		log.Warningln("TerraformJob delivery rejected")
		delivery.Reject()
		jobFail(jb)
		return
	}

	// perform task
	delivery.Ack()
	log.WithFields(log.Fields{
		"Job ID": jb.Job.ID.Hex(),
		"Name":   jb.Job.Name,
	}).Infoln("TerraformJob successfuly received")

	status(jb, "pending")

	log.WithFields(log.Fields{
		"Terraform Job ID": jb.Job.ID.Hex(),
		"Name":             jb.Job.Name,
	}).Infoln("Terraform Job changed status to pending")

	terraformRun(jb)
}

// Run starts consuming jobs into a channel of size prefetchLimit
func Run() {
	q := queue.OpenTerraformQueue()

	q.StartConsuming(1, 500 * time.Millisecond)
	q.AddConsumer(util.UniqueNew(), NewConsumer(1))
}

func terraformRun(j types.TerraformJob) {
	log.WithFields(log.Fields{
		"Terraform Job ID": j.Job.ID.Hex(),
		"Name":             j.Job.Name,
	}).Infoln("Terraform Job starting")

	// update if requested
	if j.PreviousJob != nil {
		// wait for scm update
		status(j, "waiting")

		log.WithFields(log.Fields{
			"Job ID": j.Job.ID.Hex(),
			"Name":   j.Job.Name,
		}).Infoln("Terraform Job changed status to waiting")

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
				jobError(j)
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

	start(j)

	addActivity(j.Job.ID, j.User.ID, "Job " + j.Job.ID.Hex() + " is running", j.Job.JobType)
	log.WithFields(log.Fields{
		"Terraform Job ID": j.Job.ID.Hex(),
		"Name":             j.Job.Name,
	}).Infoln("Terraform Job started")

	// Start SSH agent
	client, socket, pid, cleanup := ssh.StartAgent()

	defer func() {
		log.WithFields(log.Fields{
			"Terrraform Job ID": j.Job.ID.Hex(),
			"Name":              j.Job.Name,
			"Status":            j.Job.Status,
		}).Infoln("Stopped running Job")
		addActivity(j.Job.ID, j.User.ID, "Job " + j.Job.ID.Hex() + " finished", j.Job.JobType)
		cleanup()
	}()

	if len(j.Machine.SSHKeyData) > 0 {
		if len(j.Machine.SSHKeyUnlock) > 0 {
			key, err := ssh.GetEncryptedKey([]byte(util.CipherDecrypt(j.Machine.SSHKeyData)), util.CipherDecrypt(j.Machine.SSHKeyUnlock))
			if err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while decrypting Machine Credential")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
			if client.Add(key); err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decrypted Machine Credential to SSH Agent")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
		}

		key, err := ssh.GetKey([]byte(util.CipherDecrypt(j.Machine.SSHKeyData)))
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Error while decrypting Machine Credential")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

		if client.Add(key); err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Error while adding decrypted Machine Credential to SSH Agent")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

	}

	if len(j.Network.SSHKeyData) > 0 {
		if len(j.Network.SSHKeyUnlock) > 0 {
			key, err := ssh.GetEncryptedKey([]byte(util.CipherDecrypt(j.Machine.SSHKeyData)), util.CipherDecrypt(j.Network.SSHKeyUnlock))
			if err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while decrypting Machine Credential")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
			if client.Add(key); err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decrypted Machine Credential to SSH Agent")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
		}

		key, err := ssh.GetKey([]byte(util.CipherDecrypt(j.Machine.SSHKeyData)))
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Error while decyrpting Machine Credential")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

		if client.Add(key); err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Error while adding decyrpted Machine Credential to SSH Agent")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

	}

	cmd, cleanup, err := getCmd(&j, socket, pid)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Running terraform " + j.Job.JobType + " failed")
		j.Job.ResultStdout = "stdout capture is missing"
		j.Job.JobExplanation = err.Error()
		jobFail(j)
		return
	}

	// cleanup credential files
	defer cleanup()

	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	// Set setsid to create a new session, The new process group has no controlling
	// terminal which disables the stdin & will skip prompts
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Running terraform " + j.Job.JobType + " failed")
		j.Job.JobExplanation = err.Error()
		j.Job.ResultStdout = string(b.Bytes())
		jobFail(j)
		return
	}

	var timer *time.Timer
	timer = time.AfterFunc(time.Duration(util.Config.TerraformJobTimeOut) * time.Second, func() {
		log.Println("Killing the process. Execution exceeded threashold value")
		cmd.Process.Kill()
	})

	if err := cmd.Wait(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Running terraform " + j.Job.JobType + " failed")
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
func getCmd(j *types.TerraformJob, socket string, pid int) (cmd *exec.Cmd, cleanup func(), err error) {

	pargs := []string{}
	pargs = buildParams(*j, pargs)

	j.Job.JobARGS = pargs

	j.Job.JobARGS = []string{strings.Join(j.Job.JobARGS, " ")}

	log.Infoln("Job Arguments", append([]string{}, j.Job.JobARGS...))

	cmd = exec.Command("terraform", pargs...)
	cmd.Dir = util.Config.ProjectsHome + "/" + j.Project.ID.Hex()
	env := []string{
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
		"SSH_AUTH_SOCK=" + socket,
		"SSH_AGENT_PID=" + strconv.Itoa(pid),
	}

	// Assign job env here to ensure that sensitive information will
	// not be exposed
	j.Job.JobENV = env
	var f *os.File

	if j.Cloud.Cloud {
		env, f, err = misc.GetCloudCredential(env, j.Cloud)
		if err != nil {
			return nil, nil, err
		}
	}

	cmd.Env = env

	log.WithFields(log.Fields{
		"Dir":         cmd.Dir,
		"Environment": append([]string{}, cmd.Env...),
	}).Infoln("Job Directory and Environment")

	return cmd, func() {
		if f != nil {
			if err := os.Remove(f.Name()); err != nil {
				log.Errorln("Unable to remove cloud credential")
			}
		}
	}, nil
}

func buildParams(j types.TerraformJob, params []string) []string {
	if j.Job.JobType == "apply" {
		params = append(params, "apply", "-input=false")
	} else if j.Job.JobType == "plan" {
		params = append(params, "plan", "-input=false")
	}
	// extra variables -e EXTRA_VARS, --extra-vars=EXTRA_VARS
	if len(j.Job.Vars) > 0 {
		vars, err := json.Marshal(j.Job.Vars)
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err,
			}).Errorln("Could not marshal extra vars")
		}
		params = append(params, "-v", string(vars))
	}
	return params
}
