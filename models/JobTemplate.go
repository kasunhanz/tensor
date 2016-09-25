package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
	"github.com/gin-gonic/gin"
)

type JobTemplate struct {
	ID                  bson.ObjectId  `bson:"_id" json:"id"`

	// required
	Name                string         `bson:"name" json:"name" binding:"required"`
	JobType             string         `bson:"job_type" json:"job_type" binding:"required"`
	InventoryID         bson.ObjectId  `bson:"inventory_id" json:"inventory" binding:"required"`
	ProjectID           bson.ObjectId  `bson:"project_id" json:"project" binding:"required"`
	Playbook            string         `bson:"playbook" json:"playbook" binding:"required"`
	MachineCredentialID bson.ObjectId  `bson:"credential_id" json:"credential" binding:"required"`

	LastJobRun          *time.Time      `bson:"last_job_run,omitempty" json:"last_job_run"`
	NextJobRun          *time.Time      `bson:"next_job_run,omitempty" json:"next_job_run"`
	Status              *string         `bson:"status,omitempty" json:"status"`
	CurrentJobID        *bson.ObjectId  `bson:"current_job_id,omitempty" json:"current_job"`
	LastJobID           *bson.ObjectId  `bson:"last_job_id,omitempty" json:"last_job"`
	NextScheduleID      *bson.ObjectId  `bson:"next_schedule_id,omitempty" json:"next_schedule"`
	PolymorphicCtypeID  *bson.ObjectId  `bson:"polymorphic_ctype_id,omitempty" json:"polymorphic_ctype"`

	LastJobFailed       bool           `bson:"last_job_failed,omitempty" json:"last_job_failed"`
	HasSchedules        bool           `bson:"has_schedules,omitempty" json:"has_schedules"`

	Description         *string         `bson:"description,omitempty" json:"description"`

	Kind                *string         `bson:"kind,omitempty" json:"-"`
	Forks               *uint8          `bson:"forks,omitempty" json:"forks"`
	Limit               *string         `bson:"limit,omitempty" json:"limit"`
	Verbosity           *uint8          `bson:"verbosity,omitempty" json:"verbosity"`
	ExtraVars           *string         `bson:"extra_vars,omitempty" json:"extra_vars"`
	JobTags             *string         `bson:"job_tags,omitempty" json:"job_tags"`
	SkipTags            *string         `bson:"skip_tags,omitempty" json:"skip_tags"`
	StartAtTask         *string         `bson:"start_at_task,omitempty" json:"start_at_task"`

	ForceHandlers       bool           `bson:"force_handlers,omitempty" json:"force_handlers"`
	PromptVariables     bool           `bson:"ask_variables_on_launch,omitempty" json:"ask_variables_on_launch"`

	BecomeEnabled       bool           `bson:"become_enabled,omitempty" json:"become_enabled"`
	CloudCredentialID   *bson.ObjectId  `bson:"cloud_credential_id,omitempty" json:"cloud_credential"`
	NetworkCredentialID *bson.ObjectId  `bson:"network_credential_id,omitempty" json:"network_credential"`

	PromptLimit         bool           `bson:"prompt_limit_on_launch,omitempty" json:"ask_limit_on_launch"`
	PromptInventory     bool           `bson:"prompt_inventory,omitempty" json:"ask_inventory_on_launch"`
	PromptCredential    bool           `bson:"prompt_credential,omitempty" json:"ask_credential_on_launch"`
	PromptJobType       bool           `bson:"prompt_job_type,omitempty" json:"ask_job_type_on_launch"`
	PromptTags          bool           `bson:"prompt_tags,omitempty" json:"ask_tags_on_launch"`

	AllowSimultaneous   bool           `bson:"allow_simultaneous,omitempty" json:"allow_simultaneous"`

	CreatedByID         bson.ObjectId  `bson:"created_by_id" json:"-"`
	ModifiedByID        bson.ObjectId  `bson:"modified_by_id" json:"-"`

	Created             time.Time      `bson:"created" json:"created"`
	Modified            time.Time      `bson:"modified" json:"modified"`

	Type                string         `bson:"-" json:"type"`
	Url                 string         `bson:"-" json:"url"`
	Related             gin.H          `bson:"-" json:"related"`
	Summary             gin.H          `bson:"-" json:"summary_fields"`

	Roles               []AccessControl    `bson:"roles" json:"-"`
}