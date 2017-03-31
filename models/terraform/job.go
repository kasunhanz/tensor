package terraform

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// Job constants
const (
	JobTypeTerraformJob = "terraform_job" // A terraform job
	JobLaunchTypeManual = "manual"
	JobLaunchTypeSystem = "system"
)

type Job struct {
	ID                  bson.ObjectId `bson:"_id" json:"id"`

	Name                string `bson:"name" json:"name" binding:"required"`

	Description         string    `bson:"description,omitempty" json:"description"`
	LaunchType          string    `bson:"launch_type" json:"launch_type"`
	CancelFlag          bool      `bson:"cancel_flag" json:"cancel_flag"`
	Status              string    `bson:"status" json:"status"`
	Failed              bool      `bson:"failed" json:"failed"`
	Started             time.Time `bson:"started" json:"started"`
	Finished            time.Time `bson:"finished" json:"finished"`
	Elapsed             uint32    `bson:"elapsed" json:"elapsed"`
	ResultStdout        string    `bson:"result_stdout" json:"result_stdout"`
	ResultGetStdout     string    `bson:"result_get_stdout" json:"result_get_stdout"`
	ResultTraceback     string    `bson:"result_traceback" json:"result_traceback"`
	JobExplanation      string    `bson:"job_explanation" json:"job_explanation"`
	JobType             string    `bson:"job_type" json:"job_type,terraform_jobtype"`
	Vars                gin.H     `bson:"vars,omitempty" json:"vars"`
	Parallelism         uint8     `bson:"parallelism" json:"parallelism"`
	UpdateOnLaunch      bool      `bson:"update_on_launch" json:"update_on_launch"`
	Target              string          `bson:"target" json:"target"`
	Directory           string         `bson:"directory" json:"directory"`

	MachineCredentialID *bson.ObjectId `bson:"credential_id,omitempty" json:"credential"`
	JobTemplateID       bson.ObjectId  `bson:"job_template_id,omitempty" json:"job_template"`
	ProjectID           bson.ObjectId  `bson:"project_id,omitempty" json:"project"`
	SCMCredentialID     *bson.ObjectId `bson:"scm_credential_id,omitempty" json:"scm_credential"`
	NetworkCredentialID *bson.ObjectId `bson:"network_credential_id,omitempty" json:"network_credential"`
	CloudCredentialID   *bson.ObjectId `bson:"cloud_credential_id,omitempty" json:"cloud_credential"`

	PromptCredential    bool `bson:"prompt_credential" json:"ask_credential_on_launch"`
	PromptJobType       bool `bson:"prompt_job_type" json:"ask_job_type_on_launch"`
	PromptVariables     bool `bson:"prompt_variables" json:"ask_variables_on_launch"`
	AllowSimultaneous   bool `bson:"allow_simultaneous,omitempty" json:"allow_simultaneous"`

	// system generated items
	JobCWD              string   `bson:"job_cwd" json:"job_cwd"`
	JobARGS             []string `bson:"job_args" json:"job_args"`
	JobENV              []string `bson:"job_env" json:"job_env"`

	CreatedByID         bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID        bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created             time.Time `bson:"created" json:"created"`
	Modified            time.Time `bson:"modified" json:"modified"`

	Type                string `bson:"-" json:"type"`
	Links               gin.H  `bson:"-" json:"links"`
	Meta                gin.H  `bson:"-" json:"meta"`

	Roles               []common.AccessControl `bson:"roles" json:"-"`
}

func (Job) GetType() string {
	return "terraform_job"
}

func (job Job) GetRoles() []common.AccessControl {
	return job.Roles
}

func (job Job) GetJobTemplate() (JobTemplate, error) {
	var jobt JobTemplate
	err := db.TerrafromJobTemplates().FindId(job.JobTemplateID).One(&jobt)
	return jobt, err
}
