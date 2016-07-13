package access

import (
	"net/http"
	"net/mail"
	"strings"
	"time"

	database "pearson.com/hilbert-space/db"
	"pearson.com/hilbert-space/models"
	"pearson.com/hilbert-space/util"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2/bson"
)

func Login(c *gin.Context) {
	var login struct {
		Auth     string `json:"auth" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.Bind(&login); err != nil {
		return
	}

	login.Auth = strings.ToLower(login.Auth)

	var q bson.M

	_, err := mail.ParseAddress(login.Auth)
	if err == nil {
		q = bson.M{"email": login.Auth}

	} else {
		q = bson.M{"username": login.Auth}
	}

	var user models.User

	col := database.MongoDb.C("user")

	if err := col.Find(q).One(&user); err != nil {
		panic(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(login.Password)); err != nil {
		c.AbortWithStatus(400)
		return
	}

	session := models.Session{
		ID: bson.NewObjectId(),
		UserID:     user.ID,
		Created:    time.Now(),
		LastActive: time.Now(),
		IP:         c.ClientIP(),
		UserAgent:  c.Request.Header.Get("user-agent"),
		Expired:    false,
	}

	if err := session.Insert(); err != nil {
		panic(err)
	}

	encoded, err := util.Cookie.Encode("hilbertspace", map[string]interface{}{
		"user":    user.ID.Hex(),
		"session": session.ID.Hex(),
	})

	if err != nil {
		panic(err)
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:  "hilbertspace",
		Value: encoded,
		Path:  "/",
	})

	c.AbortWithStatus(204)
}

func Logout(c *gin.Context) {
	c.SetCookie("hilbertspace", "", -1, "/", "", false, true)
	c.AbortWithStatus(204)
}
