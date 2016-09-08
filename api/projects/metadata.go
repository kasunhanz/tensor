package projects

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
)


// Create a new organization
func setMetadata(p *models.Project) error {

	ID := p.ID.Hex()
	p.Type = "project"
	p.Url = "/v1/projects/" + ID + "/"
	p.Related = gin.H{
		"created_by": "/v1/users/" + p.CreatedBy.Hex() + "/",
		"modified_by": "/v1/users/" + p.ModifiedBy.Hex() + "/",
		"credential": "/v1/users/" + p.ScmCredential.Hex() + "/",
		"last_job": "/v1/project_updates/" + p.LastJob.Hex() + "/",
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
		"organization": "/v1/organizations/" + p.Organization.Hex() + "/",
		"last_update": "/v1/project_updates/" + p.LastJob.Hex() + "/",
	}

	if err := setSummaryFields(p); err != nil {
		return err
	}

	return nil
}

func setSummaryFields(p *models.Project) error {
	dbcu := db.MongoDb.C(models.DBC_USERS)
	dbc := db.MongoDb.C(models.DBC_CREDENTIALS)
	dbco := db.MongoDb.C(models.DBC_ORGANIZATIONS)

	var modified models.User
	var created models.User
	var cred models.Credential
	var org models.Organization

	if err := dbcu.FindId(p.CreatedBy).One(&created); err != nil {
		return err
	}

	if err := dbcu.FindId(p.ModifiedBy).One(&modified); err != nil {
		return err
	}

	if err := dbc.FindId(p.ScmCredential).One(&cred); err != nil {
		return err
	}

	if err := dbco.FindId(p.Organization).One(&org); err != nil {
		return err
	}


	//TODO: get project job information

	p.SummaryFields = gin.H{
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
		"last_job": gin.H{
			"id": "",
			"name": "Demo Project",
			"description": "",
			"finished": "2016-08-16T19:27:43.416Z",
			"status": "successful",
			"failed": false,
		},
		"last_update": gin.H{
			"id": "",
			"name": "Demo Project",
			"description": "",
			"status": "successful",
			"failed": false,
		},
		"credential":  gin.H{
			"id": cred.ID,
			"name": cred.Name,
			"description": cred.Description,
			"kind": cred.Kind,
			"cloud": cred.Cloud,
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

	return nil
}