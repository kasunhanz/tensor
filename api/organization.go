package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	cOrganization = "organization"
	cOrganizationID = "organization_id"
)

type OrganizationController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXOrganizationID from Gin Context and retrieves organization data from the collection
// and store organization data under key CTXOrganization in Gin Context
func (ctrl OrganizationController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cOrganizationID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Organization does not exist"})
		return
	}

	var organization common.Organization
	if err := db.Organizations().FindId(bson.ObjectIdHex(objectID)).One(&organization); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Organization does not exist",
			Log: log.Fields{
				"Organization ID": objectID,
				"Error":  err.Error(),
			},
		})
		return
	}

	roles := new(rbac.Organization)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(user, organization) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "DELETE":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(user, organization) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}
	c.Set(cOrganization, organization)
	c.Next()
}

// GetOrganization is a Gin handler function which returns the organization as a JSON object
func (ctrl OrganizationController) One(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)
	metadata.OrganizationMetadata(&organization)
	c.JSON(http.StatusOK, organization)
}

// GetOrganizations is a Gin handler function which returns list of organization
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl OrganizationController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"name", "description"}, match)
	query := db.Organizations().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.Organization)
	var organizations []common.Organization
	iter := query.Iter()
	var tmpOrganization common.Organization
	for iter.Next(&tmpOrganization) {
		if !roles.Read(user, tmpOrganization) {
			continue
		}
		metadata.OrganizationMetadata(&tmpOrganization)
		organizations = append(organizations, tmpOrganization)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting Organization",
			Log:     log.Fields{"Error": err.Error()},
		})
		return
	}

	count := len(organizations)
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
		Data:  organizations[pgi.Skip():pgi.End()],
	})
}

// AddOrganization is a Gin handler function which creates a new organization using request payload.
// This accepts Organization model.
func (ctrl OrganizationController) Create(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	// SuperUsers only can create organizations
	if !user.IsSuperUser {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}

	var req common.Organization
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}
	// if the Organization exist in the collection it is not unique
	if !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Organization with this Name already exists.",
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
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while creating Organization",
			Log:     log.Fields{"Error": err.Error()},
		})
		return
	}

	activity.AddOrganizationActivity(common.Create, user, req)
	metadata.OrganizationMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

// RemoveOrganization is a Gin handler function which removes a organization object from the database
func (ctrl OrganizationController) Delete(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)
	user := c.MustGet(cUser).(common.User)

	var projectIDs []bson.ObjectId
	orgIter := db.Projects().Find(bson.M{"organization_id": organization.ID}).Select(bson.M{"_id": 1}).Iter()
	var project common.Project
	for orgIter.Next(&project) {
		projectIDs = append(projectIDs, project.ID)
	}
	if err := orgIter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing Projects",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	var invIDs []bson.ObjectId
	var inventory ansible.Inventory
	invIter := db.Inventories().Find(bson.M{"organization_id": organization.ID}).Select(bson.M{"_id": 1}).Iter()
	for invIter.Next(&inventory) {
		invIDs = append(invIDs, inventory.ID)
	}
	if err := orgIter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing projects",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	// remove all jobs for the project
	if _, err := db.Jobs().RemoveAll(bson.M{"project_id": bson.M{"$in": projectIDs}}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing jobs",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	// remove all job templates
	if _, err := db.JobTemplates().RemoveAll(bson.M{"project_id": bson.M{"$in": projectIDs}}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing job tempaltes",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	// remove all jobs for the project
	if _, err := db.TerrafromJobs().RemoveAll(bson.M{"project_id": bson.M{"$in": projectIDs}}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing terraform jobs",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	// remove all job templates
	if _, err := db.TerrafromJobTemplates().RemoveAll(bson.M{"project_id": bson.M{"$in": projectIDs}}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing terraform job templates",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	// remove the project as well
	if _, err := db.Projects().RemoveAll(bson.M{"organization_id": organization.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing projects",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.Hosts().RemoveAll(bson.M{"inventory_id": bson.M{"$in": invIDs}}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing hosts",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.Groups().RemoveAll(bson.M{"inventory_id": bson.M{"$in": invIDs}}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing groups",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.Inventories().RemoveAll(bson.M{"organization_id": organization.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing inventories",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.Teams().RemoveAll(bson.M{"organization_id": organization.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing terraform teams",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.Credentials().RemoveAll(bson.M{"organization_id": organization.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing terraform credentials",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	// remove the organization as well
	if err := db.Organizations().RemoveId(organization.ID); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing organization",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddOrganizationActivity(common.Delete, user, organization)
	c.AbortWithStatus(http.StatusNoContent)
}

// UpdateOrganization is a Gin handler function which updates a organization using request payload.
// This replaces all the fields in the database, empty "" fields and
// unspecified fields will be removed from the database object.
func (ctrl OrganizationController) Update(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)
	user := c.MustGet(cUser).(common.User)
	tmpOrg := organization

	var req common.Organization
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// if the Organization exist in the collection it is not unique
	if req.Name != organization.Name && !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Organization with this Name already exists.",
		})
		return
	}

	// trim strings white space
	organization.Name = strings.Trim(req.Name, " ")
	organization.Description = strings.Trim(req.Description, " ")
	organization.Modified = time.Now()
	organization.ModifiedByID = user.ID

	if err := db.Organizations().UpdateId(organization.ID, organization); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating Organization",
			Log:     log.Fields{"Organization ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddOrganizationActivity(common.Update, user, tmpOrg, organization)
	metadata.OrganizationMetadata(&organization)
	c.JSON(http.StatusOK, organization)
}

// GetUsers Returns all Organization users
func (ctrl OrganizationController) GetUsers(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)

	var users []common.User
	if err := db.Users().Find(bson.M{"organization_id": organization.ID}).All(&users); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting Organization users",
			Log:     log.Fields{"Error": err.Error()},
		})
		return
	}

	for i, v := range users {
		metadata.UserMetadata(&v)
		users[i] = v
	}

	count := len(users)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
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
		Data:  users[pgi.Skip():pgi.End()],
	})
}

// GetAdmins returns an Organization admins
func (ctrl OrganizationController) GetAdmins(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)

	var users []common.User
	for _, v := range organization.Roles {
		if v.Type == "user" && v.Role == "admin" {
			var user common.User
			err := db.Users().FindId(v.GranteeID).One(&user)
			if err != nil {
				log.WithFields(log.Fields{
					"Organization ID": organization.ID,
					"Error":           err.Error(),
				}).Warnln("Error while getting owner users for organization")
				continue
			}
			metadata.UserMetadata(&user)
			users = append(users, user)
		}
		if v.Type == "team" && v.Role == "admin" {
			var team common.Team
			err := db.Teams().FindId(v.GranteeID).One(&team)
			if err != nil {
				log.WithFields(log.Fields{
					"Organization ID": organization.ID,
					"Error":           err.Error(),
				}).Warningln("Error while getting team for organization role")
				continue
			}
			for _, v := range team.Roles {
				var user common.User
				if v.Type == "user" {
					if err := db.Users().FindId(v.GranteeID).One(&user); err != nil {
						log.WithFields(log.Fields{
							"Organization ID": organization.ID,
							"Error":           err.Error(),
						}).Warningln("Error while getting owner users for organization")
						continue
					}
				}
				metadata.UserMetadata(&user)
				users = append(users, user)
			}
		}
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

// GetTeams will return an Organization Teams
func (ctrl OrganizationController) GetTeams(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)
	var teams []common.Team
	if err := db.Teams().Find(bson.M{"organization_id": organization.ID}).All(&teams); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting organization teams",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	for i, v := range teams {
		metadata.TeamMetadata(&v)
		teams[i] = v
	}

	count := len(teams)
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
		Data:  teams[pgi.Skip():pgi.End()],
	})
}

// GetProjects returns all projects of an Organization
func (ctrl OrganizationController) GetProjects(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)
	var projects []common.Project
	if err := db.Projects().Find(bson.M{"organization_id": organization.ID}).All(&projects); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting organization projects",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	for i, v := range projects {
		metadata.ProjectMetadata(&v)
		projects[i] = v
	}

	count := len(projects)
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
		Data:  projects[pgi.Skip():pgi.End()],
	})
}

// GetInventories returns all inventories an Organization
func (ctrl OrganizationController) GetInventories(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)
	var inventories []ansible.Inventory
	if err := db.Inventories().Find(bson.M{"organization_id": organization.ID}).All(&inventories); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting organization inventories",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	for i, v := range inventories {
		metadata.InventoryMetadata(&v)
		inventories[i] = v
	}

	count := len(inventories)
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
		Data:  inventories[pgi.Skip():pgi.End()],
	})
}

// GetCredentials returns credentials associated with an Organization
func (ctrl OrganizationController) GetCredentials(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)

	iter := db.Credentials().Find(bson.M{"organization_id": organization.ID}).Iter()
	var credentials []*common.Credential
	var credential *common.Credential
	for iter.Next(credential) {
		hideEncrypted(credential)
		metadata.CredentialMetadata(credential)
		credentials = append(credentials, credential)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting organization projects",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  credentials[pgi.Skip():pgi.End()],
	})
}

// ActivityStream returns the activities of the user on Organizations
func (ctrl OrganizationController) ActivityStream(c *gin.Context) {
	organization := c.MustGet(cOrganization).(common.Organization)

	var activities []common.ActivityOrganization
	var act common.ActivityOrganization
	iter := db.ActivityStream().Find(bson.M{"object1._id": organization.ID}).Iter()
	for iter.Next(&act) {
		metadata.ActivityOrganizationMetadata(&act)
		metadata.OrganizationMetadata(&act.Object1)
		if act.Object2 != nil {
			metadata.OrganizationMetadata(act.Object2)
		}
		activities = append(activities, act)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting activities",
			Log:     log.Fields{"Organization ID": organization.ID.Hex(), "Error": err.Error()},
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  activities[pgi.Skip():pgi.End()],
	})
}
