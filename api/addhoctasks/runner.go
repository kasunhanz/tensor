package addhoctasks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	database "github.com/gamunu/hilbertspace/db"
	"github.com/gamunu/hilbertspace/models"
	"strings"
	"errors"
	"io/ioutil"
	"os/exec"
	"github.com/gamunu/hilbertspace/util"
	"github.com/gamunu/hilbertspace/crypt"
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
		t.task.End = &now
		t.updateStatus()

		objType := "addhoc_task"
		desc := "Add-Hoc Task ID " + strconv.Itoa(t.task.ID) + " finished"
		if err := (models.Event{
			ObjectType:  &objType,
			ObjectID:    &t.task.ID,
			Description: &desc,
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

	{
		now := time.Now()
		t.task.Status = "running"
		t.task.Start = &now
		t.updateStatus()
	}

	objType := "addhoc_task"
	desc := "Add-Hoc Task ID " + strconv.Itoa(t.task.ID) + " is running"
	if err := (models.Event{
		ObjectType:  &objType,
		ObjectID:    &t.task.ID,
		Description: &desc,
	}.Insert()); err != nil {
		t.log("Fatal error inserting an event")
		panic(err)
	}

	t.log("Started: " + strconv.Itoa(t.task.ID) + "\n")

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

func (t *task) fetch(errMsg string, ptr interface{}, query string, args ...interface{}) error {
	err := database.Mysql.SelectOne(ptr, query, args...)
	if err == sql.ErrNoRows {
		t.log(errMsg)
		return err
	}

	if err != nil {
		t.fail()
		panic(err)
	}

	return nil
}

func (t *task) populateDetails() error {

	// get access key
	if err := t.fetch("Template Access Key not found!", &t.accessKey, "select * from global_access_key where id=?", t.task.AccessKeyID); err != nil {
		return err
	}

	if t.accessKey.Type != "ssh" {
		t.log("Non ssh-type keys are currently not supported: " + t.accessKey.Type)
		return errors.New("Unsupported SSH Key")
	}

	return nil
}

func (t *task) installKey(key models.GlobalAccessKey) error {
	t.log("Global access key " + key.Name + " installed")
	err := ioutil.WriteFile(key.GetPath(), []byte(*key.Secret), 0600)

	return err
}

func (t *task) runAnsible() error {

	// arguments for ansible command
	args := []string{
		"all",
	}

	// --extra-vars argument values
	var extraVars map[string]interface{}

	// inventory
	var inventory []string

	if len(t.task.Inventory) > 0 {
		err := json.Unmarshal([]byte(t.task.Inventory), &inventory)

		if err != nil {
			t.log("Could not unmarshal inventory to []string")
			return err
		}
	}

	args = append(args, "-i", strings.Join(inventory, ",") + ",")

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

	if len(t.task.ExtraVars) > 0 {
		err := json.Unmarshal([]byte(t.task.ExtraVars), &extraVars)
		if err != nil {
			t.log("Could not unmarshal arguments to map[string]interface{}")
			return err
		}
	}

	if t.accessKey.Type == "credential" {
		args = append(args, "-u", *t.accessKey.Key)
		//add ssh password as an extra argument
		extraVars["ansible_ssh_pass"] = crypt.Decrypt(*t.accessKey.Secret)
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
		args = append(args, "--extra-vars=", string(marshalVars))
	}

	cmd := exec.Command("ansible", args...)
	cmd.Env = []string{
		"HOME=" + util.Config.TmpPath,
		"PWD=" + util.Config.TmpPath,
		"PYTHONUNBUFFERED=1",
	}

	t.logCmd(cmd)
	return cmd.Run()
}
