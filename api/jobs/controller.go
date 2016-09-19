package jobs

import (
	"gopkg.in/mgo.v2/bson"
	"time"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	"log"
	"bitbucket.pearson.com/apseng/tensor/util"
	"strconv"
	"bitbucket.pearson.com/apseng/tensor/db"
)

// _CTX_JOB is the key name of the Job Template in gin.Context
const _CTX_JOB = "job"
// _CTX_USER is the key name of the User in gin.Context
const _CTX_USER = "user"
// _CTX_JOB_ID is the key name of http request Job Template ID
const _CTX_JOB_ID = "job_id"

// JobMiddleware is the middleware for job. Which
// takes _CTX_JOB_ID parameter form the request, fetches the Job
// and set it under key _CTX_JOB in gin.Context
func JobMiddleware(c *gin.Context) {
	ID := c.Params.ByName(_CTX_JOB_ID) //get Job ID

	collection := db.C(models.DBC_JOBS)

	var job models.Job

	if err := collection.FindId(bson.ObjectIdHex(ID)).One(&job); err != nil {
		log.Println("Coud not find Job", err) // log error to the system log
		// return 404 error if ID not in the database
		c.AbortWithStatus(http.StatusNotFound)
		return //done
	}

	// set Job to the gin.Context
	c.Set(_CTX_JOB, job)
	c.Next() //move to next pending handler
}

// GetJob renders the Job as JSON
// make sure to set this handler next to JobMiddleware handler
func GetJob(c *gin.Context) {
	//get Job set by the middleware
	jt := c.MustGet(_CTX_JOB).(models.Job)

	setMetadata(&jt)

	c.JSON(200, jt)
}


// GetJobs renders the Job as JSON
func GetJobs(c *gin.Context) {
	collection := db.C(models.DBC_JOBS)

	parser := util.NewQueryParser(c)

	// query match
	match := parser.Match([]string{"status", "type","failed", })

	// add filters to query
	if con := parser.IContains([]string{"id", "name", "labels"}); con != nil {
		match = con
	}

	query := collection.Find(match) // prepare the query

	count, err := query.Count(); // number of records
	if err != nil {
		log.Println("Unable to count Jobs from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// initialize Pagination
	pgi := util.NewPagination(c, count)

	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page) + ": That page contains no results."})
		return
	}

	// set sort value to the query based on request parameters
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var jobs []models.Job

	// get all values with skip limit
	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&jobs); err != nil {
		log.Println("Unable to retrive Jobs from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// set related and summary fields to every item
	for i, v := range jobs {
		// note: `v` reference doesn't modify original slice
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		jobs[i] = v // modify each object in slice
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": jobs, })
}

// AddJob creates a new Job
func AddJob(c *gin.Context) {
	var req models.Job
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// create new object to omit unnecessary fields if exists
	var jt models.Job

	//required fields
	jt.Name = req.Name
	jt.JobType = req.JobType
	jt.InventoryID = req.InventoryID
	jt.ProjectID = req.ProjectID
	jt.Playbook = req.Playbook
	jt.MachineCredentialID = req.MachineCredentialID
	jt.Verbosity = req.Verbosity

	//optional fields
	if len(req.Description) > 0 {
		jt.Description = req.Description
	}

	if len(req.CloudCredentialID) == 24 {
		jt.CloudCredentialID = req.CloudCredentialID
	}

	if len(req.NetworkCredentialID) == 24 {
		jt.CloudCredentialID = req.CloudCredentialID
	}

	if len(req.StartAtTask) > 0 {
		jt.StartAtTask = req.StartAtTask
	}

	if len(req.SkipTags) > 0 {
		jt.SkipTags = req.SkipTags
	}

	if len(req.SkipTags) > 0 {
		jt.SkipTags = req.SkipTags
	}

	if req.Forks != 0 {
		jt.Forks = req.Forks
	}

	if len(req.Limit) > 0 {
		jt.Limit = req.Limit
	}

	if len(req.JobTags) > 0 {
		jt.JobTags = req.JobTags
	}

	if len(req.ExtraVars) > 0 {
		jt.ExtraVars = req.ExtraVars
	}

	//optional boolean values, if not specified default is false
	jt.PromptInventory = req.PromptInventory
	jt.PromptCredential = req.PromptCredential
	jt.PromptLimit = req.PromptLimit
	jt.PromptTags = req.PromptTags
	jt.BecomeEnabled = req.BecomeEnabled
	jt.PromptVariables = req.PromptVariables
	jt.ForceHandlers = req.ForceHandlers


	//system generated
	jt.ID = bson.NewObjectId()
	jt.Created = time.Now()
	jt.Modified = time.Now()
	jt.CreatedByID = user.ID
	jt.ModifiedByID = user.ID

	collection := db.MongoDb.C(models.DBC_JOBS)

	// insert new object
	if err := collection.Insert(jt); err != nil {
		log.Println("Failed to create Job", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Job"})
		return
	}

	// create event in the database
	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_JOB,
		ObjectID:    jt.ID,
		Description: "Job " + jt.Name + " created",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	// set `related` and `summary` feilds
	if err := setMetadata(&jt); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	// render JSON with 201 status code
	c.JSON(http.StatusCreated, jt)
}

// RemoveJob will remove the Job
// from the models.DBC_JOB collection
func RemoveJob(c *gin.Context) {
	// get job from the gin.Context
	jt := c.MustGet(_CTX_JOB).(models.Job)

	collection := db.MongoDb.C(models.DBC_JOBS)

	// remove object from the collection
	if err := collection.RemoveId(jt.ID); err != nil {
		log.Println("Failed to remove Job", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to remove Job"})
		return
	}

	if err := (models.Event{
		Description: "Job " + jt.Name + " deleted",
		ObjectID:    jt.ID,
		ObjectType:  _CTX_JOB,
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}