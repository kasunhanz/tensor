package teams

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
)

const _CTX_TEAM = "team"
// _CTX_USER is the key name of the User in gin.Context
const _CTX_USER = "user"
const _CTX_TEAM_ID = "team_id"

// TeamMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// this set the team data under key _CTX_TEAM in gin.Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(_CTX_TEAM, c)

	if err != nil {
		log.Print("Error while getting the Team:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
	}

	var team models.Team
	err = db.Teams().FindId(bson.ObjectIdHex(ID)).One(&team);
	if err != nil {
		log.Print("Error while getting the Team:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
		return
	}

	c.Set(_CTX_TEAM, team)
	c.Next()
}

// GetTeam returns the team as a JSON object
func GetTeam(c *gin.Context) {
	team := c.MustGet(_CTX_TEAM).(models.Team)
	metadata.TeamMetadata(&team)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, team)
}


// GetTeams returns a JSON array of teams
func GetTeams(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	if con := parser.IContains([]string{"name", "description", "organization"}); con != nil {
		match = con
	}

	query := db.Teams().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var teams []models.Team
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpTeam models.Team
	// iterate over all and only get valid objects
	for iter.Next(&tmpTeam) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.TeamRead(user, tmpTeam) {
			continue
		}
		if err := metadata.TeamMetadata(&tmpTeam); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Teams",
			})
			return
		}
		// good to go add to list
		teams = append(teams, tmpTeam)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Team data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Team",
		})
		return
	}

	count := len(teams)
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
		Results: teams[pgi.Skip():pgi.End()],
	})
}


// AddTeam creates a new team
func AddTeam(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Team
	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if !helpers.OrganizationExist(req.OrganizationID, c) {
		return
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedBy = user.ID
	req.ModifiedBy = user.ID

	err := db.Teams().Insert(req);
	if err != nil {
		log.Println("Error while creating Team:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Team",
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Group " + req.Name + " created")

	err = metadata.TeamMetadata(&req);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Team",
		})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}


// UpdateTeam will update the Job Template
func UpdateTeam(c *gin.Context) {
	// get Team from the gin.Context
	team := c.MustGet(_CTX_TEAM).(models.Team)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Team
	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization exist or not
	if !helpers.OrganizationExist(req.OrganizationID, c) {
		return
	}

	req.Created = team.Created
	req.Modified = time.Now()
	req.CreatedBy = team.CreatedBy
	req.ModifiedBy = user.ID

	// update object
	if err := db.JobTemplates().UpdateId(team.ID, req); err != nil {
		log.Println("Error while updating Team:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while updating Team",
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Team " + req.Name + " updated")

	// set `related` and `summary` feilds
	if err := metadata.TeamMetadata(&req); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Team",
		})
		return
	}

	// render JSON with 200 status code
	c.JSON(http.StatusOK, req)
}

// RemoveTeam will remove the Team
// from the db.DBC_TEAMS collection
func RemoveTeam(c *gin.Context) {
	// get Team from the gin.Context
	team := c.MustGet(_CTX_TEAM).(models.Team)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	// remove object from the collection
	err := db.Teams().RemoveId(team.ID);
	if err != nil {
		log.Println("Error while removing Team:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Team",
		})
		return
	}

	// add new activity to activity stream
	addActivity(team.ID, user.ID, "Team " + team.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

func Users(c *gin.Context) {
	team := c.MustGet(_CTX_TEAM).(models.Team)

	var usrs []models.User

	for _, v := range team.Roles {
		if v.Type == "user" {
			var user models.User
			err := db.Users().FindId(v.UserID).One(&user)
			if err != nil {
				log.Println("Error while getting owner users for credential", team.ID, err)
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

func Credentials(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)
	team := c.MustGet(_CTX_TEAM).(models.Team)

	var credentials []models.Credential
	// new mongodb iterator
	iter := db.Credentials().Find(bson.M{"roles.type": "team", "roles.team_id": team.ID}).Iter()
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
				Message: "Error while getting Credentials",
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
			Message: "Error while getting Credential",
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

func Projects(c *gin.Context) {
	user := c.MustGet(_CTX_USER).(models.User)
	team := c.MustGet(_CTX_TEAM).(models.Team)

	var projects []models.Project
	// new mongodb iterator
	iter := db.Projects().Find(bson.M{"roles.type": "team", "roles.team_id": team.ID}).Iter()
	// loop through each result and modify for our needs
	var tmpProject models.Project
	// iterate over all and only get valid objects
	for iter.Next(&tmpProject) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.ProjectRead(user, tmpProject) {
			continue
		}
		if err := metadata.ProjectMetadata(&tmpProject); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Projects",
			})
			return
		}
		// good to go add to list
		projects = append(projects, tmpProject)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Projects data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting Projects",
		})
		return
	}

	count := len(projects)
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
		Results: projects[pgi.Skip():pgi.End()],
	})
}

// TODO: not complete
func ActivityStream(c *gin.Context) {
	team := c.MustGet(_CTX_TEAM).(models.Team)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": team.ID, "type": _CTX_TEAM}).All(activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while Activities",
		})
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
