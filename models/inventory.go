package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

// Inventory is the model for
// Inventory collection
type Inventory struct {
	ID                           bson.ObjectId `bson:"_id" json:"id"`

	Type                         string        `bson:"-" json:"type"`
	Url                          string        `bson:"-" json:"url"`
	Related                      gin.H         `bson:"-" json:"related"`
	SummaryFields                gin.H         `bson:"-" json:"summary_fields"`

	// required feilds
	Name                         string        `bson:"name" json:"name" binding:"required,ip|"`
	OrganizationID               bson.ObjectId `bson:"organization_id" json:"organization" binding:"required"`

	Description                  string        `bson:"description,omitempty" json:"description"`
	Variables                    string        `bson:"variables,omitempty" json:"variables"`
	TotalHosts                   uint32        `bson:"total_hosts,omitempty" json:"total_hosts"`
	HostsWithActiveFailures      uint32        `bson:"hosts_with_active_failures,omitempty" json:"hosts_with_active_failures"`
	TotalGroups                  uint32        `bson:"total_groups,omitempty" json:"total_groups"`
	GroupsWithActiveFailures     uint32        `bson:"groups_with_active_failures,omitempty" json:"groups_with_active_failures"`
	TotalInventorySources        uint32        `bson:"total_inventory_sources,omitempty" json:"total_inventory_sources"`
	InventorySourcesWithFailures uint32        `bson:"inventory_sources_with_failures,omitempty" json:"inventory_sources_with_failures"`

	HasInventorySources          bool          `bson:"has_inventory_sources" json:"has_inventory_sources"`
	HasActiveFailures            bool          `bson:"has_active_failures" json:"has_active_failures"`

	CreatedBy                    bson.ObjectId `bson:"created_by" json:"-"`
	ModifiedBy                   bson.ObjectId `bson:"modified_by" json:"-"`

	Created                      time.Time     `bson:"created" json:"created"`
	Modified                     time.Time     `bson:"modified" json:"modified"`

	Roles                        []AccessControl    `bson:"roles" json:"-"`
}