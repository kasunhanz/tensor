package models

import (
	database "github.com/gamunu/hilbert-space/db"
	"gopkg.in/mgo.v2/bson"
)

// Template is the model for project_template
// collection
type Template struct {
	ID bson.ObjectId `bson:"_id" json:"id"`

	SshKeyID      bson.ObjectId `bson:"ssh_key_id" json:"ssh_key_id"`
	ProjectID     bson.ObjectId `bson:"project_id" json:"project_id"`
	InventoryID   bson.ObjectId `bson:"inventory_id" json:"inventory_id"`
	RepositoryID  bson.ObjectId `bson:"repository_id" json:"repository_id"`
	EnvironmentID bson.ObjectId `bson:"environment_id" json:"environment_id"`

	// playbook name in the form of "some_play.yml"
	Playbook string `bson:"playbook" json:"playbook"`
	// to fit into []string
	Arguments string `bson:"arguments" json:"arguments"`
	// if true, hilbertspace will not prepend any arguments to `arguments` like inventory, etc
	OverrideArguments bool `bson:"override_args" json:"override_args"`
}

// TemplateSchedule is the model for project_template_schedule
// collection
type TemplateSchedule struct {
	TemplateID bson.ObjectId `bson:"template_id" json:"template_id"`
	CronFormat string        `bson:"cron_format" json:"cron_format"`
}

func (tpl Template) Insert() error {
	c := database.MongoDb.C("project_templates")
	return c.Insert(tpl)
}

func (tpl Template) Remove() error {
	c := database.MongoDb.C("project_templates")
	return c.RemoveId(tpl.ID)
}

func (tpl Template) Update() error {
	c := database.MongoDb.C("project_templates")
	return c.UpdateId(tpl.ID, tpl)
}
