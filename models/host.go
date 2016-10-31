package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

type Host struct {
	ID                   bson.ObjectId   `bson:"_id" json:"id"`

	// required
	Name                 string          `bson:"name" json:"name" binding:"required,iphost"`
	InventoryID          bson.ObjectId   `bson:"inventory_id" json:"inventory" binding:"required"`

	Description          string          `bson:"description,omitempty" json:"description,omitempty"`
	GroupID              *bson.ObjectId  `bson:"group_id,omitempty" json:"group,omitempty"`
	InstanceID           string          `bson:"instance_id,omitempty" json:"instance_id,omitempty"`
	Variables            string          `bson:"variables,omitempty" json:"variables,omitempty"`
	LastJobID            *bson.ObjectId  `bson:"last_job_id,omitempty" json:"last_job,omitempty"`
	LastJobHostSummaryID *bson.ObjectId  `bson:"last_job_host_summary_id,omitempty" json:"last_job_host_summary,omitempty"`

	HasActiveFailures    bool            `bson:"has_active_failures,omitempty" json:"has_active_failures,omitempty"`
	HasInventorySources  bool            `bson:"has_inventory_sources,omitempty" json:"has_inventory_sources,omitempty"`
	Enabled              bool            `bson:"enabled,omitempty" json:"enabled,omitempty"`

	CreatedByID          bson.ObjectId   `bson:"created_by_id" json:"-"`
	ModifiedByID         bson.ObjectId   `bson:"modified_by_id" json:"-"`

	Created              time.Time       `bson:"created,omitempty" json:"created,omitempty"`
	Modified             time.Time       `bson:"modified,omitempty" json:"modified,omitempty"`

	Type                 string          `bson:"-" json:"type"`
	Url                  string          `bson:"-" json:"url"`
	Related              gin.H           `bson:"-" json:"related"`
	SummaryFields        gin.H           `bson:"-" json:"summary_fields"`
}

type PatchHost struct {
	Name         string          `bson:"name,omitempty" json:"name,omitempty" binding:"iphost"`
	InventoryID  bson.ObjectId   `bson:"inventory_id,omitempty" json:"inventory,omitempty"`
	Description  string          `bson:"description,omitempty" json:"description,omitempty"`
	GroupID      *bson.ObjectId  `bson:"group_id,omitempty" json:"group,omitempty"`
	InstanceID   string          `bson:"instance_id,omitempty" json:"instance_id,omitempty"`
	Variables    string          `bson:"variables,omitempty" json:"variables,omitempty"`
	Enabled      *bool            `bson:"enabled,omitempty" json:"enabled,omitempty"`

	ModifiedByID bson.ObjectId   `bson:"modified_by_id" json:"-"`
	Modified     time.Time       `bson:"modified,omitempty" json:"-"`
}