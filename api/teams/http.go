package teams

import (
	"gopkg.in/mgo.v2/bson"
	"time"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/util/pagination"
	"strconv"
)

const _CTX_TEAM = "team"
const _CTX_TEAM_ID = "team_id"

// TeamMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// this set the team data under key _CTX_TEAM in gin.Context
func TeamMiddleware(c *gin.Context) {
	id := c.Params.ByName(_CTX_TEAM_ID)

	dbc := database.MongoDb.C(models.DBC_TEAMS)

	var org models.Team
	if err := dbc.FindId(bson.ObjectIdHex(id)).One(&org); err != nil {
		log.Print(err) // log error to the system log
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set(_CTX_TEAM, org)
	c.Next()
}

// GetTeam returns the team as a JSON object
func GetTeam(c *gin.Context) {
	o := c.MustGet(_CTX_TEAM).(models.Team)
	setMetadata(&o)

	c.JSON(200, o)
}


// GetTeams returns a JSON array of teams
func GetTeams(c *gin.Context) {
	dbc := database.MongoDb.C(models.DBC_TEAMS)

	parser := util.NewQueryParser(c)

	match := bson.M{}

	if con := parser.IContains([]string{"name", "description", "organization"}); con != nil {
		match = con
	}

	query := dbc.Find(match)

	count, err := query.Count();
	if err != nil {
		log.Println("Unable to count teams from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	pgi := pagination.NewPagination(c, count)

	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page) + ": That page contains no results."})
		return
	}

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var teams []models.Team

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&teams); err != nil {
		log.Println("Unable to retrive teams from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, v := range teams {
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		teams[i] = v
	}

	c.JSON(200, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": teams, })

}


// AddTeam creates a new team
func AddTeam(c *gin.Context) {
	var request models.Team

	u := c.MustGet("user").(models.User)

	if err := c.Bind(&request); err != nil {
		log.Println("Failed to parse payload", err)
		c.JSON(http.StatusBadRequest,
			gin.H{"status": "Bad Request", "message": "Failed to parse payload"})
		return
	}

	tm := models.Team{
		ID:bson.NewObjectId(),
		Name:request.Name,
		Description:request.Description,
		Organization: request.Organization,
		Created:time.Now(),
		Modified:time.Now(),
		CreatedBy: u.ID,
		ModifiedBy: u.ID,
	}

	dbc := database.MongoDb.C(models.DBC_TEAMS)
	dbacl := database.MongoDb.C(models.DBC_ACl)

	if err := dbc.Insert(tm); err != nil {
		log.Println("Failed to create Team", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Team"})
		return
	}

	if err := dbacl.Insert(models.ACL{ID:bson.NewObjectId(), Object:tm.ID, Type:"user", UserID:u.ID, Role: "admin"}); err != nil {
		log.Println("Failed to create acl", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create acl"})

		if err := dbc.RemoveId(tm.ID); err != nil {
			log.Println("Failed to remove Team", err)
		}

		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_TEAM,
		ObjectID:    tm.ID,
		Description: "Team " + tm.Name + " created",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	if err := setMetadata(&tm); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	c.JSON(http.StatusCreated, tm)
}

func UpdateTeam(c *gin.Context) {
	oldTemplate := c.MustGet(models.DBC_TEAMS).(models.Team)

	dbc := database.MongoDb.C(models.DBC_TEAMS)
	var tm models.Team
	if err := c.Bind(&tm); err != nil {
		return
	}

	tm.ID = oldTemplate.ID

	if err := dbc.UpdateId(tm.ID, tm); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   oldTemplate.ID,
		Description: "Template ID " + tm.Name + " updated",
		ObjectID:    oldTemplate.ID,
		ObjectType:  "template",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveTeam(c *gin.Context) {
	crd := c.MustGet(_CTX_TEAM).(models.Team)

	dbc := database.MongoDb.C(models.DBC_TEAMS)

	if err := dbc.RemoveId(crd.ID); err != nil {
		panic(err)
	}

	if err := (models.Event{
		Description: "Team " + crd.Name + " deleted",
		ObjectID:    crd.ID,
		ObjectType:  _CTX_TEAM,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}