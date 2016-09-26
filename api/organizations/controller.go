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
)

const _CTX_ORGANIZATION = "organization"
const _CTX_ORGANIZATION_ID = "organization_id"
const _CTX_USER = "user"

// OrganizationMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func Middleware(c *gin.Context) {

	projectID := c.Params.ByName(_CTX_ORGANIZATION_ID)
	collection := db.C(db.ORGANIZATIONS)
	var organization models.Organization
	err := collection.FindId(bson.ObjectIdHex(projectID)).One(&organization);

	if err != nil {
		log.Print("Error while getting the Organization:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
	}

	user := c.MustGet(_CTX_USER).(models.User)

	// reject the request if the user doesn't have permissions
	if !roles.OrganizationRead(user, organization) {
		c.JSON(http.StatusUnauthorized, models.Error{
			Code: http.StatusUnauthorized,
			Message: "Unauthorized",
		})
		return
	}

	c.Set(_CTX_ORGANIZATION, organization)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetOrganization(c *gin.Context) {

	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)
	metadata.OrganizationMetadata(&organization)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, organization)
}


// GetOrganizations returns a JSON array of projects
func GetOrganizations(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	dbc := db.C(db.ORGANIZATIONS)
	parser := util.NewQueryParser(c)
	match := bson.M{}
	con := parser.IContains([]string{"name", "description"});

	if con != nil {
		match = con
	}

	query := dbc.Find(match)
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
				Message: "Error while getting Organization",
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
			Message: "Error while getting Organization",
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

// AddOrganization creates a new project
func AddOrganization(c *gin.Context) {

	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Organization
	err := c.BindJSON(&req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	collection := db.C(db.ORGANIZATIONS)
	organization := models.Organization{
		ID: bson.NewObjectId(),
		Created : time.Now(),
		CreatedBy : user.ID,
		Modified : time.Now(),
		ModifiedBy : user.ID,
	}

	err = collection.Insert(organization);
	if err != nil {
		log.Println("Error while creating Organization:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Organization",
		})
		return
	}


	// add new activity to activity stream
	addActivity(organization.ID, user.ID, "Organization " + organization.Name + " created")

	err = metadata.OrganizationMetadata(&organization);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Organization",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, organization)
}

func UpdateOrganization(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Organization
	err := c.BindJSON(&req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	collection := db.C(db.ORGANIZATIONS)

	organization.Name = req.Name
	organization.Description = req.Description

	err = collection.UpdateId(organization.ID, organization);
	if err != nil {
		log.Println("Error while updating Organization:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while updating Organization",
		})
	}

	// add new activity to activity stream
	addActivity(organization.ID, user.ID, "Organization " + organization.Name + " updated")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

func GetUsers(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var usrs []models.User
	collection := db.C(db.USERS)

	err := collection.Find(bson.M{"organization_id": organization.ID}).All(&usrs)

	if err != nil {
		log.Println("Error while getting Organization users:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Organization users",
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

func GetAdmins(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var usrs []models.User
	collection := db.C(db.USERS)
	cTeam := db.C(db.TEAMS)

	for _, v := range organization.Roles {
		// get user with role admin
		if v.Type == "user" && v.Role == "admin" {
			var user models.User
			err := collection.FindId(v.UserID).One(&user)
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
			err := cTeam.FindId(v.TeamID).One(&team)
			if err != nil {
				log.Println("Error while getting team for organization role", organization.ID, err)
				continue // ignore and continue
			}

			for _, v := range team.Roles {
				var user models.User
				if v.Type == "user" {
					err := collection.FindId(v.UserID).One(&user)
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

func GetTeams(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var tms []models.Team
	collection := db.C(db.TEAMS)

	err := collection.Find(bson.M{"organization_id": organization.ID}).All(&tms)

	if err != nil {
		log.Println("Error while getting Organization teams:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Organization teams",
		})
	}

	for i, v := range tms {
		if err := metadata.TeamMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Organization teams",
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

func GetProjects(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var projts []models.Project
	collection := db.C(db.PROJECTS)

	err := collection.Find(bson.M{"organization_id": organization.ID}).All(&projts)

	if err != nil {
		log.Println("Error while getting Organization Projects:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Organization Projects",
		})
	}

	for i, v := range projts {
		if err := metadata.ProjectMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Organization Projects",
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

func GetInventories(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var invs []models.Inventory
	collection := db.C(db.PROJECTS)

	err := collection.Find(bson.M{"organization_id": organization.ID}).All(&invs)

	if err != nil {
		log.Println("Error while getting Organization Inventories:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Organization Inventories",
		})
	}

	for i, v := range invs {
		if err := metadata.InventoryMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Organization Inventories",
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

func GetCredentials(c *gin.Context) {
	organization := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	var creds []models.Credential
	collection := db.C(db.CREDENTIALS)

	err := collection.Find(bson.M{"organization_id": organization.ID}).All(&creds)

	if err != nil {
		log.Println("Error while getting Organization Projects:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Organization Projects",
		})
	}

	for i, v := range creds {
		if err := metadata.CredentialMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Organization Projects",
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
	collection := db.C(db.ACTIVITY_STREAM)
	err := collection.Find(bson.M{"object_id": organizatin.ID, "type": _CTX_ORGANIZATION}).All(&activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while Activities",
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