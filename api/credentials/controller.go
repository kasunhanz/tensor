package credentials

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/util"
	"time"
	"strconv"
	"bitbucket.pearson.com/apseng/tensor/crypt"
	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/roles"
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
	"bitbucket.pearson.com/apseng/tensor/api/helpers"
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
		log.Print("Error while getting the Credential:", err)
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		return
	}
	user := c.MustGet(_CTX_USER).(models.User)

	var credential models.Credential
	if err = db.Credentials().FindId(bson.ObjectIdHex(ID)).One(&credential); err != nil {
		log.Print("Error while getting the Credential:", err)
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		return
	}

	// reject the request if the user doesn't have permissions
	if !roles.CredentialRead(user, credential) {
		c.JSON(http.StatusUnauthorized, models.Error{
			Code: http.StatusUnauthorized,
			Messages: []string{"Unauthorized"},
		})
		return
	}

	c.Set(_CTX_CREDENTIAL, credential)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetCredential(c *gin.Context) {
	credential := c.MustGet(_CTX_CREDENTIAL).(models.Credential)

	hideEncrypted(&credential)

	if err := metadata.CredentialMetadata(&credential); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Credential"},
		})
		return
	}

	c.JSON(http.StatusOK, credential)
}

func GetCredentials(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	parser := util.NewQueryParser(c)
	match := parser.Match([]string{"kind"})

	if con := parser.IContains([]string{"name", "username"}); con != nil {
		if match != nil {
			for i, v := range con {
				match[i] = v
			}
		} else {
			match = con
		}
	}

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
		if err := metadata.CredentialMetadata(&tmpCred); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Credentials"},
			})
			return
		}
		// good to go add to list
		credentials = append(credentials, tmpCred)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Credential data from the db:", err)
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
		log.Println("Bad payload:", err)
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

	if req.Password != "" {
		password := crypt.Encrypt(req.Password)
		req.Password = password
	}

	if req.SshKeyData != "" {
		data := crypt.Encrypt(req.SshKeyData)
		req.SshKeyData = data

		if req.SshKeyUnlock != "" {
			unlock := crypt.Encrypt(req.SshKeyUnlock)
			req.SshKeyUnlock = unlock
		}
	}

	if req.BecomePassword != "" {
		password := crypt.Encrypt(req.BecomePassword)
		req.BecomePassword = password
	}
	if req.VaultPassword != "" {
		password := crypt.Encrypt(req.VaultPassword)
		req.VaultPassword = password
	}

	err := db.Credentials().Insert(req)
	if err != nil {
		log.Println("Error while creating Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Credential"},
		})
		return
	}

	err = roles.AddCredentialUser(req, user.ID, roles.CREDENTIAL_ADMIN)
	if err != nil {
		log.Println("Error while adding the user to roles:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while adding the user to roles"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Credential " + req.Name + " created")
	hideEncrypted(&req)
	if err := metadata.CredentialMetadata(&req); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while setting metadata"},
		})
	}

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

	if req.Password != "" {
		password := crypt.Encrypt(req.Password)
		credential.Password = password
	}

	if req.Password != "" {
		data := crypt.Encrypt(req.SshKeyData)
		credential.SshKeyData = data

		if req.SshKeyUnlock != "" {
			unlock := crypt.Encrypt(credential.SshKeyUnlock)
			credential.SshKeyUnlock = unlock
		}
	}

	if req.Password != "" {
		password := crypt.Encrypt(req.BecomePassword)
		credential.BecomeUsername = password
	}
	if req.Password != "" {
		password := crypt.Encrypt(req.VaultPassword)
		credential.VaultPassword = password
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	// system generated
	req.ID = credential.ID
	req.CreatedByID = credential.CreatedByID
	req.Created = credential.Created
	req.ModifiedByID = user.ID
	req.Modified = time.Now()

	if err := db.Credentials().UpdateId(credential.ID, req); err != nil {
		log.Println("Error while updating Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Credential"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Credential " + req.Name + " updated")

	hideEncrypted(&req)
	if err := metadata.CredentialMetadata(&req); err != nil {
		log.Println("Error while updating Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Credential"},
		})
		return
	}

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

	if len(req.Name) > 0 && req.Name != credential.Name {
		// if the Credential exist in the collection it is not unique
		if helpers.IsNotUniqueCredential(req.Name) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Credential with this Name already exists."},
			})
			return
		}
	}

	if req.Password != "" {
		password := crypt.Encrypt(req.Password)
		credential.Password = password
	}

	if req.Password != "" {
		data := crypt.Encrypt(req.SshKeyData)
		credential.SshKeyData = data

		if req.SshKeyUnlock != "" {
			unlock := crypt.Encrypt(credential.SshKeyUnlock)
			credential.SshKeyUnlock = unlock
		}
	}

	if req.Password != "" {
		password := crypt.Encrypt(req.BecomePassword)
		credential.BecomeUsername = password
	}
	if req.Password != "" {
		password := crypt.Encrypt(req.VaultPassword)
		credential.VaultPassword = password
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	// system generated
	req.ModifiedByID = user.ID
	req.Modified = time.Now()

	if err := db.Credentials().UpdateId(credential.ID, bson.M{"$set": req}); err != nil {
		log.Println("Error while updating Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Credential"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(credential.ID, user.ID, "Credential " + req.Name + " updated")


	// get newly updated group
	var resp models.Credential
	if err := db.Credentials().FindId(credential.ID).One(&resp); err != nil {
		log.Print("Error while getting the updated Credential:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Error while getting the updated Credential"},
		})
		return
	}

	hideEncrypted(&resp)
	if err := metadata.CredentialMetadata(&resp); err != nil {
		log.Println("Error while updating Credential:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Credential"},
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func RemoveCredential(c *gin.Context) {
	crd := c.MustGet(_CTX_CREDENTIAL).(models.Credential)
	u := c.MustGet(_CTX_USER).(models.User)

	if err := db.Credentials().RemoveId(crd.ID); err != nil {
		log.Println("Error while deleting Credential:", err)
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
				log.Println("Error while getting owner teams for credential", credential.ID, err)
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
				log.Println("Error while getting owner users for credential", credential.ID, err)
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
		log.Println("Error while retriving Activity data from the db:", err)
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