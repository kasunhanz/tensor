package projects

import (
	"github.com/gamunu/hilbertspace/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func EnvironmentMiddleware(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	envID := c.Params.ByName("environment_id")

	var env models.Environment
	if err := project.GetEnvironment(envID, &env); err != nil {
		panic(err)
	}

	c.Set("environment", env)
	c.Next()
}

func GetEnvironment(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	var env []models.Environment

	if err := project.GetEnvironments(&env); err != nil {
		panic(err)
	}

	c.JSON(200, env)
}

func UpdateEnvironment(c *gin.Context) {
	oldEnv := c.MustGet("environment").(models.Environment)
	var env models.Environment
	if err := c.Bind(&env); err != nil {
		return
	}

	oldEnv.Name = env.Name
	oldEnv.JSON = env.JSON

	if err := oldEnv.Update(); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func AddEnvironment(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	var env models.Environment

	if err := c.Bind(&env); err != nil {
		return
	}

	env.ProjectID = project.ID
	env.ID = bson.NewObjectId()

	if err := env.Insert(); err != nil {
		panic(err)
	}

	objType := "environment"

	desc := "Environment " + env.Name + " created"
	if err := (models.Event{
		ProjectID:   project.ID,
		ObjectType:  objType,
		ObjectID:    env.ID,
		Description: desc,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveEnvironment(c *gin.Context) {
	env := c.MustGet("environment").(models.Environment)

	if err := env.Remove(); err != nil {
		panic(err)
	}

	desc := "Environment " + env.Name + " deleted"
	if err := (models.Event{
		ProjectID:   env.ProjectID,
		Description: desc,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
