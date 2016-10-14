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
	Name                         string        `bson:"name" json:"name" binding:"required"`
	OrganizationID               bson.ObjectId `bson:"organization_id" json:"organization" binding:"required"`
	Description                  string        `bson:"description,omitempty" json:"description"`
	Variables                    string        `bson:"variables,omitempty" json:"variables"`

	// only output
	TotalHosts                   uint32        `bson:"total_hosts,omitempty" json:"total_hosts" binding:"naproperty"`
	HostsWithActiveFailures      uint32        `bson:"hosts_with_active_failures,omitempty" json:"hosts_with_active_failures" binding:"naproperty"`
	TotalGroups                  uint32        `bson:"total_groups,omitempty" json:"total_groups" binding:"naproperty"`
	GroupsWithActiveFailures     uint32        `bson:"groups_with_active_failures,omitempty" json:"groups_with_active_failures" binding:"naproperty"`
	TotalInventorySources        uint32        `bson:"total_inventory_sources,omitempty" json:"total_inventory_sources" binding:"naproperty"`
	InventorySourcesWithFailures uint32        `bson:"inventory_sources_with_failures,omitempty" json:"inventory_sources_with_failures" binding:"naproperty"`

	HasInventorySources          bool          `bson:"has_inventory_sources" json:"has_inventory_sources" binding:"naproperty"`
	HasActiveFailures            bool          `bson:"has_active_failures" json:"has_active_failures" binding:"naproperty"`

	CreatedBy                    bson.ObjectId `bson:"created_by" json:"-"`
	ModifiedBy                   bson.ObjectId `bson:"modified_by" json:"-"`

	Created                      time.Time     `bson:"created" json:"created"`
	Modified                     time.Time     `bson:"modified" json:"modified"`

	Roles                        []AccessControl    `bson:"roles" json:"-"`
}