package groups

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

const _CTX_GROUP = "group"
const _CTX_USER = "user"
const _CTX_GROUP_ID = "group_id"

// GroupMiddleware takes host_id parameter from gin.Context and
// fetches host data from the database
// it set host data under key host in gin.Context
func GroupMiddleware(c *gin.Context) {
	ID := c.Params.ByName(_CTX_GROUP_ID)

	dbc := db.C(models.DBC_GROUPS)

	var grp models.Group
	if err := dbc.FindId(bson.ObjectIdHex(ID)).One(&grp); err != nil {
		log.Print(err) // log error to the system log
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set(_CTX_GROUP, grp)
	c.Next()
}

// GetHost returns the host as a serialized JSON object
func GetGroup(c *gin.Context) {
	grp := c.MustGet(_CTX_GROUP).(models.Group)
	setMetadata(&grp)

	c.JSON(200, grp)
}


// GetGroups returns groups as a serialized JSON object
func GetGroups(c *gin.Context) {
	dbc := db.C(models.DBC_GROUPS)

	parser := util.NewQueryParser(c)

	// query map
	match := parser.Match([]string{"source", "has_active_failures", })

	// add filters to query
	if con := parser.IContains([]string{"name"}); con != nil {
		if match != nil {
			for i, v := range con {
				match[i] = v
			}
		} else {
			match = con
		}
	}

	query := dbc.Find(match) // prepare the query

	count, err := query.Count(); // number of records
	if err != nil {
		log.Println("Unable to count Groups from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// initialize Pagination
	pgi := util.NewPagination(c, count)

	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page) + ": That page contains no results."})
		return
	}

	// set sort value to the query based on request parameters
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var grps []models.Group

	// get all values with skip limit
	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit).All(&grps); err != nil {
		log.Println("Unable to retrive Groups from the db", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// set related and summary fields to every item
	for i, v := range grps {
		// note: `v` reference doesn't modify original slice
		if err := setMetadata(&v); err != nil {
			log.Println("Unable to set metadata", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		grps[i] = v // modify each object in slice
	}

	// send response with JSON rendered data
	c.JSON(200, gin.H{"count": count, "next": pgi.NextPage(), "previous": pgi.PreviousPage(), "results": grps, })

}

// AddGroup creates a new group
func AddGroup(c *gin.Context) {
	var req models.Group
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	if err := c.Bind(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// create new object to omit unnecessary fields
	grp := models.Group {
		ID : bson.NewObjectId(),
		Name: req.Name,
		Description: req.Description,
		InventoryID: req.InventoryID,
		Variables: req.Variables,
		Created: time.Now(),
		Modified: time.Now(),
		CreatedByID: user.ID,
		ModifiedByID: user.ID,
	}


	dbc := db.C(models.DBC_GROUPS)

	if err := dbc.Insert(grp); err != nil {
		log.Println("Failed to create Group", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to create Group"})
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  _CTX_GROUP,
		ObjectID:    grp.ID,
		Description: "Group " + grp.Name + " created",
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	if err := setMetadata(&grp); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, grp)
}

// UpdateGroup will update the Group
func UpdateGroup(c *gin.Context) {
	// get Group from the gin.Context
	cgroup := c.MustGet(_CTX_GROUP).(models.Group)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Group

	if err := c.Bind(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	group := models.Group{
		Name: req.Name,
		Description: req.Description,
		Variables: req.Variables,
		Modified: time.Now(),
		ModifiedByID: user.ID,
	}

	collection := db.C(models.DBC_GROUPS)

	// update object
	if err := collection.UpdateId(cgroup.ID, group); err != nil {
		log.Println("Failed to update Group", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to update Group"})
		return
	}

	if err := (models.Event{
		ProjectID:   group.ID,
		Description: "Group ID " + group.ID.Hex() + " updated",
		ObjectID:    group.ID,
		ObjectType:  "group",
	}.Insert()); err != nil {
		panic(err)
	}

	// set `related` and `summary` feilds
	if err := setMetadata(&group); err != nil {
		log.Println("Failed to fetch metadata", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to fetch metadata"})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, group)
}

// RemoveGroup will remove the Group
// from the models._CTX_GROUP collection
func RemoveGroup(c *gin.Context) {
	// get Group from the gin.Context
	cgroup := c.MustGet(_CTX_GROUP).(models.Group)

	collection := db.C(models.DBC_GROUPS)

	// remove object from the collection
	if err := collection.RemoveId(cgroup.ID); err != nil {
		log.Println("Failed to remove Group", err)

		c.JSON(http.StatusInternalServerError,
			gin.H{"status": "error", "message": "Failed to remove Group"})
		return
	}

	if err := (models.Event{
		Description: "Group " + cgroup.Name + " deleted",
		ObjectID:    cgroup.ID,
		ObjectType:  _CTX_GROUP,
	}.Insert()); err != nil {
		log.Println("Failed to create Event", err)
	}

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}