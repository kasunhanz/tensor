package inventories

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
)



// Create a new organization
func setMetadata(i *models.Inventory) error {

	ID := i.ID.Hex()
	i.Type = "inventory"
	i.Url = "/v1/inventories/" + ID + "/"
	i.Related = gin.H{
		"created_by": "/v1/users/" + i.CreatedBy.Hex() + "/",
		"job_templates": "/v1/inventories/" + ID + "/job_templates/",
		"scan_job_templates": "/v1/inventories/" + ID + "/scan_job_templates/",
		"variable_data": "/v1/inventories/" + ID + "/variable_data/",
		"root_groups": "/v1/inventories/" + ID + "/root_groups/",
		"object_roles": "/v1/inventories/" + ID + "/object_roles/",
		"ad_hoc_commands": "/v1/inventories/" + ID + "/ad_hoc_commands/",
		"script": "/v1/inventories/" + ID + "/script/",
		"tree": "/v1/inventories/" + ID + "/tree/",
		"access_list": "/v1/inventories/" + ID + "/access_list/",
		"hosts": "/v1/inventories/" + ID + "/hosts/",
		"groups": "/v1/inventories/" + ID + "/groups/",
		"activity_stream": "/v1/inventories/" + ID + "/activity_stream/",
		"inventory_sources": "/v1/inventories/" + ID + "/inventory_sources/",
		"organization": "/v1/organizations/" + i.Organization.Hex() + "/",
	}

	if err := setSummaryFields(i); err != nil {
		return err
	}

	return nil
}

func setSummaryFields(i *models.Inventory) error {

	dbu := db.MongoDb.C(models.DBC_USERS)
	dbco := db.MongoDb.C(models.DBC_ORGANIZATIONS)

	var modified models.User
	var created models.User
	var org models.Organization

	if err := dbu.FindId(i.CreatedBy).One(&created); err != nil {
		return err
	}

	if err := dbu.FindId(i.ModifiedBy).One(&modified); err != nil {
		return err
	}

	if err := dbco.FindId(i.Organization).One(&org); err != nil {
		return err
	}

	//TODO: fill these from database
	i.HasActiveFailures = false
	i.TotalHosts = 6
	i.HostsWithActiveFailures = 0
	i.TotalGroups = 2
	i.GroupsWithActiveFailures = 0
	i.HasInventorySources = false
	i.TotalInventorySources = 0
	i.InventorySourcesWithFailures = 0

	i.SummaryFields = gin.H{
		"object_roles": []gin.H{
			{
				"description": "Can use the inventory in a job template",
				"name": "use",
			},
			{
				"description": "Can manage all aspects of the inventory",
				"name": "admin",
			},
			{
				"description": "May run ad hoc commands on an inventory",
				"name": "adhoc",
			},
			{
				"description": "May update project or inventory or group using the configured source update system",
				"name": "update",
			},
			{
				"description": "May view settings for the inventory",
				"name": "read",
			},
		},
		"organization": gin.H{
			"id": org.ID.Hex(),
			"name": org.Name,
			"description": org.Description,
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
