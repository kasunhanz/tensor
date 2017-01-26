package types

import (
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"
)

// TerraformJob contains all the information required to start a job
type TerraformJob struct {
	Job         terraform.Job
	Template    terraform.JobTemplate
	MachineCred common.Credential
	NetworkCred common.Credential
	SCMCred     common.Credential
	CloudCred   common.Credential
	Project     common.Project
	User        common.User
	PreviousJob *SyncJob
	Token       string
}
