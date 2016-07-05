package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/ansible-semaphore/semaphore/models"
	database "github.com/gamunu/hilbertspace/db"
)

// Task is the model for project_task
// collection
type Task struct {
	ID          bson.ObjectId `bson:"_id" json:"id"`
	TemplateID  bson.ObjectId `bson:"template_id" json:"template_id" binding:"required"`

	Status      string `bson:"status" json:"status"`
	Debug       bool   `bson:"debug" json:"debug"`

	// override variables
	Playbook    string `bson:"playbook" json:"playbook"`
	Environment string `bson:"environment" json:"environment"`

	Created     time.Time  `bson:"created" json:"created"`
	Start       time.Time `bson:"start" json:"start"`
	End         time.Time `bson:"end" json:"end"`
}

// TaskOutput is the model for project_task_output
// collection
type TaskOutput struct {
	ID     bson.ObjectId `bson:"_id" json:"id"`
	TaskID bson.ObjectId       `bson:"task_id" json:"task_id"`
	Task   string    `bson:"task" json:"task"`
	Time   time.Time `bson:"time" json:"time"`
	Output string    `bson:"output" json:"output"`
}

// GetRepositories is returns output of a
// returns the Task.Output array and error returned by mongo driver
func (task Task) GetTaskOutput() ([]models.TaskOutput, error) {
	c := database.MongoDb.C("project_repository")

	var tasks []models.TaskOutput
	err := c.Find(bson.M{"task_id": task.ID, }).Sort("time").All(tasks)

	return tasks, err
}

// Gettask returns the inventory associated with the project
// invId parameter required
// inv parameter need to be reference
func (task Task) GetTask(taksId bson.ObjectId) (models.Task, error) {
	c := database.MongoDb.C("project_template")

	var tpl models.Template
	err := c.Find(bson.M{"_id": taksId}).One(tpl)

	return tpl, err
}

// Create a new project
func (task Task) Insert() error {
	c := database.MongoDb.C("task")
	return c.Insert(task)
}

func (task Task) Update() error {
	c := database.MongoDb.C("task")
	return c.UpdateId(task.ID, task)
}

func (tskOutput TaskOutput) Insert() error {
	c := database.MongoDb.C("task_output")
	return c.Insert(tskOutput)
}
