package credentials

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/util"
	"time"
	"strconv"
	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/roles"
	"bitbucket.pearson.com/apseng/tensor/controllers/metadata"
	"bitbucket.pearson.com/apseng/tensor/controllers/helpers"
	"strings"
	"github.com/gin-gonic/gin/binding"
)

const _CTX_CREDENTIAL = "credential"
const _CTX_CREDENTIAL_ID = "credential_id"
const _CTX_USER = "user"

// Middleware takes _CTX_CREDENTIAL_ID from gin.Context and
// retrieves credential data from the collection
// and store credential data under key _CTX_CREDENTIAL in gin.Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(_CTX_CREDENTIAL_ID, c)

	if err != nil {
		log.Errorln("Error while getting the Credential:", err)
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}
	user := c.MustGet(_CTX_USER).(models.User)

	var credential models.Credential
	if err = db.Credentials().FindId(bson.ObjectIdHex(ID)).One(&credential); err != nil {
		log.Errorln("Error while getting the Credential:", err)
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	// reject the request if the user doesn't have permissions
	if !roles.CredentialRead(user, credential) {
		c.JSON(http.StatusUnauthorized, models.Error{
			Code: http.StatusUnauthorized,
			Messages: []string{"Unauthorized"},
		})
		c.Abort()
		return
	}

	c.Set(_CTX_CREDENTIAL, credential)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetCredential(c *gin.Context) {
	credential := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	hideEncrypted(&credential)
	metadata.CredentialMetadata(&credential)

	c.JSON(http.StatusOK, credential)
}

func GetCredentials(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	parser := util.NewQueryParser(c)

	match := bson.M{}
	match = parser.Match([]string{"kind"}, match)
	match = parser.Lookups([]string{"name", "username"}, match)

	query := db.Credentials().Find(match)

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

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
			continue
		}
		// hide passwords, keys even they are already encrypted
		hideEncrypted(&tmpCred)
		metadata.CredentialMetadata(&tmpCred)
		// good to go add to list
		credentials = append(credentials, tmpCred)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Credential"},
		})
		return
	}

	count := len(credentials)
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
		Results: credentials[pgi.Skip():pgi.End()],
	})
}

func AddCredential(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Credential

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if req.OrganizationID != nil {
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	// if the Credential exist in the collection it is not unique
	if helpers.IsNotUniqueCredential(req.Name) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
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
		log.Errorln("Error while creating Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Credential"},
		})
		return
	}

	if err := roles.AddCredentialUser(req, user.ID, roles.CREDENTIAL_ADMIN); err != nil {
		log.Errorln("Error while adding the user to roles:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while adding the user to roles"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Credential " + req.Name + " created")
	hideEncrypted(&req)
	metadata.CredentialMetadata(&req)

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

func UpdateCredential(c *gin.Context) {

	user := c.MustGet(_CTX_USER).(models.User)
	credential := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	var req models.Credential
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if req.OrganizationID != nil {
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	if req.Name != credential.Name {
		// if the Credential exist in the collection it is not unique
		if helpers.IsNotUniqueCredential(req.Name) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Credential with this Name already exists."},
			})
			return
		}
	}

	// system generated
	credential.Name = strings.Trim(req.Name, " ")
	credential.Kind = req.Kind
	credential.Cloud = req.Cloud
	credential.Description = strings.Trim(req.Description, " ")
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
		log.Errorln("Error while updating Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Credential"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Credential " + credential.Name + " updated")

	hideEncrypted(&req)
	metadata.CredentialMetadata(&req)

	c.JSON(http.StatusOK, req)
}

func PatchCredential(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)
	credential := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	var req models.PatchCredential
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if req.OrganizationID != nil {
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	if req.Name != nil && *req.Name != credential.Name {
		// if the Credential exist in the collection it is not unique
		if helpers.IsNotUniqueCredential(*req.Name) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
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
		log.Errorln("Error while updating Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Credential"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(credential.ID, user.ID, "Credential " + credential.Name + " updated")

	hideEncrypted(&credential)
	metadata.CredentialMetadata(&credential)

	c.JSON(http.StatusOK, credential)
}

func RemoveCredential(c *gin.Context) {
	crd := c.MustGet(_CTX_CREDENTIAL).(models.Credential)
	u := c.MustGet(_CTX_USER).(models.User)

	if err := db.Credentials().RemoveId(crd.ID); err != nil {
		log.Errorln("Error while deleting Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while deleting Credential"},
		})

		return
	}

	// add new activity to activity stream
	addActivity(crd.ID, u.ID, "Credential " + crd.Name + " deleted")

	c.AbortWithStatus(http.StatusNoContent)
}

func OwnerTeams(c *gin.Context) {
	credential := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	var tms []models.Team

	for _, v := range credential.Roles {
		if v.Type == "team" {
			var team models.Team
			err := db.Teams().FindId(v.TeamID).One(&team)
			if err != nil {
				log.Errorln("Error while getting owner teams for credential", credential.ID, err)
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

func OwnerUsers(c *gin.Context) {
	credential := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	var usrs []models.User
	for _, v := range credential.Roles {
		if v.Type == "user" {
			var user models.User
			err := db.Users().FindId(v.UserID).One(&user)
			if err != nil {
				log.Errorln("Error while getting owner users for credential", credential.ID, err)
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
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: usrs[pgi.Skip():pgi.End()],
	})
}

// TODO: not complete
func ActivityStream(c *gin.Context) {
	credential := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": credential.ID, "type": _CTX_CREDENTIAL}).All(&activities)

	if err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while Activities"},
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