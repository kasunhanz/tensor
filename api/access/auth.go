package access

import (
	"fmt"
	"strings"
	"time"

	database "github.com/gamunu/hilbertspace/db"
	"github.com/gamunu/hilbertspace/models"
	"github.com/gamunu/hilbertspace/util"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func Authentication(c *gin.Context) {
	var userID bson.ObjectId

	if authHeader := strings.ToLower(c.Request.Header.Get("authorization")); len(authHeader) > 0 {
		var token models.APIToken
		col := database.MongoDb.C("user_token")

		if err := col.Find(bson.M{"_id": strings.Replace(authHeader, "bearer ", "", 1), "expired": false}).One(&token); err != nil {
			c.Error(err)
			c.AbortWithStatus(403)
			return
		}

		userID = token.UserID
	} else {
		// fetch session from cookie
		cookie, err := c.Request.Cookie("hilbertspace")
		if err != nil {
			c.AbortWithStatus(403)
			return
		}

		value := make(map[string]interface{})
		if err = util.Cookie.Decode("hilbertspace", cookie.Value, &value); err != nil {
			c.AbortWithStatus(403)
			return
		}

		user, ok := value["user"]
		sessionVal, okSession := value["session"]
		if !ok || !okSession {
			c.AbortWithStatus(403)
			return
		}

		userID = user.(int)
		sessionID := sessionVal.(int)

		// fetch session
		var session models.Session
		col := database.MongoDb.C("session")

		if err := col.Find(bson.M{"_id":sessionID, "user_id": userID, "expired": false}).One(&session); err != nil {
			c.AbortWithStatus(403)
			return
		}

		if time.Now().Sub(session.LastActive).Hours() > 7 * 24 {
			// more than week old unused session
			// destroy.

			if err := col.UpdateId(sessionID, bson.M{"expired": true}); err != nil {
				c.Error(err)
				c.AbortWithStatus(403)
				return
			}

			c.AbortWithStatus(403)
			return
		}

		if err := col.UpdateId(sessionID, bson.M{"last_active": time.Now()}); err != nil {
			c.Error(err)
			c.AbortWithStatus(403)
			return
		}
	}

	var user models.User
	userCol := database.MongoDb.C("user")

	if err := userCol.FindId(userID).One(&user); err != nil {
		fmt.Println("Can't find user", err)
		c.AbortWithStatus(403)
		return
	}

	c.Set("user", user)
}
