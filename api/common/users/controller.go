package users

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/api/helpers"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/util"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
)

const (
	CTXUserA  = "_user"
	CTXUserID = "user_id"
	CTXUser   = "user"
)

func Middleware(c *gin.Context) {

	userID, err := util.GetIdParam(CTXUserID, c)

	if err != nil {
		log.Errorln("Error while getting the User:", err) // log error to the system log
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var user common.User
	err = db.Users().FindId(bson.ObjectIdHex(userID)).One(&user)
	if err != nil {
		log.Errorln("Error while getting the User:", err) // log error to the system log
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	c.Set(CTXUserA, user)
	c.Next()
}

func GetUser(c *gin.Context) {
	var user common.User

	if u, exists := c.Get(CTXUserA); exists {
		user = u.(common.User)
	} else {
		user = c.MustGet("user").(common.User)
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

	var users []common.User
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpUser common.User
	// iterate over all and only get valid objects
	for iter.Next(&tmpUser) {
		metadata.UserMetadata(&tmpUser)
		// good to go add to list
		tmpUser.Password = "$encrypted$"
		users = append(users, tmpUser)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  users[pgi.Skip():pgi.End()],
	})
}

func AddUser(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	var req common.User
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Invlid JSON request")
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if helpers.IsNotUniqueEmail(req.Email) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Email alredy in use"},
		})
		return
	}

	if helpers.IsNotUniqueUsername(req.Username) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Email alredy in use"},
		})
		return
	}

	req.ID = bson.NewObjectId()
	pwdHash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 11)
	req.Password = string(pwdHash)
	req.Created = time.Now()
	req.Modified = time.Now()

	if err := db.Users().Insert(req); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while creating User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while creating User"},
		})
		return
	}
	// add new activity to activity stream
	activity.AddUserActivity(common.Create, user, req)
	req.Password = "$encrypted$"

	metadata.UserMetadata(&req)

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

func UpdateUser(c *gin.Context) {
	actor := c.MustGet(CTXUser).(common.User)
	user := c.MustGet("_user").(common.User)
	tmpUser := user

	var req common.User
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if user.Email != req.Email && helpers.IsNotUniqueEmail(req.Email) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Email alredy in use"},
		})
		return
	}

	if user.Username != req.Username && helpers.IsNotUniqueUsername(req.Username) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Username alredy in use"},
		})
		return
	}

	user.Username = req.Username
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Email = req.Email

	if req.Password != "$encrypted$" {
		pwdHash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 11)
		user.Password = string(pwdHash)
	}

	user.Modified = time.Now()

	if err := db.Users().UpdateId(user.ID, user); err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while updating User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating User"},
		})
		return
	}
	// add new activity to activity stream
	activity.AddUserActivity(common.Update, actor, tmpUser, user)

	//hide password
	user.Password = "$encrypted$"
	metadata.UserMetadata(&user)

	c.JSON(http.StatusOK, user)
}

func PatchUser(c *gin.Context) {
	actor := c.MustGet(CTXUser).(common.User)
	user := c.MustGet("_user").(common.User)
	tmpUser := user

	var req common.PatchUser
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if req.Email != nil {
		if user.Email != *req.Email && helpers.IsNotUniqueEmail(*req.Email) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Email alredy in use"},
			})
			return
		}
	}

	if req.Username != nil {
		if user.Username != *req.Username && helpers.IsNotUniqueUsername(*req.Username) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Username alredy in use"},
			})
			return
		}
	}

	if req.Username != nil {
		user.Username = strings.Trim(*req.Username, " ")
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}

	if req.LastName != nil {
		user.LastName = *req.LastName
	}

	if req.Email != nil {
		user.Email = *req.Email
	}

	if req.Password != nil && *req.Password != "$encrypted$" {
		pwdHash, _ := bcrypt.GenerateFromPassword([]byte(*req.Password), 11)
		user.Password = string(pwdHash)
	}

	user.Modified = time.Now()

	if err := db.Users().UpdateId(user.ID, user); err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while updating User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating User"},
		})
		return
	}
	// add new activity to activity stream
	activity.AddUserActivity(common.Update, actor, tmpUser, user)

	//hide password
	user.Password = "$encrypted$"
	metadata.UserMetadata(&user)

	c.JSON(http.StatusOK, user)
}

// TODO: Complete this with authentication
func DeleteUser(c *gin.Context) {
	actor := c.MustGet(CTXUser).(common.User)
	user := c.MustGet("_user").(common.User)

	// Remove user from projects
	_, err := db.Projects().UpdateAll(nil, bson.M{"$pull": bson.M{"roles": bson.M{"user_id": user.ID}}})
	if err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while deleting User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while deleting User"},
		})
	}

	// remove user from teams
	_, err = db.Teams().UpdateAll(nil, bson.M{"$pull": bson.M{"roles": bson.M{"user_id": user.ID}}})
	if err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while deleting User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while deleting User"},
		})
	}

	// remove user from organizations
	_, err = db.Organizations().UpdateAll(nil, bson.M{"$pull": bson.M{"roles": bson.M{"user_id": user.ID}}})
	if err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while deleting User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while deleting User"},
		})
	}

	// remove user from credentials
	_, err = db.Credentials().UpdateAll(nil, bson.M{"$pull": bson.M{"roles": bson.M{"user_id": user.ID}}})
	if err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while deleting User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while deleting User"},
		})
	}

	// remove user from job_templates
	_, err = db.JobTemplates().UpdateAll(nil, bson.M{"$pull": bson.M{"roles": bson.M{"user_id": user.ID}}})
	if err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while deleting User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while deleting User"},
		})
	}

	// remove user from terraform job templates
	_, err = db.TerrafromJobTemplates().UpdateAll(nil, bson.M{"$pull": bson.M{"roles": bson.M{"user_id": user.ID}}})
	if err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while deleting User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while deleting User"},
		})
	}

	if err := db.Users().RemoveId(user.ID); err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while deleting User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while deleting User"},
		})
		return
	}

	activity.AddUserActivity(common.Delete, actor, user)
	c.AbortWithStatus(http.StatusNoContent)
}

func Projects(c *gin.Context) {
	user := c.MustGet(CTXUserA).(common.User)

	var projts []common.Project
	// new mongodb iterator
	iter := db.Projects().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpProjct common.Project
	// iterate over all and only get valid objects
	for iter.Next(&tmpProjct) {
		metadata.ProjectMetadata(&tmpProjct)
		// good to go add to list
		projts = append(projts, tmpProjct)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  projts[pgi.Skip():pgi.End()],
	})
}

func Credentials(c *gin.Context) {
	user := c.MustGet(CTXUserA).(common.User)

	var creds []common.Credential
	// new mongodb iterator
	iter := db.Credentials().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpCredential common.Credential
	// iterate over all and only get valid objects
	for iter.Next(&tmpCredential) {
		// hide passwords, keys even they are already encrypted
		hideEncrypted(&tmpCredential)
		metadata.CredentialMetadata(&tmpCredential)
		// add to list
		creds = append(creds, tmpCredential)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  creds[pgi.Skip():pgi.End()],
	})
}

func Teams(c *gin.Context) {
	user := c.MustGet(CTXUserA).(common.User)

	var tms []common.Team
	// new mongodb iterator
	iter := db.Teams().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpTeam common.Team
	// iterate over all and only get valid objects
	for iter.Next(&tmpTeam) {
		metadata.TeamMetadata(&tmpTeam)
		// add to list
		tms = append(tms, tmpTeam)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving project data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  tms[pgi.Skip():pgi.End()],
	})
}

func Organizations(c *gin.Context) {
	user := c.MustGet(CTXUserA).(common.User)

	var orgs []common.Organization
	// new mongodb iterator
	iter := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	// loop through each result and modify for our needs
	var tmpOrganization common.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		metadata.OrganizationMetadata(&tmpOrganization)
		// add to list
		orgs = append(orgs, tmpOrganization)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving organization data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  orgs[pgi.Skip():pgi.End()],
	})
}

func AdminsOfOrganizations(c *gin.Context) {
	user := c.MustGet(CTXUserA).(common.User)

	var orgs []common.Organization
	// new mongodb iterator
	iter := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user", "roles.role": "admin"}).Iter()
	// loop through each result and modify for our needs
	var tmpOrganization common.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		metadata.OrganizationMetadata(&tmpOrganization)
		// add to list
		orgs = append(orgs, tmpOrganization)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving organization data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  orgs[pgi.Skip():pgi.End()],
	})
}

// ActivityStream returns the activites of the user on other Users
func ActivityStream(c *gin.Context) {
	user := c.MustGet(CTXUserA).(common.User)

	var activities []common.ActivityUser
	var activity common.ActivityUser
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": user.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&activity) {
		metadata.ActivityUserMetadata(&activity)
		metadata.UserMetadata(&activity.Object1)
		//apply metadata only when Object2 is available
		if activity.Object2 != nil {
			metadata.UserMetadata(activity.Object2)
		}
		//add to activities list
		activities = append(activities, activity)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  activities[pgi.Skip():pgi.End()],
	})
}

// hideEncrypted is replaces encrypted fields by $encrypted$ string
func hideEncrypted(c *common.Credential) {
	encrypted := "$encrypted$"
	c.Password = encrypted
	c.SSHKeyData = encrypted
	c.SSHKeyUnlock = encrypted
	c.BecomePassword = encrypted
	c.VaultPassword = encrypted
	c.AuthorizePassword = encrypted
	c.Secret = encrypted
}
