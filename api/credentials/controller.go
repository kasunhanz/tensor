package credentials

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gamunu/tensor/api/helpers"
	"github.com/gamunu/tensor/api/metadata"
	"github.com/gamunu/tensor/db"
	"github.com/gamunu/tensor/models"
	"github.com/gamunu/tensor/roles"
	"github.com/gamunu/tensor/util"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential releated items stored in the Gin Context
const (
	CTXCredential   = "credential"
	CTXCredentialID = "credential_id"
	CTXUser         = "user"
)

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXCredentialID from Gin Context and retrieves credential data from the collection
// and store credential data under key CTXCredential in Gin Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(CTXCredentialID, c)

	if err != nil {
		log.WithFields(log.Fields{
			"Credential ID": ID,
			"Error":         err.Error(),
		}).Errorln("Error while getting Credential ID url parameter")
		c.JSON(http.StatusNotFound, models.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}
	user := c.MustGet(CTXUser).(models.User)

	var credential models.Credential
	if err = db.Credentials().FindId(bson.ObjectIdHex(ID)).One(&credential); err != nil {
		log.WithFields(log.Fields{
			"Credential ID": ID,
			"Error":         err.Error(),
		}).Errorln("Error while retriving Credential form the database")
		c.JSON(http.StatusNotFound, models.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	// reject the request if the user doesn't have permissions
	if !roles.CredentialRead(user, credential) {
		c.JSON(http.StatusUnauthorized, models.Error{
			Code:     http.StatusUnauthorized,
			Messages: []string{"Unauthorized"},
		})
		c.Abort()
		return
	}

	c.Set(CTXCredential, credential)
	c.Next()
}

// GetCredential is a Gin handler function which returns the credential as a JSON object
func GetCredential(c *gin.Context) {
	credential := c.MustGet(CTXCredential).(models.Credential)

	hideEncrypted(&credential)
	metadata.CredentialMetadata(&credential)

	c.JSON(http.StatusOK, credential)
}

// GetCredentials is a Gin handler function which returns list of credentials
// This takes lookup parameters and order parameters to filter and sort output data
func GetCredentials(c *gin.Context) {
	user := c.MustGet(CTXUser).(models.User)

	parser := util.NewQueryParser(c)

	match := bson.M{}
	match = parser.Match([]string{"kind"}, match)
	match = parser.Lookups([]string{"name", "username"}, match)

	query := db.Credentials().Find(match)

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	log.WithFields(log.Fields{
		"Query": query,
	}).Debugln("Parsed query")

	var credentials []models.Credential
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpCred models.Credential
	// iterate over all and only get valid objects
	for iter.Next(&tmpCred) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.CredentialRead(user, tmpCred) {
			log.WithFields(log.Fields{
				"User ID":       user.ID.Hex(),
				"Credential ID": tmpCred.ID.Hex(),
			}).Debugln("User does not have read permissions")
			continue
		}
		// hide passwords, keys even they are already encrypted
		hideEncrypted(&tmpCred)
		metadata.CredentialMetadata(&tmpCred)
		// good to go add to list
		credentials = append(credentials, tmpCred)
	}
	if err := iter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Credential data from the database")
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Credential"},
		})
		return
	}

	count := len(credentials)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Credential page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	log.WithFields(log.Fields{
		"Count":    count,
		"Next":     pgi.NextPage(),
		"Previous": pgi.PreviousPage(),
		"Skip":     pgi.Skip(),
		"Limit":    pgi.Limit(),
	}).Debugln("Response info")
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  credentials[pgi.Skip():pgi.End()],
	})
}

// AddCredential is a Gin handler function which creates a new credential using request payload.
// This accepts Credential model.
func AddCredential(c *gin.Context) {
	user := c.MustGet(CTXUser).(models.User)

	var req models.Credential

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Invlid JSON request")
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if req.OrganizationID != nil {
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	// if the Credential exist in the collection it is not unique
	if helpers.IsNotUniqueCredential(req.Name) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Credential with this Name already exists."},
		})
		return
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	req.ID = bson.NewObjectId()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	req.Created = time.Now()
	req.Modified = time.Now()

	if len(req.Password) > 0 {
		req.Password = util.CipherEncrypt(req.Password)
	}

	if len(req.SshKeyData) > 0 {
		req.SshKeyData = util.CipherEncrypt(req.SshKeyData)

		if len(req.SshKeyUnlock) > 0 {
			req.SshKeyUnlock = util.CipherEncrypt(req.SshKeyUnlock)
		}
	}

	if len(req.BecomePassword) > 0 {
		req.BecomePassword = util.CipherEncrypt(req.BecomePassword)
	}
	if len(req.VaultPassword) > 0 {
		req.VaultPassword = util.CipherEncrypt(req.VaultPassword)
	}

	if len(req.AuthorizePassword) > 0 {
		req.AuthorizePassword = util.CipherEncrypt(req.AuthorizePassword)
	}

	if err := db.Credentials().Insert(req); err != nil {
		log.WithFields(log.Fields{
			"Credential ID": req.ID.Hex(),
			"Error":         err.Error(),
		}).Errorln("Error while creating Credential")
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while creating Credential"},
		})
		return
	}

	roles.AddCredentialUser(req, user.ID, roles.CREDENTIAL_ADMIN)

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(models.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXCredential,
		ObjectID:    req.ID,
		Description: "Credential " + req.Name + " created",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	hideEncrypted(&req)
	metadata.CredentialMetadata(&req)

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// UpdateCredential is a Gin handler function which updates a credential using request payload.
// This replaces all the fields in the database, empty "" fields and
// unspecified fields will be removed from the database object.
func UpdateCredential(c *gin.Context) {

	user := c.MustGet(CTXUser).(models.User)
	credential := c.MustGet(CTXCredential).(models.Credential)

	var req models.Credential
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if req.OrganizationID != nil {
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	if req.Name != credential.Name {
		// if the Credential exist in the collection it is not unique
		if helpers.IsNotUniqueCredential(req.Name) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Credential with this Name already exists."},
			})
			return
		}
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

	if req.Password != "$encrypted$" && len(req.Password) > 0 {
		credential.Password = util.CipherEncrypt(req.Password)
	}

	if req.SshKeyData != "$encrypted$" && len(req.SshKeyData) > 0 {
		credential.SshKeyData = util.CipherEncrypt(req.SshKeyData)

		if req.SshKeyUnlock != "$encrypted$" && len(req.SshKeyUnlock) > 0 {
			credential.SshKeyUnlock = util.CipherEncrypt(req.SshKeyUnlock)
		}
	}

	if req.BecomePassword != "$encrypted$" && len(req.BecomePassword) > 0 {
		credential.BecomePassword = util.CipherEncrypt(req.BecomePassword)
	}

	if req.VaultPassword != "$encrypted$" && len(req.VaultPassword) > 0 {
		credential.VaultPassword = util.CipherEncrypt(req.VaultPassword)
	}

	if req.AuthorizePassword != "$encrypted$" && len(req.AuthorizePassword) > 0 {
		credential.AuthorizePassword = util.CipherEncrypt(req.AuthorizePassword)
	}

	if err := db.Credentials().UpdateId(credential.ID, credential); err != nil {
		log.WithFields(log.Fields{
			"Credential ID": req.ID.Hex(),
			"Error":         err.Error(),
		}).Errorln("Error while updating Credential")
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Credential"},
		})
		return
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(models.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXCredential,
		ObjectID:    req.ID,
		Description: "Credential " + req.Name + " updated",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	hideEncrypted(&req)
	metadata.CredentialMetadata(&req)

	c.JSON(http.StatusOK, req)
}

// PatchCredential is a Gin handler function which partially updates a credential using request payload.
// This replaces specifed fields in the data, empty "" fields will be
// removed from the database object. Unspecified fields will be ignored.
func PatchCredential(c *gin.Context) {
	user := c.MustGet(CTXUser).(models.User)
	credential := c.MustGet(CTXCredential).(models.Credential)

	var req models.PatchCredential
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if req.OrganizationID != nil {
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	if req.Name != nil && *req.Name != credential.Name {
		// if the Credential exist in the collection it is not unique
		if helpers.IsNotUniqueCredential(*req.Name) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Credential with this Name already exists."},
			})
			return
		}
	}

	if req.Password != nil && *req.Password != "$encrypted$" {
		credential.Password = util.CipherEncrypt(*req.Password)
	}

	if req.SshKeyData != nil && *req.SshKeyData != "$encrypted$" {
		credential.SshKeyData = util.CipherEncrypt(*req.SshKeyData)

		if req.SshKeyUnlock != nil && *req.SshKeyUnlock != "$encrypted$" {
			credential.SshKeyUnlock = util.CipherEncrypt(*req.SshKeyUnlock)
		}
	}

	if req.BecomePassword != nil && *req.BecomePassword != "$encrypted$" {
		credential.BecomePassword = util.CipherEncrypt(*req.BecomePassword)
	}

	if req.VaultPassword != nil && *req.VaultPassword != "$encrypted$" {
		credential.VaultPassword = util.CipherEncrypt(*req.VaultPassword)
	}

	if req.AuthorizePassword != nil && *req.AuthorizePassword != "$encrypted$" {
		credential.AuthorizePassword = util.CipherEncrypt(*req.AuthorizePassword)
	}

	// replace following feilds if precent
	if req.Secret != nil && *req.Secret != "$encrypted$" {
		credential.Secret = util.CipherEncrypt(*req.Secret)
	}

	if req.Name != nil {
		credential.Name = strings.Trim(*req.Name, " ")
	}

	if req.Kind != nil {
		credential.Kind = *req.Kind
	}

	if req.Cloud != nil {
		credential.Cloud = *req.Cloud
	}

	if req.Description != nil {
		credential.Description = strings.Trim(*req.Description, " ")
	}

	if req.Host != nil {
		credential.Host = *req.Host
	}

	if req.Username != nil {
		credential.Username = *req.Username
	}

	if req.SecurityToken != nil {
		credential.SecurityToken = *req.SecurityToken
	}

	if req.Project != nil {
		credential.Project = *req.Project
	}

	if req.Domain != nil {
		credential.Domain = *req.Domain
	}

	if req.BecomeMethod != nil {
		credential.BecomeMethod = *req.BecomeMethod
	}

	if req.BecomeUsername != nil {
		credential.BecomeUsername = *req.BecomeUsername
	}

	if req.Subscription != nil {
		credential.Subscription = *req.Subscription
	}

	if req.Tenant != nil {
		credential.Tenant = *req.Tenant
	}

	if req.Client != nil {
		credential.Client = *req.Client
	}

	if req.Authorize != nil {
		credential.Authorize = *req.Authorize
	}

	if req.OrganizationID != nil {
		// if empty string then make the credential null
		if len(*req.OrganizationID) == 12 {
			credential.OrganizationID = req.OrganizationID
		} else {
			credential.OrganizationID = nil
		}
	}

	// system generated
	credential.ModifiedByID = user.ID
	credential.Modified = time.Now()

	if err := db.Credentials().UpdateId(credential.ID, credential); err != nil {
		log.WithFields(log.Fields{
			"Credential ID": credential.ID.Hex(),
			"Error":         err.Error(),
		}).Errorln("Error while updating Credential")
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Credential"},
		})
		return
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(models.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXCredential,
		ObjectID:    credential.ID,
		Description: "Credential " + credential.Name + " updated",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	hideEncrypted(&credential)
	metadata.CredentialMetadata(&credential)

	c.JSON(http.StatusOK, credential)
}

// RemoveCredential is a Gin handler function which removes a credential object from the database
func RemoveCredential(c *gin.Context) {
	crd := c.MustGet(CTXCredential).(models.Credential)
	u := c.MustGet(CTXUser).(models.User)

	if err := db.Credentials().RemoveId(crd.ID); err != nil {
		log.WithFields(log.Fields{
			"Credential ID": crd.ID.Hex(),
			"Error":         err.Error(),
		}).Errorln("Error while deleting Credential")
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while deleting Credential"},
		})

		return
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(models.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     u.ID,
		Type:        CTXCredential,
		ObjectID:    u.ID,
		Description: "Credential " + crd.Name + " deleted",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	c.AbortWithStatus(http.StatusNoContent)
}

// OwnerTeams is a Gin hander function which returns the access control list of Teams that has permissions to access
// specifed credential object.
func OwnerTeams(c *gin.Context) {
	credential := c.MustGet(CTXCredential).(models.Credential)

	var tms []models.Team

	for _, v := range credential.Roles {
		if v.Type == "team" {
			var team models.Team
			err := db.Teams().FindId(v.TeamID).One(&team)
			if err != nil {
				log.WithFields(log.Fields{
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
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Credential page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	log.WithFields(log.Fields{
		"Count":    count,
		"Next":     pgi.NextPage(),
		"Previous": pgi.PreviousPage(),
		"Skip":     pgi.Skip(),
		"Limit":    pgi.Limit(),
	}).Debugln("Response info")
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  tms[pgi.Skip():pgi.End()],
	})
}

// OwnerUsers is a Gin handler function which returns the access control list of Users that has access to
// specifed credential object.
func OwnerUsers(c *gin.Context) {
	credential := c.MustGet(CTXCredential).(models.Credential)

	var usrs []models.User
	for _, v := range credential.Roles {
		if v.Type == "user" {
			var user models.User
			err := db.Users().FindId(v.UserID).One(&user)
			if err != nil {
				log.WithFields(log.Fields{
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
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("OwnerUser page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	log.WithFields(log.Fields{
		"Count":    count,
		"Next":     pgi.NextPage(),
		"Previous": pgi.PreviousPage(),
		"Skip":     pgi.Skip(),
		"Limit":    pgi.Limit(),
	}).Debugln("Response info")
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  usrs[pgi.Skip():pgi.End()],
	})
}

// ActivityStream is a Gin handler function which returns list of activities associated with
// credential object that is in the Gin Context
// TODO: not complete
func ActivityStream(c *gin.Context) {
	credential := c.MustGet(CTXCredential).(models.Credential)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": credential.ID, "type": CTXCredential}).All(&activities)

	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Activity data from the database")
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while Activities"},
		})
		return
	}

	count := len(activities)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Activity Stream page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	log.WithFields(log.Fields{
		"Count":    count,
		"Next":     pgi.NextPage(),
		"Previous": pgi.PreviousPage(),
		"Skip":     pgi.Skip(),
		"Limit":    pgi.Limit(),
	}).Debugln("Response info")
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  activities[pgi.Skip():pgi.End()],
	})
}

// hideEncrypted is replaces encrypted fields by $encrypted$ string
func hideEncrypted(c *models.Credential) {
	encrypted := "$encrypted$"
	c.Password = encrypted
	c.SshKeyData = encrypted
	c.SshKeyUnlock = encrypted
	c.BecomePassword = encrypted
	c.VaultPassword = encrypted
	c.AuthorizePassword = encrypted
	c.Secret = encrypted
}
