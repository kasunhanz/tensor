package organizations

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/api/users"
)

// OrganizationMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func OrganizationMiddleware(c *gin.Context) {
	projectID := c.Params.ByName("organization_id")

	col := database.MongoDb.C("organizations")

	var org models.Organization
	if err := col.Find(bson.M{"_id": bson.ObjectIdHex(projectID), }).One(&org); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	(&org).IncludeMetadata()

	c.Set("organization", org)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetOrganization(c *gin.Context) {
	c.JSON(200, c.MustGet("organization"))
}

func AddOrganizationUser(c *gin.Context) {
	// get organization
	org := c.MustGet("organization").(models.Organization)

	//get the request payload
	var playload struct {
		UserId bson.ObjectId `json:"user_id"`
	}

	if err := c.Bind(&playload); err != nil {
		return
	}

	ou := bson.M{"user_id":playload.UserId}

	col := database.MongoDb.C("organizations")

	if err := col.UpdateId(org.ID, bson.M{"$addToSet": bson.M{"users": ou}}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func GetOrganizationUsers(c *gin.Context) {
	// get organization
	org := c.MustGet("organization").(models.Organization)

	col := database.MongoDb.C("organizations")

	aggregate := []bson.M{
		{"$match": bson.M{
			"_id": org.ID,
		}},
		{"$project": bson.M{//performance enhancement I guess
			"_id":0,
			"users":1, }},
		{"$unwind": "$users"},
		{"$lookup": bson.M{
			"from":         "users",
			"localField":   "users.user_id",
			"foreignField": "_id",
			"as":           "user",
		}},
		{
			"$match": bson.M{"user": bson.M{"$ne": []interface{}{} },
			}},
		{"$project": bson.M{
			"_id":0,
			"users":bson.M{"$arrayElemAt": []interface{}{"$user", 0 }},
		}},
		{"$project": bson.M{
			"_id":"$users._id",
			"created":"$users.created",
			"email":"$users.email",
			"name":"$users.name",
			"password":"$users.password",
			"username":"$users.username",
		}},
	}
	var usrs []models.User

	if err := col.Pipe(aggregate).All(&usrs); err != nil {
		panic(err)
	}

	olen := len(usrs)

	resp := make(map[string]interface{})
	resp["count"] = olen
	resp["results"] = usrs

	for i := 0; i < olen; i++ {
		users.SetMetadata(&usrs[i])
	}

	c.JSON(200, usrs)

}