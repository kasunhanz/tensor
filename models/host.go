package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

const DBC_HOSTS = "hosts"

type Host struct {
	ID                   bson.ObjectId  `bson:"_id" json:"id"`
	Name                 string         `bson:"name" json:"name" binding:"required"`
	Description          string         `bson:"description" json:"description"`
	InventoryID          bson.ObjectId  `bson:"inventory_id" json:"inventory"`
	Enabled              bool           `bson:"enabled" json:"enabled"`
	InstanceID           string         `bson:"instance_id,omitempty" json:"instance_id"`
	Variables            string         `bson:"variables" json:"variables"`
	HasActiveFailures    bool           `bson:"has_active_failures" json:"has_active_failures"`
	HasInventorySources  bool           `bson:"has_inventory_sources" json:"has_inventory_sources"`
	LastJobID            bson.ObjectId  `bson:"last_job_id,omitempty" json:"last_job"`
	LastJobHostSummaryID bson.ObjectId  `bson:"last_job_host_summary_id,omitempty" json:"last_job_host_summary"`
	CreatedByID          bson.ObjectId  `bson:"created_by_id" json:"created_by"`
	ModifiedByID         bson.ObjectId  `bson:"modified_by_id" json:"modified_by"`
	Created              time.Time      `bson:"created" json:"created"`
	Modified             time.Time      `bson:"modified" json:"modified"`

	Type                 string         `bson:"-" json:"type"`
	Url                  string         `bson:"-" json:"url"`
	Related              gin.H          `bson:"-" json:"related"`
	SummaryFields        gin.H          `bson:"-" json:"summary_fields"`
}


func (h Host) CreateIndexes()  {

}