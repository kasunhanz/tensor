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
	user := c.MustGet("user").(models.User)

	col := database.MongoDb.C("projects")

	var projects []models.Project
	if err := col.Find(bson.M{"users.user_id": user.ID}).Sort("name").All(&projects); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(200, projects)
}

// AddProject creates a new project
func AddProject(c *gin.Context) {
	var project models.Project
	user := c.MustGet("user").(models.User)

	if err := c.Bind(&project); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	project.ID = bson.NewObjectId()
	project.Created = time.Now()
	project.Users = []models.ProjectUser{{Admin: true, UserID: user.ID}}

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
