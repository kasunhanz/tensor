package types

import (
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
)

// AnsibleJob contains all the information required to start a job
type AnsibleJob struct {
	Job         ansible.Job
	Template    ansible.JobTemplate
	Machine     common.Credential
	Network     common.Credential
	SCM         common.Credential
	Cloud       common.Credential
	Inventory   ansible.Inventory
	Project     common.Project
	User        common.User
	PreviousJob *SyncJob
	Token       string
}
