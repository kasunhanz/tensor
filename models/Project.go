package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)

// Project is the model for project
// collection
type Project struct {
	ID                    bson.ObjectId  `bson:"_id" json:"id"`

	Type                  string         `bson:"-" json:"type"`
	Url                   string         `bson:"-" json:"url"`
	Related               gin.H          `bson:"-" json:"related"`
	Summary               gin.H          `bson:"-" json:"summary_fields"`

	// required feilds
	Name                  string         `bson:"name" json:"name" binding:"required"`
	ScmType               string         `bson:"scm_type" json:"scm_type" binding:"required"`
	OrganizationID        bson.ObjectId  `bson:"organization_id" json:"organization" binding:"required"`

	Description           *string         `bson:"description,omitempty" json:"description"`
	LocalPath             *string         `bson:"local_path,omitempty" json:"local_path"`
	ScmUrl                *string         `bson:"scm_url,omitempty" json:"scm_url"`
	ScmBranch             *string         `bson:"scm_branch,omitempty" json:"scm_branch"`
	ScmClean              bool           `bson:"scm_clean,omitempty" json:"scm_clean"`
	ScmDeleteOnUpdate     bool           `bson:"scm_delete_on_update,omitempty" json:"scm_delete_on_update"`
	ScmCredential         *bson.ObjectId  `bson:"credentail,omitempty" json:"credential"`
	LastJob               *bson.ObjectId  `bson:"last_job,omitempty" json:"last_job"`
	LastJobRun            *time.Time      `bson:"last_job_run,omitempty" json:"last_job_run"`
	LastJobFailed         bool           `bson:"last_job_failed,omitempty" json:"last_job_failed"`
	HasSchedules          bool           `bson:"has_schedules,omitempty" json:"has_schedules"`
	NextJobRun            *time.Time      `bson:"next_job_run,omitempty" json:"next_job_run"`
	Status                *string         `bson:"status,omitempty" json:"status"`
	ScmDeleteOnNextUpdate bool           `bson:"scm_delete_on_next_update,omitempty" json:"scm_delete_on_next_update"`
	ScmUpdateOnLaunch     bool           `bson:"scm_update_on_launch,omitempty" json:"scm_update_on_launch"`
	ScmUpdateCacheTimeout *int            `bson:"scm_update_cache_timeout,omitempty" json:"scm_update_cache_timeout"`
	LastUpdateFailed      bool           `bson:"last_update_failed,omitempty" json:"last_update_failed"`
	LastUpdated           *time.Time      `bson:"last_updated,omitempty" json:"last_updated"`

	CreatedBy             bson.ObjectId  `bson:"created_by" json:"-"`
	ModifiedBy            bson.ObjectId  `bson:"modified_by" json:"-"`

	Created               time.Time      `bson:"created" json:"created"`
	Modified              time.Time      `bson:"modified" json:"modified"`

	Roles                 []AccessControl    `bson:"roles" json:"-"`
}