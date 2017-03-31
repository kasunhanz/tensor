package metadata

import (
	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/gin-gonic/gin"
)

// Create a new organization
func GroupMetadata(grp *ansible.Group) {

	ID := grp.ID.Hex()
	grp.Type = "group"
	grp.Links = gin.H{
		"self":               "/v1/groups/" + ID,
		"created_by":         "/v1/users/" + grp.CreatedByID.Hex(),
		"job_host_summaries": "/v1/groups/" + ID + "job_host_summaries",
		"variable_data":      "/v1/groups/" + ID + "/variable_data",
		"job_events":         "/v1/groups/" + ID + "/job_events",
		"potential_children": "/v1/groups/" + ID + "/potential_children",
		"ad_hoc_commands":    "/v1/groups/" + ID + "/ad_hoc_commands",
		"all_hosts":          "/v1/groups/" + ID + "/all_hosts",
		"activity_stream":    "/v1/groups/" + ID + "/activity_stream",
		"hosts":              "/v1/groups/" + ID + "/hosts",
		"children":           "/v1/groups/" + ID + "/children",
		"inventory_sources":  "/v1/groups/" + ID + "/inventory_sources",
		"inventory":          "/v1/inventories/" + grp.InventoryID.Hex(),
		"inventory_source":   "/v1/inventory_sources/emptyid",
	}

	groupSummary(grp)
}

func groupSummary(grp *ansible.Group) {

	var modified common.User
	var created common.User
	var inv ansible.Inventory

	summary := gin.H{
		"inventory": nil,
		"inventory_source": gin.H{
			"source": "",
			"status": "none",
		},
		"modified_by": nil,
		"created_by":  nil,
	}

	if err := db.Users().FindId(grp.CreatedByID).One(&created); err != nil {
		logrus.WithFields(logrus.Fields{
			"User ID":  grp.CreatedByID.Hex(),
			"Group":    grp.Name,
			"Group ID": grp.ID.Hex(),
		}).Errorln("Error while getting created by User")
	} else {
		summary["created_by"] = gin.H{
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		}
	}

	if err := db.Users().FindId(grp.ModifiedByID).One(&modified); err != nil {
		logrus.WithFields(logrus.Fields{
			"User ID":  grp.ModifiedByID.Hex(),
			"Group":    grp.Name,
			"Group ID": grp.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	} else {
		summary["modified_by"] = gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	if err := db.Inventories().FindId(grp.InventoryID).One(&inv); err != nil {
		logrus.WithFields(logrus.Fields{
			"Inventory ID": grp.InventoryID.Hex(),
			"Group":        grp.Name,
			"Group ID":     grp.ID.Hex(),
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

	grp.Meta = summary
}
