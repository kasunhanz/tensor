package addhoctasks

import (
	"time"
	"pearson.com/hilbert-space/models"
	"github.com/gin-gonic/gin"
	database "pearson.com/hilbert-space/db"
	"pearson.com/hilbert-space/util"
	"gopkg.in/mgo.v2/bson"
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

	col := database.MongoDb.C("addhoc_task")

	if err := col.FindId(taskID).One(&task); err != nil {
		panic(err)
	}

	c.Set("addhoctask", task)
	c.Next()
}

func GetTaskOutput(c *gin.Context) {
	task := c.MustGet("addhoctask").(models.AddHocTask)
	var output []models.AddHocTaskOutput

	col := database.MongoDb.C("addhoc_task_output")

	if err := col.Find(bson.M{"task_id": task.ID}).Sort("time").All(&output); err != nil {
		panic(err)
	}

	c.JSON(200, output)
}