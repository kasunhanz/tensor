package api

import (
	"net/http"
	"strconv"

	metadata "github.com/pearsonappeng/tensor/api/metadata/terraform"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	cTerraformJob = "terraform_job"
	cTerraformJobID = "terraform_job_id"
)

type TerraformJobController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXTerraformJobID from Gin Context and retrieves credential data from the collection
// and store credential data under key CTXTerraformJob in Gin Context
func (ctrl TerraformJobController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cTerraformJobID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Job does not exist"})
		return
	}

	var job terraform.Job
	if err := db.TerrafromJobs().FindId(bson.ObjectIdHex(objectID)).One(&job); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Job does not exist",
			Log: logrus.Fields{
				"Job ID": objectID,
				"Error":           err.Error(),
			},
		})
		return
	}

	roles := new(rbac.TerraformJobTemplate)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.ReadByID(user, job.JobTemplateID) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "POST":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.WriteByID(user, job.JobTemplateID) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}

	c.Set(cTerraformJob, job)
	c.Next()
}

// GetJob is a Gin handler function which returns the job as a JSON object
func (ctrl TerraformJobController) One(c *gin.Context) {
	job := c.MustGet(cTerraformJob).(terraform.Job)
	metadata.JobMetadata(&job)
	c.JSON(http.StatusOK, job)
}

// GetJobs is a Gin handler function which returns list of jobs
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl TerraformJobController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"status", "type", "failed"}, match)
	match = parser.Lookups([]string{"id", "name", "labels"}, match)
	query := db.TerrafromJobs().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.TerraformJobTemplate)
	var jobs []terraform.Job
	iter := query.Iter()
	var tmpJob terraform.Job
	for iter.Next(&tmpJob) {
		if !roles.ReadByID(user, tmpJob.JobTemplateID) {
			continue
		}
		metadata.JobMetadata(&tmpJob)
		jobs = append(jobs, tmpJob)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting job", Log: logrus.Fields{
				"Error": err.Error(),
			},
		})
		return
	}

	count := len(jobs)
	pgi := util.NewPagination(c, count)
	if pgi.HasPage() {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound,
			Message: "#" + strconv.Itoa(pgi.Page()) + " page contains no results.",
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  jobs[pgi.Skip():pgi.End()],
	})
}

// CancelInfo to determine if the job can be cancelled.
// The response will include the following field:
// can_cancel: [boolean] Indicates whether this job can be canceled
func (ctrl TerraformJobController) CancelInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"can_cancel": false})
}

// Cancel cancels the pending job.
// The response status code will be 202 if successful, or 405 if the job cannot be
// canceled.
func (ctrl TerraformJobController) Cancel(c *gin.Context) {
	c.AbortWithStatus(http.StatusMethodNotAllowed)
}

// StdOut returns ANSI standard output of a Job
func (ctrl TerraformJobController) StdOut(c *gin.Context) {
	job := c.MustGet(cTerraformJob).(terraform.Job)
	c.JSON(http.StatusOK, job.ResultStdout)
}
