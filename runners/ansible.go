package runners

import (
	"fmt"
	"time"
	"bitbucket.pearson.com/apseng/tensor/models"
	"log"
	"io/ioutil"
	"bitbucket.pearson.com/apseng/tensor/crypt"
	"bitbucket.pearson.com/apseng/tensor/util/unique"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
	"os/exec"
	"bitbucket.pearson.com/apseng/tensor/util"
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

func StartAnsibleRunner() {
	AnsiblePool.run()
}

func (j *AnsibleJob) run() {

	// TODO: update job template if requested
	if j.Project.ScmUpdateOnLaunch {
		// TODO: update code
		// since this is already ag
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
		j.fail()
		return
	}
	//success
	j.success()
}

// runPlaybook runs a Job using ansible-playbook command
// runPlaybook uses ansible proot for environment isolation to secure host machine
//
//  PRoot is a user-space implementation of chroot, mount --bind, and binfmt_misc.
//	This means that users don't need any privileges or setup to do things like using an arbitrary directory
// 	as the new root filesystem, making files accessible somewhere else in the
// 	filesystem hierarchy, or executing programs built for another CPU architecture transparently through QEMU user-mode.
// 	Also, developers can add their own features or use
// 	PRoot as a Linux process instrumentation  engine  thanks  to  its  extension  mechanism.
// 	Technically PRoot relies on ptrace, an unprivileged system-call available in every Linux kernel.
// 	for more information use linux man pages
// 	this routine uses PRoot -b path, --bind=path, -m path, --mount=path and  -w path, --pwd=path, --cwd=path
// 	option -b is to Make the content of path accessible in the guest rootfs.
// 	option -w Set the initial working directory to path.
func (j *AnsibleJob) runPlaybook() ([]byte, error) {

	// if add this if credential type is ssh
	pSSHAgent := []string{}
	j.buildSSHParams(j.JobPaths.CredentialPath, pSSHAgent)

	// proot parameters
	/*pProot := []string{
		"proot -v 0 -r /",
		"-b " + j.JobPaths.EtcTower + ":/etc/tensor",
		"-b " + j.JobPaths.Tmp + ":/tmp",
		"-b " + j.JobPaths.VarLib + ":/opt/tensor",
		"-b " + j.JobPaths.VarLibJobStatus + ":/opt/tensor/job_status",
		"-b " + j.JobPaths.VarLibProjects + ":/opt/tensor/projects",
		"-b " + j.JobPaths.TmpRand + ":" + j.JobPaths.TmpRand,
		"-b " + j.JobPaths.ProjectRoot + ":" + j.JobPaths.ProjectRoot,
		"-b " + j.JobPaths.AnsiblePath + ":" + j.JobPaths.AnsiblePath,
		"-w " + j.JobPaths.ProjectRoot,
	}*/

	// ansible-playbook parameters
	pPlaybook := []string{
		"ansible-playbook", "-i", "/opt/tensor/plugins/inventory/tensorrest.py",
	}
	j.buildParams(pPlaybook)

	// parameters that are hidden from output
	pSecure := []string{}

	if j.MachineCred.Username != "" {
		pPlaybook = append(pPlaybook, "-u", j.MachineCred.Username)
	}

	if j.Job.BecomeEnabled && j.MachineCred.BecomeMethod != "" &&
		j.MachineCred.BecomeUsername != "" {

		pPlaybook = append(pPlaybook, "-b", j.MachineCred.BecomeUsername)

		if j.MachineCred.BecomePassword != "" {
			pSecure = append(pSecure, "-e", "'ansible_become_pass=" + crypt.Decrypt(j.MachineCred.BecomePassword) + "'")
		}
	}

	pargs := []string{}
	argproot := pPlaybook// append(pProot[:], pPlaybook[:]...)
	if len(pSSHAgent) > 0 {
		// add ssh agent parameters
		pargs = append(pargs, pSSHAgent...)
		// add proot and ansible paramters
		pargs = append(pargs, argproot...)

		j.Job.JobARGS = pargs

		// should not included in any output
		pargs = append(pargs, pSecure...)

	} else {
		// add proot and ansible paramters
		pargs = append(pargs, argproot...)

		j.Job.JobARGS = pargs

		// should not included in any output
		pargs = append(pargs, pSecure...)
	}

	//For example, if I type something like:
	//$ exec /usr/bin/ssh-agent /bin/bash
	//from my shell prompt, I end up in a bash that is setup correctly with the agent. As soon as that bash dies, or any process that replaced bash with exec dies, the agent exits.
	// add -c for shell, yes it's ugly but meh! this is golden
	arguments := strings.Join(pargs, " ")
	csh := []string{
		"-c",
		"ssh-agent -a " + j.JobPaths.CredentialPath + "/ssh_auth.sock /bin/sh -c '" + arguments + " " + j.Job.Playbook + "'",
	}
	cmd := exec.Command("/bin/sh", csh...)
	cmd.Dir = "/opt/tensor/projects/" + j.Project.ID.Hex()

	cmd.Env = append(os.Environ(), []string{
		"REST_API_TOKEN=" + j.Token,
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"PS1=(tensor)",
		"ANSIBLE_CALLBACK_PLUGINS=/opt/tensor/plugins/callback",
		"LANG=en_US.UTF-8",
		"TZ=America/New_York",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + j.Job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
		"REST_API_URL=http://127.0.0.1:" + util.Config.Port,
		"INVENTORY_HOSTVARS=True",
		"INVENTORY_ID=" + j.Inventory.ID.Hex(),
	}...)

	j.Job.JobENV = cmd.Env

	return cmd.CombinedOutput()
}

func (j *AnsibleJob) installKey() error {
	fmt.Println("SSH Credential " + j.MachineCred.Name + " installed")
	err := ioutil.WriteFile(j.JobPaths.CredentialPath + "/machine_credential", []byte(crypt.Decrypt(j.MachineCred.Secret)), 0600)
	return err
}

// createJobDirs
func (j *AnsibleJob) createJobDirs() {
	// create credential paths
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

func (j *AnsibleJob) buildSSHParams(path string, prms []string) {
	if j.MachineCred.ID != "" && j.MachineCred.Kind == models.CREDENTIAL_KIND_SSH {
		prms = []string{
			"ssh-add", path + "/machine_credential",
			"&&", "rm -f " + path + "/machine_credential &&",
		}
	}

	if j.NetworkCred.ID != "" && j.NetworkCred.Kind == models.CREDENTIAL_KIND_NET {
		prms = append(prms,
			"ssh-add", path + "/network_credential &&",
			"rm -f " + path + "/network_credential &&",
		)
	}
}

func (j *AnsibleJob) buildParams(params []string) {
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
	rp, err := yaml.Marshal(map[interface{}]interface{}{
		"tower_job_template_name": j.Template.Name,
		"tower_job_id": j.Job.ID.Hex(),
		"tower_user_id": j.Job.CreatedByID.Hex(),
		"tower_job_template_id": j.Template.ID.Hex(),
		"tower_user_name": "admin",
		"tower_job_launch_type": j.Job.LaunchType,
	});

	if err != nil {
		log.Println("Error while marshalling parameters")
	}
	params = append(params, "-e \\'{" + string(rp) + "}\\'")
}