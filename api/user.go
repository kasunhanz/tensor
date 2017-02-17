package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
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

type UserController struct{}

func (ctrl UserController) Middleware(c *gin.Context) {

	userID, err := util.GetIdParam(CTXUserID, c)
	loginUser := c.MustGet(CTXUser).(common.User)

	if err != nil {
		log.Errorln("Error while getting the User:", err) // log error to the system log
		c.JSON(http.StatusNotFound, common.Error{
			Code:   http.StatusNotFound,
			Errors: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var user common.User
	err = db.Users().FindId(bson.ObjectIdHex(userID)).One(&user)
	if err != nil {
		log.Errorln("Error while getting the User:", err) // log error to the system log
		c.JSON(http.StatusNotFound, common.Error{
			Code:   http.StatusNotFound,
			Errors: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	roles := new(rbac.User)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(loginUser, user) {
				c.JSON(http.StatusUnauthorized, common.Error{
					Code:   http.StatusUnauthorized,
					Errors: []string{"You do not have permission to perform this action."},
				})
				c.Abort()
				return
			}
		}
	case "PUT", "DELETE", "PATCH":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(loginUser, user) {
				c.JSON(http.StatusUnauthorized, common.Error{
					Code:   http.StatusUnauthorized,
					Errors: []string{"You do not have permission to perform this action."},
				})
				c.Abort()
				return
			}
		}
	case "POST":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.WriteSpecial(loginUser, user) {
				c.JSON(http.StatusUnauthorized, common.Error{
					Code:   http.StatusUnauthorized,
					Errors: []string{"You do not have permission to perform this action."},
				})
				c.Abort()
				return
			}
		}
	}

	c.Set(CTXUserA, user)
	c.Next()
}

func (ctrl UserController) One(c *gin.Context) {
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

func (ctrl UserController) All(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"username", "first_name", "last_name"}, match)

	query := db.Users().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.User)
	var users []common.User
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpUser common.User
	// iterate over all and only get valid objects
	for iter.Next(&tmpUser) {
		if !roles.Read(user, tmpUser) {
			continue
		}

		metadata.UserMetadata(&tmpUser)
		// good to go add to list
		tmpUser.Password = "$encrypted$"
		users = append(users, tmpUser)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Credential"},
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

func (ctrl UserController) Create(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	var req common.User
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Invlid JSON request")
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: validate.GetValidationErrors(err),
		})
		return
	}

	if !req.IsUniqueEmail() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: []string{"Email alredy in use"},
		})
		return
	}

	if !req.IsUniqueUsername() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: []string{"Email alredy in use"},
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while creating User"},
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

func (ctrl UserController) Update(c *gin.Context) {
	actor := c.MustGet(CTXUser).(common.User)
	user := c.MustGet("_user").(common.User)
	tmpUser := user

	var req common.User
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: validate.GetValidationErrors(err),
		})
		return
	}

	if user.Email != req.Email && !req.IsUniqueEmail() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: []string{"Email alredy in use"},
		})
		return
	}

	if user.Username != req.Username && !req.IsUniqueUsername() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: []string{"Username alredy in use"},
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while updating User"},
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

func (ctrl UserController) Patch(c *gin.Context) {
	actor := c.MustGet(CTXUser).(common.User)
	user := c.MustGet("_user").(common.User)
	tmpUser := user

	var req common.PatchUser
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: validate.GetValidationErrors(err),
		})
		return
	}

	if req.Email != nil {
		user.Email = *req.Email

		if user.Email != *req.Email && !user.IsUniqueEmail() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Email alredy in use"},
			})
			return
		}
	}

	if req.Username != nil {
		user.Username = strings.Trim(*req.Username, " ")

		if user.Username != *req.Username && !user.IsUniqueUsername() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:   http.StatusBadRequest,
				Errors: []string{"Username alredy in use"},
			})
			return
		}
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}

	if req.LastName != nil {
		user.LastName = *req.LastName
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while updating User"},
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
func (ctrl UserController) Delete(c *gin.Context) {
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while deleting User"},
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while deleting User"},
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while deleting User"},
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while deleting User"},
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while deleting User"},
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while deleting User"},
		})
	}

	if err := db.Users().RemoveId(user.ID); err != nil {
		log.WithFields(log.Fields{
			"User ID": user.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while deleting User")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while deleting User"},
		})
		return
	}

	activity.AddUserActivity(common.Delete, actor, user)
	c.AbortWithStatus(http.StatusNoContent)
}

func (ctrl UserController) Projects(c *gin.Context) {
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Projects"},
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

func (ctrl UserController) Credentials(c *gin.Context) {
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Credentials"},
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

func (ctrl UserController) Teams(c *gin.Context) {
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Credentials"},
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

func (ctrl UserController) Organizations(c *gin.Context) {
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Organizations"},
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

func (ctrl UserController) AdminsOfOrganizations(c *gin.Context) {
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
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Organizations"},
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

// ActivityStream returns the activities of the user on other Users
func (ctrl UserController) ActivityStream(c *gin.Context) {
	user := c.MustGet(CTXUserA).(common.User)

	var activities []common.ActivityUser
	var act common.ActivityUser
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": user.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&act) {
		metadata.ActivityUserMetadata(&act)
		metadata.UserMetadata(&act.Object1)
		//apply metadata only when Object2 is available
		if act.Object2 != nil {
			metadata.UserMetadata(act.Object2)
		}
		//add to activities list
		activities = append(activities, act)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Activities"},
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

func (ctrl UserController) AssignRole(c *gin.Context) {
	user := c.MustGet(CTXUserA).(common.User)

	var req common.RoleObj
	err := binding.JSON.Bind(c.Request, &req)
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:   http.StatusBadRequest,
			Errors: validate.GetValidationErrors(err),
		})
		return
	}

	switch req.ResourceType {
	case "credential":
		{
			roles := new(rbac.Credential)

			if count, _ := db.Credentials().FindId(req.ResourceID).Count(); count <= 0 {
				c.JSON(http.StatusBadRequest, common.Error{
					Code:   http.StatusBadRequest,
					Errors: []string{"Coud not find resource"},
				})
				return
			}

			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			}
		}

	case "organization":
		{
			roles := new(rbac.Organization)

			if count, _ := db.Organizations().FindId(req.ResourceID).Count(); count <= 0 {
				c.JSON(http.StatusBadRequest, common.Error{
					Code:   http.StatusBadRequest,
					Errors: []string{"Coud not find resource"},
				})
				return
			}

			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			}

		}

	case "team":
		{
			roles := new(rbac.Team)

			if count, _ := db.Teams().FindId(req.ResourceID).Count(); count <= 0 {
				c.JSON(http.StatusBadRequest, common.Error{
					Code:   http.StatusBadRequest,
					Errors: []string{"Coud not find resource"},
				})
				return
			}

			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			}
		}

	case "project":
		{
			roles := new(rbac.Project)

			if count, _ := db.Projects().FindId(req.ResourceID).Count(); count <= 0 {
				c.JSON(http.StatusBadRequest, common.Error{
					Code:   http.StatusBadRequest,
					Errors: []string{"Coud not find resource"},
				})
				return
			}

			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			}
		}

	case "job_template":
		{
			roles := new(rbac.JobTemplate)

			if count, _ := db.JobTemplates().FindId(req.ResourceID).Count(); count <= 0 {
				c.JSON(http.StatusBadRequest, common.Error{
					Code:   http.StatusBadRequest,
					Errors: []string{"Coud not find resource"},
				})
				return
			}

			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			}
		}

	case "terraform_job_template":
		{
			roles := new(rbac.TerraformJobTemplate)

			if count, _ := db.TerrafromJobTemplates().FindId(req.ResourceID).Count(); count <= 0 {
				c.JSON(http.StatusBadRequest, common.Error{
					Code:   http.StatusBadRequest,
					Errors: []string{"Coud not find resource"},
				})
				return
			}

			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			}
		}

	case "inventory":
		{
			roles := new(rbac.Inventory)

			if count, _ := db.Inventories().FindId(req.ResourceID).Count(); count <= 0 {
				c.JSON(http.StatusBadRequest, common.Error{
					Code:   http.StatusBadRequest,
					Errors: []string{"Coud not find resource"},
				})
				return
			}

			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, user.ID, rbac.RoleTypeUser, req.Role)
			}
		}
	}

	if err != nil {
		log.WithFields(log.Fields{
			"Resource ID": user.ID.Hex(),
			"User ID":     user.ID.Hex(),
			"Error":       err.Error(),
		}).Errorln("Error occured while modifying the role")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error occured while modifying the role"},
		})
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}

func (ctrl UserController) GetRoles(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, common.Error{
		Code:   http.StatusNotImplemented,
		Errors: []string{"Not implemented"},
	})
}
