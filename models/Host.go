package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

type Host struct {
	ID                   bson.ObjectId  `bson:"_id" json:"id"`

	// required
	Name                 string         `bson:"name" json:"name" binding:"required"`
	InventoryID          bson.ObjectId  `bson:"inventory_id" json:"inventory"`

	Description          *string         `bson:"description,omitempty" json:"description"`
	GroupID              *bson.ObjectId  `bson:"group_id,omitempty" json:"inventory"`
	InstanceID           *string         `bson:"instance_id,omitempty" json:"instance_id"`
	Variables            *string         `bson:"variables,omitempty" json:"variables"`
	LastJobID            *bson.ObjectId  `bson:"last_job_id,omitempty,omitempty" json:"last_job"`
	LastJobHostSummaryID *bson.ObjectId  `bson:"last_job_host_summary_id,omitempty" json:"last_job_host_summary"`

	HasActiveFailures    bool           `bson:"has_active_failures,omitempty" json:"has_active_failures"`
	HasInventorySources  bool           `bson:"has_inventory_sources,omitempty" json:"has_inventory_sources"`
	Enabled              bool           `bson:"enabled,omitempty" json:"enabled"`

	CreatedByID          bson.ObjectId  `bson:"created_by_id" json:"-"`
	ModifiedByID         bson.ObjectId  `bson:"modified_by_id" json:"-"`

	Created              time.Time      `bson:"created" json:"created"`
	Modified             time.Time      `bson:"modified" json:"modified"`

	Type                 string         `bson:"-" json:"type"`
	Url                  string         `bson:"-" json:"url"`
	Related              gin.H          `bson:"-" json:"related"`
	SummaryFields        gin.H          `bson:"-" json:"summary_fields"`
}