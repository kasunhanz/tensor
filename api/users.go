package api

import (
	"database/sql"
	"time"

	"fmt"
	database "github.com/gamunu/hilbert-space/db"
	"github.com/gamunu/hilbert-space/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

func getUsers(c *gin.Context) {
	var users []models.User

	col := database.MongoDb.C("user_token")

	if err := col.Find(nil).All(&users); err != nil {
		panic(err)
	}

	c.JSON(200, users)
}

func addUser(c *gin.Context) {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return
	}

	user.ID = bson.NewObjectId()
	user.Created = time.Now()

	if err := user.Insert(); err != nil {
		panic(err)
	}

	c.JSON(201, user)
}

func getUserMiddleware(c *gin.Context) {
	userID := c.Params.ByName("user_id")

	var user models.User

	col := database.MongoDb.C("user")

	if err := col.FindId(userID).One(&user); err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatus(404)
			return
		}

		panic(err)
	}

	c.Set("_user", user)
	c.Next()
}

func updateUser(c *gin.Context) {
	oldUser := c.MustGet("_user").(models.User)

	var user models.User
	if err := c.Bind(&user); err != nil {
		return
	}

	col := database.MongoDb.C("user")

	if err := col.UpdateId(oldUser.ID, bson.M{"name": user.Name, "username": user.Username, "email": user.Email}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func updateUserPassword(c *gin.Context) {
	user := c.MustGet("_user").(models.User)
	var pwd struct {
		Pwd string `json:"password"`
	}

	if err := c.Bind(&pwd); err != nil {
		return
	}

	password, _ := bcrypt.GenerateFromPassword([]byte(pwd.Pwd), 11)

	col := database.MongoDb.C("user")

	if err := col.UpdateId(user.ID, bson.M{"password": string(password)}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func deleteUser(c *gin.Context) {
	user := c.MustGet("_user").(models.User)

	col := database.MongoDb.C("project")

	info, err := col.UpdateAll(nil, bson.M{"$pull": bson.M{"users": bson.M{"user_id": user.ID}}})
	if err != nil {
		panic(err)
	}

	fmt.Println(info.Matched)

	userCol := database.MongoDb.C("user")

	if err := userCol.RemoveId(user.ID); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
