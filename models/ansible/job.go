package ansible

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

// Job constants
const (
	JOBTYPE_ANSIBLE_JOB = "ansible_job" // A ansible job
	JOBTYPE_UPDATE_JOB  = "update_job"  // A project scm update job

	JOB_LAUNCH_TYPE_MANUAL = "manual"
	JOB_LAUNCH_TYPE_SYSTEM = "system"
)

type Job struct {
	ID bson.ObjectId `bson:"_id" json:"id"`

	Name string `bson:"name" json:"name" binding:"required"`

	Description     string    `bson:"description,omitempty" json:"description"`
	LaunchType      string    `bson:"launch_type" json:"launch_type"`
	CancelFlag      bool      `bson:"cancel_flag" json:"cancel_flag"`
	Status          string    `bson:"status" json:"status"`
	Failed          bool      `bson:"failed" json:"failed"`
	Started         time.Time `bson:"started" json:"started"`
	Finished        time.Time `bson:"finished" json:"finished"`
	Elapsed         uint32    `bson:"elapsed" json:"elapsed"`
	ResultStdout    string    `bson:"result_stdout" json:"result_stdout"`
	ResultTraceback string    `bson:"result_traceback" json:"result_traceback"`
	JobExplanation  string    `bson:"job_explanation" json:"job_explanation"`
	JobType         string    `bson:"job_type" json:"job_type"`

	Playbook          string `bson:"playbook" json:"playbook"`
	Forks             uint8  `bson:"forks" json:"forks"`
	Limit             string `bson:"limit,omitempty" json:"limit"`
	Verbosity         uint8  `bson:"verbosity" json:"verbosity"`
	ExtraVars         gin.H  `bson:"extra_vars,omitempty" json:"extra_vars"`
	JobTags           string `bson:"job_tags,omitempty" json:"job_tags"`
	SkipTags          string `bson:"skip_tags,omitempty" json:"skip_tags"`
	ForceHandlers     bool   `bson:"force_handlers" json:"force_handlers"`
	StartAtTask       string `bson:"start_at_task,omitempty" json:"start_at_task"`
	AllowSimultaneous bool   `bson:"allow_simultaneous,omitempty" json:"allow_simultaneous"`

	MachineCredentialID bson.ObjectId  `bson:"credential_id,omitempty" json:"credential"`
	InventoryID         bson.ObjectId  `bson:"inventory_id,omitempty" json:"inventory"`
	JobTemplateID       bson.ObjectId  `bson:"job_template_id,omitempty" json:"job_template"`
	ProjectID           bson.ObjectId  `bson:"project_id,omitempty" json:"project"`
	BecomeEnabled       bool           `bson:"become_enabled" json:"become_enabled"`
	SCMCredentialID     *bson.ObjectId `bson:"scm_credential_id,omitempty" json:"scm_credential"`
	NetworkCredentialID *bson.ObjectId `bson:"network_credential_id,omitempty" json:"network_credential"`
	CloudCredentialID   *bson.ObjectId `bson:"cloud_credential_id,omitempty" json:"cloud_credential"`

	PromptLimit      bool `bson:"prompt_limit_on_launch" json:"ask_limit_on_launch"`
	PromptInventory  bool `bson:"prompt_inventory" json:"ask_inventory_on_launch"`
	PromptCredential bool `bson:"prompt_credential" json:"ask_credential_on_launch"`
	PromptJobType    bool `bson:"prompt_job_type" json:"ask_job_type_on_launch"`
	PromptTags       bool `bson:"prompt_tags" json:"ask_tags_on_launch"`
	PromptVariables  bool `bson:"prompt_variables" json:"ask_variables_on_launch"`

	// system generated items
	JobCWD  string   `bson:"job_cwd" json:"job_cwd"`
	JobARGS []string `bson:"job_args" json:"job_args"`
	JobENV  []string `bson:"job_env" json:"job_env"`

	CreatedByID  bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created  time.Time `bson:"created" json:"created"`
	Modified time.Time `bson:"modified" json:"modified"`

	Type    string `bson:"-" json:"type"`
	URL     string `bson:"-" json:"url"`
	Related gin.H  `bson:"-" json:"related"`
	Summary gin.H  `bson:"-" json:"summary_fields"`

	Roles []common.AccessControl `bson:"roles" json:"-"`
}
