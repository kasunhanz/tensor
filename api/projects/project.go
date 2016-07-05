package projects

import (
	database "pearson.com/hilbert-space/db"
	"pearson.com/hilbert-space/models"
	"pearson.com/hilbert-space/util"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func ProjectMiddleware(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	projectID, err := util.GetIntParam("project_id", c)
	if err != nil {
		return
	}

	col := database.MongoDb.C("project")

	query := bson.M{
		"_id": projectID,
		"users": bson.M{
			"$in": []bson.ObjectId{user.ID},
		},
	}

	var project models.Project
	if err := col.Find(query).One(&project); err != nil {
		panic(err)
	}

	c.Set("project", project)
	c.Next()
}

func GetProject(c *gin.Context) {
	c.JSON(200, c.MustGet("project"))
}
