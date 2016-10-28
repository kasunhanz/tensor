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
	"encoding/json"
	"bitbucket.pearson.com/apseng/tensor/controllers/metadata"
	"bitbucket.pearson.com/apseng/tensor/roles"
	"bitbucket.pearson.com/apseng/tensor/controllers/helpers"
	"github.com/gin-gonic/gin/binding"
)

const _CTX_INVENTORY = "inventory"
const _CTX_USER = "user"
const _CTX_INVENTORY_ID = "inventory_id"

// InventoryMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// this set project data under key project in gin.Context
func Middleware(c *gin.Context) {
	ID, err := util.GetIdParam(_CTX_INVENTORY_ID, c)

	if err != nil {
		log.Print("Error while getting the Inventory:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	var inventory models.Inventory
	err = db.Inventories().FindId(bson.ObjectIdHex(ID)).One(&inventory);

	if err != nil {
		log.Print("Error while getting the Inventory:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Not Found"},
		})
		c.Abort()
		return
	}

	c.Set(_CTX_INVENTORY, inventory)
	c.Next()
}

// GetInventory returns the project as a JSON object
func GetInventory(c *gin.Context) {
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)

	if err := metadata.InventoryMetadata(&inventory); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Inventory"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

// GetInventories returns a JSON array of projects
func GetInventories(c *gin.Context) {

	parser := util.NewQueryParser(c)
	match := bson.M{}
	match = parser.Match([]string{"has_inventory_sources", "has_active_failures"}, match)
	match = parser.Lookups([]string{"name", "organization"}, match)

	query := db.Inventories().Find(match)
	if order := parser.OrderBy(); order != "" {
		query.Sort(order)
	}

	var inventories []models.Inventory
	// new mongodb iterator
	iter := query.Iter()
	// loop through each result and modify for our needs
	var tmpInventory models.Inventory
	// iterate over all and only get valid objects
	for iter.Next(&tmpInventory) {
		if err := metadata.InventoryMetadata(&tmpInventory); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Inventory"},
			})
			return
		}
		// good to go add to list
		inventories = append(inventories, tmpInventory)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Inventory data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: inventories[pgi.Skip():pgi.End()],
	})
}

// AddInventory creates a new project
func AddInventory(c *gin.Context) {
	var req models.Inventory
	user := c.MustGet(_CTX_USER).(models.User)

	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization not in the collection
	// if not fail
	if helpers.OrganizationNotExist(req.OrganizationID) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	// if inventory exists in the collection
	if helpers.IsNotUniqueInventory(req.Name, req.OrganizationID) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: []string{"Inventory with this Name already exists."},
		})
		return
	}

	req.ID = bson.NewObjectId()
	req.Created = time.Now()
	req.Modified = time.Now()
	req.CreatedBy = user.ID
	req.ModifiedBy = user.ID

	if err := db.Inventories().Insert(req); err != nil {
		log.Println("Error while creating Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Inventory"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Inventory " + req.Name + " created")

	if err := metadata.InventoryMetadata(&req); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Inventory"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, req)
}

// UpdateInventory will update existing Inventory with
// request parameters
func UpdateInventory(c *gin.Context) {
	// get Inventory from the gin.Context
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Inventory
	err := binding.JSON.Bind(c.Request, &req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	// check whether the organization not in the collection
	// if not fail
	if helpers.OrganizationNotExist(req.OrganizationID) {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: []string{"Organization does not exists."},
		})
		return
	}

	if req.Name != inventory.Name {
		// if inventory exists in the collection
		if helpers.IsNotUniqueInventory(req.Name, req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Inventory with this Name already exists."},
			})
			return
		}
	}

	req.ID = bson.NewObjectId()
	req.Created = inventory.Created
	req.Modified = time.Now()
	req.CreatedBy = inventory.CreatedBy
	req.ModifiedBy = user.ID

	err = db.Inventories().UpdateId(inventory.ID, req);
	if err != nil {
		log.Println("Error while updating Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Inventory"},
		})
	}

	// add new activity to activity stream
	addActivity(req.ID, user.ID, "Inventory " + req.Name + " updated")

	// set `related` and `summary` feilds
	err = metadata.InventoryMetadata(&req);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while creating Inventory"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, req)
}

// Pathcnventory will update existing Inventory with
// request parameters
func PatchInventory(c *gin.Context) {
	// get Inventory from the gin.Context
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.PatchInventory
	if err := binding.JSON.Bind(c.Request, &req); err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Messages: util.GetValidationErrors(err),
		})
		return
	}

	if len(req.OrganizationID) == 12 {
		// check whether the organization exist or not
		if !helpers.OrganizationExist(req.OrganizationID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Organization does not exists."},
			})
			return
		}
	}

	if len(req.Name) > 0 && req.Name != inventory.Name {
		ogID := inventory.OrganizationID
		if len(req.OrganizationID) == 12 {
			ogID = req.OrganizationID
		}
		// if inventory exists in the collection
		if helpers.IsNotUniqueInventory(req.Name, ogID) {
			c.JSON(http.StatusBadRequest, models.Error{
				Code:http.StatusBadRequest,
				Messages: []string{"Inventory with this Name already exists."},
			})
			return
		}
	}

	req.Modified = time.Now()
	req.ModifiedBy = user.ID

	if err := db.Inventories().UpdateId(inventory.ID, bson.M{"$set": req}); err != nil {
		log.Println("Error while updating Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while updating Inventory"},
		})
		return
	}

	// add new activity to activity stream
	addActivity(inventory.ID, user.ID, "Inventory " + req.Name + " updated")

	// get newly updated host
	var resp models.Inventory
	if err := db.Inventories().FindId(inventory.ID).One(&resp); err != nil {
		log.Print("Error while getting the updated Inventory:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Messages: []string{"Error while getting the updated Inventory"},
		})
		return
	}

	// set `related` and `summary` feilds
	if err := metadata.InventoryMetadata(&resp); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Inventory Information"},
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, resp)
}

func RemoveInventory(c *gin.Context) {
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	changes, err := db.Hosts().RemoveAll(bson.M{"inventory_id": inventory.ID})
	if err != nil {
		log.Println("Error while removing Hosts:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while removing Inventory Hosts"},
		})
	}
	log.Println("Hosts remove info:", changes.Removed)

	changes, err = db.Groups().RemoveAll(bson.M{"inventory_id": inventory.ID})
	if err != nil {
		log.Println("Error while removing Groups:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while removing Inventory Groups"},
		})
	}
	log.Println("Groups remove info:", changes.Removed)

	err = db.Inventories().RemoveId(inventory.ID);
	if err != nil {
		log.Println("Error while removing Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while removing Inventory"},
		})
	}

	// add new activity to activity stream
	addActivity(inventory.ID, user.ID, "Inventory " + inventory.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

// note: we are not using var varname []string specially because
// output json must include [] for each array and {} for each object
func Script(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)

	// query variables
	//qall := c.Query("all")
	qhostvars := c.Query("hostvars")
	qhost := c.Query("host")

	if qhost != "" {
		//get hosts for parent group
		var host models.Host
		err := db.Hosts().Find(bson.M{"name": qhost}).One(&host);
		if err != nil {
			log.Println("Error while getting host", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": []string{"Error while getting vars"},
			})
			return
		}
		var gv gin.H
		err = json.Unmarshal([]byte(host.Variables), &gv);
		if err != nil {
			log.Println("Error while unmarshalling group vars", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": []string{"Error while getting Hosts"},
			})
			return
		}

		c.JSON(http.StatusOK, gv)
		return
	}

	resp := gin.H{}

	// First Get all groups for inventory ID
	var parents []models.Group

	q := bson.M{"inventory_id": inv.ID}

	if err := db.Groups().Find(q).All(&parents); err != nil {
		log.Println("Error while getting groups", err)
		// send a brief error description to client
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"message": []string{"Error while getting Hosts"},
		})
		return
	}

	allhosts := []models.Host{}

	// get all

	// loop through parent groups and get their hosts and
	// child groups
	// suppress key since we wont modify the array
	for _, v := range parents {

		//get hosts for parent group
		var hosts []models.Host

		q := bson.M{"inventory_id": inv.ID, "group_id": v.ID}

		if err := db.Hosts().Find(q).All(&hosts); err != nil {
			log.Println("Error while getting host for group", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": []string{"Error while getting Hosts"},
			})
			return
		}

		//add hosts to global hosts
		allhosts = append(allhosts, hosts...)

		// Second get all child groups
		var childgroups []models.Group

		q = bson.M{"inventory_id": inv.ID, "parent_group_id": v.ID}

		if err := db.Groups().Find(q).All(&childgroups); err != nil {
			log.Println("Error while getting child groups", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
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
				log.Println("Error while unmarshalling group vars", err)
				// send a brief error description to client
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": http.StatusInternalServerError,
					"message": []string{"Error while getting Hosts"},
				})
				return
			}
		}

		resp[v.Name] = gin.H{
			"hosts": hostnames,
			"children": groupnames,
			"vars": gv,
		}

	}

	//if hostvars parameter exist
	hostvars := gin.H{}
	for _, v := range allhosts {
		if v.Variables != "" {
			var gv gin.H
			if err := json.Unmarshal([]byte(v.Variables), &gv); err != nil {
				log.Println("Error while unmarshalling group vars", err)
				// send a brief error description to client
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": http.StatusInternalServerError,
					"message": []string{"Error while getting Hosts"},
				})
				return
			}
			//add host variables to hostvars
			hostvars[v.Name] = gv
		}
	}

	// add non-grouped hosts
	nghosts := []models.Host{}
	q = bson.M{"inventory_id": inv.ID, "group_id": nil}

	if err := db.Hosts().Find(q).All(&nghosts); err != nil {
		log.Println("Error while getting non-grouped hosts", err)
		// send a brief error description to client
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"message": []string{"Error while getting non-grouped Hosts"},
		})
		return
	}

	hosts := []string{}
	for _, v := range nghosts {
		if v.Variables != "" {
			var gv gin.H
			if err := json.Unmarshal([]byte(v.Variables), &gv); err != nil {
				log.Println("Error while unmarshalling group vars", err)
				// send a brief error description to client
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": http.StatusInternalServerError,
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

func JobTemplates(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var jobTemplate []models.JobTemplate
	// new mongodb iterator
	iter := db.JobTemplates().Find(bson.M{"inventory_id": inv.ID}).Iter()
	// loop through each result and modify for our needs
	var tmpJobTemplate models.JobTemplate
	// iterate over all and only get valid objects
	for iter.Next(&tmpJobTemplate) {
		// if the user doesn't have access to credential
		// skip to next
		if !roles.JobTemplateRead(user, tmpJobTemplate) {
			continue
		}
		if err := metadata.JTemplateMetadata(&tmpJobTemplate); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Job Templates"},
			})
			return
		}
		// good to go add to list
		jobTemplate = append(jobTemplate, tmpJobTemplate)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: jobTemplate[pgi.Skip():pgi.End()],
	})
}

func RootGroups(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var groups []models.Group
	query := bson.M{
		"inventory_id": inv.ID,
		"$or":[]bson.M{
			{"parent_group_id": bson.M{"$exists": false}},
			{"parent_group_id":nil},
		},
	}
	// new mongodb iterator
	iter := db.Groups().Find(query).Iter()
	// loop through each result and modify for our needs
	var tmpGroup models.Group
	// iterate over all and only get valid objects
	for iter.Next(&tmpGroup) {
		// if the user doesn't have access to inventory
		// skip to next
		if !roles.InventoryRead(user, inv) {
			continue
		}
		if err := metadata.GroupMetadata(&tmpGroup); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Groups"},
			})
			return
		}
		// good to go add to list
		groups = append(groups, tmpGroup)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: groups[pgi.Skip():pgi.End()],
	})
}

func Groups(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var groups []models.Group
	query := bson.M{
		"inventory_id": inv.ID,
	}
	// new mongodb iterator
	iter := db.Groups().Find(query).Iter()
	// loop through each result and modify for our needs
	var tmpGroup models.Group
	// iterate over all and only get valid objects
	for iter.Next(&tmpGroup) {
		// if the user doesn't have access to inventory
		// skip to next
		if !roles.InventoryRead(user, inv) {
			continue
		}
		if err := metadata.GroupMetadata(&tmpGroup); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Groups"},
			})
			return
		}
		// good to go add to list
		groups = append(groups, tmpGroup)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Credential data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: groups[pgi.Skip():pgi.End()],
	})
}

func Hosts(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var hosts []models.Host
	query := bson.M{
		"inventory_id": inv.ID,
	}
	// new mongodb iterator
	iter := db.Hosts().Find(query).Iter()
	// loop through each result and modify for our needs
	var tmpHost models.Host
	// iterate over all and only get valid objects
	for iter.Next(&tmpHost) {
		// if the user doesn't have access to host
		// skip to next
		if !roles.InventoryRead(user, inv) {
			continue
		}
		if err := metadata.HostMetadata(&tmpHost); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Hosts"},
			})
			return
		}
		// good to go add to list
		hosts = append(hosts, tmpHost)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Host data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: hosts[pgi.Skip():pgi.End()],
	})
}

// TODO: not complete
func ActivityStream(c *gin.Context) {
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)

	var activities []models.Activity
	err := db.ActivityStream().Find(bson.M{"object_id": inventory.ID, "type": _CTX_INVENTORY}).All(&activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: activities[pgi.Skip():pgi.End()],
	})
}

// TODO: complete
func Tree(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)

	var groups []models.Group
	query := bson.M{
		"inventory_id": inv.ID,
	}
	// new mongodb iterator
	iter := db.Inventories().Find(query).Iter()
	// loop through each result and modify for our needs
	var tmpGroup models.Group
	// iterate over all and only get valid objects
	for iter.Next(&tmpGroup) {
		if err := metadata.GroupMetadata(&tmpGroup); err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Groups"},
			})
			return
		}

		//TODO: children


		// good to go add to list
		groups = append(groups, tmpGroup)
	}
	if err := iter.Close(); err != nil {
		log.Println("Error while retriving Inventory data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
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
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: groups[pgi.Skip():pgi.End()],
	})
}

func VariableData(c *gin.Context) {
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)

	variables := gin.H{}

	if err := json.Unmarshal([]byte(inventory.Variables), &variables); err != nil {
		log.Println("Error while getting Inventory Variables")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"message": []string{"Error while getting Inventory Variables"},
		})
		return
	}

	c.JSON(http.StatusOK, variables)
}