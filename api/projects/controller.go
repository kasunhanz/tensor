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
)

const _CTX_PROJECT = "project"
const _CTX_USER = "user"
const _CTX_PROJECT_ID = "project_id"

// ProjectMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func ProjectMiddleware(c *gin.Context) {
	projectID := c.Params.ByName(_CTX_PROJECT_ID)

	dbcp := db.C(models.DBC_PROJECTS)

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
	dbc := db.C(models.DBC_PROJECTS)

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

	pgi := util.NewPagination(c, count)

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

	user := c.MustGet(_CTX_USER).(models.User)

	if err := c.Bind(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	proj := models.Project{
		ID: bson.NewObjectId(),
		Name:req.Name,
		Description:req.Description,
		LocalPath:req.LocalPath,
		ScmType:req.ScmType,
		ScmUrl:req.ScmUrl,
		ScmBranch:req.ScmBranch,
		ScmClean:req.ScmClean,
		ScmDeleteOnUpdate:req.ScmDeleteOnUpdate,
		ScmCredential:req.ScmCredential,
		Organization:req.Organization,
		ScmUpdateOnLaunch:req.ScmUpdateOnLaunch,
		ScmUpdateCacheTimeout:req.ScmUpdateCacheTimeout,
		CreatedBy:user.ID,
		ModifiedBy:user.ID,
		Created: time.Now(),
		Modified: time.Now(),
	}

	dbc := db.C(models.DBC_PROJECTS)

	if err := dbc.Insert(proj); err != nil {
		log.Println("Failed to create Project", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Project"})
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_PROJECT,
		ObjectID:    proj.ID,
		Description: "Project " + proj.Name + " created",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	if err := setMetadata(&proj); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	c.JSON(201, proj)
}


// UpdateProject will update the Project
func UpdateProject(c *gin.Context) {
	// get Project from the gin.Context
	oproj := c.MustGet(_CTX_PROJECT).(models.Project)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Project

	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	proj := models.Project{
		ID: bson.NewObjectId(),
		Name:oproj.Name,
		Description:oproj.Description,
		LocalPath:oproj.LocalPath,
		ScmType:oproj.ScmType,
		ScmUrl:oproj.ScmUrl,
		ScmBranch:oproj.ScmBranch,
		ScmClean:oproj.ScmClean,
		ScmDeleteOnUpdate:oproj.ScmDeleteOnUpdate,
		ScmCredential:oproj.ScmCredential,
		Organization:oproj.Organization,
		ScmUpdateOnLaunch:oproj.ScmUpdateOnLaunch,
		ScmUpdateCacheTimeout:oproj.ScmUpdateCacheTimeout,
		CreatedBy:user.ID,
		ModifiedBy:user.ID,
		Created: time.Now(),
		Modified: time.Now(),
	}

	collection := db.MongoDb.C(models.DBC_PROJECTS)

	// update object
	if err := collection.UpdateId(proj.ID, proj); err != nil {
		log.Println("Failed to update Project", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to update Project"})
		return
	}

	if err := (models.Event{
		ProjectID:   proj.ID,
		Description: "Project ID " + proj.ID.Hex() + " updated",
		ObjectID:    proj.ID,
		ObjectType:  "project",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	// set `related` and `summary` feilds
	if err := setMetadata(&proj); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	// render JSON with 200 status code
	c.JSON(http.StatusOK, proj)
}

// RemoveProject will remove the Project
// from the models.DBC_PROJECTS collection
func RemoveProject(c *gin.Context) {
	// get Project from the gin.Context
	proj := c.MustGet(_CTX_PROJECT).(models.Project)

	collection := db.MongoDb.C(models.DBC_PROJECTS)

	// remove object from the collection
	if err := collection.RemoveId(proj.ID); err != nil {
		log.Println("Failed to remove Project", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to remove Project"})
		return
	}

	if err := (models.Event{
		Description: "Project " + proj.Name + " deleted",
		ObjectID:    proj.ID,
		ObjectType:  _CTX_PROJECT,
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}