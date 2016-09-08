package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
	"github.com/gin-gonic/gin"
)

const DBC_UNIFIED_JOB_TEMPLATES = "unified_job_templates"

type UnifiedJobTemplate struct {
	ID                 bson.ObjectId  `bson:"_id" json:"id"`

	Description        string        `bson:"description" json:"description"`
	Name               string        `bson:"name" json:"name"`
	LastJobFailed      bool          `bson:"last_job_failed" json:"last_job_failed"`
	LastJobRun         time.Time     `bson:"last_job_run" json:"last_job_run"`
	HasSchedules       bool          `bson:"has_schedules" json:"has_schedules"`
	NextJobRun         time.Time     `bson:"next_job_run" json:"next_job_run"`
	Status             string        `bson:"status" json:"status"`
	CurrentJobID       bson.ObjectId `bson:"current_job_id" json:"current_job_id"`
	LastJobID          bson.ObjectId `bson:"last_job_id" json:"last_job_id"`
	NextScheduleID     bson.ObjectId `bson:"next_schedule_id" json:"next_schedule_id"`
	PolymorphicCtypeID bson.ObjectId `bson:"polymorphic_ctype_id" json:"polymorphic_ctype_id"`

	CreatedByID        time.Time     `bson:"created_by_id" json:"created_by"`
	ModifiedByID       time.Time     `bson:"modified_by_id" json:"modified_by"`
	Created            time.Time     `bson:"created" json:"created"`
	Modified           time.Time     `bson:"modified" json:"modified"`

	Type               string        `bson:"-" json:"type"`
	Url                string        `bson:"-" json:"url"`
	Related            gin.H         `bson:"-" json:"related"`
	SummaryFields      gin.H         `bson:"-" json:"summary_fields"`
}