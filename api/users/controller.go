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
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
)

const _CTX_USER = "_user"
const _CTX_USER_ID = "user_id"

func Middleware(c *gin.Context) {

	userID := c.Params.ByName(_CTX_USER_ID)
	var user models.User
	collection := db.C(db.USERS)

	err := collection.FindId(bson.ObjectIdHex(userID)).One(&user);
	if err != nil {
		log.Print("Error while getting the User:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
	}

	c.Set(_CTX_USER, user)
	c.Next()
}

func GetUser(c *gin.Context) {
	var user models.User

	if u, exists := c.Get(_CTX_USER); exists {
		user = u.(models.User)
	} else {
		user = c.MustGet("user").(models.User)
	}

	metadata.UserMetadata(&user)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:1,
		Results:user,
	})
}

func GetUsers(c *gin.Context) {

	dbc := db.C(db.USERS)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	con := parser.IContains([]string{"username", "first_name", "last_name"});
	if con != nil {
		match = con
	}

	query := dbc.Find(match)
	count, err := query.Count();

	if err != nil {
		log.Println("Error while trying to get count of Users from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Users",
		})
		return
	}

	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var users []models.User

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit()).All(&users); err != nil {
		log.Println("Error while retriving User data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Users",
		})
		return
	}
	//TODO: ?????????????????
	for i, v := range users {
		metadata.UserMetadata(&v);
		users[i] = v
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:users,
	})
}

func AddUser(c *gin.Context) {
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.User
	err := c.BindJSON(&req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	user.ID = bson.NewObjectId()
	user.Created = time.Now()

	collection := db.C(db.USERS)

	err = collection.Insert(user);
	if err != nil {
		log.Println("Error while creating User:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating User",
		})
		return
	}
	// add new activity to activity stream
	addActivity(user.ID, user.ID, "User " + user.FirstName + " " + user.LastName + " created")

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, models.Response{
		Count:1,
		Results:user,
	})
}

func UpdateUser(c *gin.Context) {
	oldUser := c.MustGet("_user").(models.User)

	var user models.User
	if err := c.BindJSON(&user); err != nil {
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

	if err := c.BindJSON(&pwd); err != nil {
		return
	}

	password, _ := bcrypt.GenerateFromPassword([]byte(pwd.Pwd), 11)

	col := db.C(db.USERS)

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

	userCol := db.C(db.USERS)

	if err := userCol.RemoveId(user.ID); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func Projects(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.C(db.PROJECTS)

	var projts []models.Project
	// new mongodb iterator
	iter := collection.Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpProjct models.Project
	// iterate over all and only get valid objects
	for iter.Next(&tmpProjct) {
		if err := metadata.ProjectMetadata(&tmpProjct); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Projects",
			})
			return
		}
		// good to go add to list
		projts = append(projts, tmpProjct)
	}

	if err := iter.Close(); err != nil {
		log.Println("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Projects",
		})
		return
	}

	count := len(projts)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: projts[pgi.Skip():pgi.End()],
	})
}

func Credentials(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.C(db.CREDENTIALS)

	var creds []models.Credential
	// new mongodb iterator
	iter := collection.Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpCredential models.Credential
	// iterate over all and only get valid objects
	for iter.Next(&tmpCredential) {
		if err := metadata.CredentialMetadata(&tmpCredential); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Credentials",
			})
			return
		}
		// add to list
		creds = append(creds, tmpCredential)
	}

	if err := iter.Close(); err != nil {
		log.Println("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Credentials",
		})
		return
	}

	count := len(creds)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: creds[pgi.Skip():pgi.End()],
	})
}

func Teams(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.C(db.TEAMS)

	var tms []models.Team
	// new mongodb iterator
	iter := collection.Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpTeam models.Team
	// iterate over all and only get valid objects
	for iter.Next(&tmpTeam) {
		if err := metadata.TeamMetadata(&tmpTeam); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Credentials",
			})
			return
		}
		// add to list
		tms = append(tms, tmpTeam)
	}

	if err := iter.Close(); err != nil {
		log.Println("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Credentials",
		})
		return
	}

	count := len(tms)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: tms[pgi.Skip():pgi.End()],
	})
}

func Organizations(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.C(db.ORGANIZATIONS)

	var orgs []models.Organization
	// new mongodb iterator
	iter := collection.Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpOrganization models.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		if err := metadata.OrganizationMetadata(&tmpOrganization); err != nil {
			log.Println("Error while getting organization metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Organizations",
			})
			return
		}
		// add to list
		orgs = append(orgs, tmpOrganization)
	}

	if err := iter.Close(); err != nil {
		log.Println("Error while retriving organization data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Organizations",
		})
		return
	}

	count := len(orgs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: orgs[pgi.Skip():pgi.End()],
	})
}

func AdminsOfOrganizations(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.C(db.ORGANIZATIONS)

	var orgs []models.Organization
	// new mongodb iterator
	iter := collection.Find(bson.M{"roles.user_id": user.ID, "roles.type": "user", "roles.role": "admin"}).Iter()
	// loop through each result and modify for our needs
	var tmpOrganization models.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		if err := metadata.OrganizationMetadata(&tmpOrganization); err != nil {
			log.Println("Error while getting organization metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Organizations",
			})
			return
		}
		// add to list
		orgs = append(orgs, tmpOrganization)
	}

	if err := iter.Close(); err != nil {
		log.Println("Error while retriving organization data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Organizations",
		})
		return
	}

	count := len(orgs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: orgs[pgi.Skip():pgi.End()],
	})
}

// TODO: not complete
func ActivityStream(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var activities []models.Activity
	collection := db.C(db.ACTIVITY_STREAM)
	err := collection.Find(bson.M{"actor_id": user.ID}).All(&activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Activities",
		})
		return
	}

	count := len(activities)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: activities[pgi.Skip():pgi.End()],
	})
}