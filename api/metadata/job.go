package metadata

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
)

func JobMetadata(job *models.Job) error {
	ID := job.ID.Hex()
	job.Type = "job"
	job.Url = "/v1/jobs/" + ID + "/"
	job.Related = gin.H{
		"created_by": "/api/v1/users/" + job.CreatedByID.Hex() + "/",
		"modified_by": "/api/v1/users/" + job.ModifiedByID.Hex() + "/",
		"labels": "/api/v1/jobs/" + ID + "/labels/",
		"inventory": "/api/v1/inventories/" + job.InventoryID.Hex() + "/",
		"project": "/api/v1/projects/" + job.ProjectID.Hex() + "/",
		"credential": "/api/v1/credentials/" + job.MachineCredentialID.Hex() + "/",
		"stdout": "/api/v1/jobs/" + ID + "/stdout/",
		"job_host_summaries": "/api/v1/jobs/" + ID + "/job_host_summaries/",
		"job_tasks": "/api/v1/jobs/" + ID + "/job_tasks/",
		"job_plays": "/api/v1/jobs/" + ID + "/job_plays/",
		"job_events": "/api/v1/jobs/" + ID + "/job_events/",
		"notifications": "/api/v1/jobs/" + ID + "/notifications/",
		"activity_stream": "/api/v1/jobs/" + ID + "/activity_stream/",
		"job_template": "/api/v1/job_templates/" + job.JobTemplateID.Hex() + "/",
		"start": "/api/v1/jobs/" + ID + "/start/",
		"cancel": "/api/v1/jobs/" + ID + "/cancel/",
		"relaunch": "/api/v1/jobs/" + ID + "/relaunch/",
	}

	if err := JobSummary(job); err != nil {
		return err
	}

	return nil
}

func JobSummary(job *models.Job) error {

	var modified models.User
	var created models.User
	var inv models.Inventory
	var jtemp models.JobTemplate
	var cred models.Credential
	var proj models.Project

	if err := db.Users().FindId(job.CreatedByID).One(&created); err != nil {
		return err
	}

	if err := db.Users().FindId(job.ModifiedByID).One(&modified); err != nil {
		return err
	}

	if err := db.Inventories().FindId(job.InventoryID).One(&inv); err != nil {
		return err
	}

	if err := db.JobTemplates().FindId(job.JobTemplateID).One(&jtemp); err != nil {
		return err
	}

	if err := db.Credentials().FindId(job.MachineCredentialID).One(&cred); err != nil {
		return err
	}

	if err := db.Projects().FindId(job.ProjectID).One(&proj); err != nil {
		return err
	}

	job.Summary = gin.H{
		"job_template": gin.H{
			"id": jtemp.ID.Hex(),
			"name": jtemp.Name,
			"description": jtemp.Description,
		},
		"inventory": gin.H{
			"id": inv.ID.Hex(),
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
		"credential": gin.H{
			"id": cred.ID,
			"name": cred.Name,
			"description": cred.Description,
			"kind": cred.Kind,
			"cloud": cred.Cloud,
		},
		"project": gin.H{
			"id": proj.ID,
			"name": proj.Name,
			"description": proj.Description,
			"status": proj.Status,
		},
		"created_by": gin.H{
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		},
		"modified_by": gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		},
		"labels": gin.H{
			"count": 0,
			"results": []gin.H{},
		},
	}

	return nil
}
