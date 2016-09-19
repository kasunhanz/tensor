package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

const DBC_INVENTORIES = "inventories"

// Inventory is the model for
// Inventory collection
type Inventory struct {
	ID                           bson.ObjectId `bson:"_id" json:"id"`

	Type                         string        `bson:"-" json:"type"`
	Url                          string        `bson:"-" json:"url"`
	Related                      gin.H         `bson:"-" json:"related"`
	SummaryFields                gin.H         `bson:"-" json:"summary_fields"`

	Name                         string        `bson:"name" json:"name" binding:"required"`
	Description                  string        `bson:"description" json:"description"`
	Variables                    string        `bson:"variables" json:"variables"`
	HasActiveFailures            bool          `bson:"has_active_failures" json:"has_active_failures"`
	TotalHosts                   uint32        `bson:"total_hosts" json:"total_hosts"`
	HostsWithActiveFailures      uint32        `bson:"hosts_with_active_failures" json:"hosts_with_active_failures"`
	TotalGroups                  uint32        `bson:"total_groups" json:"total_groups"`
	GroupsWithActiveFailures     uint32        `bson:"groups_with_active_failures" json:"groups_with_active_failures"`
	HasInventorySources          bool          `bson:"has_inventory_sources" json:"has_inventory_sources"`
	TotalInventorySources        uint32        `bson:"total_inventory_sources" json:"total_inventory_sources"`
	InventorySourcesWithFailures uint32        `bson:"inventory_sources_with_failures" json:"inventory_sources_with_failures"`
	Organization                 bson.ObjectId `bson:"organization" json:"organization"`
	CreatedBy                    bson.ObjectId `bson:"created_by" json:"created_by"`
	ModifiedBy                   bson.ObjectId `bson:"modified_by" json:"modified_by"`
	Created                      time.Time     `bson:"created" json:"created"`
	Modified                     time.Time     `bson:"modified" json:"modified"`
}


func (iv Inventory) CreateIndexes()  {

}