package inventories

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
	"github.com/pearsonappeng/tensor/roles"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/mgo.v2/bson"
)

// Keys for credential releated items stored in the Gin Context
const (
	CTXInventory   = "inventory"
	CTXUser        = "user"
	CTXInventoryID = "inventory_id"
)

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes project_id parameter from Gin Context and fetches project data from the database
// this set project data under key project in Gin Context.
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(CTXInventoryID, c)

	if err != nil {
		log.Errorln("Error while getting the Inventory:", err) // log error to the system log
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var inventory ansible.Inventory
	err = db.Inventories().FindId(bson.ObjectIdHex(ID)).One(&inventory)

	if err != nil {
		log.Errorln("Error while getting the Inventory:", err) // log error to the system log
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	c.Set(CTXInventory, inventory)
	c.Next()
}

// GetInventory is a Gin handler function which returns the project as a JSON object
func GetInventory(c *gin.Context) {
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)

	metadata.InventoryMetadata(&inventory)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

// GetInventories is a Gin handler function which returns list of inventories
// This takes lookup parameters and order parameters to filter and sort output data.
func GetInventories(c *gin.Context) {

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"has_inventory_sources", "has_active_failures"}, match)
	match = parser.Lookups([]string{"name", "organization"}, match)

	query := db.Inventories().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var inventories []ansible.Inventory
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpInventory ansible.Inventory
	// iterate over all and only get valid objects
	for iter.Next(&tmpInventory) {
		metadata.InventoryMetadata(&tmpInventory)
		// good to go add to list
		inventories = append(inventories, tmpInventory)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Inventory data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Inventory"},
		})
		return
	}

	count := len(inventories)
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
		Results:  inventories[pgi.Skip():pgi.End()],
	})
}

// AddInventory is a Gin handler function which creates a new inventory using request payload.
// This accepts Inventory model.
func AddInventory(c *gin.Context) {
	var req ansible.Inventory
	user := c.MustGet(CTXUser).(common.User)

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization not in the collection
	// if not fail
	if helpers.OrganizationNotExist(req.OrganizationID) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	// if inventory exists in the collection
	if helpers.IsNotUniqueInventory(req.Name, req.OrganizationID) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Inventory with this Name already exists."},
		})
		return
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedByID = user.ID
	req.ModifiedByID = user.ID

	if err := db.Inventories().Insert(req); err != nil {
		log.Errorln("Error while creating Inventory:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while creating Inventory"},
		})
		return
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXInventory,
		ObjectID:    req.ID,
		Description: "Inventory " + req.Name + " created",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	metadata.InventoryMetadata(&req)

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// UpdateInventory is a Gin handler function which updates a credential using request payload.
// This replaces all the fields in the database, empty "" fiels and unspecified fields will be
// removed from the database.
func UpdateInventory(c *gin.Context) {
	// get Inventory from the gin.Context
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req ansible.Inventory
	err := binding.JSON.Bind(c.Request, &req)
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization not in the collection
	// if not fail
	if helpers.OrganizationNotExist(req.OrganizationID) {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	if req.Name != inventory.Name {
		// if inventory exists in the collection
		if helpers.IsNotUniqueInventory(req.Name, req.OrganizationID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Inventory with this Name already exists."},
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

	err = db.Inventories().UpdateId(inventory.ID, inventory)
	if err != nil {
		log.Errorln("Error while updating Inventory:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Inventory"},
		})
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXInventory,
		ObjectID:    req.ID,
		Description: "Inventory " + req.Name + " updated",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	// set `related` and `summary` feilds
	metadata.InventoryMetadata(&inventory)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

// PatchInventory is a Gin handler function which partially updates a inventory using request payload.
// This replaces specified fields in the data, empty "" fields will be
// removed from the database object. unspecified fields will be ignored.
func PatchInventory(c *gin.Context) {
	// get Inventory from the gin.Context
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req ansible.PatchInventory
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if req.OrganizationID != nil {
		// check whether the organization exist or not
		if !helpers.OrganizationExist(*req.OrganizationID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	if req.Name != nil && *req.Name != inventory.Name {
		ogID := inventory.OrganizationID
		if req.OrganizationID != nil {
			ogID = *req.OrganizationID
		}
		// if inventory exists in the collection
		if helpers.IsNotUniqueInventory(*req.Name, ogID) {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Inventory with this Name already exists."},
			})
			return
		}
	}

	if req.Name != nil {
		inventory.Name = strings.Trim(*req.Name, " ")
	}

	if req.Description != nil {
		inventory.Description = strings.Trim(*req.Description, " ")
	}

	if req.OrganizationID != nil {
		inventory.OrganizationID = *req.OrganizationID
	}

	if req.Variables != nil {
		inventory.Variables = *req.Variables
	}

	inventory.Modified = time.Now()
	inventory.ModifiedByID = user.ID

	if err := db.Inventories().UpdateId(inventory.ID, inventory); err != nil {
		log.Errorln("Error while updating Inventory:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Inventory"},
		})
		return
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXInventory,
		ObjectID:    inventory.ID,
		Description: "Inventory " + inventory.Name + " updated",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	// set `related` and `summary` feilds
	metadata.InventoryMetadata(&inventory)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

// RemoveInventory is a Gin handler function which removes a inventory object from the database
func RemoveInventory(c *gin.Context) {
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	changes, err := db.Hosts().RemoveAll(bson.M{"inventory_id": inventory.ID})
	if err != nil {
		log.Errorln("Error while removing Hosts:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Inventory Hosts"},
		})
	}
	log.Infoln("Hosts remove info:", changes.Removed)

	changes, err = db.Groups().RemoveAll(bson.M{"inventory_id": inventory.ID})
	if err != nil {
		log.Errorln("Error while removing Groups:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Inventory Groups"},
		})
	}
	log.Infoln("Groups remove info:", changes.Removed)

	err = db.Inventories().RemoveId(inventory.ID)
	if err != nil {
		log.Errorln("Error while removing Inventory:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while removing Inventory"},
		})
	}

	// add new activity to activity stream
	if err := db.ActivityStream().Insert(common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     user.ID,
		Type:        CTXInventory,
		ObjectID:    inventory.ID,
		Description: "Inventory " + inventory.Name + " deleted",
		Created:     time.Now(),
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

// Script is a Gin Handler function which generates a ansible compatible
// inventory output
// note: we are not using var varname []string specially because
// output json must include [] for each array and {} for each object
func Script(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)

	// query variables
	//qall := c.Query("all")
	qhostvars := c.Query("hostvars")
	qhost := c.Query("host")

	if qhost != "" {
		//get hosts for parent group
		var host ansible.Host
		err := db.Hosts().Find(bson.M{"name": qhost}).One(&host)
		if err != nil {
			log.Errorln("Error while getting host", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": []string{"Error while getting vars"},
			})
			return
		}
		var gv gin.H
		err = json.Unmarshal([]byte(host.Variables), &gv)
		if err != nil {
			log.Errorln("Error while unmarshalling group vars", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": []string{"Error while getting Hosts"},
			})
			return
		}

		c.JSON(http.StatusOK, gv)
		return
	}

	resp := gin.H{}

	// First Get all groups for inventory ID
	var parents []ansible.Group

	q := bson.M{"inventory_id": inv.ID}

	if err := db.Groups().Find(q).All(&parents); err != nil {
		log.Errorln("Error while getting groups", err)
		// send a brief error description to client
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": []string{"Error while getting Hosts"},
		})
		return
	}

	allhosts := []ansible.Host{}

	// get all

	// loop through parent groups and get their hosts and
	// child groups
	// suppress key since we wont modify the array
	for _, v := range parents {

		//get hosts for parent group
		var hosts []ansible.Host

		q := bson.M{"inventory_id": inv.ID, "group_id": v.ID}

		if err := db.Hosts().Find(q).All(&hosts); err != nil {
			log.Errorln("Error while getting host for group", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": []string{"Error while getting Hosts"},
			})
			return
		}

		//add hosts to global hosts
		allhosts = append(allhosts, hosts...)

		// Second get all child groups
		var childgroups []ansible.Group

		q = bson.M{"inventory_id": inv.ID, "parent_group_id": v.ID}

		if err := db.Groups().Find(q).All(&childgroups); err != nil {
			log.Errorln("Error while getting child groups", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": []string{"Error while getting Hosts"},
			})
			return
		}

		hostnames := []string{}
		//add host to group
		for _, v := range hosts {
			hostnames = append(hostnames, v.Name)
		}

		groupnames := []string{}
		//add host to group
		for _, v := range childgroups {
			groupnames = append(groupnames, v.Name)
		}

		gv := gin.H{}
		if v.Variables != "" {
			if err := json.Unmarshal([]byte(v.Variables), &gv); err != nil {
				log.Errorln("Error while unmarshalling group vars", err)
				// send a brief error description to client
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": []string{"Error while getting Hosts"},
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

	//if hostvars parameter exist
	hostvars := gin.H{}
	for _, v := range allhosts {
		if v.Variables != "" {
			var gv gin.H
			if err := json.Unmarshal([]byte(v.Variables), &gv); err != nil {
				log.Errorln("Error while unmarshalling group vars", err)
				// send a brief error description to client
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": []string{"Error while getting Hosts"},
				})
				return
			}
			//add host variables to hostvars
			hostvars[v.Name] = gv
		}
	}

	// add non-grouped hosts
	nghosts := []ansible.Host{}
	q = bson.M{"inventory_id": inv.ID, "group_id": nil}

	if err := db.Hosts().Find(q).All(&nghosts); err != nil {
		log.Errorln("Error while getting non-grouped hosts", err)
		// send a brief error description to client
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": []string{"Error while getting non-grouped Hosts"},
		})
		return
	}

	hosts := []string{}
	for _, v := range nghosts {
		if v.Variables != "" {
			var gv gin.H
			if err := json.Unmarshal([]byte(v.Variables), &gv); err != nil {
				log.Errorln("Error while unmarshalling group vars", err)
				// send a brief error description to client
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": []string{"Error while getting Hosts"},
				})
				return
			}
			//add host variables to hostvars
			hostvars[v.Name] = gv
		}
		//add host names to hosts
		hosts = append(hosts, v.Name)
	}

	//if hostvars parameter exist
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
func JobTemplates(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var jobTemplate []ansible.JobTemplate
	// new mongodb iterator
	iter := db.JobTemplates().Find(bson.M{"inventory_id": inv.ID}).Iter()
	// loop through each result and modify for our needs
	var tmpJobTemplate ansible.JobTemplate
	// iterate over all and only get valid objects
	for iter.Next(&tmpJobTemplate) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.JobTemplateRead(user, tmpJobTemplate) {
			continue
		}
		metadata.JTemplateMetadata(&tmpJobTemplate)
		// good to go add to list
		jobTemplate = append(jobTemplate, tmpJobTemplate)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Job Templates"},
		})
		return
	}

	count := len(jobTemplate)
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
		Results:  jobTemplate[pgi.Skip():pgi.End()],
	})
}

// RootGroups is a Gin handler function which returns list of root groups
// of the inventory.
func RootGroups(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var groups []ansible.Group
	query := bson.M{
		"inventory_id": inv.ID,
		"$or": []bson.M{
			{"parent_group_id": bson.M{"$exists": false}},
			{"parent_group_id": nil},
		},
	}
	// new mongodb iterator
	iter := db.Groups().Find(query).Iter()
	// loop through each result and modify for our needs
	var tmpGroup ansible.Group
	// iterate over all and only get valid objects
	for iter.Next(&tmpGroup) {
		// if the user doesn't have access to inventory
		// skip to next
		if !roles.InventoryRead(user, inv) {
			continue
		}
		metadata.GroupMetadata(&tmpGroup)
		// good to go add to list
		groups = append(groups, tmpGroup)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Groups"},
		})
		return
	}

	count := len(groups)
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
		Results:  groups[pgi.Skip():pgi.End()],
	})
}

// Groups is a Gin Handler function which returns all the groups of
// an Inventory.
func Groups(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var groups []ansible.Group
	query := bson.M{
		"inventory_id": inv.ID,
	}
	// new mongodb iterator
	iter := db.Groups().Find(query).Iter()
	// loop through each result and modify for our needs
	var tmpGroup ansible.Group
	// iterate over all and only get valid objects
	for iter.Next(&tmpGroup) {
		// if the user doesn't have access to inventory
		// skip to next
		if !roles.InventoryRead(user, inv) {
			continue
		}
		metadata.GroupMetadata(&tmpGroup)
		// good to go add to list
		groups = append(groups, tmpGroup)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Groups"},
		})
		return
	}

	count := len(groups)
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
		Results:  groups[pgi.Skip():pgi.End()],
	})
}

// Hosts is a Gin handler function which returns all hosts
// associated with the inventory.
func Hosts(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var hosts []ansible.Host
	query := bson.M{
		"inventory_id": inv.ID,
	}
	// new mongodb iterator
	iter := db.Hosts().Find(query).Iter()
	// loop through each result and modify for our needs
	var tmpHost ansible.Host
	// iterate over all and only get valid objects
	for iter.Next(&tmpHost) {
		// if the user doesn't have access to host
		// skip to next
		if !roles.InventoryRead(user, inv) {
			continue
		}
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

// ActivityStream is a Gin handler function which returns list of activities associated with
// inventory object that is in the Gin Context.
// TODO: not complete
func ActivityStream(c *gin.Context) {
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)

	var activities []common.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": inventory.ID, "type": CTXInventory}).All(&activities)

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

// Tree is a Gin handler function which generete a json tree of the
// inventory.
// TODO: complete
func Tree(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)

	var groups []ansible.Group
	query := bson.M{
		"inventory_id": inv.ID,
	}
	// new mongodb iterator
	iter := db.Inventories().Find(query).Iter()
	// loop through each result and modify for our needs
	var tmpGroup ansible.Group
	// iterate over all and only get valid objects
	for iter.Next(&tmpGroup) {
		metadata.GroupMetadata(&tmpGroup)

		//TODO: children

		// good to go add to list
		groups = append(groups, tmpGroup)
	}
	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Inventory data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Groups"},
		})
		return
	}

	count := len(groups)
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
		Results:  groups[pgi.Skip():pgi.End()],
	})
}

// VariableData is a Gin Handler function which returns variable data for the inventory.
func VariableData(c *gin.Context) {
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)

	variables := gin.H{}

	if err := json.Unmarshal([]byte(inventory.Variables), &variables); err != nil {
		log.Errorln("Error while getting Inventory Variables")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": []string{"Error while getting Inventory Variables"},
		})
		return
	}

	c.JSON(http.StatusOK, variables)
}
