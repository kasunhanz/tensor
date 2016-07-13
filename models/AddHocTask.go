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
	//if no access key then it's a local connection
	//for now we will allow local connection type
	AccessKeyID bson.ObjectId `bson:"access_key_id,omitempty" json:"access_key_id,omitempty"`

	Status      string `bson:"status" json:"status"`
	Debug       bool   `bson:"debug" json:"debug"`

	Module      string `bson:"module" json:"module"`
	Arguments   string `bson:"arguments" json:"arguments"`
	ExtraVars   string `bson:"extra_vars" json:"extra_vars"`
	Forks       int    `bson:"forks" json:"forks"`
	Inventory   []string `bson:"inventory" json:"inventory"`
	Connection  string `bson:"connection" json:"connection"`
	Timeout     int    `bson:"timeout" json:"timeout"`

	//task status without log
	Log         []TaskLogItem    `bson:"log" json:"log,omitempty"`

	Created     time.Time  `bson:"created" json:"created"`
	Start       time.Time `bson:"start" json:"start"`
	End         time.Time `bson:"end" json:"end"`
}

type TaskLogItem struct {
	Record string `bson:"record" json:"record"`
	Time   time.Time `bson:"time" json:"time"`
}


func (task AddHocTask) Insert() error {
	c := database.MongoDb.C("addhoc_task")
	return c.Insert(task)
}