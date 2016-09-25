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
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
	"bitbucket.pearson.com/apseng/tensor/roles"
)

const _CTX_INVENTORY = "inventory"
const _CTX_USER = "user"
const _CTX_INVENTORY_ID = "inventory_id"

// InventoryMiddleware takes project_id parameter from gin.Context and
// fetches project data from the database
// this set project data under key project in gin.Context
func Middleware(c *gin.Context) {
	projectID := c.Params.ByName(_CTX_INVENTORY_ID)

	collection := db.C(db.INVENTORIES)

	var inventory models.Inventory
	err := collection.FindId(bson.ObjectIdHex(projectID)).One(&inventory);

	if err != nil {
		log.Print("Error while getting the Inventory:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
	}

	c.Set(_CTX_INVENTORY, inventory)
	c.Next()
}

// GetInventory returns the project as a JSON object
func GetInventory(c *gin.Context) {
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	metadata.InventoryMetadata(&inventory)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

// GetInventories returns a JSON array of projects
func GetInventories(c *gin.Context) {
	dbc := db.C(db.INVENTORIES)

	parser := util.NewQueryParser(c)
	match := parser.Match([]string{"has_inventory_sources", "has_active_failures", })
	con := parser.IContains([]string{"name", "organization"});

	if con != nil {
		if match != nil {
			for i, v := range con {
				match[i] = v
			}
		} else {
			match = con
		}
	}

	query := dbc.Find(match)
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
				Message: "Error while getting Inventory",
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
			Message: "Error while getting Inventory",
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

	if err := c.BindJSON(&req); err != nil {
		log.Println("Bad payload:", err)
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	inventory := models.Inventory{
		ID: bson.NewObjectId(),
		Name: req.Name,
		Description: req.Description,
		OrganizationID: req.OrganizationID,
		Variables: req.Variables,
		Created: time.Now(),
		Modified: time.Now(),
		CreatedBy: user.ID,
		ModifiedBy: user.ID,
	}

	collection := db.C(db.INVENTORIES)

	err := collection.Insert(inventory);
	if err != nil {
		log.Println("Error while creating Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Inventory",
		})
		return
	}

	// add new activity to activity stream
	addActivity(inventory.ID, user.ID, "Inventory " + inventory.Name + " created")

	if err := metadata.InventoryMetadata(&inventory); err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Inventory",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, inventory)
}

// UpdateInventory will update existing Inventory with
// request parameters
func UpdateInventory(c *gin.Context) {
	// get Inventory from the gin.Context
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Inventory
	err := c.BindJSON(&req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	inventory.Name = req.Name
	inventory.Description = req.Description
	inventory.Modified = time.Now()
	inventory.ModifiedBy = user.ID

	collection := db.C(db.INVENTORIES)
	err = collection.UpdateId(inventory.ID, inventory);
	if err != nil {
		log.Println("Error while updating Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while updating Inventory",
		})
	}

	// add new activity to activity stream
	addActivity(inventory.ID, user.ID, "Inventory " + inventory.Name + " updated")

	// set `related` and `summary` feilds
	err = metadata.InventoryMetadata(&inventory);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Inventory",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, inventory)
}

func RemoveInventory(c *gin.Context) {
	inventory := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	chost := db.C(db.HOSTS)
	err := chost.Remove(bson.M{"inventory_id": inventory.ID})
	if err != nil {
		log.Println("Error while removing Hosts:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Inventory Hosts",
		})
	}

	cgroup := db.C(db.GROUPS)
	err = cgroup.Remove(bson.M{"inventory_id": inventory.ID})
	if err != nil {
		log.Println("Error while removing Groups:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Inventory Groups",
		})
	}

	collection := db.C(db.INVENTORIES)
	err = collection.RemoveId(inventory.ID);
	if err != nil {
		log.Println("Error while removing Inventory:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Inventory",
		})
	}

	// add new activity to activity stream
	addActivity(inventory.ID, user.ID, "Inventory " + inventory.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}

func Script(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)

	cgroups := db.C(db.GROUPS)
	chosts := db.C(db.HOSTS)

	// query variables
	qall := c.Query("all")
	qhostvars := c.Query("hostvars")
	qhost := c.Query("host")

	if qhost != "" {
		//get hosts for parent group
		var host models.Host
		err := chosts.Find(bson.M{"name": qhost}).One(&host);
		if err != nil {
			log.Println("Error while getting host", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": "Error while getting vars",
			})
			return
		}
		var gv gin.H
		err = json.Unmarshal([]byte(*host.Variables), &gv);
		if err != nil {
			log.Println("Error while unmarshalling group vars", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": "Error while getting Hosts",
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

	if err := cgroups.Find(q).All(&parents); err != nil {
		log.Println("Error while getting groups", err)
		// send a brief error description to client
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": http.StatusInternalServerError,
			"message": "Error while getting Hosts",
		})
		return
	}

	var allhosts []models.Host

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

		//add hosts to global hosts
		allhosts = append(allhosts, hosts...)

		// Second get all child groups
		var childgroups []models.Group

		q = bson.M{"inventory_id": inv.ID, "parent_group_id": v.ID}

		if err := cgroups.Find(q).All(&childgroups); err != nil {
			log.Println("Error while getting child groups", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": "Error while getting Hosts",
			})
			return
		}

		var hostnames []string
		//add host to group
		for _, v := range hosts {
			hostnames = append(hostnames, v.Name)
		}

		var groupnames []string
		var groupvars []gin.H
		//add host to group
		for _, v := range childgroups {
			groupnames = append(groupnames, v.Name)

			var gv gin.H
			if err := json.Unmarshal([]byte(*v.Variables), &gv); err != nil {
				log.Println("Error while unmarshalling group vars", err)
				// send a brief error description to client
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": http.StatusInternalServerError,
					"message": "Error while getting Hosts",
				})
				return
			}
			groupvars = append(groupvars, gv)
		}

		resp[v.Name] = gin.H{
			"hosts": hostnames,
			"children": groupnames,
			"vars": groupvars,
		}

	}

	//if hostvars parameter exist
	var hostvars gin.H
	var hosts []string
	for _, v := range allhosts {
		var gv gin.H
		if err := json.Unmarshal([]byte(*v.Variables), &gv); err != nil {
			log.Println("Error while unmarshalling group vars", err)
			// send a brief error description to client
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": http.StatusInternalServerError,
				"message": "Error while getting Hosts",
			})
			return
		}
		//add host variables to hostvars
		hostvars[v.Name] = gv
		//add host names to hosts
		hosts = append(hosts, v.Name)
	}

	//if hostvars parameter exist
	if qhostvars != "" {
		resp["_meta"] = gin.H{
			"hostvars": hostvars,
		}
	}

	//if all parameter exist
	if qall != "" {
		resp["all"] = gin.H{
			"hosts": hosts,
		}
	}

	c.JSON(http.StatusOK, resp)
}

func JobTemplates(c *gin.Context) {
	inv := c.MustGet(_CTX_INVENTORY).(models.Inventory)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.C(db.JOB_TEMPLATES)

	var jobTemplate []models.JobTemplate
	// new mongodb iterator
	iter := collection.Find(bson.M{"inventory_id": inv.ID}).Iter()
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
				Message: "Error while getting Job Templates",
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
			Message: "Error while getting Job Templates",
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

	collection := db.C(db.GROUPS)

	var groups []models.Group
	query := bson.M{
		"inventory_id": inv.ID,
		"$or":[]bson.M{
			{"parent_group_id": bson.M{"$exists": false}},
			{"parent_group_id":nil},
		},
	}
	// new mongodb iterator
	iter := collection.Find(query).Iter()
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
				Message: "Error while getting Groups",
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
			Message: "Error while getting Groups",
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

	collection := db.C(db.GROUPS)

	var groups []models.Group
	query := bson.M{
		"inventory_id": inv.ID,
	}
	// new mongodb iterator
	iter := collection.Find(query).Iter()
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
				Message: "Error while getting Groups",
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
			Message: "Error while getting Groups",
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

	collection := db.C(db.HOSTS)

	var hosts []models.Host
	query := bson.M{
		"inventory_id": inv.ID,
	}
	// new mongodb iterator
	iter := collection.Find(query).Iter()
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
				Message: "Error while getting Hosts",
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
			Message: "Error while getting Hosts",
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
	collection := db.C(db.ACTIVITY_STREAM)
	err := collection.Find(bson.M{"object_id": inventory.ID, "type": _CTX_INVENTORY}).All(activities)

	if err != nil {
		log.Println("Error while retriving Activity data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while Activities",
		})
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