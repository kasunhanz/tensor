package api

import (
	"net/http"
	"strconv"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	cJob = "job"
	cJobID = "job_id"
)

type JobController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXJobID from Gin Context and retrieves credential data from the collection
// and store credential data under key CTXJob in Gin Context
func (ctrl JobController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cJobID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Job does not exist"})
		return
	}

	var job ansible.Job
	if err := db.Jobs().FindId(bson.ObjectIdHex(objectID)).One(&job); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Job does not exist",
			Log: log.Fields{
				"Job ID": objectID,
				"Error":  err.Error(),
			},
		})
		return
	}

	roles := new(rbac.JobTemplate)
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
	case "PUT", "DELETE", "PATCH":
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

	c.Set(cJob, job)
	c.Next()
}

// GetJob is a Gin handler function which returns the job as a JSON object
func (ctrl JobController) One(c *gin.Context) {
	job := c.MustGet(cJob).(ansible.Job)
	metadata.JobMetadata(&job)
	c.JSON(http.StatusOK, job)
}

// GetJobs is a Gin handler function which returns list of jobs
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl JobController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"status", "type", "failed"}, match)
	match = parser.Lookups([]string{"id", "name", "labels"}, match)
	query := db.Jobs().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var jobs []ansible.Job

	roles := new(rbac.JobTemplate)
	iter := query.Iter()
	var tmpJob ansible.Job
	for iter.Next(&tmpJob) {
		if !roles.ReadByID(user, tmpJob.JobTemplateID) {
			continue
		}
		metadata.JobMetadata(&tmpJob)
		jobs = append(jobs, tmpJob)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting job", Log: log.Fields{
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

	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  jobs[pgi.Skip():pgi.End()],
	})
}

// CancelInfo to determine if the job can be cancelled.
// The response will include the following field:
// can_cancel: [boolean] Indicates whether this job can be canceled
func (ctrl JobController) CancelInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"can_cancel": false})
}

// Cancel cancels the pending job.
// The response status code will be 202 if successful, or 405 if the job cannot be
// canceled.
func (ctrl JobController) Cancel(c *gin.Context) {
	c.AbortWithStatus(http.StatusMethodNotAllowed)
}

// StdOut returns ANSI standard output of a Job
func (ctrl JobController) StdOut(c *gin.Context) {
	job := c.MustGet(cJob).(ansible.Job)

	c.JSON(http.StatusOK, job.ResultStdout)
}
