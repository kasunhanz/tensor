package ansible

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/gin-gonic/gin.v1"
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
	MachineCredentialID *bson.ObjectId `bson:"credential_id,omitempty" json:"credential"`
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
	Links               gin.H  `bson:"-" json:"links"`
	Meta                gin.H  `bson:"-" json:"meta"`

	Roles               []common.AccessControl `bson:"roles" json:"-"`
}

func (JobTemplate) GetType() string {
	return "job_template"
}

func (jt JobTemplate) IsUnique() bool {
	count, err := db.JobTemplates().Find(bson.M{"name": jt.Name, "project_id": jt.ProjectID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (h JobTemplate) GetCredential() (common.Credential, error) {
	var cred common.Credential
	err := db.Credentials().FindId(h.InventoryID).One(&cred)
	return cred, err
}

func (h JobTemplate) GetInventory() (Inventory, error) {
	var inv Inventory
	err := db.Inventories().FindId(h.InventoryID).One(&inv)
	return inv, err
}

func (jt JobTemplate) GetProject() (common.Project, error) {
	var prj common.Project
	err := db.Projects().FindId(jt.ProjectID).One(&prj)
	return prj, err
}

func (h JobTemplate) GetNetworkCredential() (common.Credential, error) {
	var cred common.Credential
	err := db.Credentials().FindId(h.InventoryID).One(&cred)
	return cred, err
}

func (h JobTemplate) GetCloudCredential() (common.Credential, error) {
	var cred common.Credential
	err := db.Credentials().FindId(h.InventoryID).One(&cred)
	return cred, err
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

func (jt JobTemplate) ProjectExist() bool {
	count, err := db.Projects().FindId(jt.ProjectID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (jt JobTemplate) InventoryExist() bool {
	count, err := db.Inventories().FindId(jt.InventoryID).Count()
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

func (jt JobTemplate) GetRoles() []common.AccessControl {
	return jt.Roles
}