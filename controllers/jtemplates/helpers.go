package jtemplate

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/models"
	"time"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"github.com/gin-gonic/gin"
	"strconv"
	"bitbucket.pearson.com/apseng/tensor/util"
	"net/http"
)

func addActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string) {

	a := models.Activity{
		ID: bson.NewObjectId(),
		ActorID: userID,
		Type: _CTX_JOB_TEMPLATE,
		ObjectID: crdID,
		Description: desc,
		Created: time.Now(),
	}
	if err := db.ActivityStream().Insert(a); err != nil {
		log.Println("Failed to add new Activity", err)
	}
}

func ObjectRoles(c *gin.Context) {
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	roles := []gin.H{
		{
			"type": "role",
			"related": gin.H{
				"job_template": "/v1/job_templates/" + jobTemplate.ID.Hex() + "/",
			},
			"summary_fields": gin.H{
				"resource_name": jobTemplate.Name,
				"resource_type": "job template",
				"resource_type_display_name": "Job Template",
			},
			"name": "admin",
			"description": "Can manage all aspects of the job template",
		},
		{
			"type": "role",
			"related": gin.H{
				"job_template": "/v1/job_templates/" + jobTemplate.ID.Hex() + "/",
			},
			"summary_fields": gin.H{
				"resource_name":  jobTemplate.Name,
				"resource_type": "job template",
				"resource_type_display_name": "Job Template",
			},
			"name": "read",
			"description": "May view settings for the job template",
		},
		{
			"type": "role",
			"related": gin.H{
				"users": "/api/v1/roles/22/users/",
				"job_template": "/v1/job_templates/" + jobTemplate.ID.Hex() + "/",
			},
			"summary_fields": gin.H{
				"resource_name":  jobTemplate.Name,
				"resource_type": "job template",
				"resource_type_display_name": "Job Template",
			},
			"name": "execute",
			"description": "May run the job template",
		},
	}

	count := len(roles)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: roles[pgi.Skip():pgi.End()],
	})

}
