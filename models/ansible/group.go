package ansible

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// Group is the model for Group collection
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

	CreatedByID              bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID             bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created                  time.Time `bson:"created" json:"created"`
	Modified                 time.Time `bson:"modified" json:"modified"`

	Type                     string  `bson:"-" json:"type"`
	Children                 []Group `bson:"-" json:"children,omitempty"`
	Links                    gin.H   `bson:"-" json:"links"`
	Meta                     gin.H   `bson:"-" json:"meta"`
	LastJob                  gin.H   `bson:"-" json:"last_job"`
	LastJobHostSummary       gin.H   `bson:"-" json:"last_job_host_summary"`
}

func (*Group) GetType() string {
	return "group"
}

// GetParent returns the parent of the given Group
func (group *Group) GetParent() (Group, error) {
	var grp Group
	err := db.Groups().Find(bson.M{"_id": group.ParentGroupID}).One(&grp)
	return grp, err
}
func (grp Group) GetInventory() (inv Inventory, err error) {
	err = db.Inventories().FindId(grp.InventoryID).One(&inv)
	return inv, err
}

func (group *Group) IsUnique() bool {
	count, err := db.Groups().Find(bson.M{"name": group.Name, "inventory_id": group.InventoryID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (group *Group) GroupExist() bool {
	count, err := db.Groups().FindId(group.ID).Count()
	if err == nil && count == 1 {
		return true
	}
	return false
}

func (group *Group) ParentExist() bool {
	count, err := db.Groups().FindId(group.ParentGroupID).Count()
	if err == nil && count == 1 {
		return true
	}
	return false
}

func (group *Group) InventoryExist() bool {
	count, err := db.Inventories().FindId(group.InventoryID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}