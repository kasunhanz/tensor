package api

import (
	"database/sql"
	"time"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"golang.org/x/crypto/bcrypt"
	"fmt"
)

func getUsers(c *gin.Context) {
	var users []models.User

	col := database.MongoDb.C("user_tokens")

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

	col := database.MongoDb.C("users")

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

	col := database.MongoDb.C("users")

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

	col := database.MongoDb.C("users")

	if err := col.UpdateId(user.ID, bson.M{"password": string(password)}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func deleteUser(c *gin.Context) {
	user := c.MustGet("_user").(models.User)

	col := database.MongoDb.C("projects")

	info, err := col.UpdateAll(nil, bson.M{"$pull": bson.M{"users": bson.M{"user_id": user.ID}}})
	if err != nil {
		panic(err)
	}

	fmt.Println(info.Matched)

	userCol := database.MongoDb.C("users")

	if err := userCol.RemoveId(user.ID); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
