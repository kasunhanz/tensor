package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

type Job struct {
	ID                  bson.ObjectId  `bson:"_id" json:"id"`

	Name                string         `bson:"name" json:"name"`
	Description         string         `bson:"description" json:"description"`
	LaunchType          string         `bson:"launch_type" json:"launch_type"`
	CancelFlag          bool           `bson:"cancel_flag" json:"cancel_flag"`
	Status              string           `bson:"status" json:"status"`
	Failed              bool           `bson:"failed" json:"failed"`
	Started             time.Time      `bson:"started" json:"started"`
	Finished            time.Time      `bson:"finished" json:"finished"`
	Elapsed             uint32         `bson:"elapsed" json:"elapsed"`
	JobArgs             string         `bson:"job_args" json:"job_args"`
	StdoutText          string         `bson:"stdout_text" json:"stdout_text"`

	JobType             string         `bson:"job_type" json:"job_type"`
	Playbook            string         `bson:"playbook" json:"playbook"`
	Forks               uint8          `bson:"forks" json:"forks"`
	Limit               string         `bson:"limit" json:"limit"`
	Verbosity           uint8          `bson:"verbosity" json:"verbosity"`
	ExtraVars           string         `bson:"extra_vars" json:"extra_vars"`
	JobTags             string         `bson:"job_tags" json:"job_tags"`
	ForceHandlers       bool           `bson:"force_handlers" json:"force_handlers"`
	SkipTags            string         `bson:"skip_tags" json:"skip_tags"`
	StartAtTask         string         `bson:"start_at_task" json:"start_at_task"`

	BecomeEnabled       bool           `bson:"become_enabled" json:"become_enabled"`
	MachineCredentialID bson.ObjectId  `bson:"credential_id" json:"credential"`
	InventoryID         bson.ObjectId  `bson:"inventory_id" json:"inventory"`
	JobTemplateID       bson.ObjectId  `bson:"job_template_id" json:"job_template"`
	ProjectID           bson.ObjectId  `bson:"project_id" json:"project"`
	NetworkCredentialID bson.ObjectId  `bson:"network_credential_id" json:"network_credential"`
	CloudCredentialID   bson.ObjectId  `bson:"cloud_credential_id,omitempty" json:"cloud_credential"`

	PromptLimit         bool           `bson:"prompt_limit_on_launch" json:"ask_limit_on_launch"`
	PromptInventory     bool           `bson:"prompt_inventory" json:"ask_inventory_on_launch"`
	PromptCredential    bool           `bson:"prompt_credential" json:"ask_credential_on_launch"`
	PromptJobType       bool           `bson:"prompt_job_type" json:"ask_job_type_on_launch"`
	PromptTags          bool           `bson:"prompt_tags" json:"ask_tags_on_launch"`
	PromptVariables     bool           `bson:"prompt_variables" json:"ask_variables_on_launch"`

	JobCWD              string         `bson:"job_cwd" json:"job_cwd"`
	JobARGS             string         `bson:"job_args" json:"job_args"`
	JobENV              string         `bson:"job_env" json:"job_env"`

	CreatedByID         bson.ObjectId  `bson:"created_by_id" json:"created_by"`
	ModifiedByID        bson.ObjectId  `bson:"modified_by_id" json:"modified_by"`
	Created             time.Time      `bson:"created" json:"created"`
	Modified            time.Time      `bson:"modified" json:"modified"`

	Type                string          `bson:"-" json:"type"`
	Url                 string          `bson:"-" json:"url"`
	Related             gin.H           `bson:"-" json:"related"`
	Summary             gin.H           `bson:"-" json:"summary_fields"`

	Roles               []AccessControl    `bson:"roles" json:"-"`
}