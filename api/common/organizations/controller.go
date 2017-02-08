package organizations

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/helpers"
	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/roles"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential releated items stored in the Gin Context
const (
	CTXOrganization   = "organization"
	CTXOrganizationID = "organization_id"
	CTXUser           = "user"
)

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXOrganizationID from Gin Context and retrieves organization data from the collection
// and store organization data under key CTXOrganization in Gin Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(CTXOrganizationID, c)

	if err != nil {
		log.WithFields(log.Fields{
			"Organization ID": ID,
			"Error":           err.Error(),
		}).Errorln("Error while getting Organization ID url parameter")
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Organization Not Found"},
		})
		c.Abort()
		return
	}

	var organization common.Organization
	if err = db.Organizations().FindId(bson.ObjectIdHex(ID)).One(&organization); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": ID,
			"Error":           err.Error(),
		}).Errorln("Error while retriving Organization form the database")
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Organization Not Found"},
		})
		c.Abort()
		return
	}

	user := c.MustGet(CTXUser).(common.User)

	// reject the request if the user doesn't have permissions
	if !roles.OrganizationRead(user, organization) {
		c.JSON(http.StatusUnauthorized, common.Error{
			Code:     http.StatusUnauthorized,
			Messages: []string{"Unauthorized"},
		})
		c.Abort()
		return
	}

	c.Set(CTXOrganization, organization)
	c.Next()
}

// GetOrganization is a Gin handler function which returns the organization as a JSON object
func GetOrganization(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	metadata.OrganizationMetadata(&organization)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, organization)
}

// GetOrganizations is a Gin handler function which returns list of organization
// This takes lookup parameters and order parameters to filter and sort output data
func GetOrganizations(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"name", "description"}, match)

	query := db.Organizations().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	log.WithFields(log.Fields{
		"Query": query,
	}).Debugln("Parsed query")

	var organizations []common.Organization
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpOrganization common.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.OrganizationRead(user, tmpOrganization) {
			continue
		}
		metadata.OrganizationMetadata(&tmpOrganization)
		// good to go add to list
		organizations = append(organizations, tmpOrganization)
	}
	if err := iter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Organization data from the database")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization"},
		})
		return
	}

	count := len(organizations)
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  organizations[pgi.Skip():pgi.End()],
	})
}

// AddOrganization is a Gin handler function which creates a new organization using request payload.
// This accepts Organization model.
func AddOrganization(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	var req common.Organization
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

	// if the Organization exist in the collection it is not unique
	if helpers.IsNotUniqueOrganization(req.Name) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization with this Name already exists."},
		})
		return
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.CreatedByID = user.ID
	req.Modified = time.Now()
	req.ModifiedByID = user.ID

	if err := db.Organizations().Insert(req); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while creating Organization")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while creating Organization"},
		})
		return
	}
	// add new activity to activity stream
	activity.AddOrganizationActivity(common.Create, user, req)

	metadata.OrganizationMetadata(&req)
	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// RemoveOrganization is a Gin handler function which removes a organization object from the database
func RemoveOrganization(c *gin.Context) {
	// get Organization from the gin.Context
	organization := c.MustGet(CTXOrganization).(common.Organization)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	// remove all projects
	orgIter := db.Projects().Find(bson.M{"organization_id": organization.ID}).Iter()
	var project common.Project
	for orgIter.Next(&project) {
		// remove all jobs for the project
		changes, err := db.Jobs().RemoveAll(bson.M{"project_id": project.ID})
		if err != nil {
			log.WithFields(log.Fields{
				"Project ID": project.ID.Hex(),
				"Error":      err.Error(),
			}).Errorln("Error while deleting Project Jobs")
			c.JSON(http.StatusInternalServerError, common.Error{
				Code:     http.StatusInternalServerError,
				Messages: []string{"Error while removing Project Jobs"},
			})
			return
		}
		log.Infoln("Jobs remove info:", changes.Removed)

		// remove all job templates
		changes, err = db.JobTemplates().RemoveAll(bson.M{"project_id": project.ID})
		if err != nil {
			log.WithFields(log.Fields{
				"Project ID": project.ID.Hex(),
				"Error":      err.Error(),
			}).Errorln("Error while deleting Project Job Templates")
			c.JSON(http.StatusInternalServerError, common.Error{
				Code:     http.StatusInternalServerError,
				Messages: []string{"Error while removing Project Job Templates"},
			})
			return
		}
		log.Infoln("Job Template remove info:", changes.Removed)

		// remove the project as well
		err = db.Projects().RemoveId(project.ID)
		if err != nil {
			log.WithFields(log.Fields{
				"Project ID": project.ID.Hex(),
				"Error":      err.Error(),
			}).Errorln("Error while deleting Project")
			c.JSON(http.StatusInternalServerError, common.Error{
				Code:     http.StatusInternalServerError,
				Messages: []string{"Error while removing Project"},
			})
			return
		}
	}

	// remove all inventories associated with organization
	changes, err := db.Inventories().RemoveAll(bson.M{"organization_id": organization.ID})
	if err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":           err.Error(),
		}).Errorln("Error while removing Inventories")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Inventories"},
		})
		return
	}
	log.Infoln("Inventory remove info:", changes.Removed)

	// remove the organization as well
	err = db.Organizations().RemoveId(organization.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":           err.Error(),
		}).Errorln("Error while removing Organization")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Organization"},
		})
		return
	}
	// add new activity to activity stream
	activity.AddOrganizationActivity(common.Delete, user, organization)

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

// UpdateOrganization is a Gin handler function which updates a organization using request payload.
// This replaces all the fields in the database, empty "" fields and
// unspecified fields will be removed from the database object.
func UpdateOrganization(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)
	tmpOrg := organization
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req common.Organization
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if req.Name != organization.Name {
		// if the Organization exist in the collection it is not unique
		if helpers.IsNotUniqueOrganization(req.Name) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization with this Name already exists."},
			})
			return
		}
	}

	// trim strings white space
	organization.Name = strings.Trim(req.Name, " ")
	organization.Description = strings.Trim(req.Description, " ")
	organization.Modified = time.Now()
	organization.ModifiedByID = user.ID

	if err := db.Organizations().UpdateId(organization.ID, organization); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": req.ID.Hex(),
			"Error":           err.Error(),
		}).Errorln("Error while updating Organization")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Organization"},
		})
		return
	}

	// add new activity to activity stream
	activity.AddOrganizationActivity(common.Update, user, tmpOrg, organization)

	metadata.OrganizationMetadata(&organization)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, organization)
}

// PatchOrganization is a Gin handler function which partially updates a organization using request payload.
// This replaces specifed fields in the data, empty "" fields will be
// removed from the database object. Unspecified fields will be ignored.
func PatchOrganization(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)
	tmpOrg := organization
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req common.PatchOrganization
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

	// since this is a patch request if the name specified check the
	// Organization name is unique
	if req.Name != nil && *req.Name != organization.Name {
		// if the Organization exist in the collection it is not unique
		if helpers.IsNotUniqueOrganization(*req.Name) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization with this Name already exists."},
			})
			return
		}
	}

	// trim strings white space
	if req.Name != nil {
		organization.Name = strings.Trim(*req.Name, " ")
	}

	if req.Description != nil {
		organization.Description = strings.Trim(*req.Description, " ")
	}

	organization.Modified = time.Now()
	organization.ModifiedByID = user.ID

	if err := db.Organizations().UpdateId(organization.ID, organization); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":           err.Error(),
		}).Errorln("Error while updating Organization")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Organization"},
		})
		return
	}

	// add new activity to activity stream
	activity.AddOrganizationActivity(common.Update, user, tmpOrg, organization)

	metadata.OrganizationMetadata(&organization)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, organization)
}

// GetUsers Returns all Organization users
func GetUsers(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var usrs []common.User

	err := db.Users().Find(bson.M{"organization_id": organization.ID}).All(&usrs)

	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while getting Organization users")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization users"},
		})
	}

	for i, v := range usrs {
		metadata.UserMetadata(&v)
		usrs[i] = v
	}

	count := len(usrs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Users page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  usrs[pgi.Skip():pgi.End()],
	})
}

// GetAdmins returns an Organization admins
func GetAdmins(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var usrs []common.User

	for _, v := range organization.Roles {
		// get user with role admin
		if v.Type == "user" && v.Role == "admin" {
			var user common.User
			err := db.Users().FindId(v.UserID).One(&user)
			if err != nil {
				log.Errorln("Error while getting owner users for organization", organization.ID, err)
				continue //skip iteration
			}
			// set additional info and append to slice
			metadata.UserMetadata(&user)
			usrs = append(usrs, user)
		}
		//get teams with role admin and team users to output slice
		if v.Type == "team" && v.Role == "admin" {
			var team common.Team
			err := db.Teams().FindId(v.TeamID).One(&team)
			if err != nil {
				log.Errorln("Error while getting team for organization role", organization.ID, err)
				continue // ignore and continue
			}

			for _, v := range team.Roles {
				var user common.User
				if v.Type == "user" {
					err := db.Users().FindId(v.UserID).One(&user)
					if err != nil {
						log.Errorln("Error while getting owner users for organization", organization.ID, err)
						continue // ignore and continue
					}
				}

				// set additional info and append to slice
				metadata.UserMetadata(&user)
				usrs = append(usrs, user)
			}
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  usrs[pgi.Skip():pgi.End()],
	})
}

// GetTeams will return an Organization Teams
func GetTeams(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var tms []common.Team
	err := db.Teams().Find(bson.M{"organization_id": organization.ID}).All(&tms)

	if err != nil {
		log.Errorln("Error while getting Organization teams:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization teams"},
		})
	}

	for i, v := range tms {
		metadata.TeamMetadata(&v)
		tms[i] = v
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

// GetProjects returns all projects of an Organization
func GetProjects(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var projts []common.Project

	err := db.Projects().Find(bson.M{"organization_id": organization.ID}).All(&projts)

	if err != nil {
		log.Errorln("Error while getting Organization Projects:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization Projects"},
		})
	}

	for i, v := range projts {
		metadata.ProjectMetadata(&v)
		projts[i] = v
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

// GetInventories returns all inventories an Organization
func GetInventories(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var invs []ansible.Inventory

	err := db.Inventories().Find(bson.M{"organization_id": organization.ID}).All(&invs)

	if err != nil {
		log.Errorln("Error while getting Organization Inventories:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization Inventories"},
		})
	}

	for i, v := range invs {
		metadata.InventoryMetadata(&v)
		invs[i] = v
	}

	count := len(invs)
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
		Results:  invs[pgi.Skip():pgi.End()],
	})
}

// GetCredentials returns credentials associated with an Organization
func GetCredentials(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var creds []common.Credential

	err := db.Credentials().Find(bson.M{"organization_id": organization.ID}).All(&creds)

	if err != nil {
		log.Errorln("Error while getting Organization Projects:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization Projects"},
		})
	}

	for i, v := range creds {
		hideEncrypted(&v)
		metadata.CredentialMetadata(&v)
		creds[i] = v
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

// ActivityStream returns the activites of the user on Organizations
func ActivityStream(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var activities []common.ActivityOrganization
	var activity common.ActivityOrganization
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": organization.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&activity) {
		metadata.ActivityOrganizationMetadata(&activity)
		metadata.OrganizationMetadata(&activity.Object1)
		//apply metadata only when Object2 is available
		if activity.Object2 != nil {
			metadata.OrganizationMetadata(activity.Object2)
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
