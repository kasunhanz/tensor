package addhoctasks

import (
	"time"
	"github.com/gamunu/hilbertspace/models"
	"github.com/gin-gonic/gin"
	database "github.com/gamunu/hilbertspace/db"
	"github.com/gamunu/hilbertspace/util"
)

func AddTask(c *gin.Context) {

	var taskObj models.AddHocTask
	if err := c.Bind(&taskObj); err != nil {
		c.Error(err)
		return
	}

	taskObj.Created = time.Now()
	taskObj.Status = "waiting"

	if err := taskObj.AddHocTaskInsert(); err != nil {
		panic(err)
	}

	pool.register <- &task{
		task:      taskObj,
	}

	objType := "addhoctask"
	desc := "Add-Hoc Task ID " + taskObj.ID.String() + " queued for running"
	if err := (models.Event{
		ObjectType:  objType,
		ObjectID:    taskObj.ID,
		Description: desc,
	}.Insert()); err != nil {
		panic(err)
	}

	c.JSON(201, taskObj)
}

func GetTaskMiddleware(c *gin.Context) {
	taskID, err := util.GetIntParam("task_id", c)
	if ( err != nil) {
		panic(err)
	}

	var task models.AddHocTask
	if err := database.Mysql.SelectOne(&task, "select * from addhoc_task where id=?", taskID); err != nil {
		panic(err)
	}

	c.Set("addhoctask", task)
	c.Next()
}

func GetTaskOutput(c *gin.Context) {
	task := c.MustGet("addhoctask").(models.AddHocTask)
	var output []models.AddHocTaskOutput
	if _, err := database.Mysql.Select(&output, "select * from addhoc_task__output where task_id=? order by time asc", task.ID); err != nil {
		panic(err)
	}

	c.JSON(200, output)
}