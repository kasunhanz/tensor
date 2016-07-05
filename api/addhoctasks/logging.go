package addhoctasks

import (
	"bufio"
	"fmt"
	"os/exec"
	"time"

	database "pearson.com/hilbert-space/db"
	"pearson.com/hilbert-space/models"
)

func (t *task) log(msg string) {
	now := time.Now()

	//TODO: send event

	go func() {

		taskOutput := models.AddHocTaskOutput{
			TaskID: t.task.ID,
			Output: msg,
			Time:now,
		}
		err := taskOutput.AddHocTaskOutputInsert()
		if err != nil {
			panic(err)
		}
	}()
}

func (t *task) updateStatus() {

	c := database.MongoDb.C("addhoc_task")

	if err := c.UpdateId(t.task.ID, models.AddHocTask{
		Status:t.task.Status,
		Start:t.task.Start,
		End:t.task.End,
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
