package metadata

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
	"log"
)


// Create a new organization
func ProjectMetadata(p *models.Project) error {

	ID := p.ID.Hex()
	p.Type = "project"
	p.Url = "/v1/projects/" + ID + "/"
	related := gin.H{
		"created_by": "/v1/users/" + p.CreatedBy.Hex() + "/",
		"modified_by": "/v1/users/" + p.ModifiedBy.Hex() + "/",
		"notification_templates_error": "/v1/projects/" + ID + "/notification_templates_error/",
		"notification_templates_success": "/v1/projects/" + ID + "/notification_templates_success/",
		"object_roles": "/v1/projects/" + ID + "/object_roles/",
		"notification_templates_any": "/v1/projects/" + ID + "/notification_templates_any/",
		"project_updates": "/v1/projects/" + ID + "/project_updates/",
		"update": "/v1/projects/" + ID + "/update/",
		"access_list": "/v1/projects/" + ID + "/access_list/",
		"playbooks": "/v1/projects/" + ID + "/playbooks/",
		"schedules": "/v1/projects/" + ID + "/schedules/",
		"teams": "/v1/projects/" + ID + "/teams/",
		"activity_stream": "/v1/projects/" + ID + "/activity_stream/",
		"organization": "/v1/organizations/" + p.OrganizationID.Hex() + "/",
	}

	if p.ScmCredentialID != nil {
		related["credential"] = "/v1/credentials/" + (*p.ScmCredentialID).Hex() + "/"
	}
	if p.LastJob != nil {
		related["last_job"] = "/v1/project_updates/" + (*p.LastJob).Hex() + "/"
	}

	p.Related = related

	if err := projectSummary(p); err != nil {
		return err
	}

	return nil
}

func projectSummary(p *models.Project) error {

	var modified models.User
	var created models.User
	var cred models.Credential
	var org models.Organization

	if err := db.Users().FindId(p.CreatedBy).One(&created); err != nil {
		return err
	}

	if err := db.Users().FindId(p.ModifiedBy).One(&modified); err != nil {
		return err
	}
	if err := db.Organizations().FindId(p.OrganizationID).One(&org); err != nil {
		return err
	}

	summary := gin.H{
		"object_roles": []gin.H{
			{
				"description": "Can manage all aspects of the project",
				"name": "admin",
			},
			{
				"description": "Can use the project in a job template",
				"name": "use",
			},
			{
				"description": "May update project or inventory or group using the configured source update system",
				"name": "update",
			},
			{
				"description": "May view settings for the project",
				"name": "read",
			},
		},
		"organization": gin.H{
			"id": org.ID,
			"name": org.Name,
			"description": org.Description,
		},
		"created_by": gin.H{
			"id":         created.ID,
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		},
		"modified_by": gin.H{
			"id":         modified.ID,
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		},
	}

	if p.ScmCredentialID != nil {
		if err := db.Credentials().FindId(*p.ScmCredentialID).One(&cred); err != nil {
			return err
		}

		summary["credential"] = gin.H{
			"id": cred.ID,
			"name": cred.Name,
			"description": cred.Description,
			"kind": cred.Kind,
			"cloud": cred.Cloud,
		}
	}

	var lastupdate models.Job
	if err := db.Jobs().Find(bson.M{"job_type":"update_job", "project_id": p.ID}).Sort("-modified").One(&lastupdate); err != nil {
		log.Println("Error while getting last update")
		summary["last_update"] = nil
	} else {
		summary["last_update"] = gin.H{
			"id": lastupdate.ID,
			"name": lastupdate.Name,
			"description": lastupdate.Description,
			"finished": lastupdate.Finished,
			"status": lastupdate.Status,
			"failed": lastupdate.Failed,
		}
	}

	var lastjob models.Job
	if err := db.Jobs().Find(bson.M{"job_type":"job", "project_id": p.ID}).Sort("-modified").One(&lastjob); err != nil {
		log.Println("Error while getting last job")
		summary["last_job"] = nil
	} else {
		summary["last_job"] = gin.H{
			"id": lastjob.ID,
			"name": lastjob.Name,
			"description": lastjob.Description,
			"finished": lastjob.Finished,
			"status": lastjob.Status,
			"failed": lastjob.Failed,
		}
	}

	p.Summary = summary

	return nil
}