package addhoctasks

import (
	"bufio"
	"fmt"
	"os/exec"
	"time"

	database "pearson.com/hilbert-space/db"
	"gopkg.in/mgo.v2/bson"
	"pearson.com/hilbert-space/models"
)

func (t *task) log(msg string) {
	//TODO: send event

	go func() {

		c := database.MongoDb.C("addhoc_task")

		if _, err := c.Upsert(bson.M{"_id": t.task.ID},
			bson.M{
				"$push": bson.M{"log": models.TaskLogItem{Record:msg, Time:time.Now()}},
			}); err != nil {
			panic(err)
		}
	}()
}

func (t *task) updateStatus() {

	c := database.MongoDb.C("addhoc_task")

	if err := c.UpdateId(t.task.ID, bson.M{"$set":
	bson.M{
		"status":t.task.Status,
		"start":t.task.Start,
		"end":t.task.End,
	},
	}); err != nil {
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