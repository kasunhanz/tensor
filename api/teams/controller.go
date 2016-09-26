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
)

const _CTX_TEAM = "team"
// _CTX_USER is the key name of the User in gin.Context
const _CTX_USER = "user"
const _CTX_TEAM_ID = "team_id"

// TeamMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// this set the team data under key _CTX_TEAM in gin.Context
func Middleware(c *gin.Context) {
	ID := c.Params.ByName(_CTX_TEAM_ID)

	collection := db.C(db.TEAMS)
	var team models.Team
	err := collection.FindId(bson.ObjectIdHex(ID)).One(&team);
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
	dbc := db.C(db.TEAMS)

	parser := util.NewQueryParser(c)
	match := bson.M{}
	if con := parser.IContains([]string{"name", "description", "organization"}); con != nil {
		match = con
	}

	query := dbc.Find(match)
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
	user := c.MustGet("user").(models.User)

	var req models.Team
	if err := c.BindJSON(&req); err != nil {
		log.Println("Failed to parse payload", err)
		c.JSON(http.StatusBadRequest,
			gin.H{"status": "Bad Request", "message": "Failed to parse payload"})
		return
	}

	team := models.Team{
		ID:bson.NewObjectId(),
		Name:req.Name,
		Description:req.Description,
		OrganizationID: req.OrganizationID,
		Created:time.Now(),
		Modified:time.Now(),
		CreatedBy: user.ID,
		ModifiedBy: user.ID,
	}

	dbc := db.C(db.TEAMS)

	err := dbc.Insert(team);
	if err != nil {
		log.Println("Error while creating Team:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Team",
		})
		return
	}

	// add new activity to activity stream
	addActivity(team.ID, user.ID, "Group " + team.Name + " created")

	err = metadata.TeamMetadata(&team);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Team",
		})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusCreated, team)
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
			Message: "Bad Request",
		})
		return
	}

	team.Name = req.Name
	team.Description = req.Description
	team.OrganizationID = req.OrganizationID
	team.Modified = time.Now()
	team.ModifiedBy = user.ID

	collection := db.MongoDb.C(db.JOB_TEMPLATES)

	// update object
	if err := collection.UpdateId(team.ID, team); err != nil {
		log.Println("Error while updating Team:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while updating Team",
		})
		return
	}

	// add new activity to activity stream
	addActivity(team.ID, user.ID, "Team " + team.Name + " updated")

	// set `related` and `summary` feilds
	if err := metadata.TeamMetadata(&team); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Team",
		})
		return
	}

	// render JSON with 200 status code
	c.JSON(http.StatusOK, team)
}

// RemoveTeam will remove the Team
// from the db.DBC_TEAMS collection
func RemoveTeam(c *gin.Context) {
	// get Team from the gin.Context
	team := c.MustGet(_CTX_TEAM).(models.Team)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.MongoDb.C(db.TEAMS)

	// remove object from the collection
	err := collection.RemoveId(team.ID);
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
	collection := db.C(db.USERS)

	for _, v := range team.Roles {
		if v.Type == "user" {
			var user models.User
			err := collection.FindId(v.UserID).One(&user)
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

	collection := db.C(db.CREDENTIALS)

	var credentials []models.Credential
	// new mongodb iterator
	iter := collection.Find(bson.M{"roles.type": "team", "roles.team_id": team.ID}).Iter()
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

	collection := db.C(db.CREDENTIALS)

	var projects []models.Project
	// new mongodb iterator
	iter := collection.Find(bson.M{"roles.type": "team", "roles.team_id": team.ID}).Iter()
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
	collection := db.C(db.ACTIVITY_STREAM)
	err := collection.Find(bson.M{"object_id": team.ID, "type": _CTX_TEAM}).All(activities)

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
