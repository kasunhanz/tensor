package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
	"github.com/gin-gonic/gin"
)

const DBC_JOB_TEMPLATES = "job_templates"

type JobTemplate struct {
	ID                  bson.ObjectId  `bson:"_id" json:"id"`

	//from unified job template
	Description         string         `bson:"description" json:"description"`
	Name                string         `bson:"name" json:"name"`
	LastJobFailed       bool           `bson:"last_job_failed" json:"last_job_failed"`
	LastJobRun          time.Time      `bson:"last_job_run" json:"last_job_run"`
	HasSchedules        bool           `bson:"has_schedules" json:"has_schedules"`
	NextJobRun          time.Time      `bson:"next_job_run" json:"next_job_run"`
	Status              string         `bson:"status" json:"status"`
	CurrentJobID        bson.ObjectId  `bson:"current_job_id,omitempty" json:"current_job"`
	LastJobID           bson.ObjectId  `bson:"last_job_id,omitempty" json:"last_job"`
	NextScheduleID      bson.ObjectId  `bson:"next_schedule_id,omitempty" json:"next_schedule"`
	PolymorphicCtypeID  bson.ObjectId  `bson:"polymorphic_ctype_id,omitempty" json:"polymorphic_ctype"`

	JobType             string         `bson:"job_type" json:"job_type"`
	Kind                string         `bson:"kind" json:"-"`
	Playbook            string         `bson:"playbook" json:"playbook"`
	Forks               uint8          `bson:"forks" json:"forks"`
	Limit               string         `bson:"limit" json:"limit"`
	Verbosity           uint8          `bson:"verbosity" json:"verbosity"`
	ExtraVars           string         `bson:"extra_vars" json:"extra_vars"`
	JobTags             string         `bson:"job_tags" json:"job_tags"`
	ForceHandlers       bool           `bson:"force_handlers" json:"force_handlers"`
	SkipTags            string         `bson:"skip_tags" json:"skip_tags"`
	StartAtTask         string         `bson:"start_at_task" json:"start_at_task"`

	PromptVariables     bool           `bson:"ask_variables_on_launch" json:"ask_variables_on_launch"`

	BecomeEnabled       bool           `bson:"become_enabled" json:"become_enabled"`
	InventoryID         bson.ObjectId  `bson:"inventory_id" json:"inventory"`
	ProjectID           bson.ObjectId  `bson:"project_id" json:"project"`
	MachineCredentialID bson.ObjectId  `bson:"credential_id" json:"credential"`
	CloudCredentialID   bson.ObjectId  `bson:"cloud_credential_id,omitempty" json:"cloud_credential"`
	NetworkCredentialID bson.ObjectId  `bson:"network_credential_id,omitempty" json:"network_credential"`

	PromptLimit         bool           `bson:"prompt_limit_on_launch" json:"ask_limit_on_launch"`
	PromptInventory     bool           `bson:"prompt_inventory" json:"ask_inventory_on_launch"`
	PromptCredential    bool           `bson:"prompt_credential" json:"ask_credential_on_launch"`
	PromptJobType       bool           `bson:"prompt_job_type" json:"ask_job_type_on_launch"`
	PromptTags          bool           `bson:"prompt_tags" json:"ask_tags_on_launch"`

	AllowSimultaneous   bool           `bson:"allow_simultaneous" json:"allow_simultaneous"`

	CreatedByID         bson.ObjectId  `bson:"created_by_id" json:"created_by"`
	ModifiedByID        bson.ObjectId  `bson:"modified_by_id" json:"modified_by"`
	Created             time.Time      `bson:"created" json:"created"`
	Modified            time.Time      `bson:"modified" json:"modified"`

	Type                string         `bson:"-" json:"type"`
	Url                 string         `bson:"-" json:"url"`
	Related             gin.H          `bson:"-" json:"related"`
	Summary             gin.H          `bson:"-" json:"summary_fields"`
}


func (t JobTemplate) CreateIndexes()  {

}