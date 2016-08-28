package organizations

import (
	database "pearson.com/tensor/db"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	mdlorg "pearson.com/tensor/models/organization"
	"pearson.com/tensor/models/user"
)

// OrganizationMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func OrganizationMiddleware(c *gin.Context) {
	projectID := c.Params.ByName("organization_id")

	col := database.MongoDb.C("organizations")

	var org mdlorg.Organization
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
	org := c.MustGet("organization").(mdlorg.Organization)

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
	org := c.MustGet("organization").(mdlorg.Organization)

	col := database.MongoDb.C("organizations")

	aggregate := []bson.M{
		{"$match": bson.M{
			"_id": org.ID,
		}},
		{"$project": bson.M{ //performance enhancement I guess
			"_id":0,
			"users":1,}},
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
	var users []user.User

	if err := col.Pipe(aggregate).All(&users); err != nil {
		panic(err)
	}

	olen := len(users)

	resp := make(map[string]interface{})
	resp["count"] = olen
	resp["results"] = users

	for i := 0; i < olen; i++ {
		(&users[i]).IncludeMetadata()
	}

	c.JSON(200, users)

}