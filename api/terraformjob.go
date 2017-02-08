package api

import (
	"net/http"
	"strconv"

	metadata "github.com/pearsonappeng/tensor/api/metadata/terraform"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/gin-gonic/gin.v1"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	CTXTerraformJob = "terraform_job"
	CTXTerraformJobID = "terraform_job_id"
)

type TerraformJobController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXTerraformJobID from Gin Context and retrieves credential data from the collection
// and store credential data under key CTXTerraformJob in Gin Context
func (ctrl TerraformJobController) Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(CTXTerraformJobID, c)

	if err != nil {
		log.WithFields(log.Fields{
			"Terraform Job ID": ID,
			"Error":            err.Error(),
		}).Errorln("Error while getting Terraform Job ID url parameter")

		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var job terraform.Job
	if err = db.TerrafromJobs().FindId(bson.ObjectIdHex(ID)).One(&job); err != nil {
		log.WithFields(log.Fields{
			"Job ID": ID,
			"Error":  err.Error(),
		}).Errorln("Error while retriving Terraform Job from the database")
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	// set Job to the gin.Context
	c.Set(CTXTerraformJob, job)
	c.Next() //move to next pending handler
}

// GetJob is a Gin handler function which returns the job as a JSON object
func (ctrl TerraformJobController) One(c *gin.Context) {
	job := c.MustGet(CTXTerraformJob).(terraform.Job)

	metadata.JobMetadata(&job)

	c.JSON(http.StatusOK, job)
}

// GetJobs is a Gin handler function which returns list of jobs
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl TerraformJobController) All(c *gin.Context) {

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"status", "type", "failed"}, match)
	match = parser.Lookups([]string{"id", "name", "labels"}, match)

	query := db.TerrafromJobs().Find(match) // prepare the query

	// set sort value to the query based on request parameters
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	log.WithFields(log.Fields{
		"Query": query,
	}).Debugln("Parsed query")

	var jobs []terraform.Job

	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpJob terraform.Job
	// iterate over all and only get valid objects
	for iter.Next(&tmpJob) {
		metadata.JobMetadata(&tmpJob)
		// good to go add to list
		jobs = append(jobs, tmpJob)
	}
	if err := iter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Terraform Job data from the database")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Terraform Job"},
		})
		return
	}

	count := len(jobs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Terraform Job page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	log.WithFields(log.Fields{
		"Count":    count,
		"Next":     pgi.NextPage(),
		"Previous": pgi.PreviousPage(),
		"Skip":     pgi.Skip(),
		"Limit":    pgi.Limit(),
	}).Debugln("Response info")
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
func (ctrl TerraformJobController) CancelInfo(c *gin.Context) {
	//get Job set by the middleware
	// send response with JSON rendered data
	c.JSON(http.StatusOK, gin.H{"can_cancel": false})
}

// Cancel cancels the pending job.
// The response status code will be 202 if successful, or 405 if the job cannot be
// canceled.
func (ctrl TerraformJobController) Cancel(c *gin.Context) {
	//get Job set by the middleware
	c.AbortWithStatus(http.StatusMethodNotAllowed)
}

// StdOut returns ANSI standard output of a Job
func (ctrl TerraformJobController) StdOut(c *gin.Context) {
	//get Job set by the middleware
	job := c.MustGet(CTXTerraformJob).(terraform.Job)

	c.JSON(http.StatusOK, job.ResultStdout)
}
