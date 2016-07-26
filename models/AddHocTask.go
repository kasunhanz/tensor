package models

import (
	"time"
	database "pearson.com/hilbert-space/db"
	"gopkg.in/mgo.v2/bson"
)

const (
	TaskLogError = "error"
	TaskLogInfo = "info"
)
// AddHocTask is an exported type that
// is used as database model for record
// add hoc command execution details
type AddHocTask struct {
	ID           bson.ObjectId `bson:"_id" json:"id"`
	AccessKeyID  bson.ObjectId `bson:"access_key_id" json:"access_key_id" binding:"required"`

	Status       string `bson:"status" json:"status" binding:"omitempty"`
	Debug        bool   `bson:"debug" json:"debug" binding:"omitempty"`

	Module       string `bson:"module" json:"module" binding:"required" binding:"alpha,required"`
	Arguments    string `bson:"arguments,omitempty" json:"arguments,omitempty"`
	//become related options
	Become       bool `bson:"become,omitempty" json:"become,omitempty"`
	BecomeMethod string `bson:"become_method,omitempty" json:"become_method,omitempty" binding:"omitempty,ansible_becomemethod"`
	BecomeUser   string `bson:"become_user,omitempty" json:"become_user,omitempty" binding:"omitempty,alpha"`

	Check        bool `bson:"check,omitempty" json:"check,omitempty"`
	Diff         bool `bson:"diff,omitempty" json:"diff,omitempty"`

	ExtraVars    string `bson:"extra_vars,omitempty" json:"extra_vars,omitempty"`
	Forks        int    `bson:"forks,omitempty" json:"forks,omitempty" binding:"omitempty,gt=0,numeric"`
	Inventory    []string `bson:"inventory" json:"inventory" binding:"gt=0,dive,ip|domain_server,required"`
	Connection   string `bson:"connection,omitempty" json:"connection,omitempty" binding:"omitempty,alpha"`
	Timeout      int    `bson:"timeout,omitempty" json:"timeout,omitempty" binding:"omitempty,gt=0,numeric"`

	// task status without log
	// JSON omit (-) is a must, otherwise users will be able to inject log items
	Log          []TaskLogItem    `bson:"log" json:"-"`

	Created      time.Time  `bson:"created" json:"created"`
	Start        time.Time `bson:"start" json:"start"`
	End          time.Time `bson:"end" json:"end"`
}

// TaskLogItem is an exported type that
// is used as database model for record
// AddHocTask log items
type TaskLogItem struct {
	Record string `bson:"record" json:"record"`
	Type   string `bson:"type" json:"type"`
	Time   time.Time `bson:"time" json:"time"`
}

// Insert inserts a document in to the addhoc_task collection.  In
// case the session is in safe mode (see the SetSafe method) and an error
// happens while inserting the provided documents, the returned error will
// be of type *LastError.
func (task AddHocTask) Insert() error {
	c := database.MongoDb.C("addhoc_task")
	return c.Insert(task)
}