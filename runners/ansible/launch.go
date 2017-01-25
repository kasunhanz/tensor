package ansible

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type Launch struct {
	Limit               string        `bson:"limit,omitempty" json:"limit,omitempty" binding:"omitempty,max=1024"`
	ExtraVars           gin.H         `bson:"extra_vars,omitempty" json:"extra_vars,omitempty"`
	JobTags             string        `bson:"job_tags,omitempty" json:"job_tags,omitempty" binding:"omitempty,max=1024"`
	SkipTags            string        `bson:"skip_tags,omitempty" json:"skip_tags,omitempty" binding:"omitempty,max=1024"`
	JobType             string        `bson:"job_type,omitempty" json:"job_type,omitempty" binding:"omitempty,jobtype"`
	InventoryID         bson.ObjectId `bson:"inventory_id,omitempty" json:"inventory,omitempty"`
	MachineCredentialID bson.ObjectId `bson:"credential_id,omitempty" json:"credential,omitempty"`
}
