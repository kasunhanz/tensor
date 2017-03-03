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

	"github.com/Sirupsen/logrus"
	"github.com/gamunu/rmq"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/exec/misc"
	"github.com/pearsonappeng/tensor/exec/types"

	"github.com/adjust/uniuri"
	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/ssh"
	"github.com/pearsonappeng/tensor/util"
	"path/filepath"
	"github.com/rodaine/hclencoder"
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

// Consume will delegate jobs to appropriate runners
func (consumer *Consumer) Consume(delivery rmq.Delivery) {
	jb := types.TerraformJob{}
	if err := json.Unmarshal([]byte(delivery.Payload()), &jb); err != nil {
		// handle error
		logrus.Warningln("TerraformJob delivery rejected")
		delivery.Reject()
		jobFail(&jb)
		return
	}

	// perform task
	delivery.Ack()
	logrus.WithFields(logrus.Fields{
		"Job ID": jb.Job.ID.Hex(),
		"Name":   jb.Job.Name,
	}).Infoln("TerraformJob successfuly received")

	status(&jb, "pending")

	logrus.WithFields(logrus.Fields{
		"Terraform Job ID": jb.Job.ID.Hex(),
		"Name":             jb.Job.Name,
	}).Infoln("Terraform Job changed status to pending")

	terraformRun(&jb)
}

// Run starts consuming jobs into a channel of size prefetchLimit
func Run() {
	q := queue.OpenTerraformQueue()

	q.StartConsuming(1, 500 * time.Millisecond)
	q.AddConsumer(util.UniqueNew(), NewConsumer(1))
}

func terraformRun(j *types.TerraformJob) {
	logrus.WithFields(logrus.Fields{
		"Terraform Job ID": j.Job.ID.Hex(),
		"Name":             j.Job.Name,
	}).Infoln("Terraform Job starting")

	// update if requested
	if j.PreviousJob != nil {
		// wait for scm update
		status(j, "waiting")

		logrus.WithFields(logrus.Fields{
			"Job ID": j.Job.ID.Hex(),
			"Name":   j.Job.Name,
		}).Infoln("Terraform Job changed status to waiting")

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

	addActivity(j.Job.ID, j.User.ID, "Job " + j.Job.ID.Hex() + " is running", j.Job.JobType)
	logrus.WithFields(logrus.Fields{
		"Terraform Job ID": j.Job.ID.Hex(),
		"Name":             j.Job.Name,
	}).Infoln("Terraform Job started")

	// Start SSH agent
	client, socket, pid, sshcleanup := ssh.StartAgent()

	if len(j.Machine.SSHKeyData) > 0 {
		if len(j.Machine.SSHKeyUnlock) > 0 {
			key, err := ssh.GetKey([]byte(util.Decipher(j.Machine.SSHKeyData)),
				util.Decipher(j.Machine.SSHKeyUnlock))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while decrypting Machine Credential")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
			if client.Add(key); err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decrypted Machine Credential to SSH Agent")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
		}

		key, err := ssh.GetKey([]byte(util.Decipher(j.Machine.SSHKeyData)), nil)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while decrypting Machine Credential")
			j.Job.JobExplanation = err.Error()
			jobFail(j)
			return
		}

		if client.Add(key); err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("Error while adding decrypted Machine Credential to SSH Agent")
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
				}).Errorln("Error while decrypting Machine Credential")
				j.Job.JobExplanation = err.Error()
				jobFail(j)
				return
			}
			if client.Add(key); err != nil {
				logrus.WithFields(logrus.Fields{
					"Error": err.Error(),
				}).Errorln("Error while adding decrypted Machine Credential to SSH Agent")
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
		}).Errorln("Running terraform " + j.Job.JobType + " failed")
		j.Job.ResultStdout = "stdout capture is missing"
		j.Job.JobExplanation = err.Error()
		jobFail(j)
		return
	}
	// cleanup credential files
	defer func() {
		logrus.WithFields(logrus.Fields{
			"Terrraform Job ID": j.Job.ID.Hex(),
			"Name":              j.Job.Name,
			"Status":            j.Job.Status,
		}).Infoln("Stopped running Job")
		addActivity(j.Job.ID, j.User.ID, "Job " + j.Job.ID.Hex() + " finished", j.Job.JobType)
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
		}).Errorln("Running terraform " + j.Job.JobType + " failed")
		j.Job.JobExplanation = err.Error()
		j.Job.ResultStdout = string(b.Bytes())
		jobFail(j)
		return
	}
	var timer *time.Timer
	timer = time.AfterFunc(time.Duration(util.Config.TerraformJobTimeOut) * time.Second, func() {
		logrus.Println("Killing the process. Execution exceeded threashold value")
		cmd.Process.Kill()
	})
	if err := cmd.Wait(); err != nil {
		logrus.WithFields(logrus.Fields{
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
	// add proot and ansible parameters
	pargs := []string{"-v", "0", "-r", "/",
		"-b", j.Paths.Etc + ":/etc/tensor",
		"-b", j.Paths.Tmp + ":/tmp",
		"-b", j.Paths.VarLib + ":/var/lib/tensor",
		"-b", j.Paths.VarLibProjects + ":" + util.Config.ProjectsHome,
		"-b", j.Paths.VarLog + ":/var/log",
		"-b", j.Paths.TmpRand + ":" + j.Paths.TmpRand,
		"-b", filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()) + ":" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"-b", "/var/lib/tensor:/var/lib/tensor",
		"-w", filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
	}
	pTerraform := []string{"terraform"}
	pTerraform = buildParams(j, pTerraform)
	pargs = append(pargs, pTerraform...)
	j.Job.JobARGS = pargs
	j.Job.JobARGS = []string{strings.Join(j.Job.JobARGS, " ")}
	logrus.Infoln("Job Arguments", append([]string{}, j.Job.JobARGS...))
	cmd = exec.Command("proot", pargs...)
	cmd.Dir = filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex())
	cmd.Env = []string{
		"TERM=xterm",
		"PROJECT_PATH=" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"HOME_PATH=" + util.Config.ProjectsHome,
		"PWD=" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
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
	j.Job.JobENV = []string{
		"TERM=xterm",
		"PROJECT_PATH=" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"HOME_PATH=" + util.Config.ProjectsHome,
		"PWD=" + filepath.Join(util.Config.ProjectsHome, j.Project.ID.Hex()),
		"SHLVL=1",
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
	}, nil
}

func buildParams(j *types.TerraformJob, params []string) []string {
	if j.Job.JobType == "apply" {
		params = append(params, "apply", "-input=false")
	} else if j.Job.JobType == "plan" {
		params = append(params, "plan", "-input=false")
	}
	// extra variables -e EXTRA_VARS, --extra-vars=EXTRA_VARS
	if len(j.Job.Vars) > 0 {
		vars, err := hclencoder.Encode(j.Job.Vars)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err,
			}).Errorln("Could not marshal extra vars")
		}

		params = append(params, "-var", string(vars))
	}
	return params
}

func createTmpDirs(j *types.TerraformJob) (err error) {
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
