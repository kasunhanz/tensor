package projects

import (
	database "pearson.com/tensor/db"
	"pearson.com/tensor/models"
	"pearson.com/tensor/util"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func UserMiddleware(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	userID, err := util.GetIntParam("user_id", c)
	if err != nil {
		return
	}

	var user models.User

	userIds := make([]bson.ObjectId, 0, len(project.Users))

	for _, pUser := range project.Users {
		userIds = append(userIds, pUser.UserID)
	}

	col := database.MongoDb.C("users")

	if err := col.Find(bson.M{"_id": bson.M{"$in": userIds}}).Select(bson.M{"_id": userID}).One(&user); err != nil {
		panic(err)
	}

	c.Set("projectUser", user)
	c.Next()
}

func GetUsers(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	var users []models.User
	col := database.MongoDb.C("users")

	userIds := make([]bson.ObjectId, 0, len(project.Users))

	for _, pUser := range project.Users {
		userIds = append(userIds, pUser.UserID)
	}

	if err := col.Find(bson.M{"_id": bson.M{"$in": userIds}}).All(&users); err != nil {
		panic(err)
	}

	c.JSON(200, users)
}

func AddUser(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	var projectUser models.ProjectUser

	if err := c.Bind(&projectUser); err != nil {
		return
	}

	col := database.MongoDb.C("projects")

	if err := col.Update(bson.M{"_id": project.ID}, bson.M{"$push": bson.M{"users": bson.M{"user_id": projectUser.UserID, "admin": projectUser.Admin}}}); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   projectUser.UserID,
		ObjectType:  "user",
		ObjectID:    projectUser.UserID,
		Description: "User ID " + projectUser.UserID.String() + " added to team",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveUser(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	user := c.MustGet("projectUser").(models.User)

	col := database.MongoDb.C("projects")

	if err := col.Update(bson.M{"_id": project.ID}, bson.M{"$pull": bson.M{"users.$.user_id": user.ID}}); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   project.ID,
		ObjectType:  "user",
		ObjectID:    user.ID,
		Description: "User ID " + user.ID.String() + " removed from team",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func MakeUserAdmin(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	user := c.MustGet("projectUser").(models.User)
	admin := true

	if c.Request.Method == "DELETE" {
		// strip admin
		admin = false
	}

	col := database.MongoDb.C("projects")

	if err := col.Update(bson.M{"_id": project.ID, "users.user_id": user.ID}, bson.M{"users.$.admin": admin}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
