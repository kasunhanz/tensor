package inventories

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

const _CTX_INVENTORY = "inventory"
const _CTX_INVENTORY_ID = "inventory_id"

// InventoryMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// it set project data under key project in gin.Context
func InventoryMiddleware(c *gin.Context) {
	projectID := c.Params.ByName(_CTX_INVENTORY_ID)

	dbc := db.C(models.DBC_INVENTORIES)

	var inv models.Inventory
	if err := dbc.FindId(bson.ObjectIdHex(projectID)).One(&inv); err != nil {
		log.Print(err) // log error to the system log
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set(_CTX_INVENTORY, inv)
	c.Next()
}

// GetInventory returns the project as a JSON object
func GetInventory(c *gin.Context) {
	o := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	setMetadata(&o)

	c.JSON(200, o)
}


// GetInventories returns a JSON array of projects
func GetInventories(c *gin.Context) {
	dbc := db.C(models.DBC_INVENTORIES)

	parser := util.NewQueryParser(c)

	match := parser.Match([]string{"has_inventory_sources", "has_active_failures", })

	if con := parser.IContains([]string{"name", "organization"}); con != nil {
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

	pgi := util.NewPagination(c, count)

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

	dbc := db.C(models.DBC_INVENTORIES)

	if err := dbc.Insert(inv); err != nil {
		log.Println("Failed to create Project", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Project"})
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_INVENTORY,
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
	//TODO: this is not the request inv is the request
	req := c.MustGet(_CTX_INVENTORY).(models.Inventory)

	dbc := db.C(models.DBC_INVENTORIES)

	var inv models.Inventory
	if err := c.Bind(&inv); err != nil {
		return
	}

	inv.ID = req.ID

	if err := dbc.UpdateId(inv.ID, inv); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   req.ID,
		Description: "Template ID " + inv.ID.Hex() + " updated",
		ObjectID:    req.ID,
		ObjectType:  "template",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveInventory(c *gin.Context) {
	crd := c.MustGet(_CTX_INVENTORY).(models.Team)

	dbc := db.C(models.DBC_INVENTORIES)

	if err := dbc.RemoveId(crd.ID); err != nil {
		panic(err)
	}

	if err := (models.Event{
		Description: "Inventory " + crd.Name + " deleted",
		ObjectID:    crd.ID,
		ObjectType:  _CTX_INVENTORY,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
//TODO: performance enhancements of queries
func Script(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)

	cgroups := db.C(models.DBC_GROUPS)
	chosts := db.C(models.DBC_HOSTS)

	qall := c.Query("all")
	qhostvars := c.Query("hostvars")

	//single host vars
	qhost := c.Query("host")


	// First Get all groups for inventory ID
	var parents []models.Group

	q := bson.M{"inventory_id": inv.ID}

	if err := cgroups.Find(q).All(&parents); err != nil {
		log.Println("Error while getting groups", err)
		// send a brief error description to client
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"message": "Error while getting Hosts",
		})
		return
	}

	// loop through parent groups and get their hosts and
	// child groups
	// suppress key since we wont modify the array
	for _, v := range parents {
		//get hosts for parent group
		var hosts []models.Host
		q := bson.M{"inventory_id": inv.ID, "group_id": v.ID}

		if err := chosts.Find(q).All(&hosts); err != nil {
			log.Println("Error while getting host for group", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": "Error while getting Hosts",
			})
			return
		}

		// Second get all child groups
		var children []models.Group

		q = bson.M{"inventory_id": inv.ID, "parent_group_id": v.ID}

		if err := cgroups.Find(q).All(&children); err != nil {
			log.Println("Error while getting child groups", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": "Error while getting Hosts",
			})
			return
		}
		//TODO: we have child groups and hosts :D loop through them
		//and construct the final

	}

	var req models.Inventory
	if err := c.Bind(&inv); err != nil {
		return
	}

	inv.ID = req.ID

	if err := dbc.UpdateId(inv.ID, inv); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   req.ID,
		Description: "Template ID " + inv.ID.Hex() + " updated",
		ObjectID:    req.ID,
		ObjectType:  "template",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}