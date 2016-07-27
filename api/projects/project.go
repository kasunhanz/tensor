package projects

import (
	database "github.com/gamunu/hilbert-space/db"
	"github.com/gamunu/hilbert-space/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/http"
)

// ProjectMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func ProjectMiddleware(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	projectID := c.Params.ByName("project_id")

	col := database.MongoDb.C("project")

	query := bson.M{
		"_id": bson.ObjectIdHex(projectID),
		"users.user_id": user.ID,
	}

	var project models.Project
	if err := col.Find(query).One(&project); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Set("project", project)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetProject(c *gin.Context) {
	c.JSON(200, c.MustGet("project"))
}
