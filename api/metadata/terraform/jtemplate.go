package terraform

import (
	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

func JTemplateMetadata(jt *terraform.JobTemplate) {

	ID := jt.ID.Hex()
	jt.Type = "inventory"
	jt.URL = "/v1/terraform/job_templates/" + ID + "/"
	related := gin.H{
		"created_by":                     "/v1/terraform/users/" + jt.CreatedByID.Hex() + "/",
		"modified_by":                    "/v1/terraform/users/" + jt.ModifiedByID.Hex() + "/",
		"labels":                         "/v1/terraform/job_templates/" + ID + "/labels/",
		"project":                        "/v1/terraform/projects/" + jt.ProjectID.Hex() + "/",
		"notification_templates_error":   "/v1/terraform/job_templates/" + ID + "/notification_templates_error/",
		"notification_templates_success": "/v1/terraform/job_templates/" + ID + "/notification_templates_success/",
		"jobs":                       "/v1/terraform/job_templates/" + ID + "/jobs/",
		"object_roles":               "/v1/terraform/job_templates/" + ID + "/object_roles/",
		"notification_templates_any": "/v1/terraform/job_templates/" + ID + "/notification_templates_any/",
		"access_list":                "/v1/terraform/job_templates/" + ID + "/access_list/",
		"launch":                     "/v1/terraform/job_templates/" + ID + "/launch/",
		"schedules":                  "/v1/terraform/job_templates/" + ID + "/schedules/",
		"activity_stream":            "/v1/terraform/job_templates/" + ID + "/activity_stream/",
	}

	if jt.CurrentJobID != nil {
		related["current_job"] = "/v1/terraform/jobs/" + jt.CurrentJobID.Hex() + "/"
	}

	if jt.MachineCredentialID != nil {
		related["credential"] = "/v1/terraform/credentials/" + (*jt.MachineCredentialID).Hex() + "/"
	}

	jt.Related = related

	jTemplateSummary(jt)
}

func jTemplateSummary(jt *terraform.JobTemplate) {

	var modified common.User
	var created common.User
	var job terraform.Job
	var cjob terraform.Job
	var cupdate terraform.Job
	var cred common.Credential
	var proj common.Project
	var recentJobs []terraform.Job

	summary := gin.H{
		"object_roles": []gin.H{
			{
				"Description": "Can manage all aspects of the job template",
				"Name":        "admin",
			},
			{
				"Description": "May run the job template",
				"Name":        "execute",
			},
			{
				"Description": "May view settings for the job template",
				"Name":        "read",
			},
		},
		"current_update": nil,
		"inventory":      nil,
		"current_job":    nil,
		"credential":     nil,
		"created_by":     nil,
		"project":        nil,
		"modified_by":    nil,
		"can_copy":       true,
		"can_edit":       true,
		"recent_jobs":    nil,
	}

	if err := db.Users().FindId(jt.CreatedByID).One(&created); err != nil {
		log.WithFields(log.Fields{
			"User ID":         jt.CreatedByID.Hex(),
			"Job Template":    jt.Name,
			"Job Template ID": jt.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	} else {
		summary["created_by"] = gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	if err := db.Users().FindId(jt.ModifiedByID).One(&modified); err != nil {
		log.WithFields(log.Fields{
			"User ID":         jt.ModifiedByID.Hex(),
			"Job Template":    jt.Name,
			"Job Template ID": jt.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	} else {
		summary["modified_by"] = gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	if jt.MachineCredentialID != nil {
		if err := db.Credentials().FindId(*jt.MachineCredentialID).One(&cred); err != nil {
			log.WithFields(log.Fields{
				"Credential ID":   (*jt.MachineCredentialID).Hex(),
				"Job Template":    jt.Name,
				"Job Template ID": jt.ID.Hex(),
			}).Errorln("Error while getting Credential")
		} else {
			summary["credential"] = gin.H{
				"id":          cred.ID,
				"name":        cred.Name,
				"description": cred.Description,
				"kind":        cred.Kind,
				"cloud":       cred.Cloud,
			}
		}
	}

	if err := db.Projects().FindId(jt.ProjectID).One(&proj); err != nil {
		log.WithFields(log.Fields{
			"Project ID":      jt.ProjectID.Hex(),
			"Job Template":    jt.Name,
			"Job Template ID": jt.ID.Hex(),
		}).Errorln("Error while getting Project")
	} else {
		summary["project"] = gin.H{
			"id":          proj.ID,
			"name":        proj.Description,
			"description": proj.Description,
			"status":      proj.Status,
		}
	}

	if jt.CurrentJobID != nil {
		if err := db.TerrafromJobs().FindId(*jt.CurrentJobID).One(&cjob); err == nil {
			log.WithFields(log.Fields{
				"Current Job ID":  (*jt.CurrentJobID).Hex(),
				"Job Template":    jt.Name,
				"Job Template ID": jt.ID.Hex(),
			}).Errorln("Error while getting Current Job")
		} else {
			summary["current_job"] = gin.H{
				"id":          cjob.ID,
				"name":        cjob.Name,
				"description": job.Description,
				"status":      cjob.Status,
				"failed":      cjob.Failed,
			}
		}
	}

	if jt.CurrentUpdateID != nil {
		if err := db.TerrafromJobs().FindId(*jt.CurrentUpdateID).One(&cupdate); err == nil {
			log.WithFields(log.Fields{
				"Current Update ID": (*jt.CurrentUpdateID).Hex(),
				"Job Template":      jt.Name,
				"Job Template ID":   jt.ID.Hex(),
			}).Errorln("Error while getting Current Update Job")
		} else {
			summary["current_update"] = gin.H{
				"id":          cupdate.ID,
				"name":        cupdate.Name,
				"description": cupdate.Description,
				"status":      cupdate.Status,
				"failed":      cupdate.Failed,
			}
		}
	}

	if err := db.TerrafromJobs().Find(bson.M{"job_template_id": jt.ID}).Sort("-modified").Limit(10).All(&recentJobs); err == nil {
		log.WithFields(log.Fields{
			"Job Template":    jt.Name,
			"Job Template ID": jt.ID.Hex(),
		}).Errorln("Error while getting Current Job")
	} else {
		var a []gin.H
		for _, v := range recentJobs {
			a = append(a, gin.H{
				"status":   v.Status,
				"finished": v.Finished,
				"id":       v.ID.Hex(),
			})
		}

		summary["recent_jobs"] = a
	}

	jt.Summary = summary
}
