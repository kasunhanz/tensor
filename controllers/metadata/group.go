package metadata

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
)


// Create a new organization
func GroupMetadata(grp *models.Group) error {

	ID := grp.ID.Hex()
	grp.Type = "inventory"
	grp.Url = "/v1/inventories/" + ID + "/"
	grp.Related = gin.H{
		"created_by": "/v1/users/1/",
		"job_host_summaries": "/v1/groups/2/job_host_summaries/",
		"variable_data": "/v1/groups/2/variable_data/",
		"job_events": "/v1/groups/2/job_events/",
		"potential_children": "/v1/groups/2/potential_children/",
		"ad_hoc_commands": "/v1/groups/2/ad_hoc_commands/",
		"all_hosts": "/v1/groups/2/all_hosts/",
		"activity_stream": "/v1/groups/2/activity_stream/",
		"hosts": "/v1/groups/2/hosts/",
		"children": "/v1/groups/2/children/",
		"inventory_sources": "/v1/groups/2/inventory_sources/",
		"inventory": "/v1/inventories/1/",
		"inventory_source": "/v1/inventory_sources/7/",
	}

	if err := groupSummary(grp); err != nil {
		return err
	}

	return nil
}

func groupSummary(grp *models.Group) error {

	var modified models.User
	var created models.User
	var inv models.Inventory

	if err := db.Users().FindId(grp.CreatedByID).One(&created); err != nil {
		return err
	}

	if err := db.Users().FindId(grp.ModifiedByID).One(&modified); err != nil {
		return err
	}

	if err := db.Inventories().FindId(grp.InventoryID).One(&inv); err != nil {
		return err
	}

	grp.Summary = gin.H{
		"inventory": gin.H{
			"id": inv.ID,
			"name": inv.Name,
			"description": inv.Description,
			"has_active_failures": inv.HasActiveFailures,
			"total_hosts": inv.TotalHosts,
			"hosts_with_active_failures": inv.HostsWithActiveFailures,
			"total_groups": inv.TotalGroups,
			"groups_with_active_failures": inv.GroupsWithActiveFailures,
			"has_inventory_sources": inv.HasInventorySources,
			"total_inventory_sources": inv.TotalInventorySources,
			"inventory_sources_with_failures": inv.InventorySourcesWithFailures,
		},
		"inventory_source": gin.H{
			"source": "",
			"status": "none",
		},
		"modified_by": gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		},
		"created_by": gin.H{
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		},
	}

	return nil
}
