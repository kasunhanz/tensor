package access

import (
	"strings"
	"time"

	"errors"
	database "github.com/gamunu/tensor/db"
	"github.com/gamunu/tensor/models"
	"github.com/gamunu/tensor/util"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/http"
)

func Authentication(c *gin.Context) {
	var userID bson.ObjectId

	// Check whether the authorization header is set
	// if authorization header is available authentication the request
	// if authorization header is not available authenticate using session
	if authHeader := strings.ToLower(c.Request.Header.Get("authorization")); len(authHeader) > 0 {
		var token models.APIToken
		col := database.MongoDb.C("user_tokens")

		if err := col.Find(bson.M{"_id": bson.ObjectIdHex(strings.Replace(authHeader, "bearer ", "", 1)), "expired": false}).One(&token); err != nil {
			c.Error(errors.New("Cannot find user token for " + strings.Replace(authHeader, "bearer ", "", 1)))
			// send a informative response to user
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Request Unauthorized"})
			c.Abort() //must! without it request will continue, user will bypass authentication
			return
		}
		// user id required for fetch user information
		userID = token.UserID
	} else {
		// fetch session cookie
		cookie, err := c.Request.Cookie("tensor")

		if err != nil {
			c.Error(err)
			// send a informative response to user
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Request Unauthorized"})
			c.Abort() //must! without it request will continue, user will bypass authentication
			return
		}

		value := make(map[string]interface{})
		if err = util.Cookie.Decode("tensor", cookie.Value, &value); err != nil {

			c.Error(err)
			// send a informative response to user
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Request Unauthorized"})
			c.Abort() //must! without it request will continue, user will bypass authentication
			return
		}

		user, ok := value["user"]
		sessionVal, okSession := value["session"]
		if !ok || !okSession {
			c.Error(err)
			// send a informative response to user
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Request Unauthorized"})
			c.Abort() //must! without it request will continue, user will bypass authentication
			return
		}
		// user id required for fetch user information
		userID = bson.ObjectIdHex(user.(string))
		// session id for session update
		sessionID := bson.ObjectIdHex(sessionVal.(string))

		// fetch session
		var session models.Session
		col := database.MongoDb.C("sessions")
		if err := col.Find(bson.M{"_id": sessionID, "user_id": userID, "expired": false}).One(&session); err != nil {
			c.Error(err)
			// send a informative response to user
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid session"})
			c.Abort() //must! without it request will continue, user will bypass authentication
			return
		}

		if time.Now().Sub(session.LastActive).Hours() > 7*24 {
			// more than week old unused session
			// destroy.

			if err := col.UpdateId(sessionID, bson.M{"expired": true}); err != nil {
				c.Error(err)
				// send a informative response to user
				c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Expired session"})
				c.Abort() //must! without it request will continue, user will bypass authentication
				return
			}

			c.Error(err)
			// send a informative response to user
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Expired session"})
			c.Abort() //must! without it request will continue, user will bypass authentication
			return
		}

		if err := col.UpdateId(sessionID, bson.M{"$set": bson.M{"last_active": time.Now()}}); err != nil {
			c.Error(err)
			// send a informative response to user
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Expired session"})
			c.Abort() //must! without it request will continue, user will bypass authentication
			return
		}
	}

	// User is authenticated either session or authorization header
	var user models.User
	userCol := database.MongoDb.C("users")

	if err := userCol.FindId(userID).One(&user); err != nil {
		c.Error(err)
		// send a informative response to user
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid credentials"})
		c.Abort() //must! without it request will continue, user will bypass authentication
		return
	}

	c.Set("user", user)
}
