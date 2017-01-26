package metadata

import (
	log "github.com/Sirupsen/logrus"
	"gopkg.in/gin-gonic/gin.v1"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
)

// Create a new organization
func InventoryMetadata(i *ansible.Inventory) {

	ID := i.ID.Hex()
	i.Type = "inventory"
	i.URL = "/v1/inventories/" + ID + "/"
	i.Related = gin.H{
		"created_by":         "/v1/users/" + i.CreatedByID.Hex() + "/",
		"job_templates":      "/v1/inventories/" + ID + "/job_templates/",
		"scan_job_templates": "/v1/inventories/" + ID + "/scan_job_templates/",
		"variable_data":      "/v1/inventories/" + ID + "/variable_data/",
		"root_groups":        "/v1/inventories/" + ID + "/root_groups/",
		"object_roles":       "/v1/inventories/" + ID + "/object_roles/",
		"ad_hoc_commands":    "/v1/inventories/" + ID + "/ad_hoc_commands/",
		"script":             "/v1/inventories/" + ID + "/script/",
		"tree":               "/v1/inventories/" + ID + "/tree/",
		"access_list":        "/v1/inventories/" + ID + "/access_list/",
		"hosts":              "/v1/inventories/" + ID + "/hosts/",
		"groups":             "/v1/inventories/" + ID + "/groups/",
		"activity_stream":    "/v1/inventories/" + ID + "/activity_stream/",
		"inventory_sources":  "/v1/inventories/" + ID + "/inventory_sources/",
		"organization":       "/v1/organizations/" + i.OrganizationID.Hex() + "/",
	}

	inventorySummary(i)
}

func inventorySummary(i *ansible.Inventory) {
	var modified common.User
	var created common.User
	var org common.Organization

	summary := gin.H{
		"has_active_failures":             i.HasActiveFailures,
		"total_hosts":                     i.TotalHosts,
		"hosts_with_active_failures":      i.HostsWithActiveFailures,
		"total_groups":                    i.TotalGroups,
		"groups_with_active_failures":     i.GroupsWithActiveFailures,
		"has_inventory_sources":           i.HasInventorySources,
		"total_inventory_sources":         i.TotalInventorySources,
		"inventory_sources_with_failures": i.InventorySourcesWithFailures,
		"object_roles": []gin.H{
			{
				"description": "Can use the inventory in a job template",
				"name":        "use",
			},
			{
				"description": "Can manage all aspects of the inventory",
				"name":        "admin",
			},
			{
				"description": "May run ad hoc commands on an inventory",
				"name":        "adhoc",
			},
			{
				"description": "May update project or inventory or group using the configured source update system",
				"name":        "update",
			},
			{
				"description": "May view settings for the inventory",
				"name":        "read",
			},
		},
		"created_by":   nil,
		"modified_by":  nil,
		"organization": nil,
	}

	if err := db.Users().FindId(i.CreatedByID).One(&created); err != nil {
		log.WithFields(log.Fields{
			"User ID":      i.CreatedByID.Hex(),
			"Inventory":    i.Name,
			"Inventory ID": i.ID.Hex(),
		}).Errorln("Error while getting created by User")
	} else {
		summary["created_by"] = gin.H{
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		}
	}

	if err := db.Users().FindId(i.ModifiedByID).One(&modified); err != nil {
		log.WithFields(log.Fields{
			"User ID":      i.CreatedByID.Hex(),
			"Inventory":    i.Name,
			"Inventory ID": i.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	} else {
		summary["modified_by"] = gin.H{
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		}
	}

	if err := db.Organizations().FindId(i.OrganizationID).One(&org); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": i.OrganizationID.Hex(),
			"Inventory":       i.Name,
			"Inventory ID":    i.ID.Hex(),
		}).Errorln("Error while getting Organization")
	} else {
		summary["organization"] = gin.H{
			"id":          org.ID.Hex(),
			"name":        org.Name,
			"description": org.Description,
		}
	}

	i.Summary = summary
}
