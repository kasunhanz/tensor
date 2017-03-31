package terraform

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type JobTemplate struct {
	ID                  bson.ObjectId `bson:"_id" json:"id"`

	// required
	Name                string         `bson:"name" json:"name" binding:"required,min=1,max=500"`
	JobType             string         `bson:"job_type" json:"job_type" binding:"required,terraform_jobtype"`
	ProjectID           bson.ObjectId  `bson:"project_id" json:"project" binding:"required"`
	MachineCredentialID *bson.ObjectId `bson:"credential_id,omitempty" json:"credential"`

	Description         string         `bson:"description,omitempty" json:"description"`
	Vars                gin.H          `bson:"vars,omitempty" json:"vars"`
	PromptVariables     bool           `bson:"ask_variables_on_launch,omitempty" json:"ask_variables_on_launch"`
	CloudCredentialID   *bson.ObjectId `bson:"cloud_credential_id,omitempty" json:"cloud_credential"`
	NetworkCredentialID *bson.ObjectId `bson:"network_credential_id,omitempty" json:"network_credential"`
	SCMCredentialID     *bson.ObjectId `bson:"scm_credential_id,omitempty" json:"scm_credential_id"`
	PromptCredential    bool           `bson:"prompt_credential,omitempty" json:"ask_credential_on_launch"`
	PromptJobType       bool           `bson:"prompt_job_type,omitempty" json:"ask_job_type_on_launch"`
	AllowSimultaneous   bool           `bson:"allow_simultaneous,omitempty" json:"allow_simultaneous"`
	Parallelism         uint8          `bson:"parallelism,omitempty" json:"parallelism"`
	UpdateOnLaunch      bool           `bson:"update_on_launch" json:"update_on_launch"`
	Target              string         `bson:"target" json:"target"`
	Directory           string         `bson:"directory" json:"directory"`
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
	Links               gin.H  `bson:"-" json:"links"`
	Meta                gin.H  `bson:"-" json:"meta"`

	Roles               []common.AccessControl `bson:"roles" json:"-"`
}

func (JobTemplate) GetType() string {
	return "terraform_job_template"
}

func (org JobTemplate) GetRoles() []common.AccessControl {
	return org.Roles
}

func (jt JobTemplate) GetOrganizationID() (bson.ObjectId, error) {
	var org common.Organization
	pID, err := jt.GetProjectID()
	if err != nil {
		return org.ID, err
	}
	err = db.Organizations().FindId(pID).One(&org)
	return org.ID, err
}

func (jt JobTemplate) GetProjectID() (bson.ObjectId, error) {
	var prj common.Project
	err := db.Projects().FindId(jt.ProjectID).One(&prj)
	return prj.ID, err
}

func (jt *JobTemplate) IsUnique() bool {
	count, err := db.TerrafromJobTemplates().Find(bson.M{"name": jt.Name, "project_id": jt.ProjectID}).Count()
	if err == nil && count > 0 {
		return false
	}
	return true
}

func (jt *JobTemplate) ProjectExist() bool {
	count, err := db.Projects().FindId(jt.ProjectID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (jt *JobTemplate) MachineCredentialExist() bool {
	query := bson.M{
		"_id": jt.MachineCredentialID,
		"kind": bson.M{
			"$in": []string{
				common.CredentialKindSSH,
				common.CredentialKindWIN,
			},
		},
	}
	count, err := db.Credentials().Find(query).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (jt *JobTemplate) NetworkCredentialExist() bool {
	count, err := db.Credentials().Find(bson.M{"_id": jt.NetworkCredentialID, "kind": common.CredentialKindNET}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (jt *JobTemplate) CloudCredentialExist() bool {
	query := bson.M{
		"_id": jt.CloudCredentialID,
		"kind": bson.M{
			"$in": []string{
				common.CredentialKindAWS,
				common.CredentialKindAZURE,
				common.CredentialKindCLOUDFORMS,
				common.CredentialKindGCE,
				common.CredentialKindOPENSTACK,
				common.CredentialKindSATELLITE6,
				common.CredentialKindVMWARE,
			},
		},
	}
	count, err := db.Credentials().Find(query).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}
