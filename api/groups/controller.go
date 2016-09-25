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
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
)

const _CTX_GROUP = "group"
const _CTX_USER = "user"
const _CTX_GROUP_ID = "group_id"

// GroupMiddleware takes host_id parameter from gin.Context and
// fetches host data from the database
// it set host data under key host in gin.Context
func Middleware(c *gin.Context) {

	ID := c.Params.ByName(_CTX_GROUP_ID)
	collection := db.C(db.GROUPS)
	var group models.Group
	err := collection.FindId(bson.ObjectIdHex(ID)).One(&group);

	if err != nil {
		log.Print("Error while getting the Group:", err) // log error to the system log
		c.JSON(http.StatusNotFound, models.Error{
			Code:http.StatusNotFound,
			Message: "Not Found",
		})
		return
	}

	c.Set(_CTX_GROUP, group)
	c.Next()
}

// GetHost returns the host as a serialized JSON object
func GetGroup(c *gin.Context) {
	group := c.MustGet(_CTX_GROUP).(models.Group)
	metadata.GroupMetadata(&group)

	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:1,
		Results:group,
	})
}

// GetGroups returns groups as a serialized JSON object
func GetGroups(c *gin.Context) {
	dbc := db.C(db.GROUPS)

	parser := util.NewQueryParser(c)
	match := parser.Match([]string{"source", "has_active_failures", })
	con := parser.IContains([]string{"name"});

	if con != nil {
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
		log.Println("Error while trying to get count of Groups from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting groups",
		})
		return
	}

	// init Pagination
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// set sort value to the query based on request parameters
	order := parser.OrderBy();
	if order != "" {
		query.Sort(order)
	}

	var groups []models.Group

	// get all values with skip limit
	err = query.Skip(pgi.Offset()).Limit(pgi.Limit()).All(&groups);

	if err != nil {
		log.Println("Error while retriving Group data from the db:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while getting groups",
		})
		return
	}

	// set related and summary fields to every item
	for i, v := range groups {
		// note: `v` reference doesn't modify original slice
		err := metadata.GroupMetadata(&v);
		if err != nil {
			log.Println("Error while setting metatdata:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Message: "Error while getting Groups",
			})
			return
		}
		groups[i] = v // modify each object in slice
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:groups,
	})
}

// AddGroup creates a new group
func AddGroup(c *gin.Context) {
	var req models.Group
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	err := c.BindJSON(&req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	// create new object to omit unnecessary fields
	group := models.Group{
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

	collection := db.C(db.GROUPS)

	err = collection.Insert(group);
	if err != nil {
		log.Println("Error while creating Group:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Group",
		})
		return
	}

	// add new activity to activity stream
	addActivity(group.ID, user.ID, "Group " + group.Name + " created")

	err = metadata.GroupMetadata(&group);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Group",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusCreated, models.Response{
		Count:1,
		Results:group,
	})
}

// UpdateGroup will update the Group
func UpdateGroup(c *gin.Context) {
	// get Group from the gin.Context
	group := c.MustGet(_CTX_GROUP).(models.Group)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	var req models.Group
	err := c.BindJSON(&req);
	if err != nil {
		// Return 400 if request has bad JSON format
		c.JSON(http.StatusBadRequest, models.Error{
			Code:http.StatusBadRequest,
			Message: "Bad Request",
		})
		return
	}

	group.Name = req.Name
	group.Description = req.Description
	group.Variables = req.Variables
	group.Modified = time.Now()
	group.ModifiedByID = user.ID

	collection := db.C(db.GROUPS)

	// update object
	err = collection.UpdateId(group.ID, group);
	if err != nil {
		log.Println("Error while updating Group:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while updating Group",
		})
		return
	}

	// add new activity to activity stream
	addActivity(group.ID, user.ID, "Group " + group.Name + " updated")

	// set `related` and `summary` feilds
	err = metadata.GroupMetadata(&group);
	if err != nil {
		log.Println("Error while setting metatdata:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while creating Group",
		})
		return
	}

	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:1,
		Results:group,
	})
}

// RemoveGroup will remove the Group
// from the models._CTX_GROUP collection
func RemoveGroup(c *gin.Context) {
	// get Group from the gin.Context
	group := c.MustGet(_CTX_GROUP).(models.Group)
	// get user from the gin.Context
	user := c.MustGet(_CTX_USER).(models.User)

	collection := db.C(db.GROUPS)
	chosts := db.C(db.HOSTS)

	var childgroups []models.Group

	//find the group and all child groups
	query := bson.M{
		"$or": []bson.M{
			{"parent_group_id": group.ID},
			{"_id": group.ID},
		},
	}
	err := collection.Find(query).Select(bson.M{"_id":1}).All(&childgroups);
	if err != nil {
		log.Println("Error while getting child Groups:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Group",
		})
		return
	}

	// get group ids
	var ids []bson.ObjectId

	for _, v := range childgroups {
		ids = append(ids, v.ID)
	}

	//remove hosts that has group ids of group and child groups
	err = chosts.Remove(bson.M{"group_id": bson.M{"$in": ids}});
	if err != nil {
		log.Println("Error while removing Group Hosts:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Group Hosts",
		})
		return
	}

	// remove groups from the collection
	err = collection.Remove(query);
	if err != nil {
		log.Println("Error while removing Hosts:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Message: "Error while removing Group",
		})
		return
	}

	// add new activity to activity stream
	addActivity(group.ID, user.ID, "Group " + group.Name + " deleted")

	// abort with 204 status code
	c.AbortWithStatus(http.StatusNoContent)
}