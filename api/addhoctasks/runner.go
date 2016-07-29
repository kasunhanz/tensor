package addhoctasks

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"errors"
	"github.com/gamunu/hilbert-space/crypt"
	database "github.com/gamunu/hilbert-space/db"
	"github.com/gamunu/hilbert-space/models"
	"github.com/gamunu/hilbert-space/util"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type task struct {
	task      models.AddHocTask
	accessKey models.GlobalAccessKey
	//for future objects
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
			ID:          bson.NewObjectId(),
			ObjectType:  "addhoc_task",
			ObjectID:    t.task.ID,
			Description: "Add-Hoc Task ID " + t.task.ID.Hex() + " finished",
		}.Insert()); err != nil {
			log.Print(err)
		}
	}()

	if err := t.populateDetails(); err != nil {
		t.log("Error: "+err.Error(), models.TaskLogError)
		t.fail()
		return
	}

	now := time.Now()
	t.task.Status = "running"
	t.task.Start = now
	t.updateStatus()

	if err := (models.Event{
		ID:          bson.NewObjectId(),
		ObjectType:  "addhoc_task",
		ObjectID:    t.task.ID,
		Description: "Add-Hoc Task ID " + t.task.ID.Hex() + " is running",
		Created:     time.Now(),
	}.Insert()); err != nil {
		log.Print(err)
	}

	t.log("Started: "+t.task.ID.Hex(), models.TaskLogInfo)

	if t.accessKey.Type != "credential" {
		if err := t.installKey(t.accessKey); err != nil {
			t.log("Failed installing access key for server access: "+err.Error(), models.TaskLogError)
			t.fail()
			return
		}
	}

	if err := t.runAnsible(); err != nil {
		t.log("Running ansible failed: "+err.Error(), models.TaskLogError)
		t.fail()
		return
	}

	t.task.Status = "success"
	t.updateStatus()
}

func (t *task) populateDetails() error {

	// get access key
	if bson.IsObjectIdHex(t.task.AccessKeyID.Hex()) {
		accesskeyc := database.MongoDb.C("global_access_key")
		if err := accesskeyc.FindId(t.task.AccessKeyID).One(&t.accessKey); err != nil {
			t.log("Global Access Key not found!", models.TaskLogError)
			return errors.New("Global Access Key not found!")
		}

		if t.accessKey.Type != "ssh" && t.accessKey.Type != "credential" {
			t.log("Only ssh and credentials currently supported: "+t.accessKey.Type, models.TaskLogError)
			return errors.New("Unsupported Key")
		}
	}

	return nil
}

func (t *task) installKey(key models.GlobalAccessKey) error {
	t.log("Global access key "+key.Name+" installed", models.TaskLogInfo)
	err := ioutil.WriteFile(key.GetPath(), []byte(key.Secret), 0600)

	return err
}

// runAnsible is executes the task using Ansible command
func (t *task) runAnsible() error {

	// arguments for Ansible command
	args := []string{"all"}

	// specify inventory, comma separated host list
	if cap(t.task.Inventory) > 0 {
		args = append(args, "-i", strings.Join(t.task.Inventory, ",")+",")

	}

	if len(t.task.Module) > 0 {
		args = append(args, "-m", t.task.Module)
	} else {
		return errors.New("No argument passed to command module")
	}

	if len(t.task.Arguments) > 0 {
		args = append(args, "-a", t.task.Arguments)
	}

	if t.task.Forks > 0 {
		args = append(args, "-f", strconv.Itoa(t.task.Forks))
	}

	// don't make any changes; instead, try to predict some
	// of the changes that may occur
	if t.task.Check {
		args = append(args, "-C")
	}

	// when changing (small) files and templates, show the
	// differences in those files; works great with --check
	if t.task.Diff {
		args = append(args, "-D")
	}

	// connection type to use (default=smart)
	if len(t.task.Connection) > 0 {
		if t.task.Connection == "winrm" {
			args = append(args, "-e", "")
		}
		args = append(args, "-c", t.task.Connection)
	}

	// --extra-vars argument values
	extraVars := make(map[string]interface{})

	if len(t.task.ExtraVars) > 0 {
		if err := json.Unmarshal([]byte(t.task.ExtraVars), &extraVars); err != nil {
			return errors.New("Could not unmarshal ExtraVars, data invalid!")
		}
	}

	if t.accessKey.Type == "credential" {
		args = append(args, "-u", t.accessKey.Key)
		//add ssh password as an extra argument
		args = append(args, "-e", "ansible_ssh_pass", crypt.Decrypt(t.accessKey.Secret))
	} else if t.accessKey.Type == "ssh" {
		args = append(args, "--private-key="+t.accessKey.GetPath())
	}

	// verbose mode -nasiblevvvv to enable
	// connection debugging)
	if t.task.Debug {
		args = append(args, "-vvvv")
	}

	// run operations with become (does not imply password
	// prompting)
	if t.task.Become {
		args = append(args, "-b")

		// privilege escalation method to use (default=sudo),
		// valid choices: [ sudo | su | pbrun | pfexec | runas |
		// doas | dzdo ]
		if len(t.task.BecomeMethod) > 0 {
			args = append(args, t.task.BecomeMethod)
		}

		// run operations as this user (default=root)
		if len(t.task.BecomeUser) > 0 {
			args = append(args, t.task.BecomeUser)
		}
	}

	if len(extraVars) > 0 {
		marshalVars, err := json.Marshal(extraVars)
		if err != nil {
			return errors.New("Could not marshal arguments to json string")
		}
		args = append(args, "-e", string(marshalVars))
	}

	cmd := exec.Command("ansible", args...)
	cmd.Dir = util.Config.TmpPath

	// This is must for Ansible
	env := os.Environ()
	env = append(env, "HOME="+util.Config.TmpPath, "PWD="+cmd.Dir, "HS_TASK_ID="+t.task.ID.Hex())
	cmd.Env = env

	t.logCmd(cmd)
	return cmd.Run()
}
