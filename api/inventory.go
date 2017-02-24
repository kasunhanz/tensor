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
	cInventory   = "inventory"
	cInventoryID = "inventory_id"
)

type InventoryController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes project_id parameter from Gin Context and fetches project data from the database
// this set project data under key project in Gin Context.
func (ctrl InventoryController) Middleware(c *gin.Context) {
	objectID := c.Params.ByName(cInventoryID)
	user := c.MustGet(cUser).(common.User)

	if !bson.IsObjectIdHex(objectID) {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Inventory does not exist"})
		return
	}

	var inventory ansible.Inventory
	if err := db.Inventories().FindId(bson.ObjectIdHex(objectID)).One(&inventory); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusNotFound, Message: "Inventory does not exist",
			Log: logrus.Fields{
				"Inventory ID": objectID,
				"Error":        err.Error(),
			},
		})
		return
	}

	roles := new(rbac.Inventory)
	switch c.Request.Method {
	case "GET":
		{
			// Reject the request if the user doesn't have inventory read permissions
			if !roles.Read(user, inventory) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	case "PUT", "DELETE":
		{
			// Reject the request if the user doesn't have inventory write permissions
			if !roles.Write(user, inventory) {
				AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
					Message: "You don't have sufficient permissions to perform this action.",
				})
				return
			}
		}
	}
	c.Set(cInventory, inventory)
	c.Next()
}

// GetInventory is a Gin handler function which returns the project as a JSON object
func (ctrl InventoryController) One(c *gin.Context) {
	inventory := c.MustGet(cInventory).(ansible.Inventory)
	metadata.InventoryMetadata(&inventory)
	c.JSON(http.StatusOK, inventory)
}

// GetInventories is a Gin handler function which returns list of inventories
// This takes lookup parameters and order parameters to filter and sort output data.
func (ctrl InventoryController) All(c *gin.Context) {
	user := c.MustGet(cUser).(common.User)
	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"has_inventory_sources", "has_active_failures"}, match)
	match = parser.Lookups([]string{"name", "organization"}, match)

	query := db.Inventories().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	roles := new(rbac.Inventory)
	var inventories []ansible.Inventory
	iter := query.Iter()
	var tmpInventory ansible.Inventory
	for iter.Next(&tmpInventory) {
		if !roles.Read(user, tmpInventory) {
			continue
		}
		metadata.InventoryMetadata(&tmpInventory)
		inventories = append(inventories, tmpInventory)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting Inventory",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	count := len(inventories)
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
		Data:     inventories[pgi.Skip():pgi.End()],
	})
}

// AddInventory is a Gin handler function which creates a new inventory using request payload.
// This accepts Inventory model.
func (ctrl InventoryController) Create(c *gin.Context) {
	var req ansible.Inventory
	user := c.MustGet(cUser).(common.User)
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// check whether the organization not in the collection
	// if not fail
	if !req.OrganizationExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Organization does not exists.",
		})
		return
	}
	// Check whether the user has permissions to associate the inventory with organization
	if !(rbac.HasGlobalRead(user) || rbac.HasOrganizationRead(req.OrganizationID, user.ID)) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}
	// if inventory exists in the collection
	if !req.IsUnique() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Inventory with this Name already exists.",
		})
		return
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID
	if err := db.Inventories().Insert(req); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while creating inventory",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	roles := new(rbac.Inventory)
	if !(rbac.HasGlobalWrite(user) || rbac.IsOrganizationAdmin(req.OrganizationID, user.ID)) {
		roles.Associate(req.ID, user.ID, rbac.RoleTypeUser, rbac.InventoryAdmin)
	}

	activity.AddInventoryActivity(common.Create, user, req)
	metadata.InventoryMetadata(&req)
	c.JSON(http.StatusCreated, req)
}

// UpdateInventory is a Gin handler function which updates a credential using request payload.
// This replaces all the fields in the database, empty "" field and unspecified fields will be
// removed from the database.
func (ctrl InventoryController) Update(c *gin.Context) {
	inventory := c.MustGet(cInventory).(ansible.Inventory)
	tmpInventory := inventory
	user := c.MustGet(cUser).(common.User)

	var req ansible.Inventory
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		AbortWithErrors(c, http.StatusBadRequest,
			"Invalid JSON body",
			validate.GetValidationErrors(err)...)
		return
	}

	// check whether the organization not in the collection
	// if not fail
	if !req.OrganizationExist() {
		AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
			Message: "Organization does not exists.",
		})
		return
	}
	// Check whether the user has permissions to associate the inventory with organization
	if !(rbac.HasGlobalRead(user) || rbac.HasOrganizationRead(req.OrganizationID, user.ID)) {
		AbortWithError(LogFields{Context: c, Status: http.StatusUnauthorized,
			Message: "You don't have sufficient permissions to perform this action.",
		})
		return
	}
	if req.Name != inventory.Name {
		if !req.IsUnique() {
			AbortWithError(LogFields{Context: c, Status: http.StatusBadRequest,
				Message: "Inventory with this name already exists.",
			})
			return
		}
	}

	inventory.Name = strings.Trim(req.Name, " ")
	inventory.Description = strings.Trim(req.Description, " ")
	inventory.OrganizationID = req.OrganizationID
	inventory.Description = req.Description
	inventory.Variables = req.Variables
	inventory.Modified = time.Now()
	inventory.ModifiedByID = user.ID
	if err := db.Inventories().UpdateId(inventory.ID, inventory); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while updating inventory.",
			Log:     logrus.Fields{"Inventory ID": req.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	roles := new(rbac.Inventory)
	if !(rbac.HasGlobalWrite(user) || rbac.IsOrganizationAdmin(req.OrganizationID, user.ID)) {
		roles.Associate(req.ID, user.ID, rbac.RoleTypeUser, rbac.InventoryAdmin)
	}

	activity.AddInventoryActivity(common.Update, user, tmpInventory, inventory)
	metadata.InventoryMetadata(&inventory)
	c.JSON(http.StatusOK, inventory)
}

// RemoveInventory is a Gin handler function which removes a inventory object from the database
func (ctrl InventoryController) Delete(c *gin.Context) {
	inventory := c.MustGet(cInventory).(ansible.Inventory)
	user := c.MustGet(cUser).(common.User)

	if _, err := db.Hosts().RemoveAll(bson.M{"inventory_id": inventory.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing inventory hosts",
			Log:     logrus.Fields{"Inventory ID": inventory.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if _, err := db.Groups().RemoveAll(bson.M{"inventory_id": inventory.ID}); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing inventory groups",
			Log:     logrus.Fields{"Inventory ID": inventory.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	if err := db.Inventories().RemoveId(inventory.ID); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while removing inventory",
			Log:     logrus.Fields{"Inventory ID": inventory.ID.Hex(), "Error": err.Error()},
		})
		return
	}

	activity.AddInventoryActivity(common.Delete, user, inventory)
	c.AbortWithStatus(http.StatusNoContent)
}

// Script is a Gin Handler function which generates a ansible compatible
// inventory output
// note: we are not using var varname []string specially because
// output json must include [] for each array and {} for each object
func (ctrl InventoryController) Script(c *gin.Context) {
	inv := c.MustGet(cInventory).(ansible.Inventory)
	qhostvars := c.Query("hostvars")
	qhost := c.Query("host")

	if qhost != "" {
		//get hosts for parent group
		var host ansible.Host
		if err := db.Hosts().Find(bson.M{"name": qhost}).One(&host); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while getting vars",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}
		var gv gin.H
		if err := json.Unmarshal([]byte(host.Variables), &gv); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusInternalServerError,
				Message: "Error while getting vars",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}
		c.JSON(http.StatusOK, gv)
		return
	}

	resp := gin.H{}
	var parents []ansible.Group
	q := bson.M{"inventory_id": inv.ID}
	if err := db.Groups().Find(q).All(&parents); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting hosts",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	allhosts := []ansible.Host{}
	for _, v := range parents {
		var hosts []ansible.Host
		q := bson.M{"inventory_id": inv.ID, "group_id": v.ID}
		if err := db.Hosts().Find(q).All(&hosts); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while getting hosts",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}

		allhosts = append(allhosts, hosts...)
		var childgroups []ansible.Group
		q = bson.M{"inventory_id": inv.ID, "parent_group_id": v.ID}
		if err := db.Groups().Find(q).All(&childgroups); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while getting hosts",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}

		hostnames := []string{}
		for _, v := range hosts {
			hostnames = append(hostnames, v.Name)
		}
		groupnames := []string{}
		for _, v := range childgroups {
			groupnames = append(groupnames, v.Name)
		}
		gv := gin.H{}
		if v.Variables != "" {
			if err := json.Unmarshal([]byte(v.Variables), &gv); err != nil {
				AbortWithError(LogFields{Context: c, Status: http.StatusInternalServerError,
					Message: "Error while getting hosts",
					Log:     logrus.Fields{"Error": err.Error()},
				})
				return
			}
		}
		resp[v.Name] = gin.H{
			"hosts":    hostnames,
			"children": groupnames,
			"vars":     gv,
		}

	}

	hostvars := gin.H{}
	for _, v := range allhosts {
		if v.Variables != "" {
			var gv gin.H
			if err := json.Unmarshal([]byte(v.Variables), &gv); err != nil {
				AbortWithError(LogFields{Context: c, Status: http.StatusInternalServerError,
					Message: "Error while getting hosts",
					Log:     logrus.Fields{"Error": err.Error()},
				})
				return
			}
			hostvars[v.Name] = gv
		}
	}

	nghosts := []ansible.Host{}
	q = bson.M{"inventory_id": inv.ID, "group_id": nil}

	if err := db.Hosts().Find(q).All(&nghosts); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting non-grouped hosts",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	hosts := []string{}
	for _, v := range nghosts {
		if v.Variables != "" {
			var gv gin.H
			if err := json.Unmarshal([]byte(v.Variables), &gv); err != nil {
				AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
					Message: "Error while getting non-grouped hosts",
					Log:     logrus.Fields{"Error": err.Error()},
				})
				return
			}
			hostvars[v.Name] = gv
		}
		hosts = append(hosts, v.Name)
	}

	if qhostvars != "" {
		resp["_meta"] = gin.H{
			"hostvars": hostvars,
		}
	}
	resp["all"] = gin.H{
		"hosts": hosts,
	}

	c.JSON(http.StatusOK, resp)
}

// JobTemplates is a Gin Handler function which returns list of Job Templates
// that includes the inventory.
func (ctrl InventoryController) JobTemplates(c *gin.Context) {
	inv := c.MustGet(cInventory).(ansible.Inventory)
	user := c.MustGet(cUser).(common.User)

	var jobTemplate []ansible.JobTemplate
	iter := db.JobTemplates().Find(bson.M{"inventory_id": inv.ID}).Iter()
	roles := new(rbac.JobTemplate)
	var tmpJobTemplate ansible.JobTemplate
	for iter.Next(&tmpJobTemplate) {
		if !roles.Read(user, tmpJobTemplate) {
			continue
		}
		metadata.JTemplateMetadata(&tmpJobTemplate)
		jobTemplate = append(jobTemplate, tmpJobTemplate)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting job templates",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}

	count := len(jobTemplate)
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
		Data:     jobTemplate[pgi.Skip():pgi.End()],
	})
}

// RootGroups is a Gin handler function which returns list of root groups
// of the inventory.
func (ctrl InventoryController) RootGroups(c *gin.Context) {
	inv := c.MustGet(cInventory).(ansible.Inventory)

	var groups []ansible.Group
	iter := db.Groups().Find(bson.M{
		"inventory_id": inv.ID,
		"$or": []bson.M{
			{"parent_group_id": bson.M{"$exists": false}},
			{"parent_group_id": nil},
		},
	}).Iter()
	var tmpGroup ansible.Group
	for iter.Next(&tmpGroup) {
		metadata.GroupMetadata(&tmpGroup)
		groups = append(groups, tmpGroup)
	}
	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting groups",
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

// Groups is a Gin Handler function which returns all the groups of an Inventory.
func (ctrl InventoryController) Groups(c *gin.Context) {
	inv := c.MustGet(cInventory).(ansible.Inventory)

	var groups []ansible.Group
	iter := db.Groups().Find(bson.M{"inventory_id": inv.ID}).Iter()
	var tmpGroup ansible.Group
	for iter.Next(&tmpGroup) {
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

// Hosts is a Gin handler function which returns all hosts
// associated with the inventory.
func (ctrl InventoryController) Hosts(c *gin.Context) {
	inv := c.MustGet(cInventory).(ansible.Inventory)

	var hosts []ansible.Host
	iter := db.Hosts().Find(bson.M{"inventory_id": inv.ID}).Iter()
	var tmpHost ansible.Host
	for iter.Next(&tmpHost) {
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

// ActivityStream returns the activities of the user on Inventories
func (ctrl InventoryController) ActivityStream(c *gin.Context) {
	inventory := c.MustGet(cInventory).(ansible.Inventory)

	var activities []ansible.ActivityInventory
	var act ansible.ActivityInventory
	iter := db.ActivityStream().Find(bson.M{"object1._id": inventory.ID}).Iter()
	for iter.Next(&act) {
		metadata.ActivityInventoryMetadata(&act)
		metadata.InventoryMetadata(&act.Object1)
		if act.Object2 != nil {
			metadata.InventoryMetadata(act.Object2)
		}
		activities = append(activities, act)
	}

	if err := iter.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting activities",
			Log:     logrus.Fields{"Error": err.Error()},
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

// Tree is a Gin handler function which generate a json tree of the
// inventory.
func (ctrl InventoryController) Tree(c *gin.Context) {
	inv := c.MustGet(cInventory).(ansible.Inventory)

	var groups []ansible.Group
	iOne := db.Groups().Find(bson.M{
		"inventory_id": inv.ID,
	}).Iter()
	var gOne ansible.Group
	for iOne.Next(&gOne) {
		iTwo := db.Groups().Find(bson.M{
			"parent_group_id": gOne.ID,
		}).Iter()
		var gTwo ansible.Group
		for iTwo.Next(&gTwo) {
			iTree := db.Groups().Find(bson.M{
				"parent_group_id": gTwo.ID,
			}).Iter()
			var gThree ansible.Group
			for iTree.Next(&gThree) {
				if (*gThree.ParentGroupID).Hex() == gTwo.ID.Hex() {
					metadata.GroupMetadata(&gThree)
					gTwo.Children = append(gTwo.Children, gThree)
				}
			}

			if err := iTree.Close(); err != nil {
				AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
					Message: "Error while getting groups",
					Log:     logrus.Fields{"Error": err.Error()},
				})
				return
			}

			if (*gTwo.ParentGroupID).Hex() == gOne.ID.Hex() {
				metadata.GroupMetadata(&gTwo)
				gOne.Children = append(gOne.Children, gTwo)
			}
		}

		if err := iTwo.Close(); err != nil {
			AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
				Message: "Error while getting groups",
				Log:     logrus.Fields{"Error": err.Error()},
			})
			return
		}

		if gOne.ParentGroupID == nil {
			metadata.GroupMetadata(&gOne)
			groups = append(groups, gOne)
		}
	}

	if err := iOne.Close(); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusGatewayTimeout,
			Message: "Error while getting groups",
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

// VariableData is a Gin Handler function which returns variable data for the inventory.
func (ctrl InventoryController) VariableData(c *gin.Context) {
	inventory := c.MustGet(cInventory).(ansible.Inventory)
	variables := gin.H{}
	if err := json.Unmarshal([]byte(inventory.Variables), &variables); err != nil {
		AbortWithError(LogFields{Context: c, Status: http.StatusInternalServerError,
			Message: "Error while getting inventory variables",
			Log:     logrus.Fields{"Error": err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, variables)
}

// ObjectRoles is a Gin handler function
// This returns available roles can be associated with a Inventory model
func (ctrl InventoryController) ObjectRoles(c *gin.Context) {
	inventory := c.MustGet(cInventory).(ansible.Inventory)

	roles := []gin.H{
		{
			"type": "role",
			"links": gin.H{
				"inventory": "/v1/inventories/" + inventory.ID.Hex(),
			},
			"meta": gin.H{
				"resource_name":              inventory.Name,
				"resource_type":              "inventory",
				"resource_type_display_name": "Inventory",
			},
			"name":        "admin",
			"description": "Can manage all aspects of the inventory",
		},
		{
			"type": "role",
			"links": gin.H{
				"inventory": "/v1/inventories/" + inventory.ID.Hex(),
			},
			"meta": gin.H{
				"resource_name":              inventory.Name,
				"resource_type":              "inventory",
				"resource_type_display_name": "Inventory",
			},
			"name":        "use",
			"description": "Can use the inventory in a job template",
		},
		{
			"type": "role",
			"links": gin.H{
				"inventory": "/v1/inventories/" + inventory.ID.Hex(),
			},
			"meta": gin.H{
				"resource_name":              inventory.Name,
				"resource_type":              "inventory",
				"resource_type_display_name": "Inventory",
			},
			"name":        "update",
			"description": "May update project or inventory or group using the configured source update system",
		},
	}

	count := len(roles)
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
		Data:     roles[pgi.Skip():pgi.End()],
	})
}
