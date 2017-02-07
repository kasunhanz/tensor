package metadata

import (
	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// Create a new organization
func JTemplateMetadata(jt *ansible.JobTemplate) {

	ID := jt.ID.Hex()
	jt.Type = "job_template"
	jt.URL = "/v1/job_templates/" + ID + "/"
	related := gin.H{
		"created_by":                     "/v1/users/" + jt.CreatedByID.Hex() + "/",
		"modified_by":                    "/v1/users/" + jt.ModifiedByID.Hex() + "/",
		"labels":                         "/v1/job_templates/" + ID + "/labels/",
		"inventory":                      "/v1/inventories/" + jt.InventoryID.Hex() + "/",
		"credential":                     "/v1/credentials/" + jt.MachineCredentialID.Hex() + "/",
		"project":                        "/v1/projects/" + jt.ProjectID.Hex() + "/",
		"notification_templates_error":   "/v1/job_templates/" + ID + "/notification_templates_error/",
		"notification_templates_success": "/v1/job_templates/" + ID + "/notification_templates_success/",
		"jobs":                       "/v1/job_templates/" + ID + "/jobs/",
		"object_roles":               "/v1/job_templates/" + ID + "/object_roles/",
		"notification_templates_any": "/v1/job_templates/" + ID + "/notification_templates_any/",
		"access_list":                "/v1/job_templates/" + ID + "/access_list/",
		"launch":                     "/v1/job_templates/" + ID + "/launch/",
		"schedules":                  "/v1/job_templates/" + ID + "/schedules/",
		"activity_stream":            "/v1/job_templates/" + ID + "/activity_stream/",
	}

	if jt.CurrentJobID != nil {
		related["current_job"] = "/v1/jobs/" + jt.CurrentJobID.Hex() + "/"
	}

	jt.Related = related

	jTemplateSummary(jt)
}

func jTemplateSummary(jt *ansible.JobTemplate) {

	var modified common.User
	var created common.User
	var inv ansible.Inventory
	var job ansible.Job
	var cjob ansible.Job
	var cupdate ansible.Job
	var cred common.Credential
	var proj common.Project
	var recentJobs []ansible.Job

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
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
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

	if err := db.Inventories().FindId(jt.InventoryID).One(&inv); err != nil {
		log.WithFields(log.Fields{
			"Inventory ID":    jt.InventoryID.Hex(),
			"Job Template":    jt.Name,
			"Job Template ID": jt.ID.Hex(),
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

	if err := db.Credentials().FindId(jt.MachineCredentialID).One(&cred); err != nil {
		log.WithFields(log.Fields{
			"Credential ID":   jt.MachineCredentialID.Hex(),
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
		if err := db.Jobs().FindId(*jt.CurrentJobID).One(&cjob); err == nil {
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
		if err := db.Jobs().FindId(*jt.CurrentUpdateID).One(&cupdate); err == nil {
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

	if err := db.Jobs().Find(bson.M{"job_template_id": jt.ID}).Sort("-modified").Limit(10).All(&recentJobs); err == nil {
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
