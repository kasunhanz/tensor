package metadata

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
)



// Create a new organization
func JTemplateMetadata(jt *models.JobTemplate) error {

	ID := jt.ID.Hex()
	jt.Type = "inventory"
	jt.Url = "/v1/inventories/" + ID + "/"
	related := gin.H{
		"created_by": "/v1/users/" + jt.CreatedByID.Hex() + "/",
		"modified_by": "/v1/users/" + jt.ModifiedByID.Hex() + "/",
		"labels": "/v1/job_templates/" + ID + "/labels/",
		"inventory": "/v1/inventories/" + jt.InventoryID.Hex() + "/",
		"credential": "v1/credentials/" + jt.MachineCredentialID.Hex() + "/",
		"project": "/v1/projects/" + jt.ProjectID.Hex() + "/",
		"notification_templates_error": "/v1/job_templates/" + ID + "/notification_templates_error/",
		"notification_templates_success": "/v1/job_templates/" + ID + "/notification_templates_success/",
		"jobs": "/v1/job_templates/" + ID + "/jobs/",
		"object_roles": "/v1/job_templates/" + ID + "/object_roles/",
		"notification_templates_any": "/v1/job_templates/" + ID + "/notification_templates_any/",
		"access_list": "/v1/job_templates/" + ID + "/access_list/",
		"launch": "/v1/job_templates/" + ID + "/launch/",
		"schedules": "/v1/job_templates/" + ID + "/schedules/",
		"activity_stream": "/v1/job_templates/" + ID + "/activity_stream/",
	}

	if jt.CurrentJobID != nil {
		related["current_job"] = "v1/jobs/" + jt.CurrentJobID.Hex() + "/"
	}

	jt.Related = related

	if err := jTemplateSummary(jt); err != nil {
		return err
	}

	return nil
}

func jTemplateSummary(jt *models.JobTemplate) error {

	var modified models.User
	var created models.User
	var inv models.Inventory
	var job models.Job
	var cjob models.Job
	var cred models.Credential
	var proj models.Project

	if err := db.Users().FindId(jt.CreatedByID).One(&created); err != nil {
		return err
	}

	if err := db.Users().FindId(jt.ModifiedByID).One(&modified); err != nil {
		return err
	}

	if err := db.Inventories().FindId(jt.InventoryID).One(&inv); err != nil {
		return err
	}

	if err := db.Credentials().FindId(jt.MachineCredentialID).One(&cred); err != nil {
		return err
	}

	if err := db.Projects().FindId(jt.ProjectID).One(&proj); err != nil {
		return err
	}

	summary := gin.H{
		"object_roles": []gin.H{
			{
				"Description":"Can manage all aspects of the job template",
				"Name":"admin",
			},
			{
				"Description":"May run the job template",
				"Name":"execute",
			},
			{
				"Description":"May view settings for the job template",
				"Name":"read",
			},
		},
		"current_update": gin.H{
			"id": job.ID,
			"name": job.Name,
			"description": job.Description,
			"status": job.Status,
			"failed": job.Failed,
		},
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
		"current_job":  gin.H{},
		"credential": gin.H{
			"id": cred.ID,
			"name": cred.Name,
			"description": cred.Description,
			"kind": cred.Kind,
			"cloud": cred.Cloud,
		},
		"created_by": gin.H{
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		},
		"project": gin.H{
			"id": proj.ID,
			"name": proj.Description,
			"description": proj.Description,
			"status": proj.Status,
		},
		"modified_by": gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		},
		"can_copy": true,
		"can_edit": true,
		"recent_jobs": []gin.H{
			{
				"status": "pending",
				"finished": nil,
				"id": 15,
			},
		},
	}

	if jt.CurrentJobID != nil {
		if err := db.Jobs().FindId(*jt.CurrentJobID).One(&cjob); err == nil {
			log.Println("No current job found", err)
		} else {
			summary["current_job"] = gin.H{
				"id": job.ID,
				"name": job.Name,
				"description": job.Description,
				"status": job.Status,
				"failed": job.Failed,
			}
		}
	}

	jt.Summary = summary

	return nil
}
