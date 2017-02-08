package ansible

import (
	"time"

	"gopkg.in/gin-gonic/gin.v1"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

type JobTemplate struct {
	ID                  bson.ObjectId `bson:"_id" json:"id"`

	// required
	Name                string        `bson:"name" json:"name" binding:"required,min=1,max=500"`
	JobType             string        `bson:"job_type" json:"job_type" binding:"required,jobtype"`
	InventoryID         bson.ObjectId `bson:"inventory_id" json:"inventory" binding:"required"`
	ProjectID           bson.ObjectId `bson:"project_id" json:"project" binding:"required"`
	Playbook            string        `bson:"playbook" json:"playbook" binding:"required"`
	MachineCredentialID bson.ObjectId `bson:"credential_id" json:"credential" binding:"required"`

	Verbosity           uint8 `bson:"verbosity,omitempty" json:"verbosity" binding:"omitempty,max=5"`

	Description         string         `bson:"description,omitempty" json:"description"`
	Forks               uint8          `bson:"forks,omitempty" json:"forks"`
	Limit               string         `bson:"limit,omitempty" json:"limit" binding:"max=1024"`
	ExtraVars           gin.H          `bson:"extra_vars,omitempty" json:"extra_vars"`
	JobTags             string         `bson:"job_tags,omitempty" json:"job_tags" binding:"max=1024"`
	SkipTags            string         `bson:"skip_tags,omitempty" json:"skip_tags" binding:"max=1024"`
	StartAtTask         string         `bson:"start_at_task,omitempty" json:"start_at_task"`
	ForceHandlers       bool           `bson:"force_handlers,omitempty" json:"force_handlers"`
	PromptVariables     bool           `bson:"ask_variables_on_launch,omitempty" json:"ask_variables_on_launch"`
	BecomeEnabled       bool           `bson:"become_enabled,omitempty" json:"become_enabled"`
	CloudCredentialID   *bson.ObjectId `bson:"cloud_credential_id,omitempty" json:"cloud_credential"`
	NetworkCredentialID *bson.ObjectId `bson:"network_credential_id,omitempty" json:"network_credential"`
	PromptLimit         bool           `bson:"prompt_limit_on_launch,omitempty" json:"ask_limit_on_launch"`
	PromptInventory     bool           `bson:"prompt_inventory,omitempty" json:"ask_inventory_on_launch"`
	PromptCredential    bool           `bson:"prompt_credential,omitempty" json:"ask_credential_on_launch"`
	PromptJobType       bool           `bson:"prompt_job_type,omitempty" json:"ask_job_type_on_launch"`
	PromptTags          bool           `bson:"prompt_tags,omitempty" json:"ask_tags_on_launch"`
	PromptSkipTags      bool           `bson:"prompt_skip_tags,omitempty" json:"ask_skip_tags_on_launch"`
	AllowSimultaneous   bool           `bson:"allow_simultaneous,omitempty" json:"allow_simultaneous"`

	PolymorphicCtypeID  *bson.ObjectId `bson:"polymorphic_ctype_id,omitempty" json:"polymorphic_ctype"`

	// output only
	LastJobRun          *time.Time     `bson:"last_job_run,omitempty" json:"last_job_run" binding:"omitempty,naproperty"`
	NextJobRun          *time.Time     `bson:"next_job_run,omitempty" json:"next_job_run" binding:"omitempty,naproperty"`
	Status              string         `bson:"status,omitempty" json:"status" binding:"omitempty,naproperty"`
	CurrentJobID        *bson.ObjectId `bson:"current_job_id,omitempty" json:"current_job" binding:"omitempty,naproperty"`
	CurrentUpdateID     *bson.ObjectId `bson:"current_update_id,omitempty" json:"current_update" binding:"omitempty,naproperty"`
	LastJobID           *bson.ObjectId `bson:"last_job_id,omitempty" json:"last_job" binding:"omitempty,naproperty"`
	NextScheduleID      *bson.ObjectId `bson:"next_schedule_id,omitempty" json:"next_schedule" binding:"omitempty,naproperty"`
	LastJobFailed       bool           `bson:"last_job_failed,omitempty" json:"last_job_failed" binding:"omitempty,naproperty"`
	HasSchedules        bool           `bson:"has_schedules,omitempty" json:"has_schedules" binding:"omitempty,naproperty"`

	Kind                string `bson:"kind,omitempty" json:"-"`

	CreatedByID         bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID        bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created             time.Time `bson:"created" json:"created" binding:"omitempty,naproperty"`
	Modified            time.Time `bson:"modified" json:"modified" binding:"omitempty,naproperty"`

	Type                string `bson:"-" json:"type"`
	URL                 string `bson:"-" json:"url"`
	Related             gin.H  `bson:"-" json:"related"`
	Summary             gin.H  `bson:"-" json:"summary_fields"`

	Roles               []common.AccessControl `bson:"roles" json:"-"`
}

func (*JobTemplate) GetType() string {
	return "job_template"
}

type PatchJobTemplate struct {
	Name                *string        `json:"name" binding:"omitempty,min=1,max=500"`
	JobType             *string        `json:"job_type" binding:"omitempty,jobtype"`
	InventoryID         *bson.ObjectId `json:"inventory"`
	ProjectID           *bson.ObjectId `json:"project"`
	Playbook            *string        `json:"playbook"`
	MachineCredentialID *bson.ObjectId `json:"credential"`
	Verbosity           *uint8         `json:"verbosity" binding:"omitempty,min=0,max=5"`
	Description         *string        `json:"description"`
	Forks               *uint8         `json:"forks"`
	Limit               *string        `json:"limit" binding:"omitempty,max=1024"`
	ExtraVars           *gin.H         `json:"extra_vars"`
	JobTags             *string        `json:"job_tags" binding:"omitempty,max=1024"`
	SkipTags            *string        `json:"skip_tags" binding:"omitempty,max=1024"`
	StartAtTask         *string        `json:"start_at_task"`
	ForceHandlers       *bool          `json:"force_handlers"`
	PromptVariables     *bool          `json:"ask_variables_on_launch"`
	BecomeEnabled       *bool          `json:"become_enabled"`
	CloudCredentialID   *bson.ObjectId `json:"cloud_credential"`
	NetworkCredentialID *bson.ObjectId `json:"network_credential"`
	PromptLimit         *bool          `json:"ask_limit_on_launch"`
	PromptInventory     *bool          `json:"ask_inventory_on_launch"`
	PromptCredential    *bool          `json:"ask_credential_on_launch"`
	PromptJobType       *bool          `json:"ask_job_type_on_launch"`
	PromptTags          *bool          `json:"ask_tags_on_launch"`
	PromptSkipTags      *bool          `json:"ask_skip_tags_on_launch"`
	AllowSimultaneous   *bool          `json:"allow_simultaneous"`
	PolymorphicCtypeID  *bson.ObjectId `json:"polymorphic_ctype"`
}
