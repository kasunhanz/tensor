package projects

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"time"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
)

// GetProjects returns a JSON array of projects
func GetProjects(c *gin.Context) {
	//user := c.MustGet("user").(models.User)

	col := database.MongoDb.C("projects")

	var projects []models.Project
	if err := col.Find(nil).Sort("name").All(&projects); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	olen := len(projects)

	resp := make(map[string]interface{})
	resp["count"] = olen
	resp["results"] = projects

	for i := 0; i < olen; i++ {
		(&projects[i]).IncludeMetadata()
	}

	c.JSON(200, resp)
}

// AddProject creates a new project
func AddProject(c *gin.Context) {
	var request struct {
		Name                  string            `bson:"name" json:"name" binding:"required"`
		Description           string            `bson:"description" json:"description"`
		LocalPath             string            `bson:"local_path" json:"local_path"`
		ScmType               string            `bson:"scm_type" json:"scm_type" binding:"required"`
		ScmUrl                string            `bson:"scm_url" json:"scm_url" binding:"required"`
		ScmBranch             string            `bson:"scm_branch" json:"scm_branch"`
		ScmClean              bool              `bson:"scm_clean" json:"scm_clean"`
		ScmDeleteOnUpdate     bool              `bson:"scm_delete_on_update" json:"scm_delete_on_update"`
		ScmCredential         bson.ObjectId     `bson:"credentail" json:"credential"`
		Organization          bson.ObjectId     `bson:"organization" json:"organization" binding:"required"`
		ScmUpdateOnLaunch     bool              `bson:"scm_update_on_launch" json:"scm_update_on_launch"`
		ScmUpdateCacheTimeout int               `bson:"scm_update_cache_timeout" json:"scm_update_cache_timeout"`
	}
	//user := c.MustGet("user").(models.User)

	if err := c.Bind(&request); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	var project models.Project


	project.Name = request.Name
	project.Description = request.Description
	project.LocalPath = request.LocalPath
	project.ScmType = request.ScmType
	project.ScmUrl = request.ScmUrl
	project.ScmBranch = request.ScmBranch
	project.ScmClean = request.ScmClean
	project.ScmDeleteOnUpdate = request.ScmDeleteOnUpdate
	project.ScmCredential = request.ScmCredential
	project.Organization = request.Organization
	project.ScmUpdateOnLaunch = request.ScmUpdateOnLaunch
	project.ScmUpdateCacheTimeout = request.ScmUpdateCacheTimeout

	project.ID = bson.NewObjectId()
	project.Created = time.Now()
	project.Modified = time.Now()

	if err := project.Insert(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if err := (models.Event{
		ID:          bson.NewObjectId(),
		ProjectID:   project.ID,
		ObjectType:  "project",
		Description: "Project Created",
		Created:     project.Created,
	}.Insert()); err != nil {
		// We don't inform client about this error
		// do not ever panic :D
		c.Error(err)
		return
	}

	c.JSON(201, project)
}
