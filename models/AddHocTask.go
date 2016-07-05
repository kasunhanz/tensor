package models

import (
	"time"
	database "pearson.com/hilbert-space/db"
	"gopkg.in/mgo.v2/bson"
)

// AddHocTask is an exported type that
// is used as database model for record
// add hoc command execution details
type AddHocTask struct {
	ID          bson.ObjectId `bson:"_id" json:"id"`
	AccessKeyID int `bson:"access_key_id" json:"access_key_id"`

	Status      string `bson:"status" json:"status"`
	Debug       bool   `bson:"debug" json:"debug"`

	Module      string `bson:"module" json:"module"`
	Arguments   string `bson:"arguments" json:"arguments"`
	ExtraVars   string `bson:"extra_vars" json:"extra_vars"`
	Forks       int    `bson:"forks" json:"forks"`
	Inventory   []string `bson:"inventory" json:"inventory"`
	Connection  string `bson:"connection" json:"connection"`
	Timeout     int    `bson:"timeout" json:"timeout"`

	Created     time.Time  `bson:"created" json:"created"`
	Start       time.Time `bson:"start" json:"start"`
	End         time.Time `bson:"end" json:"end"`
}

// AddHocTaskOutput is an exported type that
// is used as database model for record
// add command database output
type AddHocTaskOutput struct {
	ID     bson.ObjectId  `bson:"_id" json:"id"`
	TaskID bson.ObjectId        `bson:"task_id" json:"task_id"`
	Task   string    `db:"task" json:"task"`
	Time   time.Time `db:"time" json:"time"`
	Output string    `db:"output" json:"output"`
}

func (task AddHocTask) AddHocTaskInsert() error {
	c := database.MongoDb.C("addhoc_task")
	err := c.Insert(task)

	return err
}

func (task AddHocTaskOutput) AddHocTaskOutputInsert() error {
	c := database.MongoDb.C("addhoc_task__output")
	err := c.Insert(task)

	return err
}
