package projects

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"net/http"
	"gopkg.in/mgo.v2/bson"
	"time"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"bitbucket.pearson.com/apseng/tensor/util"
	"strconv"
	"path/filepath"
	"os"
	"regexp"
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
	"bitbucket.pearson.com/apseng/tensor/roles"
	"bitbucket.pearson.com/apseng/tensor/api/helpers"
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
		log.Print("Error while getting the Project:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
	}

	var project models.Project
	err = db.Projects().FindId(bson.ObjectIdHex(ID)).One(&project);
	if err != nil {
		log.Print("Error while getting the Project:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
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
	match := parser.Match([]string{"type", "status"})
	con := parser.IContains([]string{"name"});
	if con != nil {
		if match != nil {
			for i, v := range con {
				match[i] = v
			}
		} else {
			match = con
		}
	}

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
		if err := metadata.ProjectMetadata(&tmpProject); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Project",
			})
			return
		}
		// good to go add to list
		projects = append(projects, tmpProject)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Project data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Project",
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
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: projects[pgi.Skip():pgi.End()],
	})
}

// AddProject creates a new project
func AddProject(c *gin.Context) {
	var req models.Project

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

	// check whether the organization exist or not
	if !helpers.OrganizationExist(req.OrganizationID, c) {
		return
	}

	// check whether the scm credential exist or not
	if req.ScmCredentialID != nil {
		if !helpers.SCMCredentialExist(*req.ScmCredentialID, c) {
			return
		}
	}

	req.ID = bson.NewObjectId()
	req.CreatedBy = user.ID
	req.ModifiedBy = user.ID
	req.Created = time.Now()
	req.Modified = time.Now()

	err = db.Projects().Insert(req);
	if err != nil {
		log.Println("Error while creating Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Project",
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Project " + req.Name + " created")

	err = metadata.ProjectMetadata(&req);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Project",
		})
		return
	}

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
	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: util.GetValidationErrors(err),
		})
		return
	}


	// check whether the organization exist or not
	if !helpers.OrganizationExist(req.OrganizationID, c) {
		return
	}

	// check whether the ScmCredential exist or not
	if req.ScmCredentialID != nil {
		if !helpers.SCMCredentialExist(*req.ScmCredentialID, c) {
			return
		}
	}

	req.CreatedBy = project.CreatedBy
	req.ModifiedBy = user.ID
	req.Created = project.Created
	req.Modified = time.Now()

	// update object
	err := db.Projects().UpdateId(project.ID, req);
	if err != nil {
		log.Println("Error while updating Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while updating Project",
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Project " + req.Name + " updated")

	// set `related` and `summary` feilds
	err = metadata.ProjectMetadata(&req);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Project",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, req)
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
		log.Println("Error while removing Project Jobs:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Project Jobs",
		})
		return
	}

	log.Println("Jobs remove info:", changes.Removed)

	changes, err = db.JobTemplates().RemoveAll(bson.M{"project_id": project.ID})
	if err != nil {
		log.Println("Error while removing Project Job Templates:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Project Job Templates",
		})
		return
	}

	log.Println("Job Template remove info:", changes.Removed)

	// remove object from the collection
	err = db.Projects().RemoveId(project.ID);
	if err != nil {
		log.Println("Error while removing Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Project",
		})
		return
	}

	// add new activity to activity stream
	addActivity(project.ID, user.ID, "Project " + project.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

func Playbooks(c *gin.Context) {
	// get Project from the gin.Context
	project := c.MustGet(_CTX_PROJECT).(models.Project)
	searchDir := util.Config.HomePath + project.ID.Hex()

	files := []string{}
	err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			r, err := regexp.MatchString(".yml", f.Name())
			if err == nil && r {
				files = append(files, f.Name())
			}
		}
		return nil
	})

	if err != nil {
		log.Println("Error while removing Project:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Project",
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
				log.Println("Error while getting Teams:", err)
				c.JSON(http.StatusInternalServerError, models.Error{
					Code:http.StatusInternalServerError,
					Message: "Error while getting Teams",
				})
				return
			}

			err = metadata.TeamMetadata(&tmpTeam)
			if err != nil {
				log.Println("Error while setting Metatdata:", err)
				c.JSON(http.StatusInternalServerError, models.Error{
					Code:http.StatusInternalServerError,
					Message: "Error while getting Teams",
				})
				return
			}

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
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: tms[pgi.Skip():pgi.End()],
	})
}

// TODO: not complete
func ActivityStream(c *gin.Context) {
	project := c.MustGet(_CTX_PROJECT).(models.Project)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": project.ID, "type": _CTX_PROJECT}).All(&activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while Activities",
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
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: activities[pgi.Skip():pgi.End()],
	})
}

// GetJobs renders the Job as JSON
func ProjectUpdates(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	parser := util.NewQueryParser(c)
	match := parser.Match([]string{"status", "type", "failed", })
	if con := parser.IContains([]string{"id", "name", "labels"}); con != nil {
		match = con
	}

	// get only project update jobs
	match["job_type"] = "project_update"

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
		if err := metadata.JobMetadata(&tmpJob); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Credentials",
			})
			return
		}
		// good to go add to list
		jobs = append(jobs, tmpJob)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Credential",
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
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: jobs[pgi.Skip():pgi.End()],
	})
}

func SCMUpdate(c *gin.Context) {
	//project := c.MustGet(_CTX_PROJECT).(models.Project)

}