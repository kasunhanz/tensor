package addhoctasks

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	database "pearson.com/hilbert-space/db"
	"pearson.com/hilbert-space/models"
	"strings"
	"errors"
	"io/ioutil"
	"os/exec"
	"pearson.com/hilbert-space/util"
	"pearson.com/hilbert-space/crypt"
	"gopkg.in/mgo.v2/bson"
	"os"
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
			ID: bson.NewObjectId(),
			ObjectType:  "addhoc_task",
			ObjectID:    t.task.ID,
			Description: "Add-Hoc Task ID " + t.task.ID.Hex() + " finished",
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

	now := time.Now()
	t.task.Status = "running"
	t.task.Start = now
	t.updateStatus()

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  "addhoc_task",
		ObjectID:    t.task.ID,
		Description: "Add-Hoc Task ID " + t.task.ID.Hex() + " is running",
		Created: time.Now(),
	}.Insert()); err != nil {
		t.log("Fatal error inserting an event")
		panic(err)
	}

	t.log("Started: " + t.task.ID.Hex() + "\n")

	if t.accessKey.Type != "credential" {
		if err := t.installKey(t.accessKey); err != nil {
			t.log("Failed installing access key for server access: " + err.Error())
			t.fail()
			return
		}
	}

	if err := t.runAnsible(); err != nil {
		t.log("Running ansible failed: " + err.Error())
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
			return errors.New("Global Access Key not found!")
		}

		if t.accessKey.Type != "ssh" && t.accessKey.Type != "credential" {
			t.log("Only ssh and credentials currently supported: " + t.accessKey.Type)
			return errors.New("Unsupported Key")
		}
	}

	return nil
}

func (t *task) installKey(key models.GlobalAccessKey) error {
	t.log("Global access key " + key.Name + " installed")
	err := ioutil.WriteFile(key.GetPath(), []byte(key.Secret), 0600)

	return err
}

// runAnsible is executes the task using Ansible command
func (t *task) runAnsible() error {

	// arguments for Ansible command
	args := []string{
		"all",
	}

	if cap(t.task.Inventory) > 0 {
		args = append(args, "-i", strings.Join(t.task.Inventory, ",") + ",")

	} else {
		t.log("No argument passed to inventory")
		return errors.New("No argument passed to inventory")
	}

	if len(t.task.Module) > 0 {
		args = append(args, "-m", t.task.Module)
	} else {
		t.log("No argument passed to command module")
		return errors.New("No argument passed to command module")
	}

	if len(t.task.Arguments) > 0 {
		args = append(args, "-a", t.task.Arguments)
	}

	if t.task.Forks > 0 {
		args = append(args, "-f", strconv.Itoa(t.task.Forks))
	}

	if len(t.task.Connection) > 0 {
		if (t.task.Connection == "winrm") {
			t.log("Windows hosts are not currently supported")
			return errors.New("Windows hosts are not currently supported")
		}
		args = append(args, "-c", t.task.Connection)
	}

	// --extra-vars argument values
	extraVars := make(map[string]interface{})

	if len(t.task.ExtraVars) > 0 {
		if err := json.Unmarshal([]byte(t.task.ExtraVars), &extraVars); err != nil {
			t.log("Could not unmarshal ExtraVars, data invalid!")
			return err
		}
	}

	if t.accessKey.Type == "credential" {
		args = append(args, "-u", t.accessKey.Key)
		//add ssh password as an extra argument
		extraVars["ansible_ssh_pass"] = crypt.Decrypt(t.accessKey.Secret)
	} else if t.accessKey.Type == "ssh" {
		args = append(args, "--private-key=" + t.accessKey.GetPath())

	}

	if t.task.Debug {
		args = append(args, "-vvvv")
	}

	if len(extraVars) > 0 {
		marshalVars, err := json.Marshal(extraVars)
		if err != nil {
			t.log("Could not marshal arguments to json string")
			return err
		}
		args = append(args, "--extra-vars", string(marshalVars))
	}

	cmd := exec.Command("ansible", args...)
	cmd.Dir = util.Config.TmpPath

	// This is must for ansible
	env := os.Environ()

	env = append(env, "HOME=" + util.Config.TmpPath, "PWD=" + cmd.Dir)
	cmd.Env = env

	t.logCmd(cmd)
	return cmd.Run()
}
