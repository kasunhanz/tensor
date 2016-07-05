package tasks

import (
	"time"

	database "github.com/gamunu/hilbertspace/db"
	"github.com/gamunu/hilbertspace/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func AddTask(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	var taskObj models.Task
	if err := c.Bind(&taskObj); err != nil {
		return
	}

	taskObj.Created = time.Now()
	taskObj.Status = "waiting"

	if err := taskObj.Insert(); err != nil {
		panic(err)
	}

	pool.register <- &task{
		task:      taskObj,
		projectID: project.ID,
	}

	if err := (models.Event{
		ProjectID:   project.ID,
		ObjectType:  "task",
		ObjectID:    taskObj.ID,
		Description: "Task ID " + taskObj.ID + " queued for running",
	}.Insert()); err != nil {
		panic(err)
	}

	c.JSON(201, taskObj)
}

func GetAll(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	col := database.MongoDb.C("task")

	aggrigrate := []bson.M{
		{"$lookup" : bson.M{
			"from":"project_template",
			"localField":"template_id",
			"foreignField":"_id",
			"as": "project_template",
		}},
		{"$match": bson.M{
			"$project_template._id":project.ID,
		}},
		{"$sort": bson.M{
			"created":-1,
		}},
	}

	var tasks []struct {
		models.Task

		TemplatePlaybook string `bson:"tpl_playbook" json:"tpl_playbook"`
	}

	if err := col.Pipe(aggrigrate).All(&tasks); err != nil {
		panic(err)
	}

	c.JSON(200, tasks)
}

func GetTaskMiddleware(c *gin.Context) {
	taskID := c.Params.ByName("task_id")

	var task models.Task
	task, err := task.GetTask(taskID);
	if err != nil {
		panic(err)
	}

	c.Set("task", task)
	c.Next()
}

func GetTaskOutput(c *gin.Context) {
	task := c.MustGet("task").(models.Task)

	output, err := task.GetTaskOutput();
	if err != nil {
		panic(err)
	}

	c.JSON(200, output)
}
