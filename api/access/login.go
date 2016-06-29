package access

import (
	"database/sql"
	"net/http"
	"net/mail"
	"strings"
	"time"

	database "github.com/gamunu/hilbertspace/db"
	"github.com/gamunu/hilbertspace/models"
	"github.com/gamunu/hilbertspace/util"
	"github.com/gin-gonic/gin"
	sq "github.com/masterminds/squirrel"
	"golang.org/x/crypto/bcrypt"
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

	q := sq.Select("*").
	From("user")

	_, err := mail.ParseAddress(login.Auth)
	if err == nil {
		q = q.Where("email=?", login.Auth)
	} else {
		q = q.Where("username=?", login.Auth)
	}

	query, args, _ := q.ToSql()

	var user models.User
	if err := database.Mysql.SelectOne(&user, query, args...); err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatus(400)
			return
		}

		panic(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(login.Password)); err != nil {
		c.AbortWithStatus(400)
		return
	}

	session := models.Session{
		UserID:     user.ID,
		Created:    time.Now(),
		LastActive: time.Now(),
		IP:         c.ClientIP(),
		UserAgent:  c.Request.Header.Get("user-agent"),
		Expired:    false,
	}
	if err := database.Mysql.Insert(&session); err != nil {
		panic(err)
	}

	encoded, err := util.Cookie.Encode("hilbertspace", map[string]interface{}{
		"user":    user.ID,
		"session": session.ID,
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
