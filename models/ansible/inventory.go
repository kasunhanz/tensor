package ansible

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// Inventory is the model for
// Inventory collection
type Inventory struct {
	ID bson.ObjectId `bson:"_id" json:"id"`

	Type  string `bson:"-" json:"type"`
	Links gin.H  `bson:"-" json:"links"`
	Meta  gin.H  `bson:"-" json:"meta"`

	// required feilds
	Name           string        `bson:"name" json:"name" binding:"required,min=1,max=500"`
	OrganizationID bson.ObjectId `bson:"organization_id" json:"organization" binding:"required"`
	Description    string        `bson:"description,omitempty" json:"description"`
	Variables      string        `bson:"variables,omitempty" json:"variables"`

	// only output
	TotalHosts                   uint32 `bson:"total_hosts,omitempty" json:"total_hosts" binding:"omitempty,naproperty"`
	HostsWithActiveFailures      uint32 `bson:"hosts_with_active_failures,omitempty" json:"hosts_with_active_failures" binding:"omitempty,naproperty"`
	TotalGroups                  uint32 `bson:"total_groups,omitempty" json:"total_groups" binding:"omitempty,naproperty"`
	GroupsWithActiveFailures     uint32 `bson:"groups_with_active_failures,omitempty" json:"groups_with_active_failures" binding:"omitempty,naproperty"`
	TotalInventorySources        uint32 `bson:"total_inventory_sources,omitempty" json:"total_inventory_sources" binding:"omitempty,naproperty"`
	InventorySourcesWithFailures uint32 `bson:"inventory_sources_with_failures,omitempty" json:"inventory_sources_with_failures" binding:"omitempty,naproperty"`

	HasInventorySources bool `bson:"has_inventory_sources" json:"has_inventory_sources" binding:"omitempty,naproperty"`
	HasActiveFailures   bool `bson:"has_active_failures" json:"has_active_failures" binding:"omitempty,naproperty"`

	CreatedByID  bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created  time.Time `bson:"created" json:"created"`
	Modified time.Time `bson:"modified" json:"modified"`

	Roles []common.AccessControl `bson:"roles" json:"-"`
}

func (Inventory) GetType() string {
	return "inventory"
}

func (inv *Inventory) IsUnique() bool {
	count, err := db.Inventories().Find(bson.M{"name": inv.Name, "organization_id": inv.OrganizationID}).Count()
	if err == nil && count > 0 {
		return false
	}
	return true
}

func (inv *Inventory) Exist() bool {
	count, err := db.Inventories().FindId(inv.ID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (inv *Inventory) OrganizationExist() bool {
	count, err := db.Organizations().FindId(inv.OrganizationID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (inv Inventory) GetRoles() []common.AccessControl {
	return inv.Roles
}
