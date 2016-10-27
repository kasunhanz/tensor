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
	"github.com/gin-gonic/gin/binding"
	"io"
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
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var jobTemplate models.JobTemplate
	err = db.JobTemplates().FindId(bson.ObjectIdHex(ID)).One(&jobTemplate);

	if err != nil {
		log.Print("Error while getting the Job Template:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	// Set the Job Template to the gin.Context
	c.Set(_CTX_JOB_TEMPLATE, jobTemplate)
	c.Next() //move to next pending handler
}

// GetJTemplate returns a single Job Template as serialized JSON
// A success will return 200 status code
// A failure will return 500 status code
func GetJTemplate(c *gin.Context) {
	//get template set by the middleware
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	if err := metadata.JTemplateMetadata(&jobTemplate); err != nil {
		log.Print("Error while setting summary and related resources:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while setting summary and related resources"},
		})
		return

	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, jobTemplate)
}


// GetJTemplates returns Job Templates as a serialized JSON
// The resulting data structure contains:
// {
//  "count\": 99,
//  "next\": null,
//  "previous\": null,
//  "results\": [
// 	...
// 	]
// The `count` field indicates the total number of job templates
// found for the given query.  The `next` and `previous` fields provides links to
// additional results if there are more than will fit on a single page.  The
// `results` list contains zero or more job template records.
// A success returns 200 status code
// A failure returns 500 status code
func GetJTemplates(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"name", "description", "labels"}, match)

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
				Messages: []string{"Error while getting Job Template"},
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
			Messages: []string{"Error while getting Credential"},
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
// fields to create a new job template:
// name: Name of this job template. string, required
// description: Optional description of this job template. string, default=""
// job_type:  choice
//   - `run: Run default
//   - `check: Check
//   - `scan: Scan
// inventory:  bson.ObjectId, default=nil
// project:  bson.ObjectId, default=nil
// playbook:  string, default=""
// credential:  bson.ObjectId, default=nil
// cloud_credential:  bson.ObjectId, default=nil
// network_credential:  bson.ObjectId, default=nil
// forks:  integer, default=0
// limit:  string, default=""
// verbosity:  choice
//   - 0: 0 Normal default
//   - 1: 1 Verbose
//   - 2: 2 More Verbose
//   - 3: 3 Debug
//   - 4: 4 Connection Debug
//   - 5: 5 WinRM Debug
// extra_vars:  string, default=""
// job_tags:  string, default=""
// force_handlers:  boolean, default=False
// skip_tags:  string, default=""
// start_at_task:  string, default=""
// host_config_key:  string, default=""
// ask_variables_on_launch:  boolean, default=False
// ask_limit_on_launch:  boolean, default=False
// ask_tags_on_launch:  boolean, default=False
// ask_skip_tags_on_launch:  boolean, default=False
// ask_job_type_on_launch:  boolean, default=False
// ask_inventory_on_launch:  boolean, default=False
// ask_credential_on_launch:  boolean, default=False
// become_enabled:  boolean, default=False
// allow_simultaneous:  boolean, default=False
func AddJTemplate(c *gin.Context) {
	var req models.JobTemplate
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	err := binding.JSON.Bind(c.Request, &req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check the project exist or not
	if !helpers.ProjectExist(req.ProjectID) {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Project does not exists"},
		})
		return
	}

	// if the JobTemplate exist in the collection it is not unique
	if helpers.IsNotUniqueJTemplate(req.Name, req.ProjectID) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: []string{"Job Template with this Name already exists."},
		})
		return
	}

	// check the inventory exist or not
	if !helpers.InventoryExist(req.InventoryID) {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Inventory does not exists"},
		})
		return
	}

	// check the machine credential exist or not
	if !helpers.MachineCredentialExist(req.MachineCredentialID) {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Machine Credential does not exists"},
		})
		return
	}

	// check the network credential exist or not
	if req.NetworkCredentialID != nil {
		if !helpers.NetworkCredentialExist(*req.NetworkCredentialID) {
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Network Credential does not exists"},
			})
			return
		}
	}

	// check the network credential exist or not
	if req.CloudCredentialID != nil {
		if !helpers.CloudCredentialExist(*req.CloudCredentialID) {
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Cloud Credential does not exists"},
			})
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
			Messages: []string{"Error while creating  Job Template"},
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
			Messages: []string{"Error while creating Job Template"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// UpdateJTemplate updates the Job Template and returns updated JSON serialized Job Template
// A success returns 200 status code
// A failure returns 500 status code
// if the request body is invalid returns serialized Error model with 400 status code
func UpdateJTemplate(c *gin.Context) {
	// get template from the gin.Context
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.JobTemplate
	err := binding.JSON.Bind(c.Request, &req);
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check the project exist or not
	if !helpers.ProjectExist(req.ProjectID) {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Project does not exists"},
		})
		return
	}

	if req.Name != jobTemplate.Name {
		// if the JobTemplate exist in the collection it is not unique
		if helpers.IsNotUniqueJTemplate(req.Name, req.ProjectID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Job Template with this Name already exists."},
			})
			return
		}
	}

	// check whether the machine credential exist or not
	if !helpers.MachineCredentialExist(req.MachineCredentialID) {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Machine Credential does not exists"},
		})
		return
	}

	// check whether the network credential exist or not
	if req.NetworkCredentialID != nil {
		if !helpers.NetworkCredentialExist(*req.NetworkCredentialID) {
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Network Credential does not exists"},
			})
			return
		}
	}

	// check whether the network credential exist or not
	if req.CloudCredentialID != nil {
		if !helpers.CloudCredentialExist(*req.CloudCredentialID) {
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Cloud Credential does not exists"},
			})
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
			Messages: []string{"Error while updating Job Template"},
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
			Messages: []string{"Error while creating Job Template"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, req)
}

// PatchJTemplate updates the Job Template and returns updated JSON serialized Job Template
// patch will only update feilds which included in the POST body
// A success returns 200 status code
// A failure returns 500 status code
// if the request body is invalid returns serialized Error model with 400 status code
func PatchJTemplate(c *gin.Context) {
	// get template from the gin.Context
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.PatchJobTemplate
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check the project exist or not
	if !helpers.ProjectExist(req.ProjectID) {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Project does not exists"},
		})
		return
	}

	if len(req.Name) > 0 && req.Name != jobTemplate.Name {
		// if the JobTemplate exist in the collection it is not unique
		if helpers.IsNotUniqueJTemplate(req.Name, req.ProjectID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Job Template with this Name already exists."},
			})
			return
		}
	}

	if len(req.MachineCredentialID) == 12 {
		// check whether the machine credential exist or not
		if !helpers.MachineCredentialExist(req.MachineCredentialID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Machine Credential does not exists."},
			})
			return
		}
	}

	// check whether the network credential exist or not
	if req.NetworkCredentialID != nil {
		if !helpers.NetworkCredentialExist(*req.NetworkCredentialID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Network Credential does not exists."},
			})
			return
		}
	}

	// check whether the network credential exist or not
	if req.CloudCredentialID != nil {
		if !helpers.CloudCredentialExist(*req.CloudCredentialID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Cloud Credential does not exists."},
			})
			return
		}
	}

	req.Modified = time.Now()
	req.ModifiedByID = user.ID

	// update object
	changeinf, err := db.JobTemplates().UpsertId(bson.M{"_id" :jobTemplate.ID}, req);
	if err != nil {
		log.Println("Error while updating Job Template:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Job Template"},
		})
		return
	}

	log.Printf("Matched: %d, Removed: %d, Updated: %d, UpsertId: %s", changeinf.Matched, changeinf.Removed, changeinf.Updated, changeinf.UpsertedId)
	// add new activity to activity stream
	addActivity(jobTemplate.ID, user.ID, "Job Template " + req.Name + " updated")

	// get newly updated JobTempate
	var resp models.JobTemplate
	if err = db.Hosts().FindId(jobTemplate.ID).One(&resp); err != nil {
		log.Print("Error while getting the updated Job Template:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Error while getting the updated Job Template"},
		})
		return
	}

	// set `related` and `summary` feilds
	err = metadata.JTemplateMetadata(&resp);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Job Template"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, resp)
}


// RemoveJTemplate removes the Job Template from the db.DBC_JOB_TEMPLATES collection
// A success returns 204 status code
// A failure returns 500 status code
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
			Messages: []string{"Error while removing Job Template"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(jobTemplate.ID, user.ID, "Job Template " + jobTemplate.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}


// ActivityStream returns serialized list of Activity models associated with the Job Template
// Resulting data structure contains:
// {
//  "count\": 99,
//  "next\": null,
//  "previous\": null,
//  "results\": [
// 	...
// 	]
// }
// The `count` field indicates the total number of activity streams found for the given query.
// The `next` and `previous` fields provides links to additional results if there are more than will fit on a single page.
// The `results` list contains zero or more activity stream records.
// success returns 200 status code
// failure reruns 500 status code
//
func ActivityStream(c *gin.Context) {
	jobTemplate := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": jobTemplate.ID, "type": _CTX_JOB_TEMPLATE}).All(&activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while Activities"},
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

// Jobs returns list of jobs associated with the Job Template
// Resulting data structure contains:
// {
//  "count\": 99,
//  "next\": null,
//  "previous\": null,
//  "results\": [
// 	...
// 	]
// }
// The `count` field indicates the total number of jobs found for the given query.
// The `next` and `previous` fields provides links to additional results if there are more than will fit on a single page.
// The `results` list contains zero or more job records.
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
				Messages: []string{"Error while getting Jobs"},
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
			Messages: []string{"Error while getting Jobs"},
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

// Launch creates a new job and adds the job into job queue. If any
// passwords, inventory, or extra variables (extra_vars) are required, they must
// be passed via POST data, with extra_vars given as a JSON string.
// If `credential_needed_to_start` is `true` then the `credential` field is required
// and if the `inventory_needed_to_start` is `True` then the `inventory` is required as well.
// success returns JSON serialized Job model with 201 status code
// if the request body is invalid returns JSON serialized Error model with 400 status code
func Launch(c *gin.Context) {
	// get job template that was set by the Middleware
	template := c.MustGet(_CTX_JOB_TEMPLATE).(models.JobTemplate)
	// get user object set by the jwt Middleware
	user := c.MustGet(_CTX_USER).(models.User)

	// create a new Launch model
	var req models.Launch
	// if the body present deserialize it
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// accept nil request body for POST request, since all the feilds are optional
		if err != io.EOF {
			// Return 400 if request has bad JSON
			// and return formatted validation errors
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: util.GetValidationErrors(err),
			})
			return // abort
		}
	}

	// create new Job
	job := models.Job{
		ID: bson.NewObjectId(),
		Name: template.Name,
		Description: template.Description,
		LaunchType: "manual",
		CancelFlag: false,
		Status: "new",
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
		PromptCredential: template.PromptCredential,
		PromptInventory: template.PromptInventory,
		PromptJobType: template.PromptJobType,
		PromptLimit: template.PromptLimit,
		PromptTags: template.PromptTags,
		PromptVariables: template.PromptVariables,
		AllowSimultaneous: template.AllowSimultaneous,
	}

	// if prompt is true override Job template
	// if not provided return an error message
	if template.PromptVariables {
		if len(req.ExtraVars) == 0 {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Additional variables required"},
			})
			return
		}

		job.ExtraVars = req.ExtraVars
	}

	if template.PromptLimit {
		if len(req.Limit) == 0 {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Limit required"},
			})
			return
		}

		job.Limit = req.Limit
	}

	if template.PromptTags {
		if len(req.JobTags) == 0 {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Job Tags required"},
			})
			return
		}

		job.JobTags = req.JobTags
	}

	if template.PromptSkipTags {
		if len(req.SkipTags) == 0 {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Skip Tags required"},
			})
			return
		}

		job.SkipTags = req.SkipTags
	}

	if template.PromptJobType {
		if len(req.JobType) == 0 {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Job Type required"},
			})
			return
		}

		job.JobType = req.JobType
	}


	// create new Ansible runner Job
	runnerJob := runners.AnsibleJob{
		Job: job,
		Template:template,
		User:user,
	}

	if template.PromptInventory {
		if len(req.InventoryID) != 24 {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Inventory required"},
			})
			return
		}
		job.InventoryID = req.InventoryID
	}

	if template.PromptCredential {
		if len(req.MachineCredentialID) != 24 {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Credential required"},
			})
			return
		}
		job.MachineCredentialID = req.MachineCredentialID
	}

	if job.NetworkCredentialID != nil {
		var credential models.Credential
		err := db.Credentials().FindId(*job.NetworkCredentialID).One(&credential)
		if err != nil {
			log.Println("Error while getting Network Credential:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Network Credential"},
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
				Messages: []string{"Error while getting Cloud Credential"},
			})
			return
		}
		runnerJob.CloudCred = credential
	}

	// get inventory information
	var inventory models.Inventory
	if err := db.Inventories().FindId(job.InventoryID).One(&inventory); err != nil {
		log.Println("Error while getting Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Inventory"},
		})
		return
	}
	runnerJob.Inventory = inventory

	var credential models.Credential
	if err := db.Credentials().FindId(job.MachineCredentialID).One(&credential); err != nil {
		log.Println("Error while getting Machine Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Machine Credential"},
		})
		return
	}
	runnerJob.MachineCred = credential

	// get project information
	var project models.Project
	if err := db.Projects().FindId(job.ProjectID).One(&project); err != nil {
		log.Println("Error while getting Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Project"},
		})
		return
	}
	runnerJob.Project = project

	// Get jwt token for authorize ansible inventory plugin
	var token jwt.LocalToken
	if err := jwt.NewAuthToken(&token); err != nil {
		log.Println("Error while getting Token:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Token"},
		})
		return
	}
	runnerJob.Token = token.Token

	// Insert new job into jobs collection
	if err := db.Jobs().Insert(job); err != nil {
		log.Println("Error while creating Job:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Job"},
		})
		return
	}

	// Add the job to channel
	runners.AnsiblePool.Register <- &runnerJob

	// set additianl information to Job
	if err := metadata.JobMetadata(&job); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while setting metatdata"},
		})
		return
	}

	// return serialized job
	c.JSON(http.StatusCreated, job)
}

// LaunchInfo returns JSON serialized launch information to determine if the job_template can be
// launched and whether any passwords are required to launch the job_template.
//
// ask_variables_on_launch: Flag indicating whether the job template is configured to prompt for variables upon launch
// ask_tags_on_launch: Flag indicating whether the job template is configured to prompt for tags upon launch
// ask_skip_tags_on_launch: Flag indicating whether the job template is configured to prompt for skip_tags upon launch
// ask_job_type_on_launch: Flag indicating whether the job template is configured to prompt for job_type upon launch
// ask_limit_on_launch: Flag indicating whether the job template is configured to prompt for limit upon launch
// ask_inventory_on_launch: Flag indicating whether the job template is configured to prompt for inventory upon launch
// ask_credential_on_launch: Flag indicating whether the job template is configured to prompt for credential upon launch
// can_start_without_user_input: Flag indicating if the job template can be launched without user-input
// variables_needed_to_start: Required variable names required to launch the job_template
// credential_needed_to_start: Flag indicating the presence of a credential associated with the job template.
// If not then one should be supplied when launching the job
// inventory_needed_to_start: Flag indicating the presence of an inventory associated with the job template.
// If not then one should be supplied when launching the job
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

	if err := db.Inventories().FindId(jt.InventoryID).One(&inven); err != nil {
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