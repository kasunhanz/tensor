package organizations

import (
	"gopkg.in/mgo.v2/bson"
	"time"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"bitbucket.pearson.com/apseng/tensor/util"
	"strconv"
	"bitbucket.pearson.com/apseng/tensor/roles"
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
	"bitbucket.pearson.com/apseng/tensor/api/helpers"
	"strings"
	"github.com/gin-gonic/gin/binding"
)

const _CTX_ORGANIZATION = "organization"
const _CTX_ORGANIZATION_ID = "organization_id"
const _CTX_USER = "user"

// Middleware takes _CTX_ORGANIZATION_ID parameter from gin.Context and
// retrieves Organization data from the collection
// it set Organization data under key organization in gin.Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(_CTX_ORGANIZATION_ID, c)

	if err != nil {
		log.Print("Error while getting the Organization:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Organization Not Found"},
		})
		c.Abort()
		return
	}

	var organization models.Organization
	err = db.Organizations().FindId(bson.ObjectIdHex(ID)).One(&organization);

	if err != nil {
		log.Print("Error while getting the Organization:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Organization Not Found"},
		})
		c.Abort()
		return
	}

	user := c.MustGet(_CTX_USER).(models.User)

	// reject the request if the user doesn't have permissions
	if !roles.OrganizationRead(user, organization) {
		c.JSON(http.StatusUnauthorized, models.Error{
			Code: http.StatusUnauthorized,
			Messages: []string{"Unauthorized"},
		})
		c.Abort()
		return
	}

	c.Set(_CTX_ORGANIZATION, organization)
	c.Next()
}

// GetOrganization returns the Organization as a JSON object
func GetOrganization(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	if err := metadata.OrganizationMetadata(&organization); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization Information"},
		})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, organization)
}


// GetOrganizations returns a JSON array of Organizations
func GetOrganizations(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"name", "description"}, match);

	query := db.Organizations().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var organizations []models.Organization
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpOrganization models.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.OrganizationRead(user, tmpOrganization) {
			continue
		}
		if err := metadata.OrganizationMetadata(&tmpOrganization); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Organization"},
			})
			return
		}
		// good to go add to list
		organizations = append(organizations, tmpOrganization)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Organization data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization"},
		})
		return
	}

	count := len(organizations)
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
		Results: organizations[pgi.Skip():pgi.End()],
	})
}

// AddOrganization creates a new Organization
func AddOrganization(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Organization
	err := binding.JSON.Bind(c.Request, &req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// if the Organization exist in the collection it is not unique
	if helpers.IsNotUniqueOrganization(req.Name) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: []string{"Organization with this Name already exists."},
		})
		return
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.CreatedBy = user.ID
	req.Modified = time.Now()
	req.ModifiedBy = user.ID

	err = db.Organizations().Insert(req);
	if err != nil {
		log.Println("Error while creating Organization:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Organization"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Organization " + req.Name + " created")

	err = metadata.OrganizationMetadata(&req);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Organization"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// RemoveOrganization will remove and Organization
// from the db.ORGANIZATIONS collection
func RemoveOrganization(c *gin.Context) {
	// get Organization from the gin.Context
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	// remove all projects
	orgIter := db.Projects().Find(bson.M{"organization_id": organization.ID}).Iter()
	var project models.Project
	for orgIter.Next(&project) {
		// remove all jobs for the project
		changes, err := db.Jobs().RemoveAll(bson.M{"project_id": project.ID})
		if err != nil {
			log.Println("Error while removing Project Jobs:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while removing Project Jobs"},
			})
			return
		}
		log.Println("Jobs remove info:", changes.Removed)

		// remove all job templates
		changes, err = db.JobTemplates().RemoveAll(bson.M{"project_id": project.ID})
		if err != nil {
			log.Println("Error while removing Project Job Templates:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while removing Project Job Templates"},
			})
			return
		}
		log.Println("Job Template remove info:", changes.Removed)

		// remove the project as well
		err = db.Projects().RemoveId(project.ID);
		if err != nil {
			log.Println("Error while removing Project:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while removing Project"},
			})
			return
		}
	}

	// remove all inventories associated with organization
	changes, err := db.Inventories().RemoveAll(bson.M{"organization_id": organization.ID})
	if err != nil {
		log.Println("Error while removing Inventories:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while removing Inventories"},
		})
		return
	}
	log.Println("Inventory remove info:", changes.Removed)


	// remove the organization as well
	err = db.Organizations().RemoveId(organization.ID);
	if err != nil {
		log.Println("Error while removing Organization:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while removing Organization"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(organization.ID, user.ID, "Organization " + organization.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

// UpdateOrganization will update an Organization
func UpdateOrganization(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Organization
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if req.Name != organization.Name {
		// if the Organization exist in the collection it is not unique
		if helpers.IsNotUniqueOrganization(req.Name) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Organization with this Name already exists."},
			})
			return
		}
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	req.ID = organization.ID
	req.Created = organization.Created
	req.CreatedBy = organization.CreatedBy
	req.Modified = time.Now()
	req.ModifiedBy = user.ID

	if err := db.Organizations().UpdateId(organization.ID, req); err != nil {
		log.Println("Error while updating Organization:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Organization"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Organization " + req.Name + " updated")

	if err := metadata.OrganizationMetadata(&req); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Organization"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, req)
}

// PatchOrganization will update selected feilds from an Organization
func PatchOrganization(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.PatchOrganization
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// since this is a patch request if the name specified check the
	// Organization name is unique
	if len(req.Name) > 0 && req.Name != organization.Name {
		// if the Organization exist in the collection it is not unique
		if helpers.IsNotUniqueOrganization(req.Name) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Organization with this Name already exists."},
			})
			return
		}
	}

	// trim strings white space
	req.Name = strings.Trim(req.Name, " ")
	req.Description = strings.Trim(req.Description, " ")

	req.Modified = time.Now()
	req.ModifiedBy = user.ID

	if err := db.Organizations().UpdateId(organization.ID, bson.M{"$set": req}); err != nil {
		log.Println("Error while updating Organization:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Organization"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(organization.ID, user.ID, "Organization " + req.Name + " updated")

	// get newly updated group
	var resp models.Organization
	if err := db.Organizations().FindId(organization.ID).One(&resp); err != nil {
		log.Print("Error while getting the updated Organization:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Error while getting the updated Organization"},
		})
		return
	}

	if err := metadata.OrganizationMetadata(&resp); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Organization"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, resp)
}

// GetUsers Returns all Organization users
func GetUsers(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var usrs []models.User

	err := db.Users().Find(bson.M{"organization_id": organization.ID}).All(&usrs)

	if err != nil {
		log.Println("Error while getting Organization users:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
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

// GetAdmins returns an Organization admins
func GetAdmins(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var usrs []models.User

	for _, v := range organization.Roles {
		// get user with role admin
		if v.Type == "user" && v.Role == "admin" {
			var user models.User
			err := db.Users().FindId(v.UserID).One(&user)
			if err != nil {
				log.Println("Error while getting owner users for organization", organization.ID, err)
				continue //skip iteration
			}
			// set additional info and append to slice
			metadata.UserMetadata(&user)
			usrs = append(usrs, user)
		}
		//get teams with role admin and team users to output slice
		if v.Type == "team" && v.Role == "admin" {
			var team models.Team
			err := db.Teams().FindId(v.TeamID).One(&team)
			if err != nil {
				log.Println("Error while getting team for organization role", organization.ID, err)
				continue // ignore and continue
			}

			for _, v := range team.Roles {
				var user models.User
				if v.Type == "user" {
					err := db.Users().FindId(v.UserID).One(&user)
					if err != nil {
						log.Println("Error while getting owner users for organization", organization.ID, err)
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: usrs[pgi.Skip():pgi.End()],
	})
}

// GetTeams will return an Organization Teams
func GetTeams(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var tms []models.Team
	err := db.Teams().Find(bson.M{"organization_id": organization.ID}).All(&tms)

	if err != nil {
		log.Println("Error while getting Organization teams:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization teams"},
		})
	}

	for i, v := range tms {
		if err := metadata.TeamMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Organization teams"},
			})
			return
		}
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: tms[pgi.Skip():pgi.End()],
	})
}

// GetProjects returns all projects of an Organization
func GetProjects(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var projts []models.Project

	err := db.Projects().Find(bson.M{"organization_id": organization.ID}).All(&projts)

	if err != nil {
		log.Println("Error while getting Organization Projects:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization Projects"},
		})
	}

	for i, v := range projts {
		if err := metadata.ProjectMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Organization Projects"},
			})
			return
		}
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: projts[pgi.Skip():pgi.End()],
	})
}

// GetInventories returns all inventories an Organization
func GetInventories(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var invs []models.Inventory

	err := db.Inventories().Find(bson.M{"organization_id": organization.ID}).All(&invs)

	if err != nil {
		log.Println("Error while getting Organization Inventories:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization Inventories"},
		})
	}

	for i, v := range invs {
		if err := metadata.InventoryMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Organization Inventories"},
			})
			return
		}
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: invs[pgi.Skip():pgi.End()],
	})
}

// GetCredentials returns credentials associated with an Organization
func GetCredentials(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var creds []models.Credential

	err := db.Credentials().Find(bson.M{"organization_id": organization.ID}).All(&creds)

	if err != nil {
		log.Println("Error while getting Organization Projects:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Organization Projects"},
		})
	}

	for i, v := range creds {
		if err := metadata.CredentialMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Organization Projects"},
			})
			return
		}
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: creds[pgi.Skip():pgi.End()],
	})
}


// TODO: not complete
func ActivityStream(c *gin.Context) {
	organizatin := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": organizatin.ID, "type": _CTX_ORGANIZATION}).All(&activities)

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