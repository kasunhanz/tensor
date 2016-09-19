package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)

const DBC_GROUPS = "groups"

// Organization is the model for organization
// collection
type Group struct {
	ID                       bson.ObjectId  `bson:"_id" json:"id"`
	Name                     string         `bson:"name" json:"name" binding:"required"`
	Description              string         `bson:"description" json:"description"`
	Variables                string         `bson:"variables" json:"variables"`
	TotalHosts               uint32         `bson:"total_hosts" json:"total_hosts"`
	HasActiveFailures        bool           `bson:"has_active_failures" json:"has_active_failures"`
	HostsWithActiveFailures  uint32         `bson:"hosts_with_active_failures" json:"hosts_with_active_failures"`
	TotalGroups              uint32         `bson:"total_groups" json:"total_groups"`
	GroupsWithActiveFailures uint32         `bson:"groups_with_active_failures" json:"groups_with_active_failures"`
	HasInventorySources      bool           `bson:"has_inventory_sources" json:"has_inventory_sources"`
	InventoryID              bson.ObjectId  `bson:"inventory_id" json:"inventory"`
	//parent child relation
	ParentGroupID            bson.ObjectId  `bson:"parent_group_id,omitempty" json:"-"`
	CreatedByID              bson.ObjectId  `bson:"created_by_id" json:"created_by"`
	ModifiedByID             bson.ObjectId  `bson:"modified_by_id" json:"modified_by"`
	Created                  time.Time      `bson:"created" json:"created"`
	Modified                 time.Time      `bson:"modified" json:"modified"`

	Type                     string         `bson:"-" json:"type"`
	Url                      string         `bson:"-" json:"url"`
	Related                  gin.H          `bson:"-" json:"related"`
	SummaryFields            gin.H          `bson:"-" json:"summary_fields"`
	LastJob                  gin.H          `bson:"-" json:"last_job"`
	LastJobHostSummary       gin.H          `bson:"-" json:"last_job_host_summary"`
}

func (g Group) CreateIndexes()  {

}