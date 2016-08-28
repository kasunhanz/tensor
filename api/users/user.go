package users

import (
	"time"

	database "pearson.com/tensor/db"
	"pearson.com/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	pkguser "pearson.com/tensor/models/user"
)

func GetUser(c *gin.Context) {
	var user pkguser.User
	if u, exists := c.Get("_user"); exists {
		user = u.(pkguser.User)
	} else {
		user = c.MustGet("user").(pkguser.User)
	}

	(&user).IncludeMetadata()
	c.JSON(200, user)
}

func GetAPITokens(c *gin.Context) {
	user := c.MustGet("user").(pkguser.User)

	var tokens []models.APIToken

	col := database.MongoDb.C("user_tokens")

	if err := col.Find(bson.M{"user_id": user.ID}).All(&tokens); err != nil {
		panic(err)
	}

	c.JSON(200, tokens)
}

func CreateAPIToken(c *gin.Context) {
	user := c.MustGet("user").(pkguser.User)

	token := models.APIToken{
		ID:      bson.NewObjectId(),
		Created: time.Now(),
		UserID:  user.ID,
		Expired: false,
	}

	if err := token.Insert(); err != nil {
		panic(err)
	}

	c.JSON(201, token)
}

func ExpireAPIToken(c *gin.Context) {
	user := c.MustGet("user").(pkguser.User)

	tokenID := c.Param("token_id")

	col := database.MongoDb.C("user_tokens")

	if err := col.Update(bson.M{"_id": tokenID, "user_id": user.ID}, bson.M{"expired": true}); err != nil {
		c.AbortWithStatus(400)
		panic(err)
	}

	c.AbortWithStatus(204)
}
