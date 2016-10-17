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
	"bitbucket.pearson.com/apseng/tensor/api/helpers"
	"bitbucket.pearson.com/apseng/tensor/runners"
	"bitbucket.pearson.com/apseng/tensor/api/jwt"
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
	ID, err := util.GetIdParam(_CTX_JOB_TEMPLATE_ID, c)

	if err != nil {
		log.Print("Error while getting the Job Template:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: []string{"Not Found"},
		})
		return
	}

	var jobTemplate models.JobTemplate
	err = db.JobTemplates().FindId(bson.ObjectIdHex(ID)).One(&jobTemplate);

	if err != nil {
		log.Print("Error while getting the Job Template:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: []string{"Not Found"},
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

	parser := util.NewQueryParser(c)
	match := bson.M{}
	con := parser.IContains([]string{"name", "description", "labels"});
	if con != nil {
		match = con
	}

	query := db.JobTemplates().Find(match) // prepare the query
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
				Message: []string{"Error while getting Job Template"},
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
			Message: []string{"Error while getting Credential"},
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
			Message: util.GetValidationErrors(err),
		})
		return
	}


	// check whether the inventory exist or not
	if !helpers.InventoryExist(req.InventoryID, c) {
		return
	}

	// check whether the machine credential exist or not
	if !helpers.MachineCredentialExist(req.MachineCredentialID, c) {
		return
	}

	// check whether the network credential exist or not
	if req.NetworkCredentialID != nil {
		if !helpers.NetworkCredentialExist(*req.NetworkCredentialID, c) {
			return
		}
	}

	// check whether the network credential exist or not
	if req.CloudCredentialID != nil {
		if !helpers.CloudCredentialExist(*req.CloudCredentialID, c) {
			return
		}
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID

	// insert new object
	err = db.JobTemplates().Insert(req);
	if err != nil {
		log.Println("Error while creating Job Template:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while creating  Job Template"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Job Template " + req.Name + " created")

	// set `related` and `summary` feilds
	err = metadata.JTemplateMetadata(&req);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while creating Job Template"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
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
			Message: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the machine credential exist or not
	if !helpers.MachineCredentialExist(req.MachineCredentialID, c) {
		return
	}

	// check whether the network credential exist or not
	if req.NetworkCredentialID != nil {
		if !helpers.NetworkCredentialExist(*req.NetworkCredentialID, c) {
			return
		}
	}

	// check whether the network credential exist or not
	if req.CloudCredentialID != nil {
		if !helpers.CloudCredentialExist(*req.CloudCredentialID, c) {
			return
		}
	}

	req.ID = jobTemplate.ID
	req.Created = jobTemplate.Created
	req.Modified = time.Now()
	req.CreatedByID = jobTemplate.CreatedByID
	req.ModifiedByID = user.ID

	// update object
	err = db.JobTemplates().UpdateId(jobTemplate.ID, req);
	if err != nil {
		log.Println("Error while updating Job Template:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while updating Job Template"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Job Template " + req.Name + " updated")

	// set `related` and `summary` feilds
	err = metadata.JTemplateMetadata(&req);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while creating Job Template"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, req)
}

// RemoveJTemplate will remove the Job Template
// from the db.DBC_JOB_TEMPLATES collection
func RemoveJTemplate(c *gin.Context) {
	// get template from the gin.Context
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	// remove object from the collection
	err := db.JobTemplates().RemoveId(jobTemplate.ID);
	if err != nil {
		log.Println("Error while removing Job Temlate:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while removing Job Template"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(jobTemplate.ID, user.ID, "Job Template " + jobTemplate.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}


// TODO: not complete
func ActivityStream(c *gin.Context) {
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": jobTemplate.ID, "type": _CTX_JOB_TEMPLATE}).All(&activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while Activities"},
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

	var jbs []models.Job
	// new mongodb iterator
	iter := db.Jobs().Find(bson.M{"job_template_id": jobTemplate.ID}).Iter()
	// loop through each result and modify for our needs
	var tmpJob models.Job
	// iterate over all and only get valid objects
	for iter.Next(&tmpJob) {
		if err := metadata.JobMetadata(&tmpJob); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: []string{"Error while getting Jobs"},
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
			Message: []string{"Error while getting Jobs"},
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

func Launch(c *gin.Context) {
	// get template from the gin.Context
	template := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Launch
	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: util.GetValidationErrors(err),
		})
		return
	}

	job := models.Job{
		ID: bson.NewObjectId(),
		Name: template.Name,
		Description: template.Description,
		LaunchType: "manual",
		CancelFlag: false,
		Status: "pending",
		JobType: models.JOBTYPE_ANSIBLE_JOB,
		Playbook:template.Playbook,
		Forks:template.Forks,
		Limit:template.Limit,
		Verbosity:template.Verbosity,
		ExtraVars:template.ExtraVars,
		JobTags:template.JobTags,
		SkipTags:template.SkipTags,
		ForceHandlers:template.ForceHandlers,
		StartAtTask:template.StartAtTask,
		MachineCredentialID:template.MachineCredentialID,
		InventoryID:template.InventoryID,
		JobTemplateID: template.ID,
		ProjectID: template.ProjectID,
		BecomeEnabled:template.BecomeEnabled,
		NetworkCredentialID:template.NetworkCredentialID,
		CloudCredentialID:template.CloudCredentialID,
		CreatedByID: user.ID,
		ModifiedByID:user.ID,
		Created:time.Now(),
		Modified:time.Now(),
	}

	// add launch parameters
	if len(req.ExtraVars) > 0 && template.PromptVariables {
		job.ExtraVars = req.ExtraVars
	}

	if len(req.Limit) > 0 && template.PromptLimit {
		job.Limit = req.Limit
	}

	if len(req.JobTags) > 0 && template.PromptTags {
		job.JobTags = req.JobTags
	}

	if len(req.SkipTags) > 0 && template.PromptSkipTags {
		job.SkipTags = req.SkipTags
	}

	if len(req.JobType) > 0 && template.PromptJobType {
		job.JobType = req.JobType
	}

	if len(req.InventoryID) == 24 {
		job.InventoryID = req.InventoryID
	}

	if len(req.MachineCredentialID) == 24 {
		job.MachineCredentialID = req.MachineCredentialID
	}

	runnerJob := runners.AnsibleJob{
		Job: job,
		Template:template,
		User:user,
	}

	var credential models.Credential
	err := db.Credentials().FindId(job.MachineCredentialID).One(&credential)
	if err != nil {
		log.Println("Error while getting Machine Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while getting Machine Credential"},
		})
		return
	}
	runnerJob.MachineCred = credential

	if job.NetworkCredentialID != nil {
		var credential models.Credential
		err := db.Credentials().FindId(*job.NetworkCredentialID).One(&credential)
		if err != nil {
			log.Println("Error while getting Network Credential:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: []string{"Error while getting Network Credential"},
			})
			return
		}
		runnerJob.NetworkCred = credential
	}

	if job.CloudCredentialID != nil {
		var credential models.Credential
		err := db.Credentials().FindId(*job.CloudCredentialID).One(&credential)
		if err != nil {
			log.Println("Error while getting Cloud Credential:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: []string{"Error while getting Cloud Credential"},
			})
			return
		}
		runnerJob.CloudCred = credential
	}

	// get inventory information
	var inventory models.Inventory
	err = db.Inventories().FindId(job.InventoryID).One(&inventory)
	if err != nil {
		log.Println("Error while getting Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while getting Inventory"},
		})
		return
	}
	runnerJob.Inventory = inventory

	// get project information
	var project models.Project
	err = db.Projects().FindId(job.ProjectID).One(&project)
	if err != nil {
		log.Println("Error while getting Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while getting Project"},
		})
		return
	}
	runnerJob.Project = project

	// Get jwt token for authorize ansible inventory plugin
	var token jwt.LocalToken
	err = jwt.NewAuthToken(&token)
	if err != nil {
		log.Println("Error while getting Token:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while getting Token"},
		})
		return
	}
	runnerJob.Token = token.Token

	// Insert new job into jobs collection
	if err := db.Jobs().Insert(job); err != nil {
		log.Println("Error while creating Job:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while creating Job"},
		})
		return
	}

	runners.AnsiblePool.Register <- &runnerJob

	if err := metadata.JobMetadata(&job); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: []string{"Error while setting metatdata"},
		})
		return
	}

	c.JSON(http.StatusOK, job)
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

	var cred models.Credential

	if err := db.Credentials().FindId(jt.MachineCredentialID).One(&cred); err != nil {
		log.Println("Cound not find Credential", err)
		defaults["credential"] = nil
		isCredentialNeeded = true
	} else {
		defaults["credential"] = gin.H{
			"id": cred.ID,
			"name": cred.Name,
		}
	}

	var inven models.Inventory

	if err := db.Inventories().FindId(jt.MachineCredentialID).One(&inven); err != nil {
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