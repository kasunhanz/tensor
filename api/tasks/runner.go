package tasks

import (
	"errors"
	"fmt"
	"time"

	"encoding/json"
	database "github.com/gamunu/hilbert-space/db"
	"github.com/gamunu/hilbert-space/models"
	"github.com/gamunu/hilbert-space/util"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"os"
	"os/exec"
)

type task struct {
	task        models.Task
	template    models.Template
	sshKey      models.AccessKey
	inventory   models.Inventory
	repository  models.Repository
	environment models.Environment
	users       []bson.ObjectId
	projectID   bson.ObjectId
}

func (t *task) fail() {
	t.task.Status = "error"
	t.updateStatus()
}

func (t *task) run() {
	pool.running = t

	defer func() {
		fmt.Println("Stopped running tasks")
		pool.running = nil

		now := time.Now()
		t.task.End = now
		t.updateStatus()

		if err := (models.Event{
			ProjectID:   t.projectID,
			ObjectType:  "task",
			ObjectID:    t.task.ID,
			Description: "Task ID " + t.task.ID.String() + " finished",
		}.Insert()); err != nil {
			t.log("Fatal error inserting an event")
			panic(err)
		}
	}()

	if err := t.populateDetails(); err != nil {
		t.log("Error: " + err.Error())
		t.fail()
		return
	}

	fmt.Println(t.users)
	now := time.Now()
	t.task.Status = "running"
	t.task.Start = now
	t.updateStatus()

	if err := (models.Event{
		ProjectID:   t.projectID,
		ObjectType:  "task",
		ObjectID:    t.task.ID,
		Description: "Task ID " + t.task.ID.String() + " is running",
	}.Insert()); err != nil {
		t.log("Fatal error inserting an event")
		panic(err)
	}

	t.log("Started: " + t.task.ID.String() + "\n")

	if err := t.installKey(t.repository.SshKey); err != nil {
		t.log("Failed installing ssh key for repository access: " + err.Error())
		t.fail()
		return
	}

	if err := t.updateRepository(); err != nil {
		t.log("Failed updating repository: " + err.Error())
		t.fail()
		return
	}

	if err := t.installInventory(); err != nil {
		t.log("Failed to install inventory: " + err.Error())
		t.fail()
		return
	}

	// todo: write environment

	if err := t.runPlaybook(); err != nil {
		t.log("Running playbook failed: " + err.Error())
		t.fail()
		return
	}

	t.task.Status = "success"
	t.updateStatus()
}

func (t *task) populateDetails() error {
	// get template
	tempCollection := database.MongoDb.C("project_templates")
	if err := tempCollection.FindId(t.task.TemplateID).One(&t.template); err != nil {
		return err
	}

	// get project users
	var users []struct {
		ID bson.ObjectId `db:"id"`
	}

	pUserc := database.MongoDb.C("project_users")

	if err := pUserc.FindId(t.template.ProjectID).Select(bson.M{"user_id": 1}).One(&users); err != nil {
		return err
	}

	t.users = []bson.ObjectId{}
	for _, user := range users {
		t.users = append(t.users, user.ID)
	}

	keyc := database.MongoDb.C("access_keys")
	// get access key
	if err := keyc.FindId(t.template.SshKeyID).One(&t.sshKey); err != nil {
		return err
	}

	if t.sshKey.Type != "ssh" {
		t.log("Non ssh-type keys are currently not supported: " + t.sshKey.Type)
		return errors.New("Unsupported SSH Key")
	}

	// get inventory
	projectInvc := database.MongoDb.C("project_inventories")

	if err := projectInvc.FindId(t.template.InventoryID).One(&t.inventory); err != nil {
		return err
	}

	// get inventory services key
	if bson.IsObjectIdHex(t.inventory.KeyID.String()) {

		accesskeyc := database.MongoDb.C("access_keys")
		if err := accesskeyc.FindId(t.inventory.KeyID).One(&t.inventory.Key); err != nil {
			return err
		}
	}

	// get inventory ssh key
	if bson.IsObjectIdHex(t.inventory.SshKeyID.String()) {
		accesskeyc := database.MongoDb.C("access_keys")
		if err := accesskeyc.FindId(t.inventory.SshKeyID).One(&t.inventory.SshKey); err != nil {
			return err
		}
	}

	// get repository
	projectRepoc := database.MongoDb.C("project_repositories")
	if err := projectRepoc.FindId(t.template.RepositoryID).One(&t.repository); err != nil {
		return err
	}

	// get repository access key
	accesskeyc := database.MongoDb.C("access_keys")
	if err := accesskeyc.FindId(t.repository.SshKeyID).One(&t.repository.SshKey); err != nil {
		return err
	}
	if t.repository.SshKey.Type != "ssh" {
		t.log("Repository Access Key is not 'SSH': " + t.repository.SshKey.Type)
		return errors.New("Unsupported SSH Key")
	}

	// get environment
	if len(t.task.Environment) == 0 && bson.IsObjectIdHex(t.template.EnvironmentID.String()) {

		projectenvc := database.MongoDb.C("project_environments")
		err := projectenvc.FindId(t.template.EnvironmentID).One(&t.environment)
		if err != nil {
			return err
		}
	} else if len(t.task.Environment) > 0 {
		t.environment.JSON = t.task.Environment
	}

	return nil
}

func (t *task) installKey(key models.AccessKey) error {
	t.log("access key " + key.Name + " installed")
	err := ioutil.WriteFile(key.GetPath(), []byte(key.Secret), 0600)

	return err
}

func (t *task) updateRepository() error {
	repoName := "repository_" + t.repository.ID.String()
	_, err := os.Stat(util.Config.TmpPath + "/" + repoName)

	cmd := exec.Command("git")
	cmd.Dir = util.Config.TmpPath
	cmd.Env = []string{
		"HOME=" + util.Config.TmpPath,
		"PWD=" + util.Config.TmpPath,
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -i " + t.repository.SshKey.GetPath(),
		// "GIT_FLUSH=1",
	}

	if err != nil && os.IsNotExist(err) {
		t.log("Cloning repository")
		cmd.Args = append(cmd.Args, "clone", t.repository.GitUrl, repoName)
	} else if err != nil {
		return err
	} else {
		t.log("Updating repository")
		cmd.Dir += "/" + repoName
		cmd.Args = append(cmd.Args, "pull", "origin", "master")
	}

	t.logCmd(cmd)
	return cmd.Run()
}

func (t *task) runPlaybook() error {
	playbookName := t.task.Playbook
	if len(playbookName) == 0 {
		playbookName = t.template.Playbook
	}

	args := []string{
		"-i", util.Config.TmpPath + "/inventory_" + t.task.ID.String(),
	}

	if bson.IsObjectIdHex(t.inventory.SshKeyID.String()) {
		args = append(args, "--private-key="+t.inventory.SshKey.GetPath())
	}

	if t.task.Debug {
		args = append(args, "-vvvv")
	}

	if len(t.environment.JSON) > 0 {
		args = append(args, "--extra-vars", t.environment.JSON)
	}

	var extraArgs []string
	if len(t.template.Arguments) > 0 {
		err := json.Unmarshal([]byte(t.template.Arguments), &extraArgs)
		if err != nil {
			t.log("Could not unmarshal arguments to []string")
			return err
		}
	}

	if t.template.OverrideArguments {
		args = extraArgs
	} else {
		args = append(args, extraArgs...)
		args = append(args, playbookName)
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = util.Config.TmpPath + "/repository_" + t.repository.ID.String()
	cmd.Env = []string{
		"HOME=" + util.Config.TmpPath,
		"PWD=" + cmd.Dir,
		"PYTHONUNBUFFERED=1",
		// "GIT_FLUSH=1",
	}

	t.logCmd(cmd)
	return cmd.Run()
}
