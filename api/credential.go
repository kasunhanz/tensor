package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	cCredential = "credential"
	cCredentialID = "credential_id"
)

type CredentialController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXCredentialID from Gin Context and retrieves credential data from the collection
// and store credential data under key CTXCredential in Gin Context
func (ctrl CredentialController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cCredentialID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Credential does not exist"})
		return
	}

	var credential common.Credential
	if err := db.Credentials().FindId(bson.ObjectIdHex(objectID)).One(&credential); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Credential does not exist",
			Log: logrus.Fields{
				"Credential": objectID,
				"Error":  err.Error(),
			},
		})
		return
	}

	roles := new(rbac.Credential)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(user, credential) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "DELETE":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(user, credential) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}

	c.Set(cCredential, credential)
	c.Next()
}

// GetCredential is a Gin handler function which returns the credential as a JSON object
func (ctrl CredentialController) One(c *gin.Context) {
	credential := c.MustGet(cCredential).(common.Credential)

	hideEncrypted(&credential)
	metadata.CredentialMetadata(&credential)

	c.JSON(http.StatusOK, credential)
}

// GetCredentials is a Gin handler function which returns list of credentials
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl CredentialController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"kind"}, match)
	match = parser.Lookups([]string{"name", "username"}, match)
	query := db.Credentials().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.Credential)
	var credentials []common.Credential
	iter := query.Iter()
	var credential common.Credential
	for iter.Next(&credential) {
		if !roles.Read(user, credential) {
			continue
		}
		hideEncrypted(&credential)
		metadata.CredentialMetadata(&credential)
		credentials = append(credentials, credential)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting credential", Log: logrus.Fields{
				"Error": err.Error(),
			},
		})
		return
	}
	count := len(credentials)
	pgi := util.NewPagination(c, count)
	if pgi.HasPage() {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound,
			Message: "#" + strconv.Itoa(pgi.Page()) + " page contains no results.",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  credentials[pgi.Skip():pgi.End()],
	})
}

// AddCredential is a Gin handler function which creates a new credential using request payload.
// This accepts Credential model.
func (ctrl CredentialController) Create(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	var req common.Credential
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	if req.OrganizationID != nil {
		if !req.OrganizationExist() {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Organization does not exists.",
			})
			return
		}

		// Check whether the user has permissions to associate the credential with organization
		if !(rbac.HasGlobalRead(user) || rbac.HasOrganizationRead(*req.OrganizationID, user.ID)) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
			return
		}
	}

	if !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Credential with this Name already exists.",
		})
		return
	}

	req.ID = bson.NewObjectId()
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")
	req.Password = util.CipherEncrypt(req.Password)
	req.SSHKeyData = util.CipherEncrypt(req.SSHKeyData)
	req.SSHKeyUnlock = util.CipherEncrypt(req.SSHKeyUnlock)
	req.BecomePassword = util.CipherEncrypt(req.BecomePassword)
	req.VaultPassword = util.CipherEncrypt(req.VaultPassword)
	req.AuthorizePassword = util.CipherEncrypt(req.AuthorizePassword)
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	req.Created = time.Now()
	req.Modified = time.Now()
	if err := db.Credentials().Insert(req); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Could not create Credential",
			Log:     logrus.Fields{"Credential ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	roles := new(rbac.Credential)
	if !(rbac.HasGlobalWrite(user) ||
		!(req.OrganizationID != nil || rbac.IsOrganizationAdmin(*req.OrganizationID, user.ID))) {
		roles.Associate(req.ID, user.ID, rbac.RoleTypeUser, rbac.CredentialAdmin)
	}

	activity.AddCredentialActivity(common.Create, user, req)
	hideEncrypted(&req)
	metadata.CredentialMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

// UpdateCredential is a Gin handler function which updates a credential using request payload.
// This replaces all the fields in the database, empty "" fields and
// unspecified fields will be removed from the database object.
func (ctrl CredentialController) Update(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	credential := c.MustGet(cCredential).(common.Credential)
	tmpCredential := credential

	var req common.Credential
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// check whether the organization exist or not
	if req.OrganizationID != nil {
		if !req.OrganizationExist() {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Organization does not exists.",
			})
			return
		}

		// Check whether the user has permissions to associate the credential with organization
		if !(rbac.HasGlobalRead(user) || rbac.HasOrganizationRead(*req.OrganizationID, user.ID)) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
			return
		}
	}

	// if the Credential exist in the collection it is not unique
	if req.Name != credential.Name && !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Credential with this Name already exists.",
		})
		return
	}

	// system generated
	credential.Name = strings.Trim(req.Name, " ")
	credential.Description = strings.Trim(req.Description, " ")
	credential.Kind = req.Kind
	credential.Cloud = req.Cloud
	credential.Host = req.Host
	credential.Username = req.Username
	credential.SecurityToken = req.SecurityToken
	credential.Project = req.Project
	credential.Domain = req.Domain
	credential.BecomeMethod = req.BecomeMethod
	credential.BecomeUsername = req.BecomeUsername
	credential.Subscription = req.Subscription
	credential.Tenant = req.Tenant
	credential.Secret = req.Secret
	credential.Client = req.Client
	credential.Authorize = req.Authorize
	credential.OrganizationID = req.OrganizationID
	credential.ModifiedByID = user.ID
	credential.Modified = time.Now()
	if req.Password != "$encrypted$" {
		credential.Password = util.CipherEncrypt(req.Password)
	}
	if req.SSHKeyData != "$encrypted$" {
		credential.SSHKeyData = util.CipherEncrypt(req.SSHKeyData)

		if req.SSHKeyUnlock != "$encrypted$" {
			credential.SSHKeyUnlock = util.CipherEncrypt(req.SSHKeyUnlock)
		}
	}
	if req.BecomePassword != "$encrypted$" {
		credential.BecomePassword = util.CipherEncrypt(req.BecomePassword)
	}
	if req.VaultPassword != "$encrypted$" {
		credential.VaultPassword = util.CipherEncrypt(req.VaultPassword)
	}
	if req.AuthorizePassword != "$encrypted$" {
		credential.AuthorizePassword = util.CipherEncrypt(req.AuthorizePassword)
	}

	if err := db.Credentials().UpdateId(credential.ID, credential); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating Credential",
			Log:     logrus.Fields{"Credential ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	roles := new(rbac.Credential)
	if !rbac.HasGlobalWrite(user) {
		roles.Associate(credential.ID, user.ID, rbac.RoleTypeUser, rbac.CredentialAdmin)
	} else if credential.OrganizationID != nil && !rbac.IsOrganizationAdmin(*credential.OrganizationID, user.ID) {
		roles.Associate(credential.ID, user.ID, rbac.RoleTypeUser, rbac.CredentialAdmin)
	}

	// add new activity to activity stream
	activity.AddCredentialActivity(common.Update, user, tmpCredential, credential)
	hideEncrypted(&credential)
	metadata.CredentialMetadata(&credential)
	c.JSON(http.StatusOK, credential)
}

// RemoveCredential is a Gin handler function which removes a credential object from the database
func (ctrl CredentialController) Delete(c *gin.Context) {
	credential := c.MustGet(cCredential).(common.Credential)
	user := c.MustGet(cUser).(common.User)

	if err := db.Credentials().RemoveId(credential.ID); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while deleting Credential",
			Log:     logrus.Fields{"Credential ID": credential.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	// add new activity to activity stream
	activity.AddCredentialActivity(common.Delete, user, credential)

	c.AbortWithStatus(http.StatusNoContent)
}

// OwnerTeams is a Gin handler function which returns the access control list of Teams that has permissions to access
// specified credential object.
func (ctrl CredentialController) OwnerTeams(c *gin.Context) {
	credential := c.MustGet(cCredential).(common.Credential)

	var tms []common.Team

	for _, v := range credential.Roles {
		if v.Type == "team" {
			var team common.Team
			err := db.Teams().FindId(v.GranteeID).One(&team)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Credential ID": credential.ID,
					"Error":         err.Error(),
				}).Errorln("Error while getting owner teams for credential")
				continue //skip iteration
			}
			// set additional info and append to slice
			metadata.TeamMetadata(&team)
			tms = append(tms, team)
		}
	}

	count := len(tms)
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
		Data:  tms[pgi.Skip():pgi.End()],
	})
}

// OwnerUsers is a Gin handler function which returns the access control list of Users that has access to
// specified credential object.
func (ctrl CredentialController) OwnerUsers(c *gin.Context) {
	credential := c.MustGet(cCredential).(common.Credential)

	var usrs []common.User
	for _, v := range credential.Roles {
		if v.Type == "user" {
			var user common.User
			err := db.Users().FindId(v.GranteeID).One(&user)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"Credential ID": credential.ID,
					"Error":         err.Error(),
				}).Errorln("Error while getting owner users for Credential")
				continue //skip iteration
			}
			// set additional info and append to slice
			metadata.UserMetadata(&user)
			usrs = append(usrs, user)
		}
	}

	count := len(usrs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
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
		Data:  usrs[pgi.Skip():pgi.End()],
	})
}

// ActivityStream returns the activities of the user on Credentials
func (ctrl CredentialController) ActivityStream(c *gin.Context) {
	credential := c.MustGet(cCredential).(common.Credential)

	var activities []common.ActivityCredential
	var act common.ActivityCredential
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": credential.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&act) {
		metadata.ActivityCredentialMetadata(&act)
		metadata.CredentialMetadata(&act.Object1)
		hideEncrypted(&act.Object1)
		//apply metadata only when Object2 is available
		if act.Object2 != nil {
			metadata.CredentialMetadata(act.Object2)
			hideEncrypted(act.Object2)
		}
		//add to activities list
		activities = append(activities, act)
	}

	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting Activities",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	count := len(activities)
	pgi := util.NewPagination(c, count)

	if pgi.HasPage() {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound,
			Message: "#" + strconv.Itoa(pgi.Page()) + " page contains no results.",
		})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  activities[pgi.Skip():pgi.End()],
	})
}
