package groups

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
)



// Create a new organization
func setMetadata(grp *models.Group) error {

	ID := grp.ID.Hex()
	grp.Type = "inventory"
	grp.Url = "/v1/inventories/" + ID + "/"
	grp.Related = gin.H{
		"created_by": "/api/v1/users/1/",
		"job_host_summaries": "/api/v1/groups/2/job_host_summaries/",
		"variable_data": "/api/v1/groups/2/variable_data/",
		"job_events": "/api/v1/groups/2/job_events/",
		"potential_children": "/api/v1/groups/2/potential_children/",
		"ad_hoc_commands": "/api/v1/groups/2/ad_hoc_commands/",
		"all_hosts": "/api/v1/groups/2/all_hosts/",
		"activity_stream": "/api/v1/groups/2/activity_stream/",
		"hosts": "/api/v1/groups/2/hosts/",
		"children": "/api/v1/groups/2/children/",
		"inventory_sources": "/api/v1/groups/2/inventory_sources/",
		"inventory": "/api/v1/inventories/1/",
		"inventory_source": "/api/v1/inventory_sources/7/",
	}

	if err := setSummaryFields(grp); err != nil {
		return err
	}

	return nil
}

func setSummaryFields(grp *models.Group) error {

	dbu := db.MongoDb.C(models.DBC_USERS)
	dbci := db.MongoDb.C(models.DBC_INVENTORIES)

	var modified models.User
	var created models.User
	var inv models.Inventory

	if err := dbu.FindId(grp.CreatedByID).One(&created); err != nil {
		return err
	}

	if err := dbu.FindId(grp.ModifiedByID).One(&modified); err != nil {
		return err
	}

	if err := dbci.FindId(grp.InventoryID).One(&inv); err != nil {
		return err
	}

	grp.SummaryFields = gin.H{
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
