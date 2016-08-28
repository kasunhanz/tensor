package models

import (
	database "pearson.com/tensor/db"
	"gopkg.in/mgo.v2/bson"
	"time"
)

// Task is the model for project_task
// collection
type Task struct {
	ID         bson.ObjectId `bson:"_id" json:"id"`
	TemplateID bson.ObjectId `bson:"template_id" json:"template_id" binding:"required"`

	Status string `bson:"status" json:"status"`
	Debug  bool   `bson:"debug" json:"debug"`

	// override variables
	Playbook    string `bson:"playbook" json:"playbook"`
	Environment string `bson:"environment" json:"environment"`

	Created time.Time `bson:"created" json:"created"`
	Start   time.Time `bson:"start" json:"start"`
	End     time.Time `bson:"end" json:"end"`
}

// TaskOutput is the model for project_task_output
// collection
type TaskOutput struct {
	ID     bson.ObjectId `bson:"_id" json:"id"`
	TaskID bson.ObjectId `bson:"task_id" json:"task_id"`
	Task   string        `bson:"task" json:"task"`
	Time   time.Time     `bson:"time" json:"time"`
	Output string        `bson:"output" json:"output"`
}

// Create a new project
func (task Task) Insert() error {
	c := database.MongoDb.C("tasks")
	return c.Insert(task)
}

func (task Task) Update() error {
	c := database.MongoDb.C("tasks")
	return c.UpdateId(task.ID, task)
}

func (tskOutput TaskOutput) Insert() error {
	c := database.MongoDb.C("task_outputs")
	return c.Insert(tskOutput)
}
