package terraform

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type Launch struct {
	Vars                gin.H          `bson:"vars,omitempty" json:"vars,omitempty"`
	JobType             string         `bson:"job_type,omitempty" json:"job_type,omitempty" binding:"omitempty,terraform_jobtype"`
	MachineCredentialID *bson.ObjectId `bson:"credential_id,omitempty" json:"credential,omitempty"`
}
