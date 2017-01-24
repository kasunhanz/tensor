package metadata

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

func ProjectMetadata(p *common.Project) {

	ID := p.ID.Hex()
	p.Type = "project"
	p.URL = "/v1/projects/" + ID + "/"
	related := gin.H{
		"created_by":                     "/v1/users/" + p.CreatedByID.Hex() + "/",
		"modified_by":                    "/v1/users/" + p.ModifiedByID.Hex() + "/",
		"notification_templates_error":   "/v1/projects/" + ID + "/notification_templates_error/",
		"notification_templates_success": "/v1/projects/" + ID + "/notification_templates_success/",
		"object_roles":                   "/v1/projects/" + ID + "/object_roles/",
		"notification_templates_any":     "/v1/projects/" + ID + "/notification_templates_any/",
		"project_updates":                "/v1/projects/" + ID + "/project_updates/",
		"update":                         "/v1/projects/" + ID + "/update/",
		"access_list":                    "/v1/projects/" + ID + "/access_list/",
		"schedules":                      "/v1/projects/" + ID + "/schedules/",
		"teams":                          "/v1/projects/" + ID + "/teams/",
		"activity_stream":                "/v1/projects/" + ID + "/activity_stream/",
		"organization":                   "/v1/organizations/" + p.OrganizationID.Hex() + "/",
	}

	if p.Kind == "ansible" {
		related["playbooks"] = "/v1/projects/" + ID + "/playbooks/"
	}

	if p.ScmCredentialID != nil {
		related["credential"] = "/v1/credentials/" + (*p.ScmCredentialID).Hex() + "/"
	}
	if p.LastJob != nil {
		related["last_job"] = "/v1/project_updates/" + (*p.LastJob).Hex() + "/"
	}

	p.Related = related
	projectSummary(p)
}

func projectSummary(p *common.Project) {

	var modified common.User
	var created common.User
	var cred common.Credential
	var org common.Organization

	if err := db.Users().FindId(p.CreatedByID).One(&created); err != nil {
		log.WithFields(log.Fields{
			"User ID":    p.CreatedByID.Hex(),
			"Project":    p.Name,
			"Project ID": p.ID.Hex(),
		}).Errorln("Error while getting created by User")
	}

	if err := db.Users().FindId(p.ModifiedByID).One(&modified); err != nil {
		log.WithFields(log.Fields{
			"User ID":    p.ModifiedByID.Hex(),
			"Project":    p.Name,
			"Project ID": p.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	}
	if err := db.Organizations().FindId(p.OrganizationID).One(&org); err != nil {
		log.WithFields(log.Fields{
			"SCM Credential ID": p.ScmCredentialID.Hex(),
			"Project":           p.Name,
			"Project ID":        p.ID.Hex(),
		})
		log.Errorln("Error while getting SCM Credential")
	}

	summary := gin.H{
		"object_roles": []gin.H{
			{
				"description": "Can manage all aspects of the project",
				"name":        "admin",
			},
			{
				"description": "Can use the project in a job template",
				"name":        "use",
			},
			{
				"description": "May update project or inventory or group using the configured source update system",
				"name":        "update",
			},
			{
				"description": "May view settings for the project",
				"name":        "read",
			},
		},
		"organization": gin.H{
			"id":          org.ID,
			"name":        org.Name,
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
			log.WithFields(log.Fields{
				"Project":       p.Name,
				"Project ID":    p.ID.Hex(),
				"Credential ID": p.ScmCredentialID.Hex(),
			}).Errorln("Error while getting SCM Credential")
		}

		summary["credential"] = gin.H{
			"id":          cred.ID,
			"name":        cred.Name,
			"description": cred.Description,
			"kind":        cred.Kind,
			"cloud":       cred.Cloud,
		}
	}

	// if the project is an ansible project show jobs related to ansible
	if p.Kind == "ansible" {
		var lastu ansible.Job
		if err := db.Jobs().Find(bson.M{"job_type": "update_job", "project_id": p.ID}).Sort("-modified").One(&lastu); err != nil {
			log.WithFields(log.Fields{
				"Project":    p.Name,
				"Project ID": p.ID,
			}).Warnln("Error while getting last update job")
			summary["last_update"] = nil
		} else {
			summary["last_update"] = gin.H{
				"id":          lastu.ID,
				"name":        lastu.Name,
				"description": lastu.Description,
				"finished":    lastu.Finished,
				"status":      lastu.Status,
				"failed":      lastu.Failed,
			}
		}

		var lastj ansible.Job
		if err := db.Jobs().Find(bson.M{"job_type": "job", "project_id": p.ID}).Sort("-modified").One(&lastj); err != nil {
			log.WithFields(log.Fields{
				"Project":    p.Name,
				"Project ID": p.ID.Hex(),
			}).Warnln("Error while getting last job")
			summary["last_job"] = nil
		} else {
			summary["last_job"] = gin.H{
				"id":          lastj.ID,
				"name":        lastj.Name,
				"description": lastj.Description,
				"finished":    lastj.Finished,
				"status":      lastj.Status,
				"failed":      lastj.Failed,
			}
		}
	}

	p.Summary = summary
}
