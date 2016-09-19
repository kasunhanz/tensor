package hosts

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
)

const _CTX_HOST = "host"
const _CTX_USER = "user"
const _CTX_HOST_ID = "host_id"

// HostMiddleware takes host_id parameter from gin.Context and
// fetches host data from the database
// it set host data under key host in gin.Context
func HostMiddleware(c *gin.Context) {
	ID := c.Params.ByName(_CTX_HOST_ID)

	dbc := db.C(models.DBC_HOSTS)

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
	h := c.MustGet(_CTX_HOST).(models.Host)
	setMetadata(&h)

	c.JSON(200, h)
}


// GetHosts returns a JSON array of projects
func GetHosts(c *gin.Context) {
	dbc := db.C(models.DBC_HOSTS)

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
		log.Println("Unable to count Hosts from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	pgi := util.NewPagination(c, count)

	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page) + ": That page contains no results."})
		return
	}

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var hosts []models.Host

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&hosts); err != nil {
		log.Println("Unable to retrive Hosts from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	for i, v := range hosts {
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		hosts[i] = v
	}

	c.JSON(200, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": hosts, })

}

// AddInventory creates a new project
func AddHost(c *gin.Context) {
	var req models.Host
	user := c.MustGet(_CTX_USER).(models.User)

	if err := c.Bind(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	var host models.Host

	host.ID = bson.NewObjectId()
	host.Name = req.Name
	host.Description = req.Description
	host.InventoryID = req.InventoryID
	host.Variables = req.Variables
	host.Enabled = req.Enabled
	host.Created = time.Now()
	host.Modified = time.Now()
	host.CreatedByID = user.ID
	host.ModifiedByID = user.ID

	dbc := db.C(models.DBC_HOSTS)

	if err := dbc.Insert(host); err != nil {
		log.Println("Failed to create Project", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Project"})
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_HOST,
		ObjectID:    host.ID,
		Description: "Host " + host.Name + " created",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	if err := setMetadata(&host); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	c.JSON(http.StatusCreated, host)
}

func UpdateHost(c *gin.Context) {
	h := c.MustGet(_CTX_HOST).(models.Host)
	u := c.MustGet(_CTX_USER).(models.User)

	var req models.Host

	if err := c.Bind(&req); err != nil {
		return
	}

	var host models.Host
	host.Name = req.Name
	host.Description = req.Description
	host.Variables = req.Variables
	host.Enabled = req.Enabled
	host.Created = time.Now()
	host.Modified = time.Now()
	host.ModifiedByID = u.ID

	dbc := db.C(models.DBC_HOSTS)

	if err := dbc.UpdateId(h.ID, host); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   host.ID,
		Description: "Host ID " + host.ID.Hex() + " updated",
		ObjectID:    host.ID,
		ObjectType:  "host",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveHost(c *gin.Context) {
	host := c.MustGet(_CTX_HOST).(models.Host)

	dbc := db.MongoDb.C(models.DBC_HOSTS)

	if err := dbc.RemoveId(host.ID); err != nil {
		panic(err)
	}

	if err := (models.Event{
		Description: "Host " + host.Name + " deleted",
		ObjectID:    host.ID,
		ObjectType:  _CTX_HOST,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}