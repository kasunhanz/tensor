package users

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"gopkg.in/mgo.v2/bson"
	"golang.org/x/crypto/bcrypt"
	"fmt"
	"time"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/util"
	"strconv"
)

const _CTX_USER = "_user"
const _CTX_USER_ID = "user_id"

func GetUserMiddleware(c *gin.Context) {
	userID := c.Params.ByName(_CTX_USER_ID)

	var usr models.User

	dbc := db.C(models.DBC_USERS)

	if err := dbc.FindId(bson.ObjectIdHex(userID)).One(&usr); err != nil {
		log.Print(err) // log error to the system log
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set(_CTX_USER, usr)
	c.Next()
}

func GetUser(c *gin.Context) {
	var usr models.User

	if u, exists := c.Get(_CTX_USER); exists {
		usr = u.(models.User)
	} else {
		usr = c.MustGet("user").(models.User)
	}

	SetMetadata(&usr)

	c.JSON(200, usr)
}

func GetUsers(c *gin.Context) {
	dbc := db.C(models.DBC_USERS)

	parser := util.NewQueryParser(c)

	match := bson.M{}

	if con := parser.IContains([]string{"username", "first_name", "last_name"}); con != nil {
		match = con
	}

	query := dbc.Find(match)

	count, err := query.Count();
	if err != nil {
		log.Println("Unable to count users from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	pgi := util.NewPagination(c, count)

	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page) + ": That page contains no results."})
		return
	}

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var usrs []models.User

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&usrs); err != nil {
		log.Println("Unable to retrive users from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, v := range usrs {
		SetMetadata(&v);
		usrs[i] = v
	}

	c.JSON(200, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": usrs, })
}

func AddUser(c *gin.Context) {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return
	}

	user.ID = bson.NewObjectId()
	user.Created = time.Now()

	col := db.C(models.DBC_USERS)

	if err := col.Insert(user); err != nil {
		panic(err)
	}

	c.JSON(201, user)
}

func UpdateUser(c *gin.Context) {
	oldUser := c.MustGet("_user").(models.User)

	var user models.User
	if err := c.Bind(&user); err != nil {
		return
	}

	col := db.C("users")

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

	col := db.C(models.DBC_USERS)

	if err := col.UpdateId(user.ID, bson.M{"password": string(password)}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func DeleteUser(c *gin.Context) {
	user := c.MustGet("_user").(models.User)

	col := db.C("projects")

	info, err := col.UpdateAll(nil, bson.M{"$pull": bson.M{"users": bson.M{"user_id": user.ID}}})
	if err != nil {
		panic(err)
	}

	fmt.Println(info.Matched)

	userCol := db.C(models.DBC_USERS)

	if err := userCol.RemoveId(user.ID); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}