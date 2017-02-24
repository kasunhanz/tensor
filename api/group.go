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

// Keys for group related items stored in the Gin Context
const (
	cGroup   = "group"
	cGroupID = "group_id"
)

type GroupController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes host_id parameter from the Gin Context and fetches host data from the database
// it will set host data under key host in the Gin Context.
func (ctrl GroupController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cGroupID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Group does not exist"})
		return
	}

	var group ansible.Group
	err := db.Groups().FindId(bson.ObjectIdHex(objectID)).One(&group)
	if err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Group does not exist",
			Log: logrus.Fields{
				"Group ID": objectID,
				"Error":    err.Error(),
			},
		})
		return
	}

	roles := new(rbac.Inventory)
	switch c.Request.Method {
	case "GET":
		{
			if !roles.ReadByID(user, group.InventoryID) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "DELETE":
		{
			// Reject the request if the user doesn't have inventory write permissions
			if !roles.WriteByID(user, group.InventoryID) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}

	c.Set(cGroup, group)
	c.Next()
}

// GetGroup is a Gin handler function which returns the host as a JSON object.
func (ctrl GroupController) One(c *gin.Context) {
	group := c.MustGet(cGroup).(ansible.Group)
	metadata.GroupMetadata(&group)
	c.JSON(http.StatusOK, group)
}

// GetGroups is a Gin handler function which returns list of Groups
// This takes lookup parameters and order parameters to filer and sort output data.
func (ctrl GroupController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"source", "has_active_failures"}, match)
	match = parser.Lookups([]string{"name", "description"}, match)
	query := db.Groups().Find(match)
	order := parser.OrderBy()
	if order != "" {
		query.Sort(order)
	}

	roles := new(rbac.Inventory)
	var groups []ansible.Group
	iter := query.Iter()
	var tmpGroup ansible.Group
	for iter.Next(&tmpGroup) {
		if !roles.ReadByID(user, tmpGroup.InventoryID) {
			continue
		}
		metadata.GroupMetadata(&tmpGroup)
		groups = append(groups, tmpGroup)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting Groups",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	count := len(groups)
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
		Data:     groups[pgi.Skip():pgi.End()],
	})
}

// AddGroup is a Gin handler function which creates a new group using request payload.
// This accepts Group model.
func (ctrl GroupController) Create(c *gin.Context) {
	var req ansible.Group
	user := c.MustGet(cUser).(common.User)
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

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

	// if the group exist in the collection it is not unique
	if !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Group with this name and inventory already exists.",
		})
		return
	}

	if req.ParentGroupID != nil {
		parent1, err := req.GetParent()
		if err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Parent group does not exist.",
			})
			return
		}
		if parent1.ParentGroupID != nil {
			parent2, err := parent1.GetParent()
			if err != nil {
				AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
					Message: "Parent Group hierarchy does not exists.",
				})
				return
			}
			if parent2.ParentGroupID != nil {
				AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
					Message: "Maximum level of nesting is 3.",
				})
				return
			}
		}
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	if err := db.Groups().Insert(req); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while creating group",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	activity.AddGroupActivity(common.Create, user, req)
	metadata.GroupMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

// UpdateGroup is a handler function which updates a group using request payload.
// This replaces all the fields in the database. empty "" fields and
// unspecified fields will be removed from the database object.
func (ctrl GroupController) Update(c *gin.Context) {
	group := c.MustGet(cGroup).(ansible.Group)
	tmpGroup := group
	user := c.MustGet(cUser).(common.User)

	var req ansible.Group
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

	if req.Name != group.Name && !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Group with this name and inventory already exists.",
		})
		return
	}

	// check whether the group exist or not
	if req.ParentGroupID != nil && *req.ParentGroupID != group.ID && !req.ParentExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Parent Group does not exists.",
		})
		return
	}

	group.Name = strings.Trim(req.Name, " ")
	group.Description = strings.Trim(req.Description, " ")
	group.Variables = req.Variables
	group.InventoryID = req.InventoryID
	group.ParentGroupID = req.ParentGroupID
	group.Modified = time.Now()
	group.ModifiedByID = user.ID
	if err := db.Groups().UpdateId(group.ID, group); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating group.",
			Log:     logrus.Fields{"Group ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddGroupActivity(common.Update, user, tmpGroup, group)
	metadata.GroupMetadata(&group)
	c.JSON(http.StatusOK, group)
}

// RemoveGroup is a Gin handler function which removes a group object from the database
func (ctrl GroupController) Delete(c *gin.Context) {
	group := c.MustGet(cGroup).(ansible.Group)
	user := c.MustGet(cUser).(common.User)

	iter := db.Groups().Find(bson.M{
		"$or": []bson.M{
			{"parent_group_id": group.ID},
			{"_id": group.ID},
		},
	}).Select(bson.M{"_id": 1}).Iter()
	var tmpGroup ansible.Group
	var groupIDs []bson.ObjectId
	for iter.Next(&tmpGroup) {
		groupIDs = append(groupIDs, tmpGroup.ID)
	}
	groupIDs = append(groupIDs, group.ID)

	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing groups",
			Log:     logrus.Fields{"Group ID": group.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.Hosts().RemoveAll(bson.M{"group_id": bson.M{"$in": groupIDs}}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing groups",
			Log:     logrus.Fields{"Group ID": group.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.Groups().RemoveAll(bson.M{"_id": bson.M{"$in": groupIDs}}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing groups",
			Log:     logrus.Fields{"Group ID": group.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddGroupActivity(common.Delete, user, group)
	c.AbortWithStatus(http.StatusNoContent)
}

// VariableData is Gin handler function which returns host group variables
func (ctrl GroupController) VariableData(c *gin.Context) {
	group := c.MustGet(cGroup).(ansible.Group)
	variables := gin.H{}
	if err := json.Unmarshal([]byte(group.Variables), &variables); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting group variables",
			Log:     logrus.Fields{"Group ID": group.ID.Hex(), "Error": err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, variables)
}

// ActivityStream returns the activities of the user on Groups
func (ctrl GroupController) ActivityStream(c *gin.Context) {
	group := c.MustGet(cGroup).(ansible.Group)
	var activities []ansible.ActivityGroup
	var act ansible.ActivityGroup
	iter := db.ActivityStream().Find(bson.M{"object1._id": group.ID}).Iter()
	for iter.Next(&act) {
		metadata.ActivityGroupMetadata(&act)
		metadata.GroupMetadata(&act.Object1)
		if act.Object2 != nil {
			metadata.GroupMetadata(act.Object2)
		}
		activities = append(activities, act)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting activities",
			Log:     logrus.Fields{"Group ID": group.ID.Hex(), "Error": err.Error()},
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
