package ansible

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

type Host struct {
	ID bson.ObjectId `bson:"_id" json:"id"`

	// required
	Name        string         `bson:"name" json:"name" binding:"required,iphost"`
	InventoryID bson.ObjectId  `bson:"inventory_id" json:"inventory" binding:"required"`
	Description string         `bson:"description,omitempty" json:"description"`
	GroupID     *bson.ObjectId `bson:"group_id,omitempty" json:"group"`
	InstanceID  string         `bson:"instance_id,omitempty" json:"instance_id"`
	Variables   string         `bson:"variables,omitempty" json:"variables"`
	Enabled     bool           `bson:"enabled,omitempty" json:"enabled"`

	LastJobID            *bson.ObjectId `bson:"last_job_id,omitempty" json:"last_job" binding:"omitempty,naproperty"`
	LastJobHostSummaryID *bson.ObjectId `bson:"last_job_host_summary_id,omitempty" json:"last_job_host_summary" binding:"omitempty,naproperty"`

	HasActiveFailures   bool          `bson:"has_active_failures,omitempty" json:"has_active_failures" binding:"omitempty,naproperty"`
	HasInventorySources bool          `bson:"has_inventory_sources,omitempty" json:"has_inventory_sources" binding:"omitempty,naproperty"`
	CreatedByID         bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID        bson.ObjectId `bson:"modified_by_id" json:"-"`
	Created             time.Time     `bson:"created" json:"created" binding:"omitempty,naproperty"`
	Modified            time.Time     `bson:"modified" json:"modified" binding:"omitempty,naproperty"`

	Type  string `bson:"-" json:"type"`
	Links gin.H  `bson:"-" json:"links"`
	Meta  gin.H  `bson:"-" json:"meta"`
}

func (*Host) GetType() string {
	return "host"
}

func (h Host) GetInventory() (Inventory, error) {
	var inv Inventory
	err := db.Inventories().FindId(h.InventoryID).One(&inv)
	return inv, err
}

func (host *Host) IsUnique() bool {
	count, err := db.Hosts().Find(bson.M{"name": host.Name, "inventory_id": host.InventoryID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (host *Host) InventoryExist() bool {
	count, err := db.Inventories().FindId(host.InventoryID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (host *Host) GroupExist() bool {
	count, err := db.Groups().FindId(host.GroupID).Count()
	if err == nil && count == 1 {
		return true
	}
	return false
}
