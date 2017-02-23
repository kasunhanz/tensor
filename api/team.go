package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	cTeam = "team"
	cTeamID = "team_id"
)

type TeamController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXTeamID from Gin Context and retrieves team data from the collection
// and store team data under key CTXTeam in Gin Context
func (ctrl TeamController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cTeamID)
	user := c.MustGet(cUser).(common.User)
	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Team does not exist"})
		return
	}

	var team common.Team
	if err := db.Teams().FindId(bson.ObjectIdHex(objectID)).One(&team); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Team does not exist",
			Log: log.Fields{
				"Team ID": objectID,
				"Error":  err.Error(),
			},
		})
		return
	}

	roles := new(rbac.Team)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.Read(user, team) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "DELETE", "PATCH":
		{
			// Reject the request if the user doesn't have write permissions
			if !roles.Write(user, team) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}

	c.Set(cTeam, team)
	c.Next()
}

// GetTeam is a Gin handler function which returns the team as a JSON object
func (ctrl TeamController) One(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)
	metadata.TeamMetadata(&team)
	c.JSON(http.StatusOK, team)
}

// GetTeams is a Gin handler function which returns list of teams
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl TeamController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Lookups([]string{"name", "description", "organization"}, match)
	query := db.Teams().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.Team)
	var teams []common.Team
	iter := query.Iter()
	var tmpTeam common.Team
	for iter.Next(&tmpTeam) {
		if !roles.Read(user, tmpTeam) {
			continue
		}
		metadata.TeamMetadata(&tmpTeam)
		teams = append(teams, tmpTeam)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting teams",
			Log:     log.Fields{"Error": err.Error()},
		})
		return
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

// AddTeam creates a new team
func (ctrl TeamController) Create(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)

	var req common.Team
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	if !req.OrganizationExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Organization does not exists.",
		})
		return
	}

	if !new(rbac.Organization).WriteByID(user, req.OrganizationID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
	}

	if !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Team with this name and organization already exists.",
		})
		return
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	if err := db.Teams().Insert(req); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while creating team",
			Log:     log.Fields{"Error": err.Error()},
		})
		return
	}

	roles := new(rbac.Team)
	if !rbac.HasGlobalWrite(user) && !rbac.IsOrganizationAdmin(req.OrganizationID, user.ID) {
		roles.Associate(req.ID, user.ID, rbac.RoleTypeUser, rbac.TeamAdmin);
	}

	activity.AddTeamActivity(common.Create, user, req)
	metadata.TeamMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

// UpdateTeam will update the Job Template
func (ctrl TeamController) Update(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)
	tmpTeam := team
	user := c.MustGet(cUser).(common.User)

	var req common.Team
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	if !req.OrganizationExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Organization does not exists.",
		})
		return
	}

	if !new(rbac.Organization).WriteByID(user, req.OrganizationID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
	}

	if req.Name != team.Name && !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Team with this name and organization already exists.",
		})
		return
	}

	team.Name = strings.Trim(req.Name, " ")
	team.Description = strings.Trim(req.Description, " ")
	team.OrganizationID = req.OrganizationID
	team.Modified = time.Now()
	team.ModifiedByID = user.ID
	if err := db.Teams().UpdateId(team.ID, team); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating team.",
			Log:     log.Fields{"Host ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddTeamActivity(common.Update, user, tmpTeam, team)
	metadata.TeamMetadata(&team)
	c.JSON(http.StatusOK, team)
}

// PatchTeam is a Gin handler function which partially updates a team using request payload.
// This replaces specified fields in the data, empty "" fields will be
// removed from the database object. Unspecified fields will be ignored.
func (ctrl TeamController) Patch(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)
	tmpTeam := team
	user := c.MustGet(cUser).(common.User)
	var req common.PatchTeam
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	if req.OrganizationID != nil {
		team.OrganizationID = *req.OrganizationID
		if !team.OrganizationExist() {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Organization does not exists.",
			})
			return
		}

		if !new(rbac.Organization).WriteByID(user, team.OrganizationID) {
			AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
				Message: "You don't have sufficient permissions to perform this action.",
			})
		}
	}

	if req.Name != nil && *req.Name != team.Name {
		team.Name = strings.Trim(*req.Name, " ")

		if req.OrganizationID != nil {
			team.OrganizationID = *req.OrganizationID
		}
		// if the team exist in the collection it is not unique
		if !team.IsUnique() {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Team with this name and organization already exists.",
			})
			return
		}
	}

	if req.Description != nil {
		team.Description = strings.Trim(*req.Description, " ")
	}
	if req.OrganizationID != nil {
		team.OrganizationID = *req.OrganizationID
	}
	team.Modified = time.Now()
	team.ModifiedByID = user.ID

	if err := db.Teams().UpdateId(team.ID, team); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating team",
			Log:     log.Fields{"Team ID": team.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddTeamActivity(common.Update, user, tmpTeam, team)
	metadata.TeamMetadata(&team)
	c.JSON(http.StatusOK, team)
}

// RemoveTeam is a Gin handler function which removes a team object from the database
func (ctrl TeamController) Delete(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)
	user := c.MustGet(cUser).(common.User)

	// Remove permissions
	access := bson.M{"$pull": bson.M{"roles": common.AccessControl{GranteeID: team.ID}}}
	if _, err := db.Projects().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing team",
			Log:     log.Fields{"Team ID": team.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	if _, err := db.Credentials().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing team",
			Log:     log.Fields{"Team ID": team.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	if _, err := db.Inventories().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing team",
			Log:     log.Fields{"Team ID": team.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	if _, err := db.JobTemplates().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing team",
			Log:     log.Fields{"Team ID": team.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	if _, err := db.TerrafromJobTemplates().UpdateAll(nil, access); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing team",
			Log:     log.Fields{"Team ID": team.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if err := db.Teams().RemoveId(team.ID); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing team",
			Log:     log.Fields{"Team ID": team.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddTeamActivity(common.Delete, user, team)
	c.AbortWithStatus(http.StatusNoContent)
}

// Users is a Gin handler function which returns users associated with a team
func (ctrl TeamController) Users(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)

	var usrs []common.User

	for _, v := range team.Roles {
		if v.Type == "user" {
			var user common.User
			err := db.Users().FindId(v.GranteeID).One(&user)
			if err != nil {
				log.WithFields(log.Fields{
					"Team ID": team.ID,
					"Error":   err.Error(),
				}).Errorln("Error while getting owner users for credential")
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
		}).Debugln("User page does not exist")
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
	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  usrs[pgi.Skip():pgi.End()],
	})
}

// Credentials is Gin handler function which returns credentials associated with a team
func (ctrl TeamController) Credentials(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)
	user := c.MustGet(cUser).(common.User)

	var credentials []common.Credential
	// new mongodb iterator
	iter := db.Credentials().Find(bson.M{"roles.type": "team", "roles.team_id": team.ID}).Iter()

	roles := new(rbac.Credential)
	// loop through each result and modify for our needs
	var tmpCred common.Credential
	// iterate over all and only get valid objects
	for iter.Next(&tmpCred) {
		// Skip if the user doesn't have read permission
		if !roles.Read(user, tmpCred) {
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
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Credential"},
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  credentials[pgi.Skip():pgi.End()],
	})
}

// Projects is a Gin handler function which returns projects associated with a team
func (ctrl TeamController) Projects(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)
	user := c.MustGet(cUser).(common.User)

	var projects []common.Project
	// new mongodb iterator
	iter := db.Projects().Find(bson.M{"roles.type": "team", "roles.team_id": team.ID}).Iter()

	roles := new(rbac.Project)
	// loop through each result and modify for our needs
	var tmpProject common.Project
	// iterate over all and only get valid objects
	for iter.Next(&tmpProject) {
		// Skip if the user doesn't have read permission
		if !roles.Read(user, tmpProject) {
			continue
		}
		// skip to next
		metadata.ProjectMetadata(&tmpProject)
		// good to go add to list
		projects = append(projects, tmpProject)
	}
	if err := iter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Projects data from the database")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Projects"},
		})
		return
	}

	count := len(projects)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Project page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:  projects[pgi.Skip():pgi.End()],
	})
}

// ActivityStream returns the activities of the user on Teams
func (ctrl TeamController) ActivityStream(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)

	var activities []common.ActivityTeam
	var act common.ActivityTeam
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": team.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&act) {
		metadata.ActivityTeamMetadata(&act)
		metadata.TeamMetadata(&act.Object1)
		//apply metadata only when Object2 is available
		if act.Object2 != nil {
			metadata.TeamMetadata(act.Object2)
		}
		//add to activities list
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
		Data:  activities[pgi.Skip():pgi.End()],
	})
}

func (ctrl TeamController) AssignRole(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)

	var req common.RoleObj
	err := binding.JSON.Bind(c.Request, &req)
	if err != nil {
		// Return 400 if request has bad JSON format
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
			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			}
		}

	case "organization":
		{
			c.JSON(http.StatusInternalServerError, common.Error{
				Code:   http.StatusInternalServerError,
				Errors: []string{"You cannot assign an Organization role as a child role for a Team."},
			})
			return
		}

	case "team":
		{
			roles := new(rbac.Team)
			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			}
		}

	case "project":
		{
			roles := new(rbac.Project)
			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			}
		}

	case "job_template":
		{
			roles := new(rbac.JobTemplate)
			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			}
		}

	case "terraform_job_template":
		{
			roles := new(rbac.TerraformJobTemplate)
			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			}
		}

	case "inventory":
		{
			roles := new(rbac.Inventory)
			if req.Disassociate {
				err = roles.Disassociate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			} else {
				err = roles.Associate(req.ResourceID, team.ID, rbac.RoleTypeTeam, req.Role)
			}
		}
	}

	if err != nil {
		log.WithFields(log.Fields{
			"Resource ID": team.ID.Hex(),
			"User ID":     team.ID.Hex(),
			"Error":       err.Error(),
		}).Errorln("Error occured while modifying the role")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error occured while adding role"},
		})
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}

func (ctrl TeamController) GetRoles(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, common.Error{
		Code:   http.StatusNotImplemented,
		Errors: []string{"Not implemented"},
	})
}
