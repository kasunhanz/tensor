package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	metadata "github.com/pearsonappeng/tensor/api/metadata/terraform"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/exec/sync"
	"github.com/pearsonappeng/tensor/exec/types"
	"github.com/pearsonappeng/tensor/jwt"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/queue"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	cTerraformJobTemplate   = "terraform_job_template"
	cTerraformJobTemplateID = "terraform_job_template_id"
)

type TJobTmplController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXTerraformJobTemplateID from Gin Context and retrieves terraform job template data from the collection
// and store terraform job template data under key CTXTerraformJobTemplate in Gin Context
func (ctrl TJobTmplController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cTerraformJobTemplateID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Job template does not exist"})
		return
	}

	var jobTemplate terraform.JobTemplate
	if err := db.TerrafromJobTemplates().FindId(bson.ObjectIdHex(objectID)).One(&jobTemplate); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Job Template does not exist",
			Log: logrus.Fields{
				"Job Template ID": objectID,
				"Error":           err.Error(),
			},
		})
		return
	}

	roles := new(rbac.TerraformJobTemplate)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(user, jobTemplate) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "POST":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(user, jobTemplate) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}

	c.Set(cTerraformJobTemplate, jobTemplate)
	c.Next()
}

// GetJTemplate is a Gin handler function which returns the Terraform Job Template as a JSON object
// A success will return 200 status code
// A failure will return 500 status code
func (ctrl TJobTmplController) One(c *gin.Context) {
	jobTemplate := c.MustGet(cTerraformJobTemplate).(terraform.JobTemplate)
	metadata.JTemplateMetadata(&jobTemplate)
	c.JSON(http.StatusOK, jobTemplate)
}

// GetJTemplates is a Gin handler function which returns list of Job Templates
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
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl TJobTmplController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"name", "description", "labels"}, match)
	query := db.TerrafromJobTemplates().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.TerraformJobTemplate)
	var jobTemplates []terraform.JobTemplate
	iter := query.Iter()
	var tmpJobTemplate terraform.JobTemplate
	for iter.Next(&tmpJobTemplate) {
		if !roles.Read(user, tmpJobTemplate) {
			continue
		}
		metadata.JTemplateMetadata(&tmpJobTemplate)
		jobTemplates = append(jobTemplates, tmpJobTemplate)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting job template", Log: logrus.Fields{
				"Error": err.Error(),
			},
		})
		return
	}

	count := len(jobTemplates)
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
		Data:     jobTemplates[pgi.Skip():pgi.End()],
	})
}

// AddJTemplate is Gin handler function which creates a new Credential using request payload
// This accepts Job Template model.
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
func (ctrl TJobTmplController) Create(c *gin.Context) {
	var req terraform.JobTemplate
	// get user from the gin.Context
	user := c.MustGet(cUser).(common.User)

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// check the project exist or not
	if !req.ProjectExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Project does not exists.",
		})
		return
	}

	if !new(rbac.Project).ReadByID(user, req.ProjectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
	}

	// if the JobTemplate exist in the collection it is not unique
	if !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Job Template with this Name already exists.",
		})
		return
	}

	roles := new(rbac.Credential)
	if req.MachineCredentialID != nil {
		if !req.MachineCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Machine Credential does not exists"},
			})
			return
		}

		if !roles.ReadByID(user, *req.MachineCredentialID) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
			return
		}
	}

	if req.NetworkCredentialID != nil {
		if !req.NetworkCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Network credential does not exists"},
			})
			return
		}

		if !roles.ReadByID(user, *req.NetworkCredentialID) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
			return
		}
	}

	if req.CloudCredentialID != nil {
		if !req.CloudCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Cloud credential does not exists"},
			})
			return
		}

		if !roles.ReadByID(user, *req.CloudCredentialID) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
			return
		}
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID

	if err := db.TerrafromJobTemplates().Insert(req); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Could not create job template",
			Log:     logrus.Fields{"Job Template ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	rolesjob := new(rbac.TerraformJobTemplate)
	if !rbac.HasGlobalWrite(user) {
		rolesjob.Associate(req.ID, user.ID, rbac.RoleTypeUser, rbac.JobTemplateAdmin)
		activity.AddJobTemplateAssociationActivity(user, req)
	} else if orgId, err := req.GetOrganizationID(); err != nil && !rbac.IsOrganizationAdmin(orgId, user.ID) {
		rolesjob.Associate(req.ID, user.ID, rbac.RoleTypeUser, rbac.JobTemplateAdmin)
		activity.AddJobTemplateAssociationActivity(user, req)
	}

	activity.AddTJobTemplateActivity(common.Create, user, req)
	metadata.JTemplateMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

// UpdateJTemplate is a Gin handler function which updates a Job Template using request payload
// A success returns 200 status code
// A failure returns 500 status code
// if the request body is invalid returns serialized Error model with 400 status code
func (ctrl TJobTmplController) Update(c *gin.Context) {
	jobTemplate := c.MustGet(cTerraformJobTemplate).(terraform.JobTemplate)
	tmpJobTemplate := jobTemplate
	user := c.MustGet(cUser).(common.User)

	var req terraform.JobTemplate
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// check the project exist or not
	if !req.ProjectExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Project does not exists.",
		})
		return
	}

	if req.Name != jobTemplate.Name && !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Job Template with this name already exists.",
		})
		return
	}

	roles := new(rbac.Credential)
	if req.MachineCredentialID != nil {
		if !req.MachineCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Machine Credential does not exists"},
			})
			return
		}

		if !roles.ReadByID(user, *req.MachineCredentialID) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
			return
		}
	}

	if req.NetworkCredentialID != nil {
		if !req.NetworkCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Network credential does not exists"},
			})
			return
		}

		if !roles.ReadByID(user, *req.NetworkCredentialID) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
			return
		}
	}

	if req.CloudCredentialID != nil {
		if !req.CloudCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Cloud credential does not exists"},
			})
			return
		}

		if !roles.ReadByID(user, *req.CloudCredentialID) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
			return
		}
	}

	jobTemplate.Name = strings.Trim(req.Name, " ")
	jobTemplate.JobType = req.JobType
	jobTemplate.ProjectID = req.ProjectID
	jobTemplate.MachineCredentialID = req.MachineCredentialID
	jobTemplate.Description = strings.Trim(req.Description, " ")
	jobTemplate.Vars = req.Vars
	jobTemplate.PromptVariables = req.PromptVariables
	jobTemplate.CloudCredentialID = req.CloudCredentialID
	jobTemplate.NetworkCredentialID = req.NetworkCredentialID
	jobTemplate.PromptCredential = req.PromptCredential
	jobTemplate.PromptJobType = req.PromptJobType
	jobTemplate.AllowSimultaneous = req.AllowSimultaneous
	jobTemplate.Modified = time.Now()
	jobTemplate.ModifiedByID = user.ID

	if err := db.TerrafromJobTemplates().UpdateId(jobTemplate.ID, jobTemplate); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating job template",
			Log:     logrus.Fields{"ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	rolesjob := new(rbac.TerraformJobTemplate)
	if !rbac.HasGlobalWrite(user) {
		rolesjob.Associate(jobTemplate.ID, user.ID, rbac.RoleTypeUser, rbac.JobTemplateAdmin)
	} else if orgId, err := req.GetOrganizationID(); err != nil && !rbac.IsOrganizationAdmin(orgId, user.ID) {
		rolesjob.Associate(jobTemplate.ID, user.ID, rbac.RoleTypeUser, rbac.JobTemplateAdmin)
	}

	activity.AddTJobTemplateActivity(common.Update, user, tmpJobTemplate, jobTemplate)
	metadata.JTemplateMetadata(&jobTemplate)
	c.JSON(http.StatusOK, jobTemplate)
}

// RemoveJTemplate is a Gin handler function which removes a Job Template object from the database
// A success returns 204 status code
// A failure returns 500 status code
func (ctrl TJobTmplController) Delete(c *gin.Context) {
	jobTemplate := c.MustGet(cTerraformJobTemplate).(terraform.JobTemplate)
	user := c.MustGet(cUser).(common.User)

	if _, err := db.TerrafromJobs().RemoveAll(bson.M{"job_template_id": jobTemplate.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing jobs",
			Log:     logrus.Fields{"Job Template ID": jobTemplate.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if err := db.TerrafromJobTemplates().RemoveId(jobTemplate.ID); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing job tempalte",
			Log:     logrus.Fields{"Job Template ID": jobTemplate.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddTJobTemplateActivity(common.Delete, user, jobTemplate)
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
func (ctrl TJobTmplController) ActivityStream(c *gin.Context) {
	jtemplate := c.MustGet(cTerraformJobTemplate).(terraform.JobTemplate)
	var activities []terraform.ActivityJobTemplate
	var act terraform.ActivityJobTemplate
	iter := db.ActivityStream().Find(bson.M{"object1._id": jtemplate.ID}).Iter()
	for iter.Next(&act) {
		metadata.ActivityJobTemplateMetadata(&act)
		metadata.JTemplateMetadata(&act.Object1)
		if act.Object2 != nil {
			metadata.JTemplateMetadata(act.Object2)
		}
		activities = append(activities, act)
	}

	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting activities",
			Log:     logrus.Fields{"Job Template ID": jtemplate.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	count := len(activities)
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
		Data:     activities[pgi.Skip():pgi.End()],
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
func (ctrl TJobTmplController) Jobs(c *gin.Context) {
	jobTemplate := c.MustGet(cTerraformJobTemplate).(terraform.JobTemplate)

	var jbs []terraform.Job
	iter := db.TerrafromJobs().Find(bson.M{"job_template_id": jobTemplate.ID}).Iter()
	var tmpJob terraform.Job
	for iter.Next(&tmpJob) {
		metadata.JobMetadata(&tmpJob)
		jbs = append(jbs, tmpJob)
	}

	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting jobs",
			Log:     logrus.Fields{"Job Template ID": jobTemplate.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	count := len(jbs)
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
		Data:     jbs[pgi.Skip():pgi.End()],
	})
}

// Launch creates a new job and adds the job into job queue. If any
// passwords, inventory, or extra variables (extra_vars) are required, they must
// be passed via POST data, with extra_vars given as a JSON string.
// If `credential_needed_to_start` is `true` then the `credential` field is required
// and if the `inventory_needed_to_start` is `True` then the `inventory` is required as well.
// success returns JSON serialized Job model with 201 status code
// if the request body is invalid returns JSON serialized Error model with 400 status code
func (ctrl TJobTmplController) Launch(c *gin.Context) {
	template := c.MustGet(cTerraformJobTemplate).(terraform.JobTemplate)
	user := c.MustGet(cUser).(common.User)
	var req terraform.Launch
	if err := binding.JSON.Bind(c.Request, &req); err != nil && err != io.EOF {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// create new Job
	job := terraform.Job{
		ID:                  bson.NewObjectId(),
		Name:                template.Name,
		Description:         template.Description,
		LaunchType:          "manual",
		CancelFlag:          false,
		Status:              "new",
		JobType:             template.JobType,
		Vars:                template.Vars,
		Parallelism:         template.Parallelism,
		MachineCredentialID: template.MachineCredentialID,
		JobTemplateID:       template.ID,
		ProjectID:           template.ProjectID,
		NetworkCredentialID: template.NetworkCredentialID,
		CloudCredentialID:   template.CloudCredentialID,
		SCMCredentialID:     nil,
		CreatedByID:         user.ID,
		ModifiedByID:        user.ID,
		Created:             time.Now(),
		Modified:            time.Now(),
		PromptCredential:    template.PromptCredential,
		PromptJobType:       template.PromptJobType,
		PromptVariables:     template.PromptVariables,
		AllowSimultaneous:   template.AllowSimultaneous,
	}

	// if prompt is true override Job template
	// if not provided return an error message
	if template.PromptVariables {
		if !(len(req.Vars) > 0) {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Additional variables required.",
			})
			return
		}

		job.Vars = req.Vars
	}

	if template.PromptJobType {
		if !(len(req.JobType) > 0) {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Job type required.",
			})
			return
		}

		job.JobType = req.JobType
	}

	// create new Ansible runner Job
	runnerJob := types.TerraformJob{
		Job:      job,
		Template: template,
		User:     user,
	}

	if template.PromptCredential {
		if req.MachineCredentialID == nil {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Credential required"},
			})
			return
		}
		job.MachineCredentialID = req.MachineCredentialID
	}

	if job.NetworkCredentialID != nil {
		var credential common.Credential
		if err := db.Credentials().FindId(*job.NetworkCredentialID).One(&credential); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while getting network credential",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}
		runnerJob.Network = credential
	}

	if job.CloudCredentialID != nil {
		var credential common.Credential
		if err := db.Credentials().FindId(*job.CloudCredentialID).One(&credential); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while getting cloud credential",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}
		runnerJob.Cloud = credential
	}

	if job.MachineCredentialID != nil {
		var credential common.Credential
		if err := db.Credentials().FindId(*job.MachineCredentialID).One(&credential); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while getting machine credential",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}
		runnerJob.Machine = credential
	}

	var project common.Project
	if err := db.Projects().FindId(job.ProjectID).One(&project); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting project",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}
	runnerJob.Project = project

	// Get jwt token for authorize API
	var token jwt.LocalToken
	if err := jwt.NewAuthToken(&token); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting token",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}
	runnerJob.Token = token.Token

	if err := db.TerrafromJobs().Insert(job); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while creating job",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	if runnerJob.Project.ScmUpdateOnLaunch {
		tj, err := sync.UpdateProject(project)
		runnerJob.PreviousJob = tj
		if err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while creating update job",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}
	}

	// Add the job to queue
	jobQueue := queue.OpenTerraformQueue()
	jobBytes, err := json.Marshal(runnerJob)
	if err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while queueing job",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	jobQueue.PublishBytes(jobBytes)
	metadata.JobMetadata(&job)
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
func (ctrl TJobTmplController) LaunchInfo(c *gin.Context) {
	// get template from the gin.Context
	jt := c.MustGet(cTerraformJobTemplate).(terraform.JobTemplate)

	var isCredentialNeeded bool

	defaults := gin.H{
		"vars":     jt.Vars,
		"job_type": jt.JobType,
	}

	var cred common.Credential

	if err := db.Credentials().FindId(jt.MachineCredentialID).One(&cred); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Cound not find Credential")
		defaults["credential"] = nil
		isCredentialNeeded = true
	} else {
		defaults["credential"] = gin.H{
			"id":   cred.ID,
			"name": cred.Name,
		}
	}

	resp := gin.H{
		"passwords_needed_to_start":  []gin.H{},
		"ask_variables_on_launch":    jt.PromptVariables,
		"ask_job_type_on_launch":     jt.PromptJobType,
		"ask_credential_on_launch":   jt.PromptCredential,
		"variables_needed_to_start":  []gin.H{},
		"credential_needed_to_start": isCredentialNeeded,
		"job_template_data": gin.H{
			"id":          jt.ID.Hex(),
			"name":        jt.Name,
			"description": jt.Description,
		},
		"defaults": defaults,
	}

	c.JSON(http.StatusOK, resp)
}

// ObjectRoles is a Gin handler function
// This returns available roles can be associated with a Terraform Job Template model
func (ctrl TJobTmplController) ObjectRoles(c *gin.Context) {
	jobTemplate := c.MustGet(cTerraformJobTemplate).(terraform.JobTemplate)

	roles := []gin.H{
		{
			"type": "role",
			"links": gin.H{
				"job_template": "/v1/terraform_job_templates/" + jobTemplate.ID.Hex(),
			},
			"summary_fields": gin.H{
				"resource_name":              jobTemplate.Name,
				"resource_type":              "terraform_job_template",
				"resource_type_display_name": "Job Template",
			},
			"name":        "admin",
			"description": "Can manage all aspects of the job template",
		},
		{
			"type": "role",
			"related": gin.H{
				"job_template": "/v1/terraform_job_templates/" + jobTemplate.ID.Hex(),
			},
			"summary_fields": gin.H{
				"resource_name":              jobTemplate.Name,
				"resource_type":              "terraform_job_template",
				"resource_type_display_name": "Job Template",
			},
			"name":        "read",
			"description": "May view settings for the job template",
		},
		{
			"type": "role",
			"related": gin.H{
				"job_template": "/v1/terraform_job_templates/" + jobTemplate.ID.Hex(),
			},
			"summary_fields": gin.H{
				"resource_name":              jobTemplate.Name,
				"resource_type":              "terraform_job_template",
				"resource_type_display_name": "Job Template",
			},
			"name":        "execute",
			"description": "May run the job template",
		},
	}

	count := len(roles)
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
		Data:     roles[pgi.Skip():pgi.End()],
	})

}
