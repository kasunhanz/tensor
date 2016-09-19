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
)

// _CTX_JOB_TEMPLATE is the key name of the Job Template in gin.Context
const _CTX_JOB_TEMPLATE = "job_template"
// _CTX_USER is the key name of the User in gin.Context
const _CTX_USER = "user"
// _CTX_JOB_TEMPLATE_ID is the key name of http request Job Template ID
const _CTX_JOB_TEMPLATE_ID = "job_template_id"

// JTemplateM is the middleware for job templates. Which
// takes _CTX_JOB_TEMPLATE_ID parameter form the request, fetches the Job Template
// and set it under key _CTX_JOB_TEMPLATE in gin.Context
func JTemplateM(c *gin.Context) {
	ID := c.Params.ByName(_CTX_JOB_TEMPLATE_ID) //get template ID

	collection := db.MongoDb.C(models.DBC_JOB_TEMPLATES)

	var jt models.JobTemplate

	if err := collection.FindId(bson.ObjectIdHex(ID)).One(&jt); err != nil {
		log.Println("Coud not find Job Template", err) // log error to the system log
		// return 404 error if ID not in the database
		c.AbortWithStatus(http.StatusNotFound)
		return //done
	}

	// set Job Template to the gin.Context
	c.Set(_CTX_JOB_TEMPLATE, jt)
	c.Next() //move to next pending handler
}

// GetJTemplate renders the Job Template as JSON
// make sure to set this handler next to JTemplateM handler
func GetJTemplate(c *gin.Context) {
	//get template set by the middleware
	jt := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	setMetadata(&jt)

	c.JSON(200, jt)
}


// GetJTemplates renders the Job Templates as JSON
func GetJTemplates(c *gin.Context) {
	collection := db.MongoDb.C(models.DBC_JOB_TEMPLATES)

	parser := util.NewQueryParser(c)

	// query map
	match := bson.M{}

	// add filters to query
	if con := parser.IContains([]string{"name", "description", "labels"}); con != nil {
		match = con
	}

	query := collection.Find(match) // prepare the query

	count, err := query.Count(); // number of records
	if err != nil {
		log.Println("Unable to count Job Templates from the db", err)
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

	var jts []models.JobTemplate

	// get all values with skip limit
	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&jts); err != nil {
		log.Println("Unable to retrive Job Template from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// set related and summary fields to every item
	for i, v := range jts {
		// note: `v` reference doesn't modify original slice
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		jts[i] = v // modify each object in slice
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": jts, })
}

// AddJTemplate creates a new Job Template
func AddJTemplate(c *gin.Context) {
	var req models.JobTemplate
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// create new object to omit unnecessary fields
	jt := models.JobTemplate{
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

	collection := db.MongoDb.C(models.DBC_JOB_TEMPLATES)

	// insert new object
	if err := collection.Insert(jt); err != nil {
		log.Println("Failed to create Job Template", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Job Template"})
		return
	}

	// create event in the database
	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_JOB_TEMPLATE,
		ObjectID:    jt.ID,
		Description: "Job Template " + jt.Name + " created",
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

// UpdateJTemplate will update the Job Template
func UpdateJTemplate(c *gin.Context) {
	// get template from the gin.Context
	ojt := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.JobTemplate

	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// create new object to omit unnecessary fields
	jt := models.JobTemplate{
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
		ID: ojt.ID,
		Modified: time.Now(),
		ModifiedByID: user.ID,
	}

	collection := db.MongoDb.C(models.DBC_JOB_TEMPLATES)

	// update object
	if err := collection.UpdateId(jt.ID, jt); err != nil {
		log.Println("Failed to update Job Template", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to update Job Template"})
		return
	}

	if err := (models.Event{
		ProjectID:   jt.ID,
		Description: "JobTemplate ID " + jt.ID.Hex() + " updated",
		ObjectID:    jt.ID,
		ObjectType:  "jt",
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

	// render JSON with 200 status code
	c.JSON(http.StatusOK, jt)
}

// RemoveJTemplate will remove the Job Template
// from the models.DBC_JOB_TEMPLATES collection
func RemoveJTemplate(c *gin.Context) {
	// get template from the gin.Context
	jt := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	collection := db.MongoDb.C(models.DBC_JOB_TEMPLATES)

	// remove object from the collection
	if err := collection.RemoveId(jt.ID); err != nil {
		log.Println("Failed to remove Job Template", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to remove Job Template"})
		return
	}

	if err := (models.Event{
		Description: "Job Template " + jt.Name + " deleted",
		ObjectID:    jt.ID,
		ObjectType:  _CTX_JOB_TEMPLATE,
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

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

	ccred := db.C(models.DBC_CREDENTIALS)
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

	cinven := db.C(models.DBC_INVENTORIES)
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
		"job_template_data": {
			"id": jt.ID,
			"name": jt.Name,
			"description": jt.Description,
		},
		"defaults": defaults,
	}

	// render JSON with 200 status code
	c.JSON(http.StatusOK, resp)
}

func LaunchJob(c *gin.Context) {
	// get template from the gin.Context
	jt := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// render JSON with 200 status code
	c.JSON(http.StatusOK, jt)

}