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
	cUserA = "_user"
	cUserID = "user_id"
	cUser = "user"
)

type UserController struct{}

func (ctrl UserController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cUserID)
	loginUser := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "User does not exist"})
		return
	}

	var user common.User
	if err := db.Users().FindId(bson.ObjectIdHex(objectID)).One(&user); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "User does not exist",
			Log: log.Fields{
				"User ID": objectID,
				"Error": err.Error()},
		})
		return
	}

	roles := new(rbac.User)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(loginUser, user) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "DELETE", "PATCH":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(loginUser, user) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "POST":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.WriteSpecial(loginUser, user) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}

	c.Set(cUserA, user)
	c.Next()
}

func (ctrl UserController) One(c *gin.Context) {
	var user common.User
	if u, exists := c.Get(cUserA); exists {
		user = u.(common.User)
	} else {
		user = c.MustGet("user").(common.User)
	}
	metadata.UserMetadata(&user)
	c.JSON(http.StatusOK, user)
}

func (ctrl UserController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"username", "first_name", "last_name"}, match)

	query := db.Users().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.User)
	var users []common.User
	iter := query.Iter()
	var tmpUser common.User
	for iter.Next(&tmpUser) {
		if !roles.Read(user, tmpUser) {
			continue
		}
		metadata.UserMetadata(&tmpUser)
		tmpUser.Password = "$encrypted$"
		users = append(users, tmpUser)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting users",
			Log:     log.Fields{"Error": err.Error()},
		})
		return
	}

	count := len(users)
	pgi := util.NewPagination(c, count)
	if pgi.HasPage() {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound,
			Message: "#" + strconv.Itoa(pgi.Page()) + " page contains no results.",
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  users[pgi.Skip():pgi.End()],
	})
}

func (ctrl UserController) Create(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)

	// SuperUsers only can create users
	if !user.IsSuperUser {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}

	var req common.User
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	if !req.IsUniqueEmail() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Email is alredy in use.",
		})
		return
	}

	if !req.IsUniqueUsername() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Username is alredy in use.",
		})
		return
	}

	req.ID = bson.NewObjectId()
	pwdHash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 11)
	req.Password = string(pwdHash)
	req.Created = time.Now()
	req.Modified = time.Now()

	if err := db.Users().Insert(req); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while creating user",
			Log:     log.Fields{"Error": err.Error()},
		})
		return
	}

	activity.AddUserActivity(common.Create, user, req)
	req.Password = "$encrypted$"
	metadata.UserMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

func (ctrl UserController) Update(c *gin.Context) {
	actor := c.MustGet(cUser).(common.User)
	user := c.MustGet("_user").(common.User)
	tmpUser := user

	var req common.User
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	if user.Email != req.Email && !req.IsUniqueEmail() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Email is alredy in use.",
		})
		return
	}

	if user.Username != req.Username && !req.IsUniqueUsername() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Username is alredy in use.",
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
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating user.",
			Log:     log.Fields{"Host ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddUserActivity(common.Update, actor, tmpUser, user)
	user.Password = "$encrypted$"
	metadata.UserMetadata(&user)
	c.JSON(http.StatusOK, user)
}

func (ctrl UserController) Patch(c *gin.Context) {
	actor := c.MustGet(cUser).(common.User)
	user := c.MustGet("_user").(common.User)
	tmpUser := user

	var req common.PatchUser
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	if req.Email != nil {
		user.Email = *req.Email
		if user.Email != *req.Email && !user.IsUniqueEmail() {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Email is alredy in use.",
			})
			return
		}
	}

	if req.Username != nil {
		user.Username = strings.Trim(*req.Username, " ")

		if user.Username != *req.Username && !user.IsUniqueUsername() {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Username is alredy in use.",
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
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating user.",
			Log:     log.Fields{"Host ID": user.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddUserActivity(common.Update, actor, tmpUser, user)
	user.Password = "$encrypted$"
	metadata.UserMetadata(&user)
	c.JSON(http.StatusOK, user)
}

func (ctrl UserController) Delete(c *gin.Context) {
	loginUser := c.MustGet(cUser).(common.User)
	user := c.MustGet("_user").(common.User)

	// Remove permissions
	access := bson.M{"$pull": bson.M{"roles": common.AccessControl{GranteeID: user.ID}}}
	if _, err := db.Projects().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing user",
			Log:     log.Fields{"User ID": user.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	if _, err := db.Credentials().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing user",
			Log:     log.Fields{"User ID": user.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	if _, err := db.Inventories().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing user",
			Log:     log.Fields{"User ID": user.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	if _, err := db.JobTemplates().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing user",
			Log:     log.Fields{"User ID": user.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	if _, err := db.TerrafromJobTemplates().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing user",
			Log:     log.Fields{"User ID": user.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if err := db.Users().RemoveId(user.ID); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing user",
			Log:     log.Fields{"User ID": user.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddUserActivity(common.Delete, loginUser, user)
	c.AbortWithStatus(http.StatusNoContent)
}

func (ctrl UserController) Projects(c *gin.Context) {
	user := c.MustGet(cUserA).(common.User)

	var projts []common.Project
	iter := db.Projects().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	var tmpProjct common.Project
	for iter.Next(&tmpProjct) {
		metadata.ProjectMetadata(&tmpProjct)
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
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  projts[pgi.Skip():pgi.End()],
	})
}

func (ctrl UserController) Credentials(c *gin.Context) {
	user := c.MustGet(cUserA).(common.User)

	var creds []common.Credential
	iter := db.Credentials().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	var tmpCredential common.Credential
	for iter.Next(&tmpCredential) {
		hideEncrypted(&tmpCredential)
		metadata.CredentialMetadata(&tmpCredential)
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

	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  creds[pgi.Skip():pgi.End()],
	})
}

func (ctrl UserController) Teams(c *gin.Context) {
	user := c.MustGet(cUserA).(common.User)

	var tms []common.Team
	iter := db.Teams().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	var tmpTeam common.Team
	for iter.Next(&tmpTeam) {
		metadata.TeamMetadata(&tmpTeam)
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
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  tms[pgi.Skip():pgi.End()],
	})
}

func (ctrl UserController) Organizations(c *gin.Context) {
	user := c.MustGet(cUserA).(common.User)

	var orgs []common.Organization
	iter := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user"}).Iter()
	var tmpOrganization common.Organization
	for iter.Next(&tmpOrganization) {
		metadata.OrganizationMetadata(&tmpOrganization)
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
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  orgs[pgi.Skip():pgi.End()],
	})
}

func (ctrl UserController) AdminsOfOrganizations(c *gin.Context) {
	user := c.MustGet(cUserA).(common.User)

	var orgs []common.Organization
	iter := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "roles.type": "user", "roles.role": "admin"}).Iter()
	var tmpOrganization common.Organization
	for iter.Next(&tmpOrganization) {
		metadata.OrganizationMetadata(&tmpOrganization)
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
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  orgs[pgi.Skip():pgi.End()],
	})
}

// ActivityStream returns the activities of the user on other Users
func (ctrl UserController) ActivityStream(c *gin.Context) {
	user := c.MustGet(cUserA).(common.User)

	var activities []common.ActivityUser
	var act common.ActivityUser
	iter := db.ActivityStream().Find(bson.M{"object1._id": user.ID}).Iter()
	for iter.Next(&act) {
		metadata.ActivityUserMetadata(&act)
		metadata.UserMetadata(&act.Object1)
		if act.Object2 != nil {
			metadata.UserMetadata(act.Object2)
		}
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

	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  activities[pgi.Skip():pgi.End()],
	})
}

func (ctrl UserController) AssignRole(c *gin.Context) {
	user := c.MustGet(cUserA).(common.User)

	var req common.RoleObj
	err := binding.JSON.Bind(c.Request, &req)
	if err != nil {
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
