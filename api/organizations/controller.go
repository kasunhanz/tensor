package organizations

import (
	"gopkg.in/mgo.v2/bson"
	"time"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/api/users"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/util/pagination"
	"strconv"
)

const _CTX_ORGANIZATION = "organization"
const _CTX_ORGANIZATION_ID = "organization_id"

// OrganizationMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func OrganizationMiddleware(c *gin.Context) {
	projectID := c.Params.ByName(_CTX_ORGANIZATION_ID)

	dbc := database.MongoDb.C(models.DBC_ORGANIZATIONS)

	var org models.Organization
	if err := dbc.FindId(bson.ObjectIdHex(projectID)).One(&org); err != nil {
		log.Print(err) // log error to the system log
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set(_CTX_ORGANIZATION, org)
	c.Next()
}

// GetProject returns the project as a JSON object
func GetOrganization(c *gin.Context) {
	o := c.MustGet(_CTX_ORGANIZATION).(models.Organization)
	setMetadata(&o)

	c.JSON(200, o)
}


// GetOrganizations returns a JSON array of projects
func GetOrganizations(c *gin.Context) {
	dbc := database.MongoDb.C(models.DBC_ORGANIZATIONS)

	parser := util.NewQueryParser(c)

	match := bson.M{}

	if con := parser.IContains([]string{"name", "description"}); con != nil {
		match = con
	}

	query := dbc.Find(match)

	count, err := query.Count();
	if err != nil {
		log.Println("Unable to count organizations from the db", err)
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

	var orgs []models.Organization

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&orgs); err != nil {
		log.Println("Unable to retrive organizations from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, v := range orgs {
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		orgs[i] = v
	}

	c.JSON(200, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": orgs, })

}

func AddOrganizationUser(c *gin.Context) {
	// get organization
	org := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	//get the request payload
	var playload struct {
		UserId bson.ObjectId `json:"user_id"`
	}

	if err := c.Bind(&playload); err != nil {
		return
	}

	ou := bson.M{"user_id":playload.UserId}

	col := database.MongoDb.C(models.DBC_ORGANIZATIONS)

	if err := col.UpdateId(org.ID, bson.M{"$addToSet": bson.M{"users": ou}}); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func GetOrganizationUsers(c *gin.Context) {
	// get organization
	org := c.MustGet("organization").(models.Organization)

	col := database.MongoDb.C("organizations")

	aggregate := []bson.M{
		{"$match": bson.M{
			"_id": org.ID,
		}},
		{"$project": bson.M{//performance enhancement I guess
			"_id":0,
			"users":1, }},
		{"$unwind": "$users"},
		{"$lookup": bson.M{
			"from":         "users",
			"localField":   "users.user_id",
			"foreignField": "_id",
			"as":           "user",
		}},
		{
			"$match": bson.M{"user": bson.M{"$ne": []interface{}{} },
			}},
		{"$project": bson.M{
			"_id":0,
			"users":bson.M{"$arrayElemAt": []interface{}{"$user", 0 }},
		}},
		{"$project": bson.M{
			"_id":"$users._id",
			"created":"$users.created",
			"email":"$users.email",
			"name":"$users.name",
			"password":"$users.password",
			"username":"$users.username",
		}},
	}
	var usrs []models.User

	if err := col.Pipe(aggregate).All(&usrs); err != nil {
		panic(err)
	}

	olen := len(usrs)

	resp := make(map[string]interface{})
	resp["count"] = olen
	resp["results"] = usrs

	for i := 0; i < olen; i++ {
		users.SetMetadata(&usrs[i])
	}

	c.JSON(200, usrs)

}

// AddOrganization creates a new project
func AddOrganization(c *gin.Context) {
	var org models.Organization
	user := c.MustGet("user").(models.User)

	if err := c.Bind(&org); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	org.ID = bson.NewObjectId()
	org.Created = time.Now()
	org.Modified = time.Now()
	org.CreatedBy = user.ID
	org.ModifiedBy = user.ID

	if err := org.Insert(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if err := (models.Event{
		ID:          bson.NewObjectId(),
		ProjectID:   org.ID,
		ObjectType:  "organization",
		Description: "Organization Created",
		Created:     org.Created,
	}.Insert()); err != nil {
		// We don't inform client about this error
		// do not ever panic :D
		c.Error(err)
		return
	}

	if err := setMetadata(&org); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	c.JSON(http.StatusCreated, org)
}

func UpdateOrganization(c *gin.Context) {
	req := c.MustGet(_CTX_ORGANIZATION).(models.Organization)

	col := database.MongoDb.C("organizations")
	var org models.Organization
	if err := c.Bind(&org); err != nil {
		return
	}

	org.ID = req.ID

	if err := col.UpdateId(org.ID, org); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   req.ID,
		Description: "Template ID " + org.ID.Hex() + " updated",
		ObjectID:    req.ID,
		ObjectType:  "template",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
