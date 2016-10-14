package runners

import (
	"log"
	"os/exec"
	"gopkg.in/yaml.v2"
	"time"
	"fmt"
	"bitbucket.pearson.com/apseng/tensor/models"
	"strings"
	"os"
	"io/ioutil"
	"bitbucket.pearson.com/apseng/tensor/crypt"
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
	Job               models.Job
	Project           models.Project
	Credential        models.Credential
	User              models.User
	ScmCredentialPath string
}

func (j *SystemJob) run() {
	SystemPool.running = j

	defer func() {
		fmt.Println("Stopped running tasks")
		SystemPool.running = nil
		addActivity(j.Job.ID, j.User.ID, "Project Update Job " + j.Job.ID.Hex() + " finished")
	}()

	j.start()

	// create job directories
	j.createJobDirs()

	addSystemActivity(j.Job.ID, j.User.ID, "Project Update Job " + j.Job.ID.Hex() + " is running")
	log.Println("Started: " + j.Job.ID.Hex() + "\n")

	output, err := j.runUpdate();
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

func (j *SystemJob) runUpdate() ([]byte, error) {

	// ansible-playbook parameters
	arguments := []string{}
	j.buildSSHParams(arguments)
	j.buildParams(arguments)

	// set job arguments, exclude unencrypted passwords etc.
	j.Job.JobARGS = []string{"ssh-agent -a " + j.ScmCredentialPath + "/ssh_auth.sock /bin/sh -c '", strings.Join(arguments, " ")}

	csh := []string{
		"-c",
		"ssh-agent -a " + j.ScmCredentialPath + "/ssh_auth.sock /bin/sh -c '" + strings.Join(arguments, " ") + " " + j.Job.Playbook + "'",
	}

	cmd := exec.Command("ansible-playbook", csh...)
	cmd.Dir = "/opt/tensor/projects/" + j.Project.ID.Hex()

	cmd.Env = append(os.Environ(), []string{
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"PS1=(tensor)",
		"ANSIBLE_CALLBACK_PLUGINS=/opt/tensor/plugins/callback",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + j.Job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
	}...)

	j.Job.JobENV = cmd.Env

	return cmd.CombinedOutput()
}

func (j *SystemJob) createJobDirs() {
	if err := os.MkdirAll(j.ScmCredentialPath, 0770); err != nil {
		log.Println("Unable to create directory: ", j.ScmCredentialPath)
	}
}

func (j *SystemJob) buildParams(params []string) {
	// host information
	params = append(params, "ansible-playbook")
	params = append(params, "-i", "localhost")

	// verbosity
	params = append(params, "-v")

	// Parameters required by the system
	rp, err := yaml.Marshal(map[interface{}]interface{}{
		"scm_branch": j.Project.ScmBranch,
		"scm_type": j.Project.ScmType,
		"project_path": j.Project.LocalPath,
		"scm_clean": j.Project.ScmClean,
		"scm_url": j.Project.ScmUrl,
		"scm_delete_on_update": j.Project.ScmDeleteOnUpdate,
	});

	if err != nil {
		log.Println("Error while marshalling parameters")
	}
	params = append(params, "-e '{" + string(rp) + "}'")
}

func (j *SystemJob) buildSSHParams(prms []string) {
	if j.Credential.ID != "" && j.Credential.Kind == models.CREDENTIAL_KIND_SCM && j.Credential.Secret != "" {
		j.installScmCred()
		prms = []string{
			"ssh-add", j.ScmCredentialPath + "/scm_credential",
			"&&", "rm -f " + j.ScmCredentialPath + "/scm_credential &&",
		}
	}
}

func (j *SystemJob) installScmCred() error {
	fmt.Println("SCM Credentials " + j.Credential.Name + " installed")
	err := ioutil.WriteFile(j.ScmCredentialPath + "/scm_crendetial", []byte(crypt.Decrypt(j.Credential.Secret)), 0600)
	return err
}
