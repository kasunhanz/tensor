package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)


// Organization is the model for organization
// collection
type Group struct {
	ID                       bson.ObjectId  `bson:"_id" json:"id"`
	Name                     string         `bson:"name" json:"name" binding:"required,min=1,max=500"`
	Description              string         `bson:"description" json:"description"`
	Variables                string         `bson:"variables" json:"variables"`
	TotalHosts               uint32         `bson:"total_hosts" json:"total_hosts"`
	HasActiveFailures        bool           `bson:"has_active_failures" json:"has_active_failures"`
	HostsWithActiveFailures  uint32         `bson:"hosts_with_active_failures" json:"hosts_with_active_failures"`
	TotalGroups              uint32         `bson:"total_groups" json:"total_groups"`
	GroupsWithActiveFailures uint32         `bson:"groups_with_active_failures" json:"groups_with_active_failures"`
	HasInventorySources      bool           `bson:"has_inventory_sources" json:"has_inventory_sources"`
	InventoryID              bson.ObjectId  `bson:"inventory_id" json:"inventory"`
	ParentGroupID            *bson.ObjectId `bson:"parent_group_id,omitempty" json:"parent_group,omitempty"`

	CreatedByID              bson.ObjectId  `bson:"created_by_id" json:"-"`
	ModifiedByID             bson.ObjectId  `bson:"modified_by_id" json:"-"`

	Created                  time.Time      `bson:"created" json:"created"`
	Modified                 time.Time      `bson:"modified" json:"modified"`

	Type                     string         `bson:"-" json:"type"`
	Url                      string         `bson:"-" json:"url"`
	Related                  gin.H          `bson:"-" json:"related"`
	Summary                  gin.H          `bson:"-" json:"summary_fields"`
	LastJob                  gin.H          `bson:"-" json:"last_job"`
	LastJobHostSummary       gin.H          `bson:"-" json:"last_job_host_summary"`
}

type PatchGroup struct {
	Name          *string        `json:"name" binding:"omitempty,min=1,max=500"`
	Description   *string        `json:"description"`
	Variables     *string        `json:"variables"`
	InventoryID   *bson.ObjectId `json:"inventory"`
	ParentGroupID *bson.ObjectId `json:"parent_group"`
}