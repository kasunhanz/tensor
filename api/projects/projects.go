package projects

import (
	database "github.com/gamunu/hilbertspace/db"
	"github.com/gamunu/hilbertspace/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func GetProjects(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	col := database.MongoDb.C("project")

	query := bson.M{
		"users": bson.M{
			"$in": []bson.ObjectId{user.ID},
		},
	}

	var projects []models.Project
	if err := col.Find(query).Sort("name").All(&projects); err != nil {
		panic(err)
	}

	c.JSON(200, projects)
}

func AddProject(c *gin.Context) {
	var project models.Project
	user := c.MustGet("user").(*models.User)

	if err := c.Bind(&project); err != nil {
		return
	}

	project.ID = bson.NewObjectId()
	project.Created = time.Now()
	project.Users = []bson.ObjectId{user.ID}

	if err := project.Insert(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   project.ID,
		Description: "Project Created",
	}.Insert()); err != nil {
		panic(err)
	}

	c.JSON(201, project)
}
