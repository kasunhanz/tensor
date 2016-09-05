package users

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"gopkg.in/mgo.v2/bson"
	"golang.org/x/crypto/bcrypt"
	"fmt"
	"time"
	database "bitbucket.pearson.com/apseng/tensor/db"
)

func GetUser(c *gin.Context) {
	var usr models.User
	if u, exists := c.Get("_user"); exists {
		usr = u.(models.User)
	} else {
		usr = c.MustGet("user").(models.User)
	}

	SetMetadata(&usr)

	c.JSON(200, usr)
}

func GetUsers(c *gin.Context) {
	var users []models.User

	col := database.MongoDb.C("users")

	if err := col.Find(nil).All(&users); err != nil {
		panic(err)
	}

	resp := models.Response{}
	resp.Count = len(users)
	resp.Results = users

	if users != nil {
		for k, v := range users {
			SetMetadata(&v)
			users[k] = v
		}

		resp.Results = users
	}

	c.JSON(200, resp)
}

func AddUser(c *gin.Context) {
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

func GetUserMiddleware(c *gin.Context) {
	userID := c.Params.ByName("user_id")

	var u models.User

	col := database.MongoDb.C("users")

	if err := col.FindId(bson.ObjectIdHex(userID)).One(&u); err != nil {
		panic(err)
	}

	c.Set("_user", u)
	c.Next()
}

func UpdateUser(c *gin.Context) {
	oldUser := c.MustGet("_user").(models.User)

	var user models.User
	if err := c.Bind(&user); err != nil {
		return
	}

	col := database.MongoDb.C("users")

	if err := col.UpdateId(oldUser.ID,
		bson.M{"first_name": user.FirstName, "last_name":user.LastName, "username": user.Username, "email": user.Email}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func UpdateUserPassword(c *gin.Context) {
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

func DeleteUser(c *gin.Context) {
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