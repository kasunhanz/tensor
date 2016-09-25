package runners

import (
	"fmt"
	"time"
	"os/exec"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/util"
	"log"
)

type AnsibleJob struct {
	job         models.Job
	template    models.JobTemplate
	credentials models.Credential
	inventory   models.Inventory
	project     models.Project
	user        models.User
}

type AnsibleJobPool struct {
	queue    []*AnsibleJob
	register chan *AnsibleJob
	running  *AnsibleJob
}

var AnsiblePool = AnsibleJobPool{
	queue:    make([]*AnsibleJob, 0),
	register: make(chan *AnsibleJob),
	running:  nil,
}

func (p *AnsibleJobPool) run() {
	ticker := time.NewTicker(2 * time.Second)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case task := <-p.register:
			fmt.Println(task)
			if p.running == nil {
				go task.run()
				continue
			}

			p.queue = append(p.queue, task)
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

func (t *AnsibleJob) run() {
	AnsiblePool.running = t

	defer func() {
		fmt.Println("Stopped running tasks")
		AnsiblePool.running = nil

		t.success()

		/*if err := (models.Activity{
			Type:  "job",
			ObjectID:    t.job.ID,
			Description: "Job " + t.job.ID.Hex() + " finished",
		}.Insert()); err != nil {
			log.Println("Fatal error inserting an event")
		}*/
	}()

	t.start()

	/*if err := (models.Activity{
		ProjectID:   t.project.ID,
		Type:  "job",
		ObjectID:    t.job.ID,
		Description: "Job " + t.job.ID.Hex() + " is running",
	}.Insert()); err != nil {
		log.Println("Fatal error inserting an event")
	}*/

	log.Println("Started: " + t.job.ID.Hex() + "\n")

	output, err := t.runPlaybook();

	if err != nil {
		log.Println("Running playbook failed", err)
		t.fail()
		return
	}

	t.job.StdoutText = string(output[:])
}

func (t *AnsibleJob) runPlaybook() ([]byte, error) {

	args := []string{}
	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = util.Config.HomePath

	// This is must for Ansible
	//env := os.Environ()
	cmd.Env = []string{
		"REST_API_TOKEN=" + "token",
		"ANSIBLE_PARAMIKO_RECORD_HOST_KEYS=False",
		"HOME=" + util.Config.HomePath,
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin",
		"PS1=(tensor)",
		"ANSIBLE_CALLBACK_PLUGINS=" + util.Config.HomePath + "/plugins/callback",
		"LANG=en_US.UTF-8",
		"TZ=America/New_York",
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"JOB_ID=" + t.job.ID.Hex(),
		"ANSIBLE_FORCE_COLOR=True",
		"REST_API_URL=http://127.0.0.1:" + util.Config.Port,
		"INVENTORY_HOSTVARS=True",
		"INVENTORY_ID=" + t.inventory.ID.Hex(),
		"PWD=" + util.Config.HomePath,
		"USER=tensor",
	}

	return cmd.CombinedOutput()
}
