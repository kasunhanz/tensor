package access

import (
	"net/http"
	"net/mail"
	"strings"
	"time"

	database "pearson.com/tensor/db"
	"pearson.com/tensor/models"
	"pearson.com/tensor/util"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

// Login creates a session for a requested user
func Login(c *gin.Context) {
	// Model for store credentials
	var login struct {
		Auth     string `json:"auth" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.Bind(&login); err != nil {
		// Give user an informative error
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "status": "error"})
		c.Abort() // abort the request if JSON payload is invalid
		return
	}

	// Lowercase email or username
	login.Auth = strings.ToLower(login.Auth)

	var q bson.M

	if _, err := mail.ParseAddress(login.Auth); err == nil {
		q = bson.M{"email": login.Auth}

	} else {
		q = bson.M{"username": login.Auth}
	}

	var user models.User

	col := database.MongoDb.C("users")

	if err := col.Find(q).One(&user); err != nil {
		// Give the user an informative error
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unable to find user", "status": "error"})
		c.Abort()
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(login.Password)); err != nil {
		// Give the user an informative error
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalied password", "status": "error"})
		c.Abort()
		return
	}

	session := models.Session{
		ID:         bson.NewObjectId(),
		UserID:     user.ID,
		Created:    time.Now(),
		LastActive: time.Now(),
		IP:         c.ClientIP(),
		UserAgent:  c.Request.Header.Get("user-agent"),
		Expired:    false,
	}

	if err := session.Insert(); err != nil {
		// Give the user an informative error
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to create session", "status": "error"})
		c.Abort()
		return
	}

	encoded, err := util.Cookie.Encode("tensor", map[string]interface{}{
		"user":    user.ID.Hex(),
		"session": session.ID.Hex(),
	})

	if err != nil {
		// Give the user an informative error
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to create session", "status": "error"})
		c.Abort()
	}

	// set a new cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:  "tensor",
		Value: encoded,
		Path:  "/",
	})

	c.AbortWithStatus(204)
}

// Logout will remove the browser cookie
func Logout(c *gin.Context) {
	c.SetCookie("tensor", "", -1, "/", "", false, true)
	c.AbortWithStatus(204)
}
