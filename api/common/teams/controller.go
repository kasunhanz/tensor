package teams

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/helpers"
	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	CTXTeam = "team"
	CTXUser = "user"
	CTXTeamID = "team_id"
)

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes CTXTeamID from Gin Context and retrieves team data from the collection
// and store team data under key CTXTeam in Gin Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(CTXTeamID, c)

	if err != nil {
		log.WithFields(log.Fields{
			"Team ID": ID,
			"Error":   err.Error(),
		}).Errorln("Error while getting Team ID url parameter")
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var team common.Team
	err = db.Teams().FindId(bson.ObjectIdHex(ID)).One(&team)
	if err != nil {
		log.WithFields(log.Fields{
			"Team ID": ID,
			"Error":   err.Error(),
		}).Errorln("Error while retriving Team form the database")
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	c.Set(CTXTeam, team)
	c.Next()
}

// GetTeam is a Gin handler function which returns the team as a JSON object
func GetTeam(c *gin.Context) {
	team := c.MustGet(CTXTeam).(common.Team)
	metadata.TeamMetadata(&team)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, team)
}

// GetTeams is a Gin handler function which returns list of teams
// This takes lookup parameters and order parameters to filter and sort output data
func GetTeams(c *gin.Context) {

	parser := util.NewQueryParser(c)

	match := bson.M{}
	match = parser.Lookups([]string{"name", "description", "organization"}, match)

	query := db.Teams().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	log.WithFields(log.Fields{
		"Query": query,
	}).Debugln("Parsed query")

	var teams []common.Team
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpTeam common.Team
	// iterate over all and only get valid objects
	for iter.Next(&tmpTeam) {
		// TODO: if the user doesn't have access to credential
		// skip to next
		metadata.TeamMetadata(&tmpTeam)
		// good to go add to list
		teams = append(teams, tmpTeam)
	}
	if err := iter.Close(); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Error while retriving Team data from the database")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Team"},
		})
		return
	}

	count := len(teams)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		log.WithFields(log.Fields{
			"Page number": pgi.Page(),
		}).Debugln("Team page does not exist")
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  teams[pgi.Skip():pgi.End()],
	})
}

// AddTeam creates a new team
func AddTeam(c *gin.Context) {
	user := c.MustGet(CTXUser).(common.User)

	var req common.Team
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

	// check whether the organization exist or not
	if !helpers.OrganizationExist(req.OrganizationID) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	// if the team exist in the collection it is not unique
	if helpers.IsNotUniqueTeam(req.Name, req.OrganizationID) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Team with this Name and Organization already exists."},
		})
		return
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID

	err := db.Teams().Insert(req)
	if err != nil {
		log.WithFields(log.Fields{
			"Team ID": req.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while creating Team")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while creating Team"},
		})
		return
	}
	// add new activity to activity stream
	activity.AddTeamActivity(common.Create, user, req)

	metadata.TeamMetadata(&req)
	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// UpdateTeam will update the Job Template
func UpdateTeam(c *gin.Context) {
	// get Team from the gin.Context
	team := c.MustGet(CTXTeam).(common.Team)
	tmpTeam := team
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req common.Team
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if !helpers.OrganizationExist(req.OrganizationID) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	if req.Name != team.Name {
		// if the team exist in the collection it is not unique
		if helpers.IsNotUniqueTeam(req.Name, req.OrganizationID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Team with this Name and Organization already exists."},
			})
			return
		}
	}

	team.Name = strings.Trim(req.Name, " ")
	team.Description = strings.Trim(req.Description, " ")
	team.OrganizationID = req.OrganizationID
	team.Modified = time.Now()
	team.ModifiedByID = user.ID

	// update object
	if err := db.Teams().UpdateId(team.ID, team); err != nil {
		log.WithFields(log.Fields{
			"Team ID": team.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while updating Team")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Team"},
		})
		return
	}

	// add new activity to activity stream
	activity.AddTeamActivity(common.Update, user, tmpTeam, team)

	// set `related` and `summary` fields
	metadata.TeamMetadata(&team)

	// render JSON with 200 status code
	c.JSON(http.StatusOK, team)
}

// PatchTeam is a Gin handler function which partially updates a team using request payload.
// This replaces specifed fields in the data, empty "" fields will be
// removed from the database object. Unspecified fields will be ignored.
func PatchTeam(c *gin.Context) {
	// get Team from the gin.Context
	team := c.MustGet(CTXTeam).(common.Team)
	tmpTeam := team
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req common.PatchTeam
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if req.OrganizationID != nil {
		// check whether the organization exist or not
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	if req.Name != nil && *req.Name != team.Name {
		ogID := team.OrganizationID
		if req.OrganizationID != nil {
			ogID = *req.OrganizationID
		}
		// if the team exist in the collection it is not unique
		if helpers.IsNotUniqueTeam(*req.Name, ogID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Team with this Name and Organization already exists."},
			})
			return
		}
	}

	if req.Name != nil {
		team.Name = strings.Trim(*req.Name, " ")
	}

	if req.Description != nil {
		team.Description = strings.Trim(*req.Description, " ")
	}

	if req.OrganizationID != nil {
		team.OrganizationID = *req.OrganizationID
	}

	if req.OrganizationID != nil {
		team.OrganizationID = *req.OrganizationID
	}

	team.Modified = time.Now()
	team.ModifiedByID = user.ID

	// update object
	if err := db.Teams().UpdateId(team.ID, team); err != nil {
		log.WithFields(log.Fields{
			"Team ID": team.ID.Hex(),
			"Error":   err.Error(),
		}).Errorln("Error while updating Team")
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Team"},
		})
		return
	}

	// add new activity to activity stream
	activity.AddTeamActivity(common.Update, user, tmpTeam, team)

	// set `related` and `summary` feilds
	metadata.TeamMetadata(&team)

	// render JSON with 200 status code
	c.JSON(http.StatusOK, team)
}

// RemoveTeam is a Gin handler function which removes a team object from the database
func RemoveTeam(c *gin.Context) {
	// get Team from the gin.Context
	team := c.MustGet(CTXTeam).(common.Team)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	// remove object from the collection
	err := db.Teams().RemoveId(team.ID)
	if err != nil {
		log.Errorln("Error while removing Team:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Team"},
		})
		return
	}
	// add new activity to activity stream
	activity.AddTeamActivity(common.Delete, user, team)

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

// Users is a Gin handler function which returns users associated with a team
func Users(c *gin.Context) {
	team := c.MustGet(CTXTeam).(common.Team)

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
		Results:  usrs[pgi.Skip():pgi.End()],
	})
}

// Credentials is Gin handler function which returns credentials associated with a team
func Credentials(c *gin.Context) {
	team := c.MustGet(CTXTeam).(common.Team)

	var credentials []common.Credential
	// new mongodb iterator
	iter := db.Credentials().Find(bson.M{"roles.type": "team", "roles.team_id": team.ID}).Iter()
	// loop through each result and modify for our needs
	var tmpCred common.Credential
	// iterate over all and only get valid objects
	for iter.Next(&tmpCred) {
		// TODO: if the user doesn't have access to credential
		// skip to next
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  credentials[pgi.Skip():pgi.End()],
	})
}

// Projects is a Gin handler function which returns projects associated with a team
func Projects(c *gin.Context) {
	team := c.MustGet(CTXTeam).(common.Team)

	var projects []common.Project
	// new mongodb iterator
	iter := db.Projects().Find(bson.M{"roles.type": "team", "roles.team_id": team.ID}).Iter()
	// loop through each result and modify for our needs
	var tmpProject common.Project
	// iterate over all and only get valid objects
	for iter.Next(&tmpProject) {
		// TODO: if the user doesn't have access to credential
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
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Projects"},
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
		Results:  projects[pgi.Skip():pgi.End()],
	})
}

// ActivityStream returns the activites of the user on Teams
func ActivityStream(c *gin.Context) {
	team := c.MustGet(CTXTeam).(common.Team)

	var activities []common.ActivityTeam
	var activity common.ActivityTeam
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": team.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&activity) {
		metadata.ActivityTeamMetadata(&activity)
		metadata.TeamMetadata(&activity.Object1)
		//apply metadata only when Object2 is available
		if activity.Object2 != nil {
			metadata.TeamMetadata(activity.Object2)
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

// hideEncrypted is replace encrypted fields by $encrypted$
func hideEncrypted(c *common.Credential) {
	encrypted := "$encrypted$"
	c.Password = encrypted
	c.SSHKeyData = encrypted
	c.SSHKeyUnlock = encrypted
	c.BecomePassword = encrypted
	c.VaultPassword = encrypted
	c.AuthorizePassword = encrypted
}
