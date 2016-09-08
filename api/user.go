package api

import (
	"time"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
)

func getAPITokens(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	var tokens []models.APIToken

	col := database.MongoDb.C("user_tokens")

	if err := col.Find(bson.M{"user_id": user.ID}).All(&tokens); err != nil {
		panic(err)
	}

	c.JSON(200, tokens)
}

func createAPIToken(c *gin.Context) {
	user := c.MustGet("user").(models.User)

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

func expireAPIToken(c *gin.Context) {
	user := c.MustGet("user").(models.User)

	tokenID := c.Param("token_id")

	col := database.MongoDb.C("user_tokens")

	if err := col.Update(bson.M{"_id": tokenID, "user_id": user.ID}, bson.M{"expired": true}); err != nil {
		c.AbortWithStatus(400)
		panic(err)
	}

	c.AbortWithStatus(204)
}
