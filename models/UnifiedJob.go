package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

const DBC_UNIFIED_JOB = "unified_jobs"

type UnifiedJob struct {
	ID                   bson.ObjectId  `bson:"_id" json:"id"`

	Name                 string         `bson:"name" json:"name"`
	Description          string         `bson:"description" json:"description"`
	LaunchType           string         `bson:"launch_type" json:"launch_type"`
	CancelFlag           bool           `bson:"cancel_flag" json:"cancel_flag"`
	Status               bool           `bson:"status" json:"status"`
	Failed               bool           `bson:"failed" json:"failed"`
	Started              time.Time      `bson:"started" json:"started"`
	Finished             time.Time      `bson:"finished" json:"finished"`
	Elapsed              uint32         `bson:"finished" json:"finished"`
	JobArgs              string         `bson:"job_args" json:"job_args"`
	JobCWD               string         `bson:"job_cwd" json:"job_cwd"`
	JobENV               string         `bson:"job_env" json:"job_env"`
	JobExplanation       string         `bson:"job_explanation" json:"job_explanation"`
	StartArgs            string         `bson:"start_args" json:"start_args"`
	ResultStdoutText     string         `bson:"result_stdout_text" json:"result_stdout_text"`
	ResultStdoutFile     string         `bson:"result_stdout_file" json:"result_stdout_file"`
	ResultTraceback      string         `bson:"result_traceback" json:"result_traceback"`
	CeleryTaskID         bson.ObjectId  `bson:"celery_task_id" json:"celery_task_id"`
	PolymorphicCtypeID   bson.ObjectId  `bson:"polymorphic_ctype_id" json:"polymorphic_ctype_id"`
	ScheduleID           time.Time      `bson:"schedule_id" json:"schedule_id"`
	UnifiedJobTemplateID time.Time      `bson:"unified_job_template_id" json:"unified_job_template_id"`
	CreatedByID          time.Time      `bson:"created_by_id" json:"created_by"`
	ModifiedByID         time.Time      `bson:"modified_by_id" json:"modified_by"`
	Created              time.Time      `bson:"created" json:"created"`
	Modified             time.Time      `bson:"modified" json:"modified"`

	Type                 string         `bson:"-" json:"type"`
	Url                  string         `bson:"-" json:"url"`
	Related              gin.H          `bson:"-" json:"related"`
	SummaryFields        gin.H          `bson:"-" json:"summary_fields"`
}