package inventories

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

const _CTX_HOST = "host"
const _CTX_HOST_ID = "host_id"

// HostMiddleware takes host_id parameter from gin.Context and
// fetches host data from the database
// it set host data under key host in gin.Context
func HostMiddleware(c *gin.Context) {
	ID := c.Params.ByName(_CTX_HOST_ID)

	dbc := database.MongoDb.C(models.DBC_HOSTS)

	var h models.Host
	if err := dbc.FindId(bson.ObjectIdHex(ID)).One(&h); err != nil {
		log.Print(err) // log error to the system log
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set(_CTX_HOST, h)
	c.Next()
}

// GetHost returns the hsot as a JSON object
func GetHost(c *gin.Context) {
	o := c.MustGet(_CTX_HOST).(models.Inventory)
	setMetadata(&o)

	c.JSON(200, o)
}


// GetHosts returns a JSON array of projects
func GetHosts(c *gin.Context) {
	dbc := database.MongoDb.C(models.DBC_HOSTS)

	parser := util.NewQueryParser(c)

	match := parser.Match([]string{"enabled", "has_active_failures", })
	//TODO: has_active_failures `gt` true

	if con := parser.IContains([]string{"name"}); con != nil {
		if match != nil {
			for i, v := range con {
				match[i] = v
			}
		} else {
			match = con
		}
	}

	query := dbc.Find(match)

	count, err := query.Count();
	if err != nil {
		log.Println("Unable to count inventories from the db", err)
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

	var invs []models.Inventory

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&invs); err != nil {
		log.Println("Unable to retrive inventories from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, v := range invs {
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		invs[i] = v
	}

	c.JSON(200, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": invs, })

}

// AddInventory creates a new project
func AddInventory(c *gin.Context) {
	var req models.Inventory
	user := c.MustGet("user").(models.User)

	if err := c.Bind(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	var inv models.Inventory

	inv.ID = bson.NewObjectId()
	inv.Name = req.Name
	inv.Description = req.Description
	inv.Organization = req.Organization
	inv.Variables = req.Variables
	inv.Created = time.Now()
	inv.Modified = time.Now()
	inv.CreatedBy = user.ID
	inv.ModifiedBy = user.ID

	dbc := database.MongoDb.C(models.DBC_INVENTORIES)

	if err := dbc.Insert(inv); err != nil {
		log.Println("Failed to create Project", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Project"})
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_HOST,
		ObjectID:    inv.ID,
		Description: "Inventory " + inv.Name + " created",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	if err := setMetadata(&inv); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	c.JSON(http.StatusCreated, inv)
}

func UpdateInventory(c *gin.Context) {
	oldTemplate := c.MustGet("template").(models.Template)

	var template models.Template
	if err := c.Bind(&template); err != nil {
		return
	}

	template.ID = oldTemplate.ID

	if err := template.Update(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   oldTemplate.ProjectID,
		Description: "Template ID " + template.ID.String() + " updated",
		ObjectID:    oldTemplate.ID,
		ObjectType:  "template",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveInventory(c *gin.Context) {
	crd := c.MustGet(_CTX_HOST).(models.Team)

	dbc := database.MongoDb.C(models.DBC_INVENTORIES)

	if err := dbc.RemoveId(crd.ID); err != nil {
		panic(err)
	}

	if err := (models.Event{
		Description: "Inventory " + crd.Name + " deleted",
		ObjectID:    crd.ID,
		ObjectType:  _CTX_HOST,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}