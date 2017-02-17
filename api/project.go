package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/exec/sync"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
)

// Keys for project related items stored in the Gin Context
const (
	CTXProject   = "project"
	CTXProjectID = "project_id"
)

type ProjectController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXProjectID from Gin Context and retrieves project data from the collection
// and store credential data under key CTXProject in Gin Context
func (ctrl ProjectController) Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(CTXProjectID, c)
	user := c.MustGet(CTXUser).(common.User)

	if err != nil {
		log.WithFields(log.Fields{
			"Project ID": ID,
			"Error":      err.Error(),
		}).Errorln("Error while getting Project ID url parameter")
		c.JSON(http.StatusNotFound, common.Error{
			Code:   http.StatusNotFound,
			Errors: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var project common.Project
	err = db.Projects().FindId(bson.ObjectIdHex(ID)).One(&project)
	if err != nil {
		log.WithFields(log.Fields{
			"Project ID": ID,
			"Error":      err.Error(),
		}).Errorln("Error while retriving Project form the database")
		c.JSON(http.StatusNotFound, common.Error{
			Code:   http.StatusNotFound,
			Errors: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	roles := new(rbac.Project)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(user, project) {
				c.JSON(http.StatusUnauthorized, common.Error{
					Code:   http.StatusUnauthorized,
					Errors: []string{"You do not have permission to perform this action."},
				})
				c.Abort()
				return
			}
		}
	case "PUT", "DELETE", "PATCH":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(user, project) {
				c.JSON(http.StatusUnauthorized, common.Error{
					Code:   http.StatusUnauthorized,
					Errors: []string{"You do not have permission to perform this action."},
				})
				c.Abort()
				return
			}
		}
	}

	c.Set(CTXProject, project)
	c.Next()
}

// GetProject returns the project as a JSON object
func (ctrl ProjectController) One(c *gin.Context) {
	project := c.MustGet(CTXProject).(common.Project)
	metadata.ProjectMetadata(&project)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, project)
}

// GetProjects returns a JSON array of projects
func (ctrl ProjectController) All(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"type", "status"}, match)
	match = parser.Lookups([]string{"name"}, match)

	query := db.Projects().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	log.WithFields(log.Fields{
		"Query": query,
	}).Debugln("Parsed query")

	roles := new(rbac.Project)
	var projects []common.Project
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpProject common.Project
	// iterate over all and only get valid objects
	for iter.Next(&tmpProject) {
		// Skip if the user doesn't have read permission
		if !roles.Read(user, tmpProject) {
			continue
		}
		// skip to next
		metadata.ProjectMetadata(&tmpProject)
		// good to go add to list
		projects = append(projects, tmpProject)
	}
	if err := iter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Project data from the database")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Project"},
		})
		return
	}

	count := len(projects)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Project page does not exist")
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
		Results:  projects[pgi.Skip():pgi.End()],
	})
}

// AddProject is a Gin handler function which creates a new project using request payload.
// This accepts Project model.
func (ctrl ProjectController) Create(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	var req common.Project
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Invlid JSON request")
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: validate.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if !req.OrganizationExist() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: []string{"Organization does not exists."},
		})
		return
	}

	// if a project exists within the Organization, reject the request
	if !req.IsUnique() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: []string{"Project with this Name and Organization already exists."},
		})
		return
	}

	// check whether the scm credential exist or not
	if req.ScmCredentialID != nil {
		if !req.SCMCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"SCM Credential does not exists."},
			})
			return
		}
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	req.ID = bson.NewObjectId()
	req.LocalPath = util.Config.ProjectsHome + "/" + req.ID.Hex()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	req.Created = time.Now()
	req.Modified = time.Now()

	if err := db.Projects().Insert(req); err != nil {
		log.WithFields(log.Fields{
			"Project ID": req.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while creating Project")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while creating Project"},
		})
		return
	}
	// before set metadata update the project
	if sysJobID, err := sync.UpdateProject(req); err != nil {
		log.WithFields(log.Fields{
			"SystemJob ID": sysJobID.Job.ID.Hex(),
			"Error":        err.Error(),
		}).Errorln("Error while scm update")
	}

	roles := new(rbac.Project)
	if !rbac.HasGlobalWrite(user) && !rbac.IsOrganizationAdmin(req.OrganizationID, user.ID) {
		if err := roles.Associate(req.ID, user.ID, rbac.RoleTypeUser, rbac.ProjectAdmin); err != nil {
			log.WithFields(log.Fields{
				"User ID":   user.ID,
				"Object ID": req.ID,
				"Error":     err.Error(),
			}).Errorln("Admin roles association failed")
		}
	}

	// add new activity to activity stream
	activity.AddProjectActivity(common.Create, user, req)

	metadata.ProjectMetadata(&req)

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// UpdateProject is a Gin handler function which updates a project using request payload.
// This replaces all the fields in the database, empty "" fields and
// unspecified fields will be removed from the database object.
func (ctrl ProjectController) Update(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(CTXProject).(common.Project)
	tmpProject := project
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req common.Project
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: validate.GetValidationErrors(err),
		})
		return
	}

	// Reject the request if project type is going to change
	if project.Kind != req.Kind && req.Kind != "" {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: []string{"Project Kind property cannot be modified."},
		})
		return
	}

	// check whether the organization exist or not
	if !req.OrganizationExist() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: []string{"Organization does not exists."},
		})
		return
	}

	if req.Name != project.Name {
		// if a project exists within the Organization, reject the request
		if !req.IsUnique() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Project with this Name and Organization already exists."},
			})
			return
		}
	}

	// check whether the ScmCredential exist or not
	if req.ScmCredentialID != nil {
		if !req.SCMCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"SCM Credential does not exists."},
			})
			return
		}
	}

	// trim strings white space
	project.Name = strings.Trim(req.Name, " ")
	project.Description = strings.Trim(req.Description, " ")
	project.ScmType = req.ScmType
	project.OrganizationID = req.OrganizationID
	project.Description = req.Description
	project.ScmURL = req.ScmURL
	project.ScmBranch = req.ScmBranch
	project.ScmClean = req.ScmClean
	project.ScmDeleteOnUpdate = req.ScmDeleteOnUpdate
	project.ScmCredentialID = req.ScmCredentialID
	project.ScmDeleteOnNextUpdate = req.ScmDeleteOnNextUpdate
	project.ScmUpdateOnLaunch = req.ScmUpdateOnLaunch
	project.ScmUpdateCacheTimeout = req.ScmUpdateCacheTimeout
	project.Modified = time.Now()

	// update object
	if err := db.Projects().UpdateId(project.ID, project); err != nil {
		log.WithFields(log.Fields{
			"Project ID": req.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while updating Project")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while updating Project"},
		})
		return
	}
	// before set metadata update the project
	if sysJobID, err := sync.UpdateProject(project); err != nil {
		log.WithFields(log.Fields{
			"SystemJob ID": sysJobID.Job.ID.Hex(),
			"Error":        err.Error(),
		}).Errorln("Error while scm update")
	}
	// add new activity to activity stream
	activity.AddProjectActivity(common.Update, user, tmpProject, project)
	// set `related` and `summary` fields
	metadata.ProjectMetadata(&project)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, project)
}

// PatchProject partially updates a project
// this only updates given files in the request playload
func (ctrl ProjectController) Patch(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(CTXProject).(common.Project)
	tmpProject := project
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req common.PatchProject
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: validate.GetValidationErrors(err),
		})
		return
	}

	if req.OrganizationID != nil {
		project.OrganizationID = *req.OrganizationID
		// check whether the organization exist or not
		if !project.OrganizationExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Organization does not exists."},
			})
			return
		}
	}

	if req.Name != nil && *req.Name != project.Name {
		// if a project exists within the Organization, reject the request
		if !project.IsUnique() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Project with this Name and Organization already exists."},
			})
			return
		}
	}

	// check whether the ScmCredential exist
	// if the credential is empty
	if req.ScmCredentialID != nil {
		project.ScmCredentialID = req.ScmCredentialID
		if !project.SCMCredentialExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"SCM Credential does not exists."},
			})
			return
		}
	}

	// trim strings white space
	if req.Name != nil {
		project.Name = strings.Trim(*req.Name, " ")
	}

	if req.Description != nil {
		project.Description = strings.Trim(*req.Description, " ")
	}

	if req.ScmType != nil {
		project.ScmType = *req.ScmType
	}
	if req.Description != nil {
		project.Description = *req.Description
	}

	if req.ScmURL != nil {
		project.ScmURL = *req.ScmURL
	}

	if req.ScmBranch != nil {
		project.ScmBranch = *req.ScmBranch
	}

	if req.ScmClean != nil {
		project.ScmClean = *req.ScmClean
	}

	if req.ScmDeleteOnUpdate != nil {
		project.ScmDeleteOnUpdate = *req.ScmDeleteOnUpdate
	}

	if req.ScmDeleteOnNextUpdate != nil {
		project.ScmDeleteOnNextUpdate = *req.ScmDeleteOnNextUpdate
	}

	if req.ScmUpdateOnLaunch != nil {
		project.ScmUpdateOnLaunch = *req.ScmUpdateOnLaunch
	}

	if req.ScmUpdateCacheTimeout != nil {
		project.ScmUpdateCacheTimeout = *req.ScmUpdateCacheTimeout
	}

	project.ModifiedByID = user.ID
	project.Modified = time.Now()

	// update object
	if err := db.Projects().UpdateId(project.ID, project); err != nil {
		log.WithFields(log.Fields{
			"Project ID": project.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while updating Project")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while updating Project"},
		})
		return
	}
	// before set metadata update the project
	if sysJobID, err := sync.UpdateProject(project); err != nil {
		log.WithFields(log.Fields{
			"SystemJob ID": sysJobID.Job.ID.Hex(),
			"Error":        err.Error(),
		}).Errorln("Error while scm update")
	}
	// add new activity to activity stream
	activity.AddProjectActivity(common.Update, user, tmpProject, project)
	// set `related` and `summary` feilds
	metadata.ProjectMetadata(&project)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, project)
}

// RemoveProject is a Gin handler function which removes a project object from the database
func (ctrl ProjectController) Delete(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(CTXProject).(common.Project)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	changes, err := db.Jobs().RemoveAll(bson.M{"project_id": project.ID})
	if err != nil {
		log.WithFields(log.Fields{
			"Project ID": project.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while removing Project Jobs")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while removing Project Jobs"},
		})
		return
	}

	log.Infoln("Jobs remove info:", changes.Removed)

	changes, err = db.JobTemplates().RemoveAll(bson.M{"project_id": project.ID})
	if err != nil {
		log.Errorln("Error while removing Project Job Templates:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while removing Project Job Templates"},
		})
		return
	}

	log.Infoln("Job Template remove info:", changes.Removed)

	// remove object from the collection
	if err = db.Projects().RemoveId(project.ID); err != nil {
		log.Errorln("Error while removing Project:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while removing Project"},
		})
		return
	}

	// cleanup directories from a concurrent thread
	go func() {
		if err := os.RemoveAll(project.LocalPath); err != nil {
			log.Errorln("An error occured while removing project directory", err.Error())
		}
	}()

	// add new activity to activity stream
	activity.AddProjectActivity(common.Delete, user, project)
	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

// Playbooks returs array of playbooks contains in project directory
func (ctrl ProjectController) Playbooks(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(CTXProject).(common.Project)

	// if project is an terraform project return no content header

	if project.Kind == "terraform" {
		c.JSON(http.StatusNoContent, common.Error{
			Code:   http.StatusNoContent,
			Errors: []string{"Invalid project kind"},
		})
		return
	}

	files := []string{}

	if _, err := os.Stat(project.LocalPath); err != nil {
		if os.IsNotExist(err) {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Errorln("Project directory does not exist")
			c.JSON(http.StatusNoContent, common.Error{
				Code:   http.StatusNoContent,
				Errors: []string{"Project directory does not exist"},
			})
			return
		}

		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Could not read project directory")
		c.JSON(http.StatusNoContent, common.Error{
			Code:   http.StatusNoContent,
			Errors: []string{"Could not read project directory"},
		})
		return
	}

	err := filepath.Walk(project.LocalPath, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			r, err := regexp.MatchString(".yml|.yaml|.json", f.Name())
			if err == nil && r {
				files = append(files, strings.TrimPrefix(path, project.LocalPath+"/"))
			}
		}
		return nil
	})

	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while getting Playbooks")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Playbooks"},
		})
		return
	}

	c.JSON(http.StatusOK, files)
}

// Teams returns the list of teams that has permission to access
// project object in the gin.Context
func (ctrl ProjectController) OwnerTeams(c *gin.Context) {
	team := c.MustGet(CTXProject).(common.Project)

	var tms []common.Team

	var tmpTeam common.Team
	for _, v := range team.Roles {
		if v.Type == "team" {
			err := db.Teams().FindId(v.GranteeID).One(&tmpTeam)
			if err != nil {
				log.WithFields(log.Fields{
					"Error": err.Error(),
				}).Errorln("Error while getting Teams")
				c.JSON(http.StatusInternalServerError, common.Error{
					Code:   http.StatusInternalServerError,
					Errors: []string{"Error while getting Teams"},
				})
				return
			}

			metadata.TeamMetadata(&tmpTeam)
			tms = append(tms, tmpTeam)
		}
	}

	count := len(tms)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  tms[pgi.Skip():pgi.End()],
	})
}

// ActivityStream returns the activities of the user on projects
func (ctrl ProjectController) ActivityStream(c *gin.Context) {
	project := c.MustGet(CTXProject).(common.Project)

	var activities []common.ActivityProject
	var act common.ActivityProject
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": project.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&act) {
		metadata.ActivityProjectMetadata(&act)
		metadata.ProjectMetadata(&act.Object1)
		//apply metadata only when Object2 is available
		if act.Object2 != nil {
			metadata.ProjectMetadata(act.Object2)
		}
		//add to activities list
		activities = append(activities, act)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Activities"},
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  activities[pgi.Skip():pgi.End()],
	})
}

// ProjectUpdates is a Gin handler function which returns project update jobs
func (ctrl ProjectController) ProjectUpdates(c *gin.Context) {

	parser := util.NewQueryParser(c)

	match := bson.M{}
	match = parser.Match([]string{"status", "type", "failed"}, match)
	match = parser.Lookups([]string{"id", "name", "labels"}, match)
	log.Infoln(match)

	// get only project update jobs
	match["job_type"] = "update_job"

	query := db.Jobs().Find(match) // prepare the query

	// set sort value to the query based on request parameters
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	log.WithFields(log.Fields{
		"Query": query,
	}).Debugln("Parsed query")

	var jobs []ansible.Job

	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpJob ansible.Job
	// iterate over all and only get valid objects
	for iter.Next(&tmpJob) {
		// TODO: if the user doesn't have access to credential
		// skip to next
		metadata.JobMetadata(&tmpJob)
		// good to go add to list
		jobs = append(jobs, tmpJob)
	}
	if err := iter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Credential data from the database")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Credential"},
		})
		return
	}

	count := len(jobs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Project Updates page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
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

// SCMUpdateInfo returns whether a project can be updated or not
func (ctrl ProjectController) SCMUpdateInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"can_update": true})
}

// SCMUpdate creates a new system job to update a project
func (ctrl ProjectController) SCMUpdate(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(CTXProject).(common.Project)
	user := c.MustGet(CTXUser).(common.User)

	roles := new(rbac.Project)
	if !roles.Update(user, project) {
		c.JSON(http.StatusUnauthorized, common.Error{
			Code:   http.StatusUnauthorized,
			Errors: []string{"Unauthorized"},
		})
		c.Abort()
		return
	}

	var req common.SCMUpdate
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// accept nil request body for POST request, since all the fields are optional
		if err != io.EOF {
			// Return 400 if request has bad JSON format
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: validate.GetValidationErrors(err),
			})
			return
		}
	}

	updateID, err := sync.UpdateProject(project)

	if err != nil {
		c.JSON(http.StatusMethodNotAllowed, common.Error{
			Code:    http.StatusMethodNotAllowed,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"project_update": updateID.Job.ID.Hex()})
}
