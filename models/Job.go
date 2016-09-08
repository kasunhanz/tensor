package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

const DBC_JOBS = "jobs"

type Job struct {
	ID                  bson.ObjectId  `bson:"_id" json:"id"`
	JobType             string         `bson:"job_type" json:"job_type"`
	Playbook            string         `bson:"playbook" json:"playbook"`
	Forks               uint8          `bson:"forks" json:"forks"`
	Limit               string         `bson:"limit" json:"limit"`
	Verbosity           uint8          `bson:"verbosity" json:"verbosity"`
	ExtraVars           string         `bson:"extra_vars" json:"extra_vars"`
	JobTags             string         `bson:"job_tags" json:"job_tags"`
	ForceHandlers       bool           `bson:"force_handlers" json:"force_handlers"`
	SkipTags            string         `bson:"skip_tags" json:"skip_tags"`
	StartAtTask         string         `bson:"start_at_task" json:"start_at_task"`
	BecomeEnabled       bool           `bson:"become_enabled" json:"become_enabled"`
	CredentialID        bson.ObjectId  `bson:"credential_id" json:"credential_id"`
	InventoryID         time.Time      `bson:"inventory_id" json:"inventory_id"`
	JobTemplateID       time.Time      `bson:"job_template_id" json:"job_template_id"`
	ProjectID           time.Time      `bson:"project_id" json:"project_id"`
	NetworkCredentialID time.Time      `bson:"network_credential_id" json:"network_credential_id"`
	CreatedByID         time.Time      `bson:"created_by_id" json:"created_by"`
	ModifiedByID        time.Time      `bson:"modified_by_id" json:"modified_by"`
	Created             time.Time      `bson:"created" json:"created"`
	Modified            time.Time      `bson:"modified" json:"modified"`

	Type                string         `bson:"-" json:"type"`
	Url                 string         `bson:"-" json:"url"`
	Related             gin.H          `bson:"-" json:"related"`
	SummaryFields       gin.H          `bson:"-" json:"summary_fields"`
}