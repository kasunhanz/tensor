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

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
	"github.com/gin-gonic/gin"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
	"path"
)

// Keys for project related items stored in the Gin Context
const (
	cProject = "project"
	cProjectID = "project_id"
)

type ProjectController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXProjectID from Gin Context and retrieves project data from the collection
// and store credential data under key CTXProject in Gin Context
func (ctrl ProjectController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cProjectID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Project does not exist"})
		return
	}

	var project common.Project
	err := db.Projects().FindId(bson.ObjectIdHex(objectID)).One(&project)
	if err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Project does not exist",
			Log: logrus.Fields{
				"Project ID": objectID,
				"Error":      err.Error(),
			},
		})
		return
	}

	roles := new(rbac.Project)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(user, project) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "DELETE", "PATCH":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(user, project) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}

	c.Set(cProject, project)
	c.Next()
}

// GetProject returns the project as a JSON object
func (ctrl ProjectController) One(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)
	metadata.ProjectMetadata(&project)
	c.JSON(http.StatusOK, project)
}

// GetProjects returns a JSON array of projects
func (ctrl ProjectController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"type", "status"}, match)
	match = parser.Lookups([]string{"name"}, match)

	query := db.Projects().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.Project)
	var projects []common.Project
	iter := query.Iter()
	var tmpProject common.Project
	for iter.Next(&tmpProject) {
		if !roles.Read(user, tmpProject) {
			continue
		}
		metadata.ProjectMetadata(&tmpProject)
		projects = append(projects, tmpProject)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting project", Log: logrus.Fields{
				"Error": err.Error(),
			},
		})
		return
	}

	count := len(projects)
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
		Data:     projects[pgi.Skip():pgi.End()],
	})
}

// AddProject is a Gin handler function which creates a new project using request payload.
// This accepts Project model.
func (ctrl ProjectController) Create(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	var req common.Project
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	if !req.OrganizationExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Organization does not exists.",
		})
		return
	}
	// Check whether the user has permissions to associate the credential with organization
	if !(rbac.HasGlobalRead(user) || rbac.HasOrganizationRead(req.OrganizationID, user.ID)) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}
	if !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Project with this Name and Organization already exists.",
		})
		return
	}
	// check whether the scm credential exist or not
	if req.ScmCredentialID != nil {
		if !req.SCMCredentialExist() {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "SCM Credential does not exists.",
			})
			return
		}

		roles := new(rbac.Credential)
		cred, err := req.GetCredential()
		if err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "SCM Credential does not exists.",
			})
			return
		}

		if !roles.Read(user, cred) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
		}
	}

	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")
	req.ID = bson.NewObjectId()
	req.LocalPath = path.Join(util.Config.ProjectsHome, req.ID.Hex())
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	req.Created = time.Now()
	req.Modified = time.Now()

	if err := db.Projects().Insert(req); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Could not create project",
			Log:     logrus.Fields{"Project ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	roles := new(rbac.Project)
	if !(rbac.HasGlobalWrite(user) && rbac.IsOrganizationAdmin(req.OrganizationID, user.ID)) {
		roles.Associate(req.ID, user.ID, rbac.RoleTypeUser, rbac.ProjectAdmin)
		activity.AddActivity(activity.Associate, user.ID, req, user)
	}

	// before set metadata update the project
	if sysJobID, err := sync.UpdateProject(req); err != nil {
		logrus.WithFields(logrus.Fields{
			"SystemJob ID": sysJobID.Job.ID.Hex(),
			"Error":        err.Error(),
		}).Errorln("Error while scm update")
	}

	activity.AddActivity(activity.Create, user.ID, req, nil)
	metadata.ProjectMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

// UpdateProject is a Gin handler function which updates a project using request payload.
// This replaces all the fields in the database, empty "" fields and
// unspecified fields will be removed from the database object.
func (ctrl ProjectController) Update(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)
	tmpProject := project
	user := c.MustGet(cUser).(common.User)

	var req common.Project
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// Reject the request if project type is going to change
	if project.Kind != req.Kind && req.Kind != "" {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Project Kind property cannot be modified.",
		})
		return
	}
	// check whether the organization exist or not
	if !req.OrganizationExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Organization does not exists.",
		})
		return
	}

	// Check whether the user has permissions to associate the credential with organization
	if !(rbac.HasGlobalRead(user) || rbac.HasOrganizationRead(req.OrganizationID, user.ID)) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}

	if req.Name != project.Name && !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Project with this name and organization already exists.",
		})
		return
	}

	if req.ScmCredentialID != nil && !req.SCMCredentialExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "SCM Credential does not exists.",
		})
		return
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
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating project",
			Log:     logrus.Fields{"Project ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	roles := new(rbac.Project)
	if !(rbac.HasGlobalWrite(user) && rbac.IsOrganizationAdmin(project.OrganizationID, user.ID)) {
		roles.Associate(project.ID, user.ID, rbac.RoleTypeUser, rbac.ProjectAdmin)
		activity.AddActivity(activity.Associate, user.ID, project, user)
	}

	// before set metadata update the project
	if sysJobID, err := sync.UpdateProject(project); err != nil {
		logrus.WithFields(logrus.Fields{
			"SystemJob ID": sysJobID.Job.ID.Hex(),
			"Error":        err.Error(),
		}).Errorln("Error while scm update")
	}

	activity.AddActivity(activity.Update, user.ID, tmpProject, project)
	metadata.ProjectMetadata(&project)
	c.JSON(http.StatusOK, project)
}

// RemoveProject is a Gin handler function which removes a project object from the database
func (ctrl ProjectController) Delete(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)
	user := c.MustGet(cUser).(common.User)

	if _, err := db.Jobs().RemoveAll(bson.M{"project_id": project.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing project jobs",
			Log:     logrus.Fields{"Project ID": project.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.JobTemplates().RemoveAll(bson.M{"project_id": project.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing job templates",
			Log:     logrus.Fields{"Project ID": project.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if err := db.Projects().RemoveId(project.ID); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing project",
			Log:     logrus.Fields{"Project ID": project.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	// cleanup directories from a concurrent thread
	go func() {
		if err := os.RemoveAll(project.LocalPath); err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err.Error(),
			}).Errorln("An error occured while removing project directory")
		}
	}()

	activity.AddActivity(activity.Delete, user.ID, project, nil)
	c.AbortWithStatus(http.StatusNoContent)
}

// Playbooks returns array of playbooks contains in project directory
func (ctrl ProjectController) Playbooks(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)

	if project.Kind == "terraform" {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Invalid project kind.",
		})
		return
	}
	files := []string{}
	if _, err := os.Stat(project.LocalPath); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNoContent,
			Message: "Project directory does not exist",
			Log:     logrus.Fields{"Project ID": project.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if err := filepath.Walk(project.LocalPath, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			r, err := regexp.MatchString(".yml|.yaml|.json", f.Name())
			if err == nil && r {
				files = append(files, strings.TrimPrefix(path, project.LocalPath + "/"))
			}
		}
		return nil
	}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusInternalServerError,
			Message: "Error while getting playbooks",
			Log:     logrus.Fields{"Project ID": project.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, files)
}

// Teams returns the list of teams that has permission to access
// project object in the gin.Context
func (ctrl ProjectController) OwnerTeams(c *gin.Context) {
	team := c.MustGet(cProject).(common.Project)

	var tms []common.Team

	var tmpTeam common.Team
	for _, v := range team.Roles {
		if v.Type == "team" {
			err := db.Teams().FindId(v.GranteeID).One(&tmpTeam)
			if err != nil {
				AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
					Message: "Error while getting teams",
					Log:     logrus.Fields{"Error": err.Error()},
				})
				return
			}

			metadata.TeamMetadata(&tmpTeam)
			tms = append(tms, tmpTeam)
		}
	}

	count := len(tms)
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
		Data:     tms[pgi.Skip():pgi.End()],
	})
}

// ActivityStream returns the activities of the user on projects
func (ctrl ProjectController) ActivityStream(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)

	var activities []common.Activity
	var act common.Activity
	iter := db.ActivityStream().Find(bson.M{"object1_id": project.ID}).Iter()
	for iter.Next(&act) {
		metadata.ActivityProjectMetadata(&act)
		activities = append(activities, act)
	}

	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting activities",
			Log:     logrus.Fields{"Project ID": project.ID.Hex(), "Error": err.Error()},
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

// ProjectUpdates is a Gin handler function which returns project update jobs
func (ctrl ProjectController) ProjectUpdates(c *gin.Context) {
	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"status", "type", "failed"}, match)
	match = parser.Lookups([]string{"id", "name", "labels"}, match)
	match["job_type"] = "update_job"
	query := db.Jobs().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}
	var jobs []ansible.Job
	iter := query.Iter()
	var tmpJob ansible.Job
	for iter.Next(&tmpJob) {
		metadata.JobMetadata(&tmpJob)
		jobs = append(jobs, tmpJob)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusInternalServerError,
			Message: "Error while getting credential",
			Log:     logrus.Fields{"Error": err.Error()},
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

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:     jobs[pgi.Skip():pgi.End()],
	})
}

// SCMUpdateInfo returns whether a project can be updated or not
func (ctrl ProjectController) SCMUpdateInfo(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)
	user := c.MustGet(cUser).(common.User)

	roles := new(rbac.Project)
	if !roles.Update(user, project) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"can_update": true})
}

// SCMUpdate creates a new system job to update a project
func (ctrl ProjectController) SCMUpdate(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)
	user := c.MustGet(cUser).(common.User)

	roles := new(rbac.Project)
	if !roles.Update(user, project) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}

	var req common.SCMUpdate
	if err := binding.JSON.Bind(c.Request, &req); err != nil && err != io.EOF {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	updateID, err := sync.UpdateProject(project)

	if err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusMethodNotAllowed,
			Message: "SCM Update failed",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"project_update": updateID.Job.ID.Hex()})
}

// ObjectRoles is a Gin handler function
// This returns available roles can be associated with a Project model
func (ctrl ProjectController) ObjectRoles(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)

	roles := []gin.H{
		{
			"type": "role",
			"links": gin.H{
				"project": "/v1/projects/" + project.ID.Hex(),
			},
			"meta": gin.H{
				"resource_name":              project.Name,
				"resource_type":              "project",
				"resource_type_display_name": "Project",
			},
			"name":        "admin",
			"description": "Can manage all aspects of the project",
		},
		{
			"type": "role",
			"related": gin.H{
				"project": "/v1/projects/" + project.ID.Hex(),
			},
			"summary_fields": gin.H{
				"resource_name":              project.Name,
				"resource_type":              "project",
				"resource_type_display_name": "Project",
			},
			"name":        "use",
			"description": "Can use the project in a job template",
		},
		{
			"type": "role",
			"related": gin.H{
				"project": "/v1/projects/" + project.ID.Hex(),
			},
			"summary_fields": gin.H{
				"resource_name":              project.Name,
				"resource_type":              "project",
				"resource_type_display_name": "Project",
			},
			"name":        "update",
			"description": "May update project or inventory or group using the configured source update system",
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
