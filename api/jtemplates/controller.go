package jtemplate

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
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
	"bitbucket.pearson.com/apseng/tensor/roles"
)

// _CTX_JOB_TEMPLATE is the key name of the Job Template in gin.Context
const _CTX_JOB_TEMPLATE = "job_template"
// _CTX_USER is the key name of the User in gin.Context
const _CTX_USER = "user"
// _CTX_JOB_TEMPLATE_ID is the key name of http request Job Template ID
const _CTX_JOB_TEMPLATE_ID = "job_template_id"

// Middleware is the middleware for job templates. Which
// takes _CTX_JOB_TEMPLATE_ID parameter form the request, fetches the Job Template
// and set it under key _CTX_JOB_TEMPLATE in gin.Context
func Middleware(c *gin.Context) {
	ID := c.Params.ByName(_CTX_JOB_TEMPLATE_ID) //get template ID

	collection := db.MongoDb.C(db.JOB_TEMPLATES)
	var jobTemplate models.JobTemplate
	err := collection.FindId(bson.ObjectIdHex(ID)).One(&jobTemplate);

	if err != nil {
		log.Print("Error while getting the Job Template:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
	}

	// set Job Template to the gin.Context
	c.Set(_CTX_JOB_TEMPLATE, jobTemplate)
	c.Next() //move to next pending handler
}

// GetJTemplate renders the Job Template as JSON
// make sure to set this handler next to JTemplateM handler
func GetJTemplate(c *gin.Context) {
	//get template set by the middleware
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	metadata.JTemplateMetadata(&jobTemplate)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, jobTemplate)
}


// GetJTemplates renders the Job Templates as JSON
func GetJTemplates(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)
	collection := db.MongoDb.C(db.JOB_TEMPLATES)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	con := parser.IContains([]string{"name", "description", "labels"});
	if con != nil {
		match = con
	}

	query := collection.Find(match) // prepare the query
	// set sort value to the query based on request parameters
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var jobTemplates []models.JobTemplate
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpJobTemplate models.JobTemplate
	// iterate over all and only get valid objects
	for iter.Next(&tmpJobTemplate) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.JobTemplateRead(user, tmpJobTemplate) {
			continue
		}
		if err := metadata.JTemplateMetadata(&tmpJobTemplate); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Job Template",
			})
			return
		}
		// good to go add to list
		jobTemplates = append(jobTemplates, tmpJobTemplate)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Credential",
		})
		return
	}

	count := len(jobTemplates)
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
		Results: jobTemplates[pgi.Skip():pgi.End()],
	})
}

// AddJTemplate creates a new Job Template
func AddJTemplate(c *gin.Context) {
	var req models.JobTemplate
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	err := c.BindJSON(&req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	// create new object to omit unnecessary fields
	jobTemplate := models.JobTemplate{
		Name: req.Name,
		JobType: req.JobType,
		InventoryID: req.InventoryID,
		ProjectID: req.ProjectID,
		Playbook: req.Playbook,
		MachineCredentialID: req.MachineCredentialID,
		Verbosity: req.Verbosity,
		Description: req.Description,
		CloudCredentialID: req.CloudCredentialID,
		NetworkCredentialID: req.NetworkCredentialID,
		StartAtTask: req.StartAtTask,
		SkipTags: req.SkipTags,
		Forks: req.Forks,
		Limit: req.Limit,
		JobTags: req.JobTags,
		ExtraVars: req.ExtraVars,
		PromptInventory: req.PromptInventory,
		PromptCredential: req.PromptCredential,
		PromptLimit: req.PromptLimit,
		PromptTags: req.PromptTags,
		BecomeEnabled: req.BecomeEnabled,
		PromptVariables: req.PromptVariables,
		AllowSimultaneous: req.AllowSimultaneous,
		ForceHandlers: req.ForceHandlers,
		ID: bson.NewObjectId(),
		Created: time.Now(),
		Modified: time.Now(),
		CreatedByID: user.ID,
		ModifiedByID: user.ID,
	}

	collection := db.MongoDb.C(db.JOB_TEMPLATES)

	// insert new object
	err = collection.Insert(jobTemplate);
	if err != nil {
		log.Println("Error while creating Job Template:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating  Job Template",
		})
		return
	}

	// add new activity to activity stream
	addActivity(jobTemplate.ID, user.ID, "Job Template " + jobTemplate.Name + " created")

	// set `related` and `summary` feilds
	err = metadata.JTemplateMetadata(&jobTemplate);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Job Template",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, jobTemplate)
}

// UpdateJTemplate will update the Job Template
func UpdateJTemplate(c *gin.Context) {
	// get template from the gin.Context
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.JobTemplate
	err := c.BindJSON(&req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	// create new object to omit unnecessary fields
	jobTemplate.Name = req.Name
	jobTemplate.JobType = req.JobType
	jobTemplate.InventoryID = req.InventoryID
	jobTemplate.ProjectID = req.ProjectID
	jobTemplate.Playbook = req.Playbook
	jobTemplate.MachineCredentialID = req.MachineCredentialID
	jobTemplate.Verbosity = req.Verbosity
	jobTemplate.Description = req.Description
	jobTemplate.CloudCredentialID = req.CloudCredentialID
	jobTemplate.NetworkCredentialID = req.NetworkCredentialID
	jobTemplate.StartAtTask = req.StartAtTask
	jobTemplate.SkipTags = req.SkipTags
	jobTemplate.Forks = req.Forks
	jobTemplate.Limit = req.Limit
	jobTemplate.JobTags = req.JobTags
	jobTemplate.ExtraVars = req.ExtraVars
	jobTemplate.PromptInventory = req.PromptInventory
	jobTemplate.PromptCredential = req.PromptCredential
	jobTemplate.PromptLimit = req.PromptLimit
	jobTemplate.PromptTags = req.PromptTags
	jobTemplate.BecomeEnabled = req.BecomeEnabled
	jobTemplate.PromptVariables = req.PromptVariables
	jobTemplate.AllowSimultaneous = req.AllowSimultaneous
	jobTemplate.ForceHandlers = req.ForceHandlers
	jobTemplate.Modified = time.Now()
	jobTemplate.ModifiedByID = user.ID

	collection := db.MongoDb.C(db.JOB_TEMPLATES)

	// update object
	err = collection.UpdateId(jobTemplate.ID, jobTemplate);
	if err != nil {
		log.Println("Error while updating Job Template:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while updating Job Template",
		})
		return
	}

	// add new activity to activity stream
	addActivity(jobTemplate.ID, user.ID, "Job Template " + jobTemplate.Name + " updated")

	// set `related` and `summary` feilds
	err = metadata.JTemplateMetadata(&jobTemplate);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Job Template",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, jobTemplate)
}

// RemoveJTemplate will remove the Job Template
// from the db.DBC_JOB_TEMPLATES collection
func RemoveJTemplate(c *gin.Context) {
	// get template from the gin.Context
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.MongoDb.C(db.JOB_TEMPLATES)

	// remove object from the collection
	err := collection.RemoveId(jobTemplate.ID);
	if err != nil {
		log.Println("Error while removing Job Temlate:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Job Template",
		})
		return
	}

	// add new activity to activity stream
	addActivity(jobTemplate.ID, user.ID, "Job Template " + jobTemplate.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

func LaunchInfo(c *gin.Context) {
	// get template from the gin.Context
	jt := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	var isCredentialNeeded bool
	var isInventoryNeeded bool

	defaults := gin.H{
		"job_tags": jt.JobTags,
		"extra_vars": jt.ExtraVars,
		"job_type": jt.JobType,
		"skip_tags": jt.SkipTags,
		"limit": jt.Limit,
		"inventory": gin.H{
			"id": jt.InventoryID,
			"name": "Demo Inventory",
		},
	}

	ccred := db.C(db.CREDENTIALS)
	var cred models.Credential

	if err := ccred.FindId(jt.MachineCredentialID).One(&cred); err != nil {
		log.Println("Cound not find Credential", err)
		defaults["credential"] = nil
		isCredentialNeeded = true
	} else {
		defaults["credential"] = gin.H{
			"id": cred.ID,
			"name": cred.Name,
		}
	}

	cinven := db.C(db.INVENTORIES)
	var inven models.Inventory

	if err := cinven.FindId(jt.MachineCredentialID).One(&inven); err != nil {
		log.Println("Cound not find Inventory", err)
		defaults["inventory"] = nil
		isInventoryNeeded = true
	} else {
		defaults["inventory"] = gin.H{
			"id": inven.ID,
			"name": inven.Name,
		}
	}

	resp := gin.H{
		"passwords_needed_to_start": []gin.H{},
		"ask_variables_on_launch": jt.PromptVariables,
		"ask_tags_on_launch": jt.PromptTags,
		"ask_job_type_on_launch": jt.PromptJobType,
		"ask_limit_on_launch": jt.PromptInventory,
		"ask_inventory_on_launch": jt.PromptInventory,
		"ask_credential_on_launch": jt.PromptCredential,
		"variables_needed_to_start": []gin.H{},
		"credential_needed_to_start": isCredentialNeeded,
		"inventory_needed_to_start": isInventoryNeeded,
		"job_template_data": gin.H{
			"id": jt.ID.Hex(),
			"name": jt.Name,
			"description": jt.Description,
		},
		"defaults": defaults,
	}

	// render JSON with 200 status code
	c.JSON(http.StatusOK, resp)
}

// TODO: not complete
func ActivityStream(c *gin.Context) {
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	var activities []models.Activity
	collection := db.C(db.ACTIVITY_STREAM)
	err := collection.Find(bson.M{"object_id": jobTemplate.ID, "type": _CTX_JOB_TEMPLATE}).All(&activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while Activities",
		})
		return
	}

	count := len(activities)
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
		Results: activities[pgi.Skip():pgi.End()],
	})
}

func Jobs(c *gin.Context) {
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	collection := db.C(db.JOBS)

	var jbs []models.Job
	// new mongodb iterator
	iter := collection.Find(bson.M{"job_template_id": jobTemplate.ID}).Iter()
	// loop through each result and modify for our needs
	var tmpJob models.Job
	// iterate over all and only get valid objects
	for iter.Next(&tmpJob) {
		if err := metadata.JobMetadata(&tmpJob); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Jobs",
			})
			return
		}
		// good to go add to list
		jbs = append(jbs, tmpJob)
	}

	if err := iter.Close(); err != nil {
		log.Println("Error while retriving jobs data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Jobs",
		})
		return
	}

	count := len(jbs)
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
		Results: jbs[pgi.Skip():pgi.End()],
	})
}

// TODO: implement
func Launch(c *gin.Context) {
	// get template from the gin.Context
	jt := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// render JSON with 200 status code
	c.JSON(http.StatusOK, jt)

}