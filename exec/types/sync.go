package types

import (
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

// SyncJob contains all the information required to start a job
type SyncJob struct {
	Job            ansible.Job
	ProjectID      bson.ObjectId
	JobTemplateID  bson.ObjectId
	SCM            common.Credential
	Project        common.Project
	User           common.User
	Token          string
	CredentialPath string // for system jobs
}
