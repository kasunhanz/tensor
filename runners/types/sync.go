package types

import (
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
)

// SyncJob contains all the information required to start a job
type SyncJob struct {
	Job            ansible.Job
	Template       ansible.JobTemplate
	SCMCred        common.Credential
	Project        common.Project
	User           common.User
	Token          string
	CredentialPath string // for system jobs
}

// AnsibleJob contains all the information required to start a job
type AnsibleJob struct {
	Job         ansible.Job
	Template    ansible.JobTemplate
	MachineCred common.Credential
	NetworkCred common.Credential
	SCMCred     common.Credential
	CloudCred   common.Credential
	Inventory   ansible.Inventory
	Project     common.Project
	User        common.User
	PreviousJob *SyncJob
	Token       string
}
