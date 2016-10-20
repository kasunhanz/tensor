package runners

import (
	"fmt"
	"time"
	"bitbucket.pearson.com/apseng/tensor/models"
	"log"
	"bitbucket.pearson.com/apseng/tensor/crypt"
	"bitbucket.pearson.com/apseng/tensor/util/unique"
	"os"
	"os/exec"
	"io"
	"bytes"
	"bitbucket.pearson.com/apseng/tensor/ssh"
	"strings"
	"gopkg.in/mgo.v2/bson"
)

// JobPaths
type JobPaths struct {
	EtcTower        string
	Tmp             string
	VarLib          string
	VarLibJobStatus string
	VarLibProjects  string
	VarLog          string
	TmpRand         string
	ProjectRoot     string
	AnsiblePath     string
	CredentialPath  string
}

type AnsibleJobPool struct {
	queue    []*AnsibleJob
	Register chan *AnsibleJob
	running  *AnsibleJob
}

var AnsiblePool = AnsibleJobPool{
	queue:    make([]*AnsibleJob, 0),
	Register: make(chan *AnsibleJob),
	running:  nil,
}

func (p *AnsibleJobPool) run() {
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
			go AnsiblePool.queue[0].run()
			AnsiblePool.queue = AnsiblePool.queue[1:]
		}
	}
}

func (p *AnsibleJobPool) RemoveFromPool(id bson.ObjectId) bool {
	for k, v := range p.queue {
		if v.Job.ID == id {
			p.queue = append(p.queue[:k], p.queue[k + 1:]...)
			v.jobCancel() // update job in database
			return true
		}
	}
	return false
}

func (p *AnsibleJobPool) CanCancel(id bson.ObjectId) bool {
	for _, v := range p.queue {
		if v.Job.ID == id {
			return true
		}
	}
	return false
}

func CancelJob(id bson.ObjectId) bool {
	return AnsiblePool.RemoveFromPool(id)
}

func CanCancel(id bson.ObjectId) bool {
	return AnsiblePool.CanCancel(id)
}

func StartAnsibleRunner() {
	AnsiblePool.run()
}

type AnsibleJob struct {
	Job         models.Job
	Template    models.JobTemplate
	MachineCred models.Credential
	NetworkCred models.Credential
	CloudCred   models.Credential
	Inventory   models.Inventory
	Project     models.Project
	User        models.User
	Token       string
	JobPaths    JobPaths
}

func (j *AnsibleJob) run() {

	j.status("pending")
	// update if requested
	if j.Project.ScmUpdateOnLaunch {
		// wait for scm update
		j.status("waiting")
		err, updateID := UpdateProject(j.Project)

		if err != nil {
			j.Job.JobExplanation = "Previous Task Failed: {\"job_type\": \"project_update\", \"job_name\": \"" + j.Job.Name + "\", \"job_id\": \"" + updateID.Hex() + "\"}"
			j.Job.ResultStdout = "stdout capture is missing"
			j.jobError()
			return
		}

		ticker := time.NewTicker(time.Second * 2)

		for range ticker.C {
			status, err := getJobStatus(updateID)
			if status == "failed" || status == "error" || err != nil {
				j.Job.JobExplanation = "Previous Task Failed: {\"job_type\": \"project_update\", \"job_name\": \"" + j.Job.Name + "\", \"job_id\": \"" + updateID.Hex() + "\"}"
				j.Job.ResultStdout = "stdout capture is missing"
				j.jobError()
				return
			}
			if status == "successful" {
				// stop the ticker and break the loop
				ticker.Stop()
				break
			}

		}
	}

	AnsiblePool.running = j

	defer func() {
		fmt.Println("Stopped running tasks")
		AnsiblePool.running = nil
		addActivity(j.Job.ID, j.User.ID, "Job " + j.Job.ID.Hex() + " finished")
	}()

	j.start()

	addActivity(j.Job.ID, j.User.ID, "Job " + j.Job.ID.Hex() + " is running")
	log.Println("Started: " + j.Job.ID.Hex() + "\n")

	//Generate directory paths and create directories
	tmp := "/tmp/tensor_proot_" + uniuri.New() + "/"
	j.JobPaths = JobPaths{
		EtcTower: tmp + uniuri.New(),
		Tmp: tmp + uniuri.New(),
		VarLib: tmp + uniuri.New(),
		VarLibJobStatus: tmp + uniuri.New(),
		VarLibProjects: tmp + uniuri.New(),
		VarLog: tmp + uniuri.New(),
		TmpRand: "/tmp/tensor__" + uniuri.New(),
		ProjectRoot: "/opt/tensor/projects/" + j.Project.ID.Hex(),
		AnsiblePath: "/opt/ansible/bin",
		CredentialPath: "/tmp/tensor_" + uniuri.New(),
	}

	// create job directories
	j.createJobDirs()

	output, err := j.runPlaybook();
	j.Job.ResultStdout = string(output)
	if err != nil {
		log.Println("Running playbook failed", err)
		j.Job.JobExplanation = err.Error()
		j.jobFail()
		return
	}
	//success
	j.jobSuccess()
}

// runPlaybook runs a Job using ansible-playbook command
func (j *AnsibleJob) runPlaybook() ([]byte, error) {

	// ansible-playbook parameters
	pPlaybook := []string{
		"-i", "/opt/tensor/plugins/inventory/tensorrest.py",
	}
	pPlaybook = j.buildParams(pPlaybook)

	// parameters that are hidden from output
	pSecure := []string{}

	if j.MachineCred.Username != "" {
		pPlaybook = append(pPlaybook, "-u", j.MachineCred.Username)

		if j.MachineCred.Password != "" && j.MachineCred.Kind == models.CREDENTIAL_KIND_SSH {
			pSecure = append(pSecure, "-e", "'ansible_ssh_pass=" + crypt.Decrypt(j.MachineCred.Password) + "'")
		}

		// if credential type is windows the issue a kinit to acquire a kerberos ticket
		if j.MachineCred.Password != "" && j.MachineCred.Kind == models.CREDENTIAL_KIND_WIN {
			j.kinit()
		}
	}

	if j.Job.BecomeEnabled {
		pPlaybook = append(pPlaybook, "-b")

		// default become method is sudo
		if j.MachineCred.BecomeMethod != "" {
			pPlaybook = append(pPlaybook, "--become-method=" + j.MachineCred.BecomeMethod)
		}

		// default become user is root
		if j.MachineCred.BecomeUsername != "" {
			pPlaybook = append(pPlaybook, "--become-user=" + j.MachineCred.BecomeUsername)
		}

		// for now this is more convenient than --ask-become-pass with sshpass
		if j.MachineCred.BecomePassword != "" {
			pSecure = append(pSecure, "-e", "'ansible_become_pass=" + crypt.Decrypt(j.MachineCred.BecomePassword) + "'")
		}
	}

	pargs := []string{}
	// add proot and ansible paramters
	pargs = append(pargs, pPlaybook...)
	j.Job.JobARGS = pargs
	// should not included in any output
	pargs = append(pargs, pSecure...)

	// Start SSH agent
	client, socket, cleanup := ssh.StartAgent()

	if j.MachineCred.SshKeyData != "" {

		if j.MachineCred.SshKeyUnlock != "" {
			key, err := ssh.GetEncryptedKey([]byte(crypt.Decrypt(j.MachineCred.SshKeyData)), crypt.Decrypt(j.MachineCred.SshKeyUnlock))
			if err != nil {
				return []byte("stdout capture is missing"), err
			}
			if client.Add(key); err != nil {
				return []byte("stdout capture is missing"), err
			}
		}

		key, err := ssh.GetKey([]byte(crypt.Decrypt(j.MachineCred.SshKeyData)))
		if err != nil {
			return []byte("stdout capture is missing"), err
		}

		if client.Add(key); err != nil {
			return []byte("stdout capture is missing"), err
		}

	}

	if j.NetworkCred.SshKeyData != "" {
		if j.NetworkCred.SshKeyUnlock != "" {
			key, err := ssh.GetEncryptedKey([]byte(crypt.Decrypt(j.MachineCred.SshKeyData)), crypt.Decrypt(j.NetworkCred.SshKeyUnlock))
			if err != nil {
				return []byte("stdout capture is missing"), err
			}
			if client.Add(key); err != nil {
				return []byte("stdout capture is missing"), err
			}
		}

		key, err := ssh.GetKey([]byte(crypt.Decrypt(j.MachineCred.SshKeyData)))
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


	// set job arguments, exclude unencrypted passwords etc.
	j.Job.JobARGS = []string{strings.Join(j.Job.JobARGS, " ") + " " + j.Job.Playbook + "'"}

	// For example, if I type something like:
	// $ exec /usr/bin/ssh-agent /bin/bash
	// from my shell prompt, I end up in a bash that is setup correctly with the agent.
	// As soon as that bash dies, or any process that replaced bash with exec dies, the agent exits.
	// add -c for shell, yes it's ugly but meh! this is golden
	pargs = append(pargs, j.Job.Playbook)

	cmd := exec.Command("ansible-playbook", pargs...)
	cmd.Dir = "/opt/tensor/projects/" + j.Project.ID.Hex()

	cmd.Env = append(os.Environ(), []string{
		"REST_API_TOKEN=" + j.Token,
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"PS1=(tensor)",
		"ANSIBLE_CALLBACK_PLUGINS=/opt/tensor/plugins/callback",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + j.Job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
		"REST_API_URL=http://localhost:8010",
		"INVENTORY_HOSTVARS=True",
		"INVENTORY_ID=" + j.Inventory.ID.Hex(),
		"SSH_AUTH_SOCK=" + socket,
	}...)

	j.Job.JobENV = cmd.Env

	return cmd.CombinedOutput()
}

// createJobDirs
func (j *AnsibleJob) createJobDirs() {
	// create credential paths
	if err := os.MkdirAll(j.JobPaths.EtcTower, 0770); err != nil {
		log.Println("Unable to create directory: ", j.JobPaths.EtcTower)
	}
	if err := os.MkdirAll(j.JobPaths.CredentialPath, 0770); err != nil {
		log.Println("Unable to create directory: ", j.JobPaths.CredentialPath)
	}
	if err := os.MkdirAll(j.JobPaths.Tmp, 0770); err != nil {
		log.Println("Unable to create directory: ", j.JobPaths.Tmp)
	}
	if err := os.MkdirAll(j.JobPaths.TmpRand, 0770); err != nil {
		log.Println("Unable to create directory: ", j.JobPaths.TmpRand)
	}
	if err := os.MkdirAll(j.JobPaths.VarLib, 0770); err != nil {
		log.Println("Unable to create directory: ", j.JobPaths.VarLib)
	}
	if err := os.MkdirAll(j.JobPaths.VarLibJobStatus, 0770); err != nil {
		log.Println("Unable to create directory: ", j.JobPaths.VarLibJobStatus)
	}
	if err := os.MkdirAll(j.JobPaths.VarLibProjects, 0770); err != nil {
		log.Println("Unable to create directory: ", j.JobPaths.VarLibProjects)
	}
	if err := os.MkdirAll(j.JobPaths.VarLog, 0770); err != nil {
		log.Println("Unable to create directory: ", j.JobPaths.VarLog)
	}
}

func (j *AnsibleJob) buildParams(params []string) []string {
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
	if j.Job.ExtraVars != "" {
		params = append(params, "-e", "'" + j.Job.ExtraVars + "'")
	}

	// -t, TAGS, --tags=TAGS
	if j.Job.JobTags != "" {
		params = append(params, "-t", j.Job.JobTags)
	}

	// --skip-tags=SKIP_TAGS
	if j.Job.SkipTags != "" {
		params = append(params, "--skip-tags=" + j.Job.SkipTags)
	}

	// --force-handlers
	if j.Job.ForceHandlers {
		params = append(params, "--force-handlers")
	}

	if j.Job.StartAtTask != "" {
		params = append(params, "--start-at-task=" + j.Job.StartAtTask)
	}

	// Parameters required by the system
	/*rp, err := yaml.Marshal(map[interface{}]interface{}{
		"tensor_job_template_name": j.Template.Name,
		"tensor_job_id": j.Job.ID.Hex(),
		"tensor_user_id": j.Job.CreatedByID.Hex(),
		"tensor_job_template_id": j.Template.ID.Hex(),
		"tensor_user_name": "admin",
		"tensor_job_launch_type": j.Job.LaunchType,
	});

	if err != nil {
		log.Println("Error while marshalling parameters")
	}
	params = append(params, "-e '{" + string(rp) + "}'")*/

	return params
}

func (j *AnsibleJob) kinit() error {

	// Create two command structs for echo and kinit
	echo := exec.Command("echo", "-n", crypt.Decrypt(j.MachineCred.Password))
	kinit := exec.Command("kinit", j.MachineCred.Username)
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
		log.Println(err.Error())
		return err
	}

	if err := kinit.Start(); err != nil {
		log.Println(err.Error())
		return err
	}

	if err := echo.Wait(); err != nil {
		log.Println(err.Error())
		return err
	}

	if err := w.Close(); err != nil {
		log.Println(err.Error())
		return err
	}

	if err := kinit.Wait(); err != nil {
		log.Println(err.Error())
		return err
	}

	if _, err := io.Copy(os.Stdout, &buffer); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}