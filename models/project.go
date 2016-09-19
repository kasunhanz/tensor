package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)

const DBC_PROJECTS = "projects"

// Project is the model for project
// collection
type Project struct {
	ID                    bson.ObjectId  `bson:"_id" json:"id"`
	Type                  string         `bson:"-" json:"type"`
	Url                   string         `bson:"-" json:"url"`
	Related               gin.H          `bson:"-" json:"related"`
	SummaryFields         gin.H          `bson:"-" json:"summary_fields"`
	Name                  string         `bson:"name" json:"name" binding:"required"`
	Description           string         `bson:"description" json:"description"`
	LocalPath             string         `bson:"local_path" json:"local_path"`
	ScmType               string         `bson:"scm_type" json:"scm_type" binding:"required"`
	ScmUrl                string         `bson:"scm_url" json:"scm_url" binding:"required"`
	ScmBranch             string         `bson:"scm_branch" json:"scm_branch"`
	ScmClean              bool           `bson:"scm_clean" json:"scm_clean"`
	ScmDeleteOnUpdate     bool           `bson:"scm_delete_on_update" json:"scm_delete_on_update"`
	ScmCredential         bson.ObjectId  `bson:"credentail" json:"credential"`
	LastJob               bson.ObjectId  `bson:"last_job,omitempty" json:"last_job"`
	LastJobRun            time.Time      `bson:"last_job_run" json:"last_job_run"`
	LastJobFailed         bool           `bson:"last_job_failed" json:"last_job_failed"`
	HasSchedules          bool           `bson:"has_schedules" json:"has_schedules"`
	NextJobRun            time.Time      `bson:"next_job_run" json:"next_job_run"`
	Status                string         `bson:"status" json:"status"`
	Organization          bson.ObjectId  `bson:"organization" json:"organization" binding:"required"`
	ScmDeleteOnNextUpdate bool           `bson:"scm_delete_on_next_update" json:"scm_delete_on_next_update"`
	ScmUpdateOnLaunch     bool           `bson:"scm_update_on_launch" json:"scm_update_on_launch"`
	ScmUpdateCacheTimeout int            `bson:"scm_update_cache_timeout" json:"scm_update_cache_timeout"`
	LastUpdateFailed      bool           `bson:"last_update_failed" json:"last_update_failed"`
	LastUpdated           time.Time      `bson:"last_updated" json:"last_updated"`
	CreatedBy             bson.ObjectId  `bson:"created_by" json:"created_by"`
	ModifiedBy            bson.ObjectId  `bson:"modified_by" json:"modified_by"`
	Created               time.Time      `bson:"created" json:"created"`
	Modified              time.Time      `bson:"modified" json:"modified"`
}


func (p Project) CreateIndexes()  {

}