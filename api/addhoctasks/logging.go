package addhoctasks

import (
	"bufio"
	"fmt"
	"os/exec"
	"time"

	database "github.com/gamunu/hilbertspace/db"
)


func (t *task) log(msg string) {
	now := time.Now()

	//TODO: send event

	go func() {
		_, err := database.Mysql.Exec("insert into addhoc_task__output set task_id=?, output=?, time=?", t.task.ID, msg, now)
		if err != nil {
			panic(err)
		}
	}()
}

func (t *task) updateStatus() {
	if _, err := database.Mysql.Exec("update addhoc_task set status=?, start=?, end=? where id=?", t.task.Status, t.task.Start, t.task.End, t.task.ID); err != nil {
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
