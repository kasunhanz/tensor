package models

import "gopkg.in/mgo.v2/bson"

type Launch struct {
	Limit               string         `bson:"limit,omitempty" json:"limit" binding:"max=1024"`
	ExtraVars           string         `bson:"extra_vars,omitempty" json:"extra_vars"`
	JobTags             string         `bson:"job_tags,omitempty" json:"job_tags" binding:"max=1024"`
	SkipTags            string         `bson:"skip_tags,omitempty" json:"skip_tags" binding:"max=1024"`
	JobType             string         `bson:"job_type" json:"job_type" binding:"required,jobtype"`
	InventoryID         bson.ObjectId  `bson:"inventory_id" json:"inventory" binding:"required"`
	MachineCredentialID bson.ObjectId  `bson:"credential_id" json:"credential" binding:"required"`
}
