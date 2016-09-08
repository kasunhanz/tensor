package projects

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"net/http"
	"gopkg.in/mgo.v2/bson"
	"time"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/util/pagination"
	"strconv"
)

const _CTX_PROJECT = "project"
const _CTX_USER = "user"
const _CTX_PROJECT_ID = "project_id"

// ProjectMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func ProjectMiddleware(c *gin.Context) {
	projectID := c.Params.ByName(_CTX_PROJECT_ID)

	dbcp := database.MongoDb.C(models.DBC_PROJECTS)

	var project models.Project
	if err := dbcp.FindId(bson.ObjectIdHex(projectID)).One(&project); err != nil {
		log.Print(err) // log error to the system log
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set(_CTX_PROJECT, project)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetProject(c *gin.Context) {
	p := c.MustGet(_CTX_PROJECT).(models.Project)
	setMetadata(&p)

	c.JSON(200, p)
}

// GetProjects returns a JSON array of projects
func GetProjects(c *gin.Context) {
	dbc := database.MongoDb.C(models.DBC_PROJECTS)

	parser := util.NewQueryParser(c)

	match := parser.Match([]string{"type", "status"})

	if con := parser.IContains([]string{"name"}); con != nil {
		if match != nil {
			for i, v := range con {
				match[i] = v
			}
		} else {
			match = con
		}
	}

	query := dbc.Find(match)

	count, err := query.Count();
	if err != nil {
		log.Println("Unable to count projects from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	pgi := pagination.NewPagination(c, count)

	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page) + ": That page contains no results."})
		return
	}

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var projs []models.Project

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&projs); err != nil {
		log.Println("Unable to retrive projects from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, v := range projs {
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		projs[i] = v
	}

	c.JSON(200, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": projs, })
}

// AddProject creates a new project
func AddProject(c *gin.Context) {
	var req models.Project

	u := c.MustGet(_CTX_USER).(models.User)

	if err := c.Bind(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	var p models.Project

	p.Name = req.Name
	p.Description = req.Description
	p.LocalPath = req.LocalPath
	p.ScmType = req.ScmType
	p.ScmUrl = req.ScmUrl
	p.ScmBranch = req.ScmBranch
	p.ScmClean = req.ScmClean
	p.ScmDeleteOnUpdate = req.ScmDeleteOnUpdate
	p.ScmCredential = req.ScmCredential
	p.Organization = req.Organization
	p.ScmUpdateOnLaunch = req.ScmUpdateOnLaunch
	p.ScmUpdateCacheTimeout = req.ScmUpdateCacheTimeout
	p.CreatedBy = u.ID
	p.ModifiedBy = u.ID

	p.ID = bson.NewObjectId()
	p.Created = time.Now()
	p.Modified = time.Now()

	dbc := database.MongoDb.C(models.DBC_PROJECTS)

	if err := dbc.Insert(p); err != nil {
		log.Println("Failed to create Project", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Project"})
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_PROJECT,
		ObjectID:    p.ID,
		Description: "Project " + p.Name + " created",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	if err := setMetadata(&p); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	c.JSON(201, p)
}
