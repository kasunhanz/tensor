package users

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models"
	"github.com/pearsonappeng/tensor/util"
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/mgo.v2/bson"
)

const _CTX_USER = "_user"
const _CTX_USER_ID = "user_id"

func Middleware(c *gin.Context) {

	userID, err := util.GetIdParam(_CTX_USER_ID, c)

	if err != nil {
		log.Errorln("Error while getting the User:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var user models.User
	err = db.Users().FindId(bson.ObjectIdHex(userID)).One(&user)
	if err != nil {
		log.Errorln("Error while getting the User:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
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
	c.JSON(http.StatusOK, user)
}

func GetUsers(c *gin.Context) {

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"username", "first_name", "last_name"}, match)

	query := db.Users().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var users []models.User
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpUser models.User
	// iterate over all and only get valid objects
	for iter.Next(&tmpUser) {
		metadata.UserMetadata(&tmpUser)
		// good to go add to list
		users = append(users, tmpUser)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Credential"},
		})
		return
	}

	count := len(users)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  users[pgi.Skip():pgi.End()],
	})
}

func AddUser(c *gin.Context) {
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.User
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	user.ID = bson.NewObjectId()
	user.Created = time.Now()

	if err := db.Users().Insert(user); err != nil {
		log.Errorln("Error while creating User:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while creating User"},
		})
		return
	}
	// add new activity to activity stream
	addActivity(user.ID, user.ID, "User "+user.FirstName+" "+user.LastName+" created")

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, user)
}

func UpdateUser(c *gin.Context) {
	oldUser := c.MustGet("_user").(models.User)

	var user models.User
	if err := c.BindJSON(&user); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if err := db.Users().UpdateId(oldUser.ID,
		bson.M{"first_name": user.FirstName, "last_name": user.LastName, "username": user.Username, "email": user.Email}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func DeleteUser(c *gin.Context) {
	user := c.MustGet("_user").(models.User)

	info, err := db.Projects().UpdateAll(nil, bson.M{"$pull": bson.M{"users": bson.M{"user_id": user.ID}}})
	if err != nil {
		panic(err)
	}

	log.Info(info.Matched)

	if err := db.Users().RemoveId(user.ID); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func Projects(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var projts []models.Project
	// new mongodb iterator
	iter := db.Projects().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpProjct models.Project
	// iterate over all and only get valid objects
	for iter.Next(&tmpProjct) {
		metadata.ProjectMetadata(&tmpProjct)
		// good to go add to list
		projts = append(projts, tmpProjct)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Projects"},
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
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  projts[pgi.Skip():pgi.End()],
	})
}

func Credentials(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var creds []models.Credential
	// new mongodb iterator
	iter := db.Credentials().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpCredential models.Credential
	// iterate over all and only get valid objects
	for iter.Next(&tmpCredential) {
		metadata.CredentialMetadata(&tmpCredential)
		// add to list
		creds = append(creds, tmpCredential)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Credentials"},
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
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  creds[pgi.Skip():pgi.End()],
	})
}

func Teams(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var tms []models.Team
	// new mongodb iterator
	iter := db.Teams().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpTeam models.Team
	// iterate over all and only get valid objects
	for iter.Next(&tmpTeam) {
		metadata.TeamMetadata(&tmpTeam)
		// add to list
		tms = append(tms, tmpTeam)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Credentials"},
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
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  tms[pgi.Skip():pgi.End()],
	})
}

func Organizations(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var orgs []models.Organization
	// new mongodb iterator
	iter := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpOrganization models.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		metadata.OrganizationMetadata(&tmpOrganization)
		// add to list
		orgs = append(orgs, tmpOrganization)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving organization data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Organizations"},
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
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  orgs[pgi.Skip():pgi.End()],
	})
}

func AdminsOfOrganizations(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var orgs []models.Organization
	// new mongodb iterator
	iter := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user", "roles.role": "admin"}).Iter()
	// loop through each result and modify for our needs
	var tmpOrganization models.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		metadata.OrganizationMetadata(&tmpOrganization)
		// add to list
		orgs = append(orgs, tmpOrganization)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving organization data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Organizations"},
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
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  orgs[pgi.Skip():pgi.End()],
	})
}

// TODO: not complete
func ActivityStream(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"actor_id": user.ID}).All(&activities)

	if err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Activities"},
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
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  activities[pgi.Skip():pgi.End()],
	})
}
