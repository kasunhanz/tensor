package types

import (
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"
)

// TerraformJob contains all the information required to start a job
type TerraformJob struct {
	Job         terraform.Job
	Template    terraform.JobTemplate
	Machine     common.Credential
	Network     common.Credential
	SCM         common.Credential
	Cloud       common.Credential
	Project     common.Project
	User        common.User
	PreviousJob *SyncJob
	Token       string
}
