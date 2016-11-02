package metadata

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
)

// Create a new organization
func HostMetadata(host *models.Host) error {

	ID := host.ID.Hex()
	host.Type = "host"
	host.Url = "/v1/hosts/" + ID + "/"
	host.Related = gin.H{
		"created_by": "/v1/users/" + host.CreatedByID.Hex() + "/",
		"modified_by": "/v1/users/" + host.CreatedByID.Hex() + "/",
		"job_host_summaries": "/v1/hosts/" + ID + "/job_host_summaries/",
		"variable_data": "/v1/hosts/" + ID + "/variable_data/",
		"job_events": "/v1/hosts/" + ID + "/job_events/",
		"ad_hoc_commands": "/v1/hosts/" + ID + "/ad_hoc_commands/",
		"fact_versions": "/v1/hosts/" + ID + "/fact_versions/",
		"inventory_sources": "/v1/hosts/" + ID + "/inventory_sources/",
		"groups": "/v1/hosts/" + ID + "/groups/",
		"activity_stream": "/v1/hosts/" + ID + "/activity_stream/",
		"all_groups": "/v1/hosts/" + ID + "/all_groups/",
		"ad_hoc_command_events": "/v1/hosts/" + ID + "/ad_hoc_command_events/",
		"inventory": "/v1/inventories/" + host.InventoryID + "/",
	}

	if err := hostSummary(host); err != nil {
		return err
	}

	return nil
}

func hostSummary(host *models.Host) error {

	var modified models.User
	var created models.User
	var inv models.Inventory

	if err := db.Users().FindId(host.CreatedByID).One(&created); err != nil {
		return err
	}

	if err := db.Users().FindId(host.ModifiedByID).One(&modified); err != nil {
		return err
	}

	if err := db.Inventories().FindId(host.InventoryID).One(&inv); err != nil {
		return err
	}

	host.SummaryFields = gin.H{
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
		"recent_jobs": []gin.H{},
	}

	return nil
}
