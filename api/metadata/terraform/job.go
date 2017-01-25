package terraform

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"
)

func JobMetadata(job *terraform.Job) {
	ID := job.ID.Hex()
	job.Type = job.JobType
	job.URL = "/v1/jobs/" + ID + "/"
	related := gin.H{
		"labels":          "/v1/jobs/" + ID + "/labels/",
		"project":         "/v1/projects/" + job.ProjectID.Hex() + "/",
		"stdout":          "/v1/jobs/" + ID + "/stdout/",
		"notifications":   "/v1/jobs/" + ID + "/notifications/",
		"activity_stream": "/v1/jobs/" + ID + "/activity_stream/",
		"start":           "/v1/jobs/" + ID + "/start/",
		"cancel":          "/v1/jobs/" + ID + "/cancel/",
		"relaunch":        "/v1/jobs/" + ID + "/relaunch/",
	}

	if len(job.CreatedByID) == 12 {
		related["created_by"] = "/v1/users/" + job.CreatedByID.Hex() + "/"
	}

	if len(job.ModifiedByID) == 12 {
		related["modified_by"] = "/v1/users/" + job.ModifiedByID.Hex() + "/"
	}

	if len(job.MachineCredentialID) == 12 {
		related["credential"] = "/v1/credentials/" + job.MachineCredentialID.Hex() + "/"
	}

	if len(job.JobTemplateID) == 12 {
		related["job_template"] = "/v1/job_templates/" + job.JobTemplateID.Hex() + "/"
	}

	job.Related = related
	JobSummary(job)
}

func JobSummary(job *terraform.Job) {
	var proj common.Project

	summary := gin.H{
		"credential": nil,
		"project":    nil,
		"labels": gin.H{
			"count":   0,
			"results": []gin.H{},
		},
		"modified_by":  nil,
		"created_by":   nil,
		"job_template": nil,
	}

	if len(job.ModifiedByID) == 12 {
		var modified common.User
		if err := db.Users().FindId(job.ModifiedByID).One(&modified); err != nil {
			log.WithFields(log.Fields{
				"User ID": job.ModifiedByID.Hex(),
				"Job":     job.Name,
				"Job ID":  job.ID.Hex(),
			}).Errorln("Error while getting modified by User")
		} else {
			summary["modified_by"] = gin.H{
				"id":         modified.ID.Hex(),
				"username":   modified.Username,
				"first_name": modified.FirstName,
				"last_name":  modified.LastName,
			}
		}
	}

	if len(job.CreatedByID) == 12 {
		var created common.User
		if err := db.Users().FindId(job.CreatedByID).One(&created); err != nil {
			log.WithFields(log.Fields{
				"User ID": job.CreatedByID.Hex(),
				"Job":     job.Name,
				"Job ID":  job.ID.Hex(),
			}).Errorln("Error while getting created by User")
		} else {
			summary["created_by"] = gin.H{
				"id":         created.ID.Hex(),
				"username":   created.Username,
				"first_name": created.FirstName,
				"last_name":  created.LastName,
			}
		}
	}

	if len(job.JobTemplateID) == 12 {
		var jtemp ansible.JobTemplate
		if err := db.TerrafromJobTemplates().FindId(job.JobTemplateID).One(&jtemp); err != nil {
			log.WithFields(log.Fields{
				"Job Template ID": job.JobTemplateID.Hex(),
				"Job":             job.Name,
				"Job ID":          job.ID.Hex(),
			}).Warnln("Error while getting Job Template")
		} else {
			summary["job_template"] = gin.H{
				"id":          jtemp.ID.Hex(),
				"name":        jtemp.Name,
				"description": jtemp.Description,
			}
		}
	}

	if len(job.MachineCredentialID) == 12 {
		var cred common.Credential
		if err := db.Credentials().FindId(job.MachineCredentialID).One(&cred); err != nil {
			log.WithFields(log.Fields{
				"Credential ID": job.MachineCredentialID.Hex(),
				"Job":           job.Name,
				"Job ID":        job.ID.Hex(),
			}).Warnln("Error while getting Machine Credential")
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

	if err := db.Projects().FindId(job.ProjectID).One(&proj); err != nil {
		log.WithFields(log.Fields{
			"Project ID": job.ProjectID.Hex(),
			"Job":        job.Name,
			"Job ID":     job.ID.Hex(),
		}).Warnln("Error while getting Project")
	} else {
		summary["project"] = gin.H{
			"id":          proj.ID,
			"name":        proj.Name,
			"description": proj.Description,
			"status":      proj.Status,
		}
	}

	job.Summary = summary
}
