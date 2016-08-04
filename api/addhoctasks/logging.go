package addhoctasks

import (
	"bufio"
	"fmt"
	"os/exec"
	"time"

	database "github.com/gamunu/tensor/db"
	"github.com/gamunu/tensor/models"
	"gopkg.in/mgo.v2/bson"
)

func (t *task) log(msg string, logType string) {
	//TODO: send event

	go func() {

		c := database.MongoDb.C("addhoc_tasks")

		if _, err := c.Upsert(bson.M{"_id": t.task.ID},
			bson.M{
				"$push": bson.M{"log": models.TaskLogItem{Record: msg, Type: logType, Time: time.Now()}},
			}); err != nil {
			panic(err)
		}
	}()
}

func (t *task) updateStatus() {

	c := database.MongoDb.C("addhoc_tasks")

	if err := c.UpdateId(t.task.ID, bson.M{"$set": bson.M{
		"status": t.task.Status,
		"start":  t.task.Start,
		"end":    t.task.End,
	},
	}); err != nil {
		fmt.Println("Failed to update task status")
		t.log("Fatal error with database!", models.TaskLogError)
		panic(err)
	}
}

func (t *task) logPipe(scanner *bufio.Scanner, logType string) {
	for scanner.Scan() {
		t.log(scanner.Text(), logType)
	}
}

func (t *task) logCmd(cmd *exec.Cmd) {

	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	go t.logPipe(bufio.NewScanner(stderr), models.TaskLogError)
	go t.logPipe(bufio.NewScanner(stdout), models.TaskLogInfo)
}
