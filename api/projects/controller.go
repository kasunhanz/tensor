package projects

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bitbucket.pearson.com/apseng/tensor/api/helpers"
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/roles"
	"bitbucket.pearson.com/apseng/tensor/runners"
	"bitbucket.pearson.com/apseng/tensor/util"
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/mgo.v2/bson"
)

const _CTX_PROJECT = "project"
const _CTX_USER = "user"
const _CTX_PROJECT_ID = "project_id"

// ProjectMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(_CTX_PROJECT_ID, c)

	if err != nil {
		log.Errorln("Error while getting the Project:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var project models.Project
	err = db.Projects().FindId(bson.ObjectIdHex(ID)).One(&project)
	if err != nil {
		log.Errorln("Error while getting the Project:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	c.Set(_CTX_PROJECT, project)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetProject(c *gin.Context) {
	project := c.MustGet(_CTX_PROJECT).(models.Project)
	metadata.ProjectMetadata(&project)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, project)
}

// GetProjects returns a JSON array of projects
func GetProjects(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"type", "status"}, match)
	match = parser.Lookups([]string{"name"}, match)

	query := db.Projects().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var projects []models.Project
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpProject models.Project
	// iterate over all and only get valid objects
	for iter.Next(&tmpProject) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.ProjectRead(user, tmpProject) {
			continue
		}
		metadata.ProjectMetadata(&tmpProject)
		// good to go add to list
		projects = append(projects, tmpProject)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Project data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Project"},
		})
		return
	}

	count := len(projects)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  projects[pgi.Skip():pgi.End()],
	})
}

// AddProject creates a new project
func AddProject(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Project
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if !helpers.OrganizationExist(req.OrganizationID) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	// if a project exists within the Organization, reject the request
	if helpers.IsNotUniqueProject(req.Name, req.OrganizationID) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Project with this Name and Organization already exists."},
		})
		return
	}

	// check whether the scm credential exist or not
	if req.ScmCredentialID != nil {
		if !helpers.SCMCredentialExist(*req.ScmCredentialID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"SCM Credential does not exists."},
			})
			return
		}
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	req.ID = bson.NewObjectId()
	req.LocalPath = "/opt/tensor/projects/" + req.ID.Hex()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	req.Created = time.Now()
	req.Modified = time.Now()

	if err := db.Projects().Insert(req); err != nil {
		log.Errorln("Error while creating Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while creating Project"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Project "+req.Name+" created")

	// before set metadata update the project
	if sysJobID, err := runners.UpdateProject(req); err != nil {
		log.Errorln("Error while scm update "+sysJobID.Job.ID.Hex(), err)
	}

	metadata.ProjectMetadata(&req)

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// UpdateProject will update the Project
func UpdateProject(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(_CTX_PROJECT).(models.Project)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Project
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if !helpers.OrganizationExist(req.OrganizationID) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	if req.Name != project.Name {
		// if a project exists within the Organization, reject the request
		if helpers.IsNotUniqueProject(req.Name, req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Project with this Name and Organization already exists."},
			})
			return
		}
	}

	// check whether the ScmCredential exist or not
	if req.ScmCredentialID != nil {
		if !helpers.SCMCredentialExist(*req.ScmCredentialID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"SCM Credential does not exists."},
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
	project.ScmUrl = req.ScmUrl
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
		log.Errorln("Error while updating Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Project"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(project.ID, user.ID, "Project "+project.Name+" updated")

	// before set metadata update the project
	if sysJobID, err := runners.UpdateProject(project); err != nil {
		log.Errorln("Error while scm update "+sysJobID.Job.ID.Hex(), err)
	}

	// set `related` and `summary` feilds
	metadata.ProjectMetadata(&project)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, project)
}

// UpdateProject will update the Project
func PatchProject(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(_CTX_PROJECT).(models.Project)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.PatchProject
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if req.OrganizationID != nil {
		// check whether the organization exist or not
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	if req.Name != nil && *req.Name != project.Name {
		ogID := project.OrganizationID
		if req.OrganizationID != nil {
			ogID = *req.OrganizationID
		}
		// if a project exists within the Organization, reject the request
		if helpers.IsNotUniqueProject(*req.Name, ogID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Project with this Name and Organization already exists."},
			})
			return
		}
	}

	// check whether the ScmCredential exist
	// if the credential is empty
	if req.ScmCredentialID != nil && len(*req.ScmCredentialID) == 12 {
		if !helpers.SCMCredentialExist(*req.ScmCredentialID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"SCM Credential does not exists."},
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

	if req.OrganizationID != nil {
		project.OrganizationID = *req.OrganizationID
	}

	if req.Description != nil {
		project.Description = *req.Description
	}

	if req.ScmUrl != nil {
		project.ScmUrl = *req.ScmUrl
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

	if req.ScmCredentialID != nil {
		// if empty string then make the credential null
		if len(*req.ScmCredentialID) == 12 {
			project.ScmCredentialID = req.ScmCredentialID
		} else {
			project.ScmCredentialID = nil
		}
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
		log.Errorln("Error while updating Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Project"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(project.ID, user.ID, "Project "+project.Name+" updated")

	// before set metadata update the project
	if sysJobID, err := runners.UpdateProject(project); err != nil {
		log.Errorln("Error while scm update "+sysJobID.Job.ID.Hex(), err)
	}

	// set `related` and `summary` feilds
	metadata.ProjectMetadata(&project)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, project)
}

// RemoveProject will remove the Project
// from the db.DBC_PROJECTS collection
func RemoveProject(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(_CTX_PROJECT).(models.Project)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	changes, err := db.Jobs().RemoveAll(bson.M{"project_id": project.ID})
	if err != nil {
		log.Errorln("Error while removing Project Jobs:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Project Jobs"},
		})
		return
	}

	log.Infoln("Jobs remove info:", changes.Removed)

	changes, err = db.JobTemplates().RemoveAll(bson.M{"project_id": project.ID})
	if err != nil {
		log.Errorln("Error while removing Project Job Templates:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Project Job Templates"},
		})
		return
	}

	log.Infoln("Job Template remove info:", changes.Removed)

	// remove object from the collection
	if err = db.Projects().RemoveId(project.ID); err != nil {
		log.Errorln("Error while removing Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Project"},
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
	addActivity(project.ID, user.ID, "Project "+project.Name+" deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

// Playbooks returs array of playbooks contains in project directory
func Playbooks(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(_CTX_PROJECT).(models.Project)

	files := []string{}

	if _, err := os.Stat(project.LocalPath); err != nil {
		if os.IsNotExist(err) {
			log.Errorln("Project directory does not exist", err)
			c.JSON(http.StatusNoContent, models.Error{
				Code:     http.StatusNoContent,
				Messages: []string{"Project directory does not exist"},
			})
			return
		}

		log.Errorln("Could not read project directory", err)
		c.JSON(http.StatusNoContent, models.Error{
			Code:     http.StatusNoContent,
			Messages: []string{"Could not read project directory"},
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
		log.Errorln("Error while getting Playbooks:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Playbooks"},
		})
		return
	}

	c.JSON(http.StatusOK, files)
}

func Teams(c *gin.Context) {
	team := c.MustGet(_CTX_PROJECT).(models.Project)

	var tms []models.Team

	var tmpTeam models.Team
	for _, v := range team.Roles {
		if v.Type == "team" {
			err := db.Teams().FindId(v.TeamID).One(&tmpTeam)
			if err != nil {
				log.Errorln("Error while getting Teams:", err)
				c.JSON(http.StatusInternalServerError, models.Error{
					Code:     http.StatusInternalServerError,
					Messages: []string{"Error while getting Teams"},
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
	c.JSON(http.StatusOK, models.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  tms[pgi.Skip():pgi.End()],
	})
}

// TODO: not complete
func ActivityStream(c *gin.Context) {
	project := c.MustGet(_CTX_PROJECT).(models.Project)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": project.ID, "type": _CTX_PROJECT}).All(&activities)

	if err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while Activities"},
		})
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
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  activities[pgi.Skip():pgi.End()],
	})
}

// GetJobs renders the Job as JSON
func ProjectUpdates(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

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

	var jobs []models.Job

	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpJob models.Job
	// iterate over all and only get valid objects
	for iter.Next(&tmpJob) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.JobRead(user, tmpJob) {
			continue
		}
		metadata.JobMetadata(&tmpJob)
		// good to go add to list
		jobs = append(jobs, tmpJob)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Credential"},
		})
		return
	}

	count := len(jobs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  jobs[pgi.Skip():pgi.End()],
	})
}

func SCMUpdateInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"can_update": true})
}

func SCMUpdate(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(_CTX_PROJECT).(models.Project)

	var req models.SCMUpdate
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// accept nil request body for POST request, since all the fields are optional
		if err != io.EOF {
			// Return 400 if request has bad JSON format
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: util.GetValidationErrors(err),
			})
		}
		return
	}

	updateId, err := runners.UpdateProject(project)

	if err != nil {
		c.JSON(http.StatusMethodNotAllowed, models.Error{
			Code:     http.StatusMethodNotAllowed,
			Messages: err,
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"project_update": updateId.Job.ID.Hex()})
}
