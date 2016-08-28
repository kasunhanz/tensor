package tasks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"bitbucket.pearson.com/apseng/tensor/api/sockets"
	"bitbucket.pearson.com/apseng/tensor/models"
	"gopkg.in/mgo.v2/bson"
)

func (t *task) log(msg string) {
	now := time.Now()

	for _, user := range t.users {
		b, err := json.Marshal(&map[string]interface{}{
			"type":       "log",
			"output":     msg,
			"time":       now,
			"task_id":    t.task.ID,
			"project_id": t.projectID,
		})

		if err != nil {
			panic(err)
		}

		sockets.Message(user, b)
	}

	go func() {

		if err := (models.TaskOutput{
			ID:     bson.NewObjectId(),
			Output: msg,
			Time:   now,
			TaskID: t.task.ID,
		}.Insert()); err != nil {
			panic(err)
		}
	}()
}

func (t *task) updateStatus() {
	for _, user := range t.users {
		b, err := json.Marshal(&map[string]interface{}{
			"type":       "update",
			"start":      t.task.Start,
			"end":        t.task.End,
			"status":     t.task.Status,
			"task_id":    t.task.ID,
			"project_id": t.projectID,
		})

		if err != nil {
			panic(err)
		}

		sockets.Message(user, b)
	}

	if err := t.task.Update(); err != nil {
		fmt.Println("Failed to update task status")
		t.log("Fatal error with database!")
		panic(err)
	}
}

func (t *task) logPipe(scanner *bufio.Scanner) {
	for scanner.Scan() {
		t.log(scanner.Text())
	}
}

func (t *task) logCmd(cmd *exec.Cmd) {
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	go t.logPipe(bufio.NewScanner(stderr))
	go t.logPipe(bufio.NewScanner(stdout))
}
