package addhoctasks

import (
	database "pearson.com/tensor/db"
	"pearson.com/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"time"
)

// AddTask creates a new add-hoc task
// It validates post task object and ads new task to database and to task pool
func AddTask(c *gin.Context) {

	var taskObj models.AddHocTask

	// bind request JSON with model
	if err := c.BindJSON(&taskObj); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	taskObj.ID = bson.NewObjectId()
	taskObj.Created = time.Now()
	taskObj.Status = "waiting"

	// Insert the task object to database
	if err := taskObj.Insert(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// add newly created task in to addHocTaskPool
	pool.register <- &task{
		task: taskObj,
	}

	// Create new event ins the database
	if err := (models.Event{
		ID:          bson.NewObjectId(),
		ObjectType:  "addhoc_task",
		ObjectID:    taskObj.ID,
		Description: "Add-Hoc Task ID " + taskObj.ID.Hex() + " queued for running",
		Created:     time.Now(),
	}.Insert()); err != nil {
		// We don't inform client about this error
		// do not ever panic :D
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, taskObj)
}

// GetTaskMiddleware takes task_id parameter and
// fetches task data from the database
// it set task data under key addhoc_task in gin.Context
func GetTaskMiddleware(c *gin.Context) {
	taskID := c.Params.ByName("task_id")

	var task models.AddHocTask

	col := database.MongoDb.C("addhoc_tasks")

	if err := col.FindId(bson.ObjectIdHex(taskID)).One(&task); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Set("addhoc_task", task)
	c.Next()
}

// GetTaskWithoutLogMiddleware takes task_id parameter from and
// fetches task data from the database without log data
// it set task data under key addhoc_task_wlog in gin.Context
func GetTaskWithoutLogMiddleware(c *gin.Context) {
	taskID := c.Params.ByName("task_id")
	var task models.AddHocTask

	col := database.MongoDb.C("addhoc_tasks")

	if err := col.FindId(bson.ObjectIdHex(taskID)).Select(bson.M{"log": 0}).One(&task); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Set("addhoc_task_wlog", task)

	c.Next() //process next handler
}

// GetTaskWithoutLog takes addhoc_task_wlog from gin.Context
// which added by GetTaskWithoutLogMiddleware
// and returns task formatted as JSON
func GetTaskWithoutLog(c *gin.Context) {
	task := c.MustGet("addhoc_task_wlog").(models.AddHocTask)

	c.JSON(http.StatusOK, task)
}

// GetTaskOutput takes addhoc_task from gin.Context
// which added by GetTaskMiddleware and
// returns task.Log as formatted as JSON
func GetTaskOutput(c *gin.Context) {
	task := c.MustGet("addhoc_task").(models.AddHocTask)

	c.JSON(http.StatusOK, task.Log)
}
