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

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/log/activity"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/gin-gonic/gin.v1/binding"
	"gopkg.in/mgo.v2/bson"
	"github.com/pearsonappeng/tensor/validate"
)

// Keys for credential related items stored in the Gin Context
const (
	CTXInventory = "inventory"
	CTXInventoryID = "inventory_id"
)

type InventoryController struct{}

// Middleware generates a middleware handler function that works inside of a Gin request.
// This function takes project_id parameter from Gin Context and fetches project data from the database
// this set project data under key project in Gin Context.
func (ctrl InventoryController) Middleware(c *gin.Context) {
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
func (ctrl InventoryController) One(c *gin.Context) {
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)

	metadata.InventoryMetadata(&inventory)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

// GetInventories is a Gin handler function which returns list of inventories
// This takes lookup parameters and order parameters to filter and sort output data.
func (ctrl InventoryController) All(c *gin.Context) {

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
func (ctrl InventoryController) Create(c *gin.Context) {
	var req ansible.Inventory
	user := c.MustGet(CTXUser).(common.User)

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: validate.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization not in the collection
	// if not fail
	if !req.OrganizationExist() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	// if inventory exists in the collection
	if !req.IsUnique() {
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
	activity.AddInventoryActivity(common.Create, user, req)

	metadata.InventoryMetadata(&req)

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// UpdateInventory is a Gin handler function which updates a credential using request payload.
// This replaces all the fields in the database, empty "" fiels and unspecified fields will be
// removed from the database.
func (ctrl InventoryController) Update(c *gin.Context) {
	// get Inventory from the gin.Context
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)
	tmpInventory := inventory
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req ansible.Inventory
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: validate.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization not in the collection
	// if not fail
	if !req.OrganizationExist() {
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	if req.Name != inventory.Name {
		// if inventory exists in the collection
		if !req.IsUnique() {
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

	if err := db.Inventories().UpdateId(inventory.ID, inventory); err != nil {
		log.Errorln("Error while updating Inventory:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while updating Inventory"},
		})
	}

	// add new activity to activity stream
	activity.AddInventoryActivity(common.Update, user, tmpInventory, inventory)

	// set `related` and `summary` fields
	metadata.InventoryMetadata(&inventory)
	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

// PatchInventory is a Gin handler function which partially updates a inventory using request payload.
// This replaces specified fields in the data, empty "" fields will be
// removed from the database object. unspecified fields will be ignored.
func (ctrl InventoryController) Patch(c *gin.Context) {
	// get Inventory from the gin.Context
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)
	tmpInventory := inventory
	// get user from the gin.Context
	user := c.MustGet(CTXUser).(common.User)

	var req ansible.PatchInventory
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, common.Error{
			Code:     http.StatusBadRequest,
			Messages: validate.GetValidationErrors(err),
		})
		return
	}

	if req.OrganizationID != nil {
		inventory.OrganizationID = *req.OrganizationID
		// check whether the organization exist or not
		if !inventory.OrganizationExist() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Organization does not exist."},
			})
			return
		}
	}

	if req.Name != nil && *req.Name != inventory.Name {
		inventory.Name = strings.Trim(*req.Name, " ")

		// if inventory exists in the collection
		if !inventory.IsUnique() {
			c.JSON(http.StatusBadRequest, common.Error{
				Code:     http.StatusBadRequest,
				Messages: []string{"Inventory with this Name already exists."},
			})
			return
		}
	}

	if req.Description != nil {
		inventory.Description = strings.Trim(*req.Description, " ")
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
	activity.AddInventoryActivity(common.Update, user, tmpInventory, inventory)

	// set `related` and `summary` feilds
	metadata.InventoryMetadata(&inventory)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

// RemoveInventory is a Gin handler function which removes a inventory object from the database
func (ctrl InventoryController) Delete(c *gin.Context) {
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
	activity.AddInventoryActivity(common.Delete, user, inventory)

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

// Script is a Gin Handler function which generates a ansible compatible
// inventory output
// note: we are not using var varname []string specially because
// output json must include [] for each array and {} for each object
func (ctrl InventoryController) Script(c *gin.Context) {
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
func (ctrl InventoryController) JobTemplates(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context

	var jobTemplate []ansible.JobTemplate
	// new mongodb iterator
	iter := db.JobTemplates().Find(bson.M{"inventory_id": inv.ID}).Iter()
	// loop through each result and modify for our needs
	var tmpJobTemplate ansible.JobTemplate
	// iterate over all and only get valid objects
	for iter.Next(&tmpJobTemplate) {
		// TODO: if the user doesn't have access to credential
		// skip to next
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
func (ctrl InventoryController) RootGroups(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context

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
		// TODO: if the user doesn't have access to inventory
		// skip to next
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
func (ctrl InventoryController) Groups(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context

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
		// TODO: if the user doesn't have access to inventory
		// skip to next
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
func (ctrl InventoryController) Hosts(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)
	// get user from the gin.Context

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
		// TODO: if the user doesn't have access to host
		// skip to next
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

// ActivityStream returns the activites of the user on Inventories
func (ctrl InventoryController) ActivityStream(c *gin.Context) {
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)

	var activities []ansible.ActivityInventory
	var activity ansible.ActivityInventory
	// new mongodb iterator
	iter := db.ActivityStream().Find(bson.M{"object1._id": inventory.ID}).Iter()
	// iterate over all and only get valid objects
	for iter.Next(&activity) {
		metadata.ActivityInventoryMetadata(&activity)
		metadata.InventoryMetadata(&activity.Object1)
		//apply metadata only when Object2 is available
		if activity.Object2 != nil {
			metadata.InventoryMetadata(activity.Object2)
		}
		//add to activities list
		activities = append(activities, activity)
	}

	if err := iter.Close(); err != nil {
		log.Errorln("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Activities"},
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
func (ctrl InventoryController) Tree(c *gin.Context) {
	inv := c.MustGet(CTXInventory).(ansible.Inventory)

	var groups []ansible.Group
	// new mongodb iterator for groups
	iOne := db.Groups().Find(bson.M{
		"inventory_id": inv.ID,
	}).Iter()

	var gOne ansible.Group
	// iterate over all and only get valid objects
	for iOne.Next(&gOne) {
		// new mongodb iterator for children groups
		iTwo := db.Groups().Find(bson.M{
			"parent_group_id": gOne.ID,
		}).Iter()
		var gTwo ansible.Group
		// iterate over all and only get valid children objects
		for iTwo.Next(&gTwo) {
			// new mongodb iterator for grandchildren groups
			iTree := db.Groups().Find(bson.M{
				"parent_group_id": gTwo.ID,
			}).Iter()
			var gThree ansible.Group
			// iterate over all and only get valid grandchildren objects
			for iTree.Next(&gThree) {
				if (*gThree.ParentGroupID).Hex() == gTwo.ID.Hex() {
					// attach metadata
					metadata.GroupMetadata(&gThree)
					gTwo.Children = append(gTwo.Children, gThree)
				}
			}

			if err := iTree.Close(); err != nil {
				log.Errorln("Error while retriving Inventory data from the db:", err)
				c.JSON(http.StatusInternalServerError, common.Error{
					Code:     http.StatusInternalServerError,
					Messages: []string{"Error while getting Groups"},
				})
				return
			}

			if (*gTwo.ParentGroupID).Hex() == gOne.ID.Hex() {
				// attach metadata
				metadata.GroupMetadata(&gTwo)
				gOne.Children = append(gOne.Children, gTwo)
			}
		}

		if err := iTwo.Close(); err != nil {
			log.Errorln("Error while retriving Inventory data from the db:", err)
			c.JSON(http.StatusInternalServerError, common.Error{
				Code:     http.StatusInternalServerError,
				Messages: []string{"Error while getting Groups"},
			})
			return
		}

		if gOne.ParentGroupID == nil {
			// attach metadata
			metadata.GroupMetadata(&gOne)
			groups = append(groups, gOne)
		}
	}

	if err := iOne.Close(); err != nil {
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
func (ctrl InventoryController) VariableData(c *gin.Context) {
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
