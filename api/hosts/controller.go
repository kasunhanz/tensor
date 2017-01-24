package hosts

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/helpers"
	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential releated items stored in the Gin Context
const (
	CTXHost   = "host"
	CTXUser   = "user"
	CTXHostID = "host_id"
)

// Middleware generates a middleware handler function that works inside of a Gin request.
// Middleware takes CTXHostID parameter from the Gin Context and fetches host data from the database
// it set host data under key CTXHost in the Gin Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(CTXHostID, c)

	if err != nil {
		log.Errorln("Error while getting the Host:", err) // log error to the system log
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var h ansible.Host
	if err := db.Hosts().FindId(bson.ObjectIdHex(ID)).One(&h); err != nil {
		log.Errorln("Error while getting the Host:", err) // log error to the system log
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	c.Set(CTXHost, h)
	c.Next()
}

// GetHost is a Gin Handler function, returns the host as a JSON object
func GetHost(c *gin.Context) {
	host := c.MustGet(CTXHost).(ansible.Host)
	metadata.HostMetadata(&host)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, host)
}

// GetHosts is Gin handler function which returns list of hosts
// This takes lookup parameters and order parameters to filter and sort output data
func GetHosts(c *gin.Context) {

	parser := util.NewQueryParser(c)

	match := bson.M{}
	match = parser.Match([]string{"enabled", "has_active_failures"}, match)
	match = parser.Lookups([]string{"name", "description"}, match)

	//prepare the query
	query := db.Hosts().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var hosts []ansible.Host
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpHost ansible.Host
	// iterate over all and only get valid objects
	for iter.Next(&tmpHost) {
		metadata.HostMetadata(&tmpHost)
		// good to go add to list
		hosts = append(hosts, tmpHost)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Host data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Hosts"},
		})
		return
	}

	count := len(hosts)
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
		Results:  hosts[pgi.Skip():pgi.End()],
	})
}

// AddHost is a Gin handler function which creates a new host using request payload
// This accepts Host model.
func AddHost(c *gin.Context) {
	var req ansible.Host
	user := c.MustGet(CTXUser).(common.User)

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// if the host exist in the collection it is not unique
	if helpers.IsNotUniqueHost(req.Name, req.InventoryID) {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Host with this Name and Inventory already exists."},
		})
		return
	}

	// check whether the inventory exist or not
	if !helpers.InventoryExist(req.InventoryID) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Inventory does not exists."},
		})
		return
	}

	// check whether the group exist or not
	if req.GroupID != nil {
		if !helpers.GroupExist(*req.GroupID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Group does not exists."},
			})
			return
		}
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID

	if err := db.Hosts().Insert(req); err != nil {
		log.Errorln("Error while creating Host:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while creating Host"},
		})
		return
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXHost,
		ObjectID:    req.ID,
		Description: "Host " + req.Name + " created",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	metadata.HostMetadata(&req)

	c.JSON(http.StatusCreated, req)
}

// UpdateHost is a handler function which updates a credential using request payload.
// This replaces all the fields in the database, empty "" fields and
// unspecified fields will be removed from the database object
func UpdateHost(c *gin.Context) {
	host := c.MustGet(CTXHost).(ansible.Host)
	user := c.MustGet(CTXUser).(common.User)

	var req ansible.Host

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
	}

	// check whether the inventory exist or not
	if !helpers.InventoryExist(req.InventoryID) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Inventory does not exists."},
		})
		return
	}

	if req.Name != host.Name {
		// if the host exist in the collection it is not unique
		if helpers.IsNotUniqueHost(req.Name, req.InventoryID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Host with this Name and Inventory already exists."},
			})
			return
		}
	}

	// check whether the group exist or not
	if req.GroupID != nil {
		if !helpers.GroupExist(*req.GroupID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Group does not exists."},
			})
			return
		}
	}

	host.Name = strings.Trim(req.Name, " ")
	host.InventoryID = req.InventoryID
	host.Description = strings.Trim(req.Description, " ")
	host.GroupID = req.GroupID
	host.InstanceID = req.InstanceID
	host.Variables = req.Variables
	host.Enabled = req.Enabled
	host.Modified = req.Modified
	host.ModifiedByID = user.ID

	//update object
	if err := db.Hosts().UpdateId(host.ID, host); err != nil {
		log.Errorln("Error while updating Host:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Host"},
		})
	}

	if err := db.ActivityStream().Insert(common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXHost,
		ObjectID:    req.ID,
		Description: "Host " + req.Name + " updated",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	metadata.HostMetadata(&host)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, host)
}

// PatchHost is a Gin handler function which partially updates a credential using request payload.
// This replaces specified fields in the database, empty "" fields will be
// removed from the database object. Unspecified fields will ignored.
func PatchHost(c *gin.Context) {
	host := c.MustGet(CTXHost).(ansible.Host)
	user := c.MustGet(CTXUser).(common.User)

	var req ansible.PatchHost

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
	}

	// check whether the inventory exist or not
	if req.InventoryID != nil {
		if !helpers.InventoryExist(*req.InventoryID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Inventory does not exists."},
			})
			return
		}
	}

	if req.Name != nil && *req.Name != host.Name {
		invID := host.ID
		if req.InventoryID != nil {
			invID = *req.InventoryID
		}
		// if the host exist in the collection it is not unique
		if helpers.IsNotUniqueHost(*req.Name, invID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Host with this Name and Inventory already exists."},
			})
			return
		}
	}

	// check whether the group exist or not
	if req.GroupID != nil {
		if !helpers.GroupExist(*req.GroupID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Group does not exists."},
			})
			return
		}
	}

	if req.Name != nil {
		host.Name = *req.Name
	}

	if req.InventoryID != nil {
		host.InventoryID = *req.InventoryID
	}

	if req.Description != nil {
		host.Description = *req.Description
	}

	if req.GroupID != nil {
		// if empty string then make the credential null
		if len(*req.GroupID) == 12 {
			host.GroupID = req.GroupID
		} else {
			host.GroupID = nil
		}
	}

	if req.InstanceID != nil {
		host.InstanceID = *req.InstanceID
	}

	if req.Variables != nil {
		host.Variables = *req.Variables
	}

	if req.Enabled != nil {
		host.Enabled = *req.Enabled
	}

	host.Modified = time.Now()
	host.ModifiedByID = user.ID

	//update object
	if err := db.Hosts().UpdateId(host.ID, host); err != nil {
		log.Errorln("Error while updating Host:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Host"},
		})
		return
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXHost,
		ObjectID:    host.ID,
		Description: "Host " + host.Name + " updated",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	// set `related` and `summary` feilds
	metadata.HostMetadata(&host)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, host)
}

// RemoveHost is a Gin handler function which removes a host object from the database
func RemoveHost(c *gin.Context) {
	// get Host from the gin.Context
	host := c.MustGet(CTXHost).(ansible.Host)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	if err := db.Hosts().RemoveId(host.ID); err != nil {
		log.Errorln("Error while removing Host:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Host"},
		})
		return
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXHost,
		ObjectID:    host.ID,
		Description: "Host " + host.Name + " deleted",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	c.AbortWithStatus(204)
}

// VariableData is a Gin Handler function which returns variables for
// the host as JSON formatted object.
func VariableData(c *gin.Context) {
	host := c.MustGet(CTXHost).(ansible.Host)

	variables := gin.H{}

	if err := json.Unmarshal([]byte(host.Variables), &variables); err != nil {
		log.Errorln("Error while getting Host variables")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": []string{"Error while getting Host variables"},
		})
		return
	}

	c.JSON(http.StatusOK, variables)

}

// Groups is a Gin handler function which returns parent group of the host
// TODO: not implemented
func Groups(c *gin.Context) {
	host := c.MustGet(CTXHost).(ansible.Host)

	var group ansible.Group

	if host.GroupID != nil {
		// find group for the host
		if err := db.Groups().FindId(host.GroupID).One(&group); err != nil {
			log.Errorln("Error while getting groups")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": []string{"Error while getting groups"},
			})
			return
		}

		// set group metadata
		metadata.GroupMetadata(&group)

		// send response with JSON rendered data
		c.JSON(http.StatusOK, group)
		return
	}

	// no assigned groups
	// send response with JSON rendered data
	c.JSON(http.StatusOK, nil)
}

// AllGroups is a Gin handler function which returns parent groups of a host
// TODO: not implemented
func AllGroups(c *gin.Context) {
	host := c.MustGet(CTXHost).(ansible.Host)

	var outobjects []ansible.Group
	var group ansible.Group

	if host.GroupID != nil {
		// find group for the host
		if err := db.Groups().FindId(host.GroupID).One(&group); err != nil {
			log.Errorln("Error while getting groups")
			c.JSON(http.StatusInternalServerError, common.Error{
				Code:     http.StatusInternalServerError,
				Messages: []string{"Error while getting groups"},
			})
			return
		}

		// set group metadata
		metadata.GroupMetadata(&group)
		//add group to outobjects
		outobjects = append(outobjects, group)
		// clean object
		group = ansible.Group{}

		for group.ParentGroupID != nil {
			// find group for the host
			if err := db.Groups().FindId(host.GroupID).One(&group); err != nil {
				log.Errorln("Error while getting groups")
				c.JSON(http.StatusInternalServerError, common.Error{
					Code:     http.StatusInternalServerError,
					Messages: []string{"Error while getting groups"},
				})
				return
			}

			// set group metadata
			metadata.GroupMetadata(&group)
			//add group to outobjects
			outobjects = append(outobjects, group)

			// clean object
			group = ansible.Group{}
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
		c.JSON(http.StatusOK, common.Response{
			Count:    nobj,
			Next:     pgi.NextPage(),
			Previous: pgi.PreviousPage(),
			Results:  outobjects[pgi.Limit():pgi.Offset()],
		})

		return
	}

	// no assigned groups
	// send response with JSON rendered data
	c.JSON(http.StatusOK, common.Response{
		Count: 0,
	})
}

// ActivityStream is a Gin handler function which returns list of activities associated with
// credential object that is in the Gin Context
// TODO: not complete
func ActivityStream(c *gin.Context) {
	host := c.MustGet(CTXHost).(ansible.Host)

	var activities []common.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": host.ID, "type": CTXHost}).All(&activities)

	if err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while Activities"},
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
