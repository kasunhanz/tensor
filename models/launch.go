package models

import "gopkg.in/mgo.v2/bson"

type Launch struct {
	Limit               string         `bson:"limit,omitempty" json:"limit,omitempty" binding:"max=1024,omitempty"`
	ExtraVars           string         `bson:"extra_vars,omitempty" json:"extra_vars,omitempty"`
	JobTags             string         `bson:"job_tags,omitempty" json:"job_tags,omitempty" binding:"max=1024,omitempty"`
	SkipTags            string         `bson:"skip_tags,omitempty" json:"skip_tags,omitempty" binding:"max=1024,omitempty"`
	JobType             string         `bson:"job_type,omitempty" json:"job_type" binding:"jobtype,omitempty"`
	InventoryID         bson.ObjectId  `bson:"inventory_id,omitempty" json:"inventory,omitempty"`
	MachineCredentialID bson.ObjectId  `bson:"credential_id,omitempty" json:"credential,omitempty"`
}
