package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"github.com/pearsonappeng/tensor/validate"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential related items stored in the Gin Context
const (
	cHost = "host"
	cHostID = "host_id"
)

type HostController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// Middleware takes CTXHostID parameter from the Gin Context and fetches host data from the database
// it set host data under key CTXHost in the Gin Context
func (ctrl HostController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cHostID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Host does not exist"})
		return
	}

	var host ansible.Host
	if err := db.Hosts().FindId(bson.ObjectIdHex(objectID)).One(&host); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Host does not exist",
			Log: logrus.Fields{
				"Host ID": objectID,
				"Error":   err.Error(),
			},
		})
		return
	}

	roles := new(rbac.Inventory)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.ReadByID(user, host.InventoryID) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "DELETE":
		{
			// Reject the request if the user doesn't have inventory write permissions
			if !roles.WriteByID(user, host.InventoryID) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}

	c.Set(cHost, host)
	c.Next()
}

// GetHost is a Gin Handler function, returns the host as a JSON object
func (ctrl HostController) One(c *gin.Context) {
	host := c.MustGet(cHost).(ansible.Host)
	metadata.HostMetadata(&host)
	c.JSON(http.StatusOK, host)
}

// GetHosts is Gin handler function which returns list of hosts
// This takes lookup parameters and order parameters to filter and sort output data
func (ctrl HostController) All(c *gin.Context) {

	user := c.MustGet(cUser).(common.User)
	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"enabled", "has_active_failures"}, match)
	match = parser.Lookups([]string{"name", "description"}, match)
	query := db.Hosts().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.Inventory)
	var hosts []ansible.Host
	iter := query.Iter()
	var tmpHost ansible.Host
	for iter.Next(&tmpHost) {
		if !roles.ReadByID(user, tmpHost.InventoryID) {
			continue
		}

		metadata.HostMetadata(&tmpHost)
		hosts = append(hosts, tmpHost)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting hosts",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	count := len(hosts)
	pgi := util.NewPagination(c, count)
	if pgi.HasPage() {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound,
			Message: "#" + strconv.Itoa(pgi.Page()) + " page contains no results.",
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:     hosts[pgi.Skip():pgi.End()],
	})
}

// AddHost is a Gin handler function which creates a new host using request payload
// This accepts Host model.
func (ctrl HostController) Create(c *gin.Context) {
	var req ansible.Host
	user := c.MustGet(cUser).(common.User)

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// check whether the inventory exist or not
	if !req.InventoryExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Inventory does not exists.",
		})
		return
	}

	// Reject the request if the user doesn't have inventory write permissions
	if !new(rbac.Inventory).WriteByID(user, req.InventoryID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}

	// if the host exist in the collection it is not unique
	if !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Host with this name and inventory already exists.",
		})
		return
	}

	// check whether the group exist or not
	if req.GroupID != nil && !req.GroupExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Group does not exist.",
		})
		return
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	if err := db.Hosts().Insert(req); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while creating host",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	activity.AddActivity(activity.Create, user.ID, req, nil)
	metadata.HostMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

// UpdateHost is a handler function which updates a credential using request payload.
// This replaces all the fields in the database, empty "" fields and
// unspecified fields will be removed from the database object
func (ctrl HostController) Update(c *gin.Context) {
	host := c.MustGet(cHost).(ansible.Host)
	tmpHost := host
	user := c.MustGet(cUser).(common.User)
	var req ansible.Host
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// check whether the inventory exist or not
	if !req.InventoryExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Inventory does not exists.",
		})
		return
	}

	if req.Name != host.Name && !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Host with this name and inventory already exists.",
		})
		return
	}

	// check whether the group exist or not
	if req.GroupID != nil && !req.GroupExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Group does not exists.",
		})
		return
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

	if err := db.Hosts().UpdateId(host.ID, host); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating host.",
			Log:     logrus.Fields{"Host ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddActivity(activity.Update, user.ID, tmpHost, host)
	metadata.HostMetadata(&host)
	c.JSON(http.StatusOK, host)
}

// RemoveHost is a Gin handler function which removes a host object from the database
func (ctrl HostController) Delete(c *gin.Context) {
	host := c.MustGet(cHost).(ansible.Host)
	user := c.MustGet(cUser).(common.User)

	if err := db.Hosts().RemoveId(host.ID); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing hosts",
			Log:     logrus.Fields{"Host ID": host.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddActivity(activity.Delete, user.ID, host, nil)
	c.AbortWithStatus(204)
}

// VariableData is a Gin Handler function which returns variables for
// the host as JSON formatted object.
func (ctrl HostController) VariableData(c *gin.Context) {
	host := c.MustGet(cHost).(ansible.Host)

	variables := gin.H{}

	if err := json.Unmarshal([]byte(host.Variables), &variables); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusInternalServerError,
			Message: "Error while getting host variables",
			Log:     logrus.Fields{"Host ID": host.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, variables)

}

// Groups is a Gin handler function which returns parent group of the host
// TODO: not implemented
func (ctrl HostController) Groups(c *gin.Context) {
	host := c.MustGet(cHost).(ansible.Host)
	var group ansible.Group
	if host.GroupID != nil {
		if err := db.Groups().FindId(host.GroupID).One(&group); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while getting Groups",
				Log:     logrus.Fields{"Host ID": host.ID.Hex(), "Error": err.Error()},
			})
			return
		}
		metadata.GroupMetadata(&group)
		c.JSON(http.StatusOK, group)
		return
	}

	c.JSON(http.StatusOK, nil)
}

// AllGroups is a Gin handler function which returns parent groups of a host
// TODO: not implemented
func (ctrl HostController) AllGroups(c *gin.Context) {
	host := c.MustGet(cHost).(ansible.Host)

	var outobjects []ansible.Group
	var group ansible.Group

	if host.GroupID != nil {
		if err := db.Groups().FindId(host.GroupID).One(&group); err != nil {
			logrus.Errorln("Error while getting groups")
			c.JSON(http.StatusInternalServerError, common.Error{
				Code:   http.StatusInternalServerError,
				Errors: []string{"Error while getting groups"},
			})
			return
		}

		metadata.GroupMetadata(&group)
		outobjects = append(outobjects, group)
		group = ansible.Group{}
		for group.ParentGroupID != nil {
			if err := db.Groups().FindId(host.GroupID).One(&group); err != nil {
				logrus.Errorln("Error while getting groups")
				c.JSON(http.StatusInternalServerError, common.Error{
					Code:   http.StatusInternalServerError,
					Errors: []string{"Error while getting groups"},
				})
				return
			}

			metadata.GroupMetadata(&group)
			outobjects = append(outobjects, group)
			group = ansible.Group{}
		}

		nobj := len(outobjects)
		pgi := util.NewPagination(c, nobj)
		if pgi.HasPage() {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
			return
		}

		c.JSON(http.StatusOK, common.Response{
			Count:    nobj,
			Next:     pgi.NextPage(),
			Previous: pgi.PreviousPage(),
			Data:     outobjects[pgi.Limit():pgi.Offset()],
		})
		return
	}
	c.JSON(http.StatusOK, common.Response{
		Count: 0,
	})
}

// ActivityStream returns the activities of the user on Hosts
func (ctrl HostController) ActivityStream(c *gin.Context) {
	host := c.MustGet(cHost).(ansible.Host)
	var activities []common.Activity
	var act common.Activity
	iter := db.ActivityStream().Find(bson.M{"object1_id": host.ID}).Iter()
	for iter.Next(&act) {
		metadata.ActivityHostMetadata(&act)
		activities = append(activities, act)
	}

	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting activities",
			Log:     logrus.Fields{"Host ID": host.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	count := len(activities)
	pgi := util.NewPagination(c, count)
	if pgi.HasPage() {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound,
			Message: "#" + strconv.Itoa(pgi.Page()) + " page contains no results.",
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Data:     activities[pgi.Skip():pgi.End()],
	})
}
