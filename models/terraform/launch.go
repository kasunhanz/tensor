package terraform

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type Launch struct {
	Limit               string        `bson:"limit,omitempty" json:"limit,omitempty" binding:"omitempty,max=1024"`
	Vars                gin.H         `bson:"vars,omitempty" json:"vars,omitempty"`
	JobType             string        `bson:"job_type,omitempty" json:"job_type,omitempty" binding:"omitempty,jobtype"`
	MachineCredentialID bson.ObjectId `bson:"credential_id,omitempty" json:"credential,omitempty"`
}
