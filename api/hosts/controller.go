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
	"encoding/json"
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
)

const _CTX_HOST = "host"
const _CTX_USER = "user"
const _CTX_HOST_ID = "host_id"

// Middleware takes host_id parameter from gin.Context and
// fetches host data from the database
// it set host data under key host in gin.Context
func Middleware(c *gin.Context) {
	ID := c.Params.ByName(_CTX_HOST_ID)

	dbc := db.C(db.HOSTS)

	var h models.Host
	if err := dbc.FindId(bson.ObjectIdHex(ID)).One(&h); err != nil {
		log.Print("Error while getting the Host:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
	}

	c.Set(_CTX_HOST, h)
	c.Next()
}

// GetHost returns the hsot as a JSON object
func GetHost(c *gin.Context) {
	host := c.MustGet(_CTX_HOST).(models.Host)
	metadata.HostMetadata(&host)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:1,
		Results:host,
	})
}


// GetHosts returns a JSON array of projects
func GetHosts(c *gin.Context) {
	dbc := db.C(db.HOSTS)

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

	//prepare the query
	query := dbc.Find(match)

	//get number of records fro pagination
	count, err := query.Count();
	if err != nil {
		log.Println("Error while trying to get count of Hosts from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting hosts",
		})
		return
	}

	// init pagination
	pgi := util.NewPagination(c, count)

	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}

	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var hosts []models.Host

	if err := query.Skip(pgi.Offset()).Limit(pgi.Limit()).All(&hosts); err != nil {
		log.Println("Error while retriving Host data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting hosts",
		})
		return
	}
	for i, v := range hosts {
		if err := metadata.HostMetadata(&v); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting hosts",
			})
			return
		}

		hosts[i] = v
	}


	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:hosts,
	})
}

// AddInventory creates a new project
func AddHost(c *gin.Context) {
	var req models.Host
	user := c.MustGet(_CTX_USER).(models.User)

	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	host := models.Host{
		ID:bson.NewObjectId(),
		Name: req.Name,
		Description: req.Description,
		InventoryID: req.InventoryID,
		Variables: req.Variables,
		Enabled: req.Enabled,
		Created: time.Now(),
		Modified: time.Now(),
		CreatedByID: user.ID,
		ModifiedByID: user.ID,
	}

	dbc := db.C(db.HOSTS)

	if err := dbc.Insert(host); err != nil {
		log.Println("Error while creating Host:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Host",
		})
		return
	}


	// add new activity to activity stream
	addActivity(host.ID, user.ID, "Host " + host.Name + " created")

	if err := metadata.HostMetadata(&host); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Host",
		})
		return
	}

	c.JSON(http.StatusCreated, models.Response{
		Count:1,
		Results:host,
	})
}
// update will update a Host
// from request values
func UpdateHost(c *gin.Context) {
	host := c.MustGet(_CTX_HOST).(models.Host)
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Host

	if err := c.BindJSON(&req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
	}

	host.Name = req.Name
	host.Description = req.Description
	host.Variables = req.Variables
	host.Enabled = req.Enabled
	host.Modified = time.Now()
	host.ModifiedByID = user.ID

	dbc := db.C(db.HOSTS)

	//update object
	if err := dbc.UpdateId(host.ID, host); err != nil {
		log.Println("Error while updating Host:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while updating Host",
		})
	}

	// add new activity to activity stream
	addActivity(host.ID, user.ID, "Host " + host.Name + " created")

	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:1,
		Results:host,
	})
}

func RemoveHost(c *gin.Context) {
	// get Host from the gin.Context
	host := c.MustGet(_CTX_HOST).(models.Host)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	dbc := db.MongoDb.C(db.HOSTS)

	if err := dbc.RemoveId(host.ID); err != nil {
		log.Println("Error while removing Host:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Host",
		})
		return
	}

	// add new activity to activity stream
	addActivity(host.ID, user.ID, "Group " + host.Name + " deleted")

	c.AbortWithStatus(204)
}

func VariableData(c *gin.Context) {
	host := c.MustGet(_CTX_HOST).(models.Host)

	variables := gin.H{}

	if err := json.Unmarshal([]byte(host.Variables), &variables); err != nil {
		log.Println("Error while getting host variables")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"message": "Error while getting host variables",
		})
		return
	}

	c.JSON(http.StatusOK, variables)

}

func Groups(c *gin.Context) {
	host := c.MustGet(_CTX_HOST).(models.Host)

	collection := db.MongoDb.C(db.GROUPS)

	var group models.Group

	if len(host.GroupID) == 24 {
		// find group for the host
		if err := collection.FindId(host.GroupID).One(&group); err != nil {
			log.Println("Error while getting groups")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": "Error while getting groups",
			})
			return
		}

		// set group metadata
		metadata.GroupMetadata(&group)

		// send response with JSON rendered data
		c.JSON(http.StatusOK, models.Response{
			Count: 1,
			Results: []models.Group{group, },
		})
		return
	}

	// no assigned groups
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count: 0,
	})
}

func AllGroups(c *gin.Context) {
	host := c.MustGet(_CTX_HOST).(models.Host)

	collection := db.MongoDb.C(db.GROUPS)

	var outobjects []models.Group
	var group models.Group

	if len(host.GroupID) == 24 {
		// find group for the host
		if err := collection.FindId(host.GroupID).One(&group); err != nil {
			log.Println("Error while getting groups")
			c.JSON(http.StatusInternalServerError, models.Error{
				Code: http.StatusInternalServerError,
				Message: "Error while getting groups",
			})
			return
		}

		// set group metadata
		metadata.GroupMetadata(&group)
		//add group to outobjects
		outobjects = append(outobjects, group)
		// clean object
		group = models.Group{}

		for len(group.ParentGroupID) == 12 {
			// find group for the host
			if err := collection.FindId(host.GroupID).One(&group); err != nil {
				log.Println("Error while getting groups")
				c.JSON(http.StatusInternalServerError, models.Error{
					Code: http.StatusInternalServerError,
					Message: "Error while getting groups",
				})
				return
			}

			// set group metadata
			metadata.GroupMetadata(&group)
			//add group to outobjects
			outobjects = append(outobjects, group)

			// clean object
			group = models.Group{}
		}

		nobj := len(outobjects)

		// initialize Pagination
		pgi := util.NewPagination(c, nobj)

		//if page is incorrect return 404
		if pgi.HasPage() {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
			return
		}

		// send response with JSON rendered data
		c.JSON(http.StatusOK, models.Response{
			Count: nobj,
			Next: pgi.NextPage(),
			Previous: pgi.PreviousPage(),
			Results: outobjects[pgi.Limit():pgi.Offset()],
		})

		return
	}

	// no assigned groups
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count: 0,
	})
}