package common

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// Project is the model for project
// collection
type Project struct {
	ID                    bson.ObjectId `bson:"_id" json:"id"`

	Type                  string `bson:"-" json:"type"`
	URL                   string `bson:"-" json:"url"`
	Related               gin.H  `bson:"-" json:"related"`
	Summary               gin.H  `bson:"-" json:"summary_fields"`

	// required fields
	Name                  string        `bson:"name" json:"name" binding:"required,min=1,max=500"`
	ScmType               string        `bson:"scm_type" json:"scm_type" binding:"required,scmtype"`
	OrganizationID        bson.ObjectId `bson:"organization_id" json:"organization" binding:"required"`

	Description           string         `bson:"description,omitempty" json:"description"`
	LocalPath             string         `bson:"local_path,omitempty" json:"local_path" binding:"omitempty,naproperty"`
	ScmURL                string         `bson:"scm_url,omitempty" json:"scm_url" binding:"url"`
	Kind                  string         `bson:"kind,omitempty" json:"kind" binding:"project_kind"`
	ScmBranch             string         `bson:"scm_branch,omitempty" json:"scm_branch"`
	ScmClean              bool           `bson:"scm_clean,omitempty" json:"scm_clean"`
	ScmDeleteOnUpdate     bool           `bson:"scm_delete_on_update,omitempty" json:"scm_delete_on_update"`
	ScmCredentialID       *bson.ObjectId `bson:"credentail_id,omitempty" json:"credential"`
	ScmDeleteOnNextUpdate bool           `bson:"scm_delete_on_next_update,omitempty" json:"scm_delete_on_next_update"`
	ScmUpdateOnLaunch     bool           `bson:"scm_update_on_launch,omitempty" json:"scm_update_on_launch"`
	ScmUpdateCacheTimeout int            `bson:"scm_update_cache_timeout,omitempty" json:"scm_update_cache_timeout"`

	// only output
	LastJob               *bson.ObjectId `bson:"last_job,omitempty" json:"last_job" binding:"omitempty,naproperty"`
	LastJobRun            *time.Time     `bson:"last_job_run,omitempty" json:"last_job_run" binding:"omitempty,naproperty"`
	LastJobFailed         bool           `bson:"last_job_failed,omitempty" json:"last_job_failed" binding:"omitempty,naproperty"`
	HasSchedules          bool           `bson:"has_schedules,omitempty" json:"has_schedules" binding:"omitempty,naproperty"`
	NextJobRun            *time.Time     `bson:"next_job_run,omitempty" json:"next_job_run" binding:"omitempty,naproperty"`
	Status                string         `bson:"status,omitempty" json:"status" binding:"omitempty,naproperty"`
	LastUpdateFailed      bool           `bson:"last_update_failed,omitempty" json:"last_update_failed" binding:"omitempty,naproperty"`
	LastUpdated           *time.Time     `bson:"last_updated,omitempty" json:"last_updated" binding:"omitempty,naproperty"`

	CreatedByID           bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID          bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created               time.Time `bson:"created" json:"created" binding:"omitempty,naproperty"`
	Modified              time.Time `bson:"modified" json:"modified" binding:"omitempty,naproperty"`

	Roles                 []AccessControl `bson:"roles" json:"-"`
}

func (Project) GetType() string {
	return "project"
}

func (p Project) GetRoles() []AccessControl {
	return p.Roles
}

func (p Project) GetOrganizationID() (bson.ObjectId, error) {
	var org Organization
	err := db.Organizations().FindId(p.OrganizationID).One(&org)
	return org.ID, err
}

func (project Project) IsUnique() bool {
	count, err := db.Projects().Find(bson.M{"name": project.Name, "organization_id": project.OrganizationID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (project Project) Exist() bool {
	count, err := db.Projects().FindId(project.ID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (project Project) OrganizationExist() bool {
	count, err := db.Organizations().FindId(project.OrganizationID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (project *Project) SCMCredentialExist() bool {
	count, err := db.Credentials().Find(bson.M{"_id": project.ScmCredentialID, "kind": CredentialKindSCM}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

// PatchProject is the model for PATCH requests
type PatchProject struct {
	Name                  *string        `json:"name" binding:"omitempty,min=1,max=500"`
	ScmType               *string        `json:"scm_type" binding:"omitempty,scmtype"`
	OrganizationID        *bson.ObjectId `json:"organization"`
	Description           *string        `json:"description"`
	ScmURL                *string        `json:"scm_url" binding:"omitempty,url"`
	ScmBranch             *string        `json:"scm_branch"`
	ScmClean              *bool          `json:"scm_clean"`
	ScmDeleteOnUpdate     *bool          `json:"scm_delete_on_update"`
	ScmCredentialID       *bson.ObjectId `json:"credential"`
	ScmDeleteOnNextUpdate *bool          `json:"scm_delete_on_next_update"`
	ScmUpdateOnLaunch     *bool          `json:"scm_update_on_launch"`
	ScmUpdateCacheTimeout *int           `json:"scm_update_cache_timeout"`
}
