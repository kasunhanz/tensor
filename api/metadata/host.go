package metadata

import (
	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/gin-gonic/gin"
)

// Create a new organization
func HostMetadata(host *ansible.Host) {

	ID := host.ID.Hex()
	host.Type = "host"
	host.Links = gin.H{
		"self":                  "/v1/hosts/" + ID,
		"created_by":            "/v1/users/" + host.CreatedByID.Hex(),
		"modified_by":           "/v1/users/" + host.CreatedByID.Hex(),
		"job_host_summaries":    "/v1/hosts/" + ID + "/job_host_summaries",
		"variable_data":         "/v1/hosts/" + ID + "/variable_data",
		"job_events":            "/v1/hosts/" + ID + "/job_events",
		"ad_hoc_commands":       "/v1/hosts/" + ID + "/ad_hoc_commands",
		"fact_versions":         "/v1/hosts/" + ID + "/fact_versions",
		"inventory_sources":     "/v1/hosts/" + ID + "/inventory_sources",
		"groups":                "/v1/hosts/" + ID + "/groups",
		"activity_stream":       "/v1/hosts/" + ID + "/activity_stream",
		"all_groups":            "/v1/hosts/" + ID + "/all_groups",
		"ad_hoc_command_events": "/v1/hosts/" + ID + "/ad_hoc_command_events",
		"inventory":             "/v1/inventories/" + host.InventoryID.Hex(),
	}

	hostSummary(host)
}

func hostSummary(host *ansible.Host) {

	var modified common.User
	var created common.User
	var inv ansible.Inventory

	summary := gin.H{
		"recent_jobs": []gin.H{}, //TODO: recent_jobs
		"inventory":   nil,
		"modified_by": nil,
		"created_by":  nil,
	}

	if err := db.Users().FindId(host.CreatedByID).One(&created); err != nil {
		logrus.WithFields(logrus.Fields{
			"User ID": host.CreatedByID.Hex(),
			"Host":    host.Name,
			"Host ID": host.ID.Hex(),
		}).Errorln("Error while getting created by User")
	} else {
		summary["created_by"] = gin.H{
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		}
	}

	if err := db.Users().FindId(host.ModifiedByID).One(&modified); err != nil {
		logrus.WithFields(logrus.Fields{
			"User ID": host.ModifiedByID.Hex(),
			"Host":    host.Name,
			"Host ID": host.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	} else {
		summary["modified_by"] = gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	if err := db.Inventories().FindId(host.InventoryID).One(&inv); err != nil {
		logrus.WithFields(logrus.Fields{
			"Inventory ID": host.InventoryID.Hex(),
			"Host":         host.Name,
			"Host ID":      host.ID.Hex(),
		}).Errorln("Error while getting Inventory")
	} else {
		summary["inventory"] = gin.H{
			"id":                              inv.ID,
			"name":                            inv.Name,
			"description":                     inv.Description,
			"has_active_failures":             inv.HasActiveFailures,
			"total_hosts":                     inv.TotalHosts,
			"hosts_with_active_failures":      inv.HostsWithActiveFailures,
			"total_groups":                    inv.TotalGroups,
			"groups_with_active_failures":     inv.GroupsWithActiveFailures,
			"has_inventory_sources":           inv.HasInventorySources,
			"total_inventory_sources":         inv.TotalInventorySources,
			"inventory_sources_with_failures": inv.InventorySourcesWithFailures,
		}
	}

	host.Meta = summary
}
