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
	CTXOrganization = "organization"
	CTXOrganizationID = "organization_id"
)

type OrganizationController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXOrganizationID from Gin Context and retrieves organization data from the collection
// and store organization data under key CTXOrganization in Gin Context
func (ctrl OrganizationController) Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(CTXOrganizationID, c)
	user := c.MustGet(CTXUser).(common.User)

	if err != nil {
		log.WithFields(log.Fields{
			"Organization ID": ID,
			"Error":           err.Error(),
		}).Errorln("Error while getting Organization ID url parameter")
		AbortWithError(c, http.StatusNotFound, "Organization does not exist")
		return
	}

	var organization common.Organization
	if err = db.Organizations().FindId(bson.ObjectIdHex(ID)).One(&organization); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": ID,
			"Error":           err.Error(),
		}).Errorln("Error while retriving Organization form the database")
		AbortWithError(c, http.StatusNotFound, "Organization does not exist")
		return
	}

	roles := new(rbac.Organization)

	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(user, organization) {
				AbortWithError(c, http.StatusUnauthorized, "You don't have sufficient permissions to perform this action.")
				return
			}
		}
	case "PUT", "PATCH", "DELETE":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(user, organization) {
				AbortWithError(c, http.StatusUnauthorized, "You don't have sufficient permissions to perform this action.")
				return
			}
		}
	}

	c.Set(CTXOrganization, organization)
	c.Next()
}

// GetOrganization is a Gin handler function which returns the organization as a JSON object
func (ctrl OrganizationController) One(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	metadata.OrganizationMetadata(&organization)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, organization)
}

// GetOrganizations is a Gin handler function which returns list of organization
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl OrganizationController) All(c *gin.Context) {
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

	roles := new(rbac.Organization)
	var organizations []common.Organization
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpOrganization common.Organization
	// iterate over all and only get valid objects
	for iter.Next(&tmpOrganization) {
		// Skip if the user doesn't have read permission
		if !roles.Read(user, tmpOrganization) {
			continue
		}
		// skip to next
		metadata.OrganizationMetadata(&tmpOrganization)
		// good to go add to list
		organizations = append(organizations, tmpOrganization)
	}
	if err := iter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Organization data from the database")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while getting Organization")
		return
	}

	count := len(organizations)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		AbortWithError(c, http.StatusNotFound, "#" + strconv.Itoa(pgi.Page()) + " page contains no results.")
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  organizations[pgi.Skip():pgi.End()],
	})
}

// AddOrganization is a Gin handler function which creates a new organization using request payload.
// This accepts Organization model.
func (ctrl OrganizationController) Create(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	// SuperUsers only can create organizations
	if !user.IsSuperUser {
		c.JSON(http.StatusUnauthorized, common.Error{
			Code:   http.StatusUnauthorized,
			Errors: []string{"Unauthorized"},
		})
		c.Abort()
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
		AbortWithError(c, http.StatusBadRequest, "Organization with this Name already exists.")
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
		AbortWithError(c, http.StatusGatewayTimeout, "Error while creating Organization")
		return
	}

	// add new activity to activity stream
	activity.AddOrganizationActivity(common.Create, user, req)

	metadata.OrganizationMetadata(&req)
	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// RemoveOrganization is a Gin handler function which removes a organization object from the database
func (ctrl OrganizationController) Delete(c *gin.Context) {
	// get Organization from the gin.Context
	organization := c.MustGet(CTXOrganization).(common.Organization)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var projectIDs []bson.ObjectId
	orgIter := db.Projects().Find(bson.M{"organization_id": organization.ID, }).Select(bson.M{"_id":1}).Iter()
	var project common.Project
	for orgIter.Next(&project) {
		projectIDs = append(projectIDs, project.ID)
	}
	if err := orgIter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while removing Projects")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Projects")
		return
	}

	var invIDs []bson.ObjectId
	var inventory ansible.Inventory
	invIter := db.Inventories().Find(bson.M{"organization_id": organization.ID}).Select(bson.M{"_id":1}).Iter()
	for invIter.Next(&inventory) {
		invIDs = append(invIDs, inventory.ID)
	}
	if err := orgIter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while removing Projects")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Projects")
		return
	}

	// remove all jobs for the project
	if _, err := db.Jobs().RemoveAll(bson.M{"project_id": bson.M{"$in": projectIDs}}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while deleting Project Jobs")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Project Jobs")
		return
	}

	// remove all job templates
	if _, err := db.JobTemplates().RemoveAll(bson.M{"project_id": bson.M{"$in": projectIDs}}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while deleting Project Job Templates")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Project Job Templates")
		return
	}

	// remove all jobs for the project
	if _, err := db.TerrafromJobs().RemoveAll(bson.M{"project_id": bson.M{"$in": projectIDs}}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while deleting Project Jobs")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Project Jobs")
		return
	}

	// remove all job templates
	if _, err := db.TerrafromJobTemplates().RemoveAll(bson.M{"project_id": bson.M{"$in": projectIDs}}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while deleting Project Job Templates")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Project Job Templates")
		return
	}
	// remove the project as well
	if _, err := db.Projects().RemoveAll(bson.M{"organization_id": organization.ID}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while deleting Project")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Project")
		return
	}

	if _, err := db.Hosts().RemoveAll(bson.M{"inventory_id": bson.M{"$in": invIDs}}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while deleting Project Jobs")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing inventory hosts")
		return
	}

	if _, err := db.Groups().RemoveAll(bson.M{"inventory_id": bson.M{"$in": invIDs}}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":      err.Error(),
		}).Errorln("Error while deleting Project Job Templates")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing inventory groups")
		return
	}

	if _, err := db.Inventories().RemoveAll(bson.M{"organization_id": organization.ID}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":           err.Error(),
		}).Errorln("Error while removing Inventories")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Inventories")
		return
	}

	if _, err := db.Teams().RemoveAll(bson.M{"organization_id": organization.ID}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":           err.Error(),
		}).Errorln("Error while removing Teams")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Teams")
		return
	}

	if _, err := db.Credentials().RemoveAll(bson.M{"organization_id": organization.ID}); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":           err.Error(),
		}).Errorln("Error while removing Credentials")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Credentials")
		return
	}

	// remove the organization as well
	if err := db.Organizations().RemoveId(organization.ID); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": organization.ID.Hex(),
			"Error":           err.Error(),
		}).Errorln("Error while removing Organization")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while removing Organization")
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
func (ctrl OrganizationController) Update(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)
	tmpOrg := organization
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req common.Organization
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// if the Organization exist in the collection it is not unique
	if req.Name != organization.Name && !req.IsUnique() {
		AbortWithError(c, http.StatusBadRequest, "Organization with this Name already exists.")
		return
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
		AbortWithError(c, http.StatusGatewayTimeout, "Error while updating Organization")
		return
	}

	// add new activity to activity stream
	activity.AddOrganizationActivity(common.Update, user, tmpOrg, organization)

	metadata.OrganizationMetadata(&organization)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, organization)
}

// PatchOrganization is a Gin handler function which partially updates a organization using request payload.
// This replaces specified fields in the data, empty "" fields will be
// removed from the database object. Unspecified fields will be ignored.
func (ctrl OrganizationController) Patch(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)
	tmpOrg := organization
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req common.PatchOrganization
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// since this is a patch request if the name specified check the
	// Organization name is unique
	if req.Name != nil && *req.Name != organization.Name {
		organization.Name = strings.Trim(*req.Name, " ")

		// if the Organization exist in the collection it is not unique
		if !organization.IsUnique() {
			AbortWithError(c, http.StatusBadRequest, "Organization with this Name already exists.")
			return
		}
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
		AbortWithError(c, http.StatusGatewayTimeout, "Error while updating Organization")
		return
	}

	// add new activity to activity stream
	activity.AddOrganizationActivity(common.Update, user, tmpOrg, organization)

	metadata.OrganizationMetadata(&organization)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, organization)
}

// GetUsers Returns all Organization users
func (ctrl OrganizationController) GetUsers(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var usrs []common.User

	err := db.Users().Find(bson.M{"organization_id": organization.ID}).All(&usrs)

	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while getting Organization users")
		AbortWithError(c, http.StatusGatewayTimeout, "Error while getting Organization users")
	}

	for i, v := range usrs {
		metadata.UserMetadata(&v)
		usrs[i] = v
	}

	count := len(usrs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		AbortWithError(c, http.StatusNotFound, "#" + strconv.Itoa(pgi.Page()) + " page contains no results.")
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
func (ctrl OrganizationController) GetAdmins(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var usrs []common.User

	for _, v := range organization.Roles {
		// get user with role admin
		if v.Type == "user" && v.Role == "admin" {
			var user common.User
			err := db.Users().FindId(v.GranteeID).One(&user)
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
			err := db.Teams().FindId(v.GranteeID).One(&team)
			if err != nil {
				log.Errorln("Error while getting team for organization role", organization.ID, err)
				continue // ignore and continue
			}

			for _, v := range team.Roles {
				var user common.User
				if v.Type == "user" {
					err := db.Users().FindId(v.GranteeID).One(&user)
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
		AbortWithError(c, http.StatusNotFound, "#" + strconv.Itoa(pgi.Page()) + " page contains no results.")
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
func (ctrl OrganizationController) GetTeams(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var tms []common.Team
	if err := db.Teams().Find(bson.M{"organization_id": organization.ID}).All(&tms); err != nil {
		log.Errorln("Error while getting Organization teams:", err)
		AbortWithError(c, http.StatusGatewayTimeout, "Error while getting Organization teams")
		return
	}

	for i, v := range tms {
		metadata.TeamMetadata(&v)
		tms[i] = v
	}

	count := len(tms)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		AbortWithError(c, http.StatusNotFound, "#" + strconv.Itoa(pgi.Page()) + " page contains no results.")
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
func (ctrl OrganizationController) GetProjects(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var projts []common.Project

	if err := db.Projects().Find(bson.M{"organization_id": organization.ID}).All(&projts); err != nil {
		log.Errorln("Error while getting Organization Projects:", err)
		AbortWithError(c, http.StatusGatewayTimeout, "Error while getting Organization Projects")
		return
	}

	for i, v := range projts {
		metadata.ProjectMetadata(&v)
		projts[i] = v
	}

	count := len(projts)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		AbortWithError(c, http.StatusNotFound, "#" + strconv.Itoa(pgi.Page()) + " page contains no results.")
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
func (ctrl OrganizationController) GetInventories(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var invs []ansible.Inventory

	if err := db.Inventories().Find(bson.M{"organization_id": organization.ID}).All(&invs); err != nil {
		log.Errorln("Error while getting Organization Inventories:", err)
		AbortWithError(c, http.StatusGatewayTimeout, "Error while getting Organization Inventories")
		return
	}

	for i, v := range invs {
		metadata.InventoryMetadata(&v)
		invs[i] = v
	}

	count := len(invs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		AbortWithError(c, http.StatusNotFound, "#" + strconv.Itoa(pgi.Page()) + " page contains no results.")
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
func (ctrl OrganizationController) GetCredentials(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var creds []common.Credential

	if err := db.Credentials().Find(bson.M{"organization_id": organization.ID}).All(&creds); err != nil {
		log.Errorln("Error while getting Organization Projects:", err)
		AbortWithError(c, http.StatusGatewayTimeout, "Error while getting Organization Projects")
		return
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
		AbortWithError(c, http.StatusNotFound, "#" + strconv.Itoa(pgi.Page()) + " page contains no results.")
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

// ActivityStream returns the activities of the user on Organizations
func (ctrl OrganizationController) ActivityStream(c *gin.Context) {
	organization := c.MustGet(CTXOrganization).(common.Organization)

	var activities []common.ActivityOrganization
	var act common.ActivityOrganization
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": organization.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&act) {
		metadata.ActivityOrganizationMetadata(&act)
		metadata.OrganizationMetadata(&act.Object1)
		//apply metadata only when Object2 is available
		if act.Object2 != nil {
			metadata.OrganizationMetadata(act.Object2)
		}
		//add to activities list
		activities = append(activities, act)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		AbortWithError(c, http.StatusGatewayTimeout, "Error while getting Activities")
		return
	}

	count := len(activities)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		AbortWithError(c, http.StatusNotFound, "#" + strconv.Itoa(pgi.Page()) + " page contains no results.")
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
