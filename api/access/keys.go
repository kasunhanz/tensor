package access

import (
	"bitbucket.pearson.com/apseng/tensor/crypt"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/util"
	"github.com/gin-gonic/gin"
	"net/http"
	"gopkg.in/mgo.v2/bson"
	"time"
)

// KeyMiddleware is taking key_id request parameter and
// will find, assign the correct object to gin context as globalAccessKey
// key_id must be a bson.ObjectId otherwise request will terminate
func KeyMiddleware(c *gin.Context) {
	keyID, err := util.GetObjectIdParam("key_id", c)

	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid ID", "status": "error"})
		c.Abort() // terminate request
		return
	}

	var key models.GlobalAccessKey

	col := database.MongoDb.C("global_access_keys")

	if err := col.FindId(bson.ObjectIdHex(keyID)).One(&key); err != nil {
		// Give user an informative error
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID doesn't exisit", "status": "error"})
		c.Abort() // abort the request if the key doesn't exist in the db
		return
	}

	c.Set("globalAccessKey", key) // set key object to gin context
	c.Next()                      // move to next handler
}

// GetKeys will returns all global_access_keys
// if type query string available it will filter accordingly
func GetKeys(c *gin.Context) {
	var keys []models.GlobalAccessKey

	col := database.MongoDb.C("global_access_keys") // get the key from gin context

	var query bson.M

	// Query
	if len(c.Query("type")) > 0 {
		query = bson.M{"type": c.Query("type")}
	}

	if err := col.Find(query).Select(bson.M{"_id": 1, "name": 1, "type": 1, "key": 1}).All(&keys); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Cound not find keys", "status": "error"})
		c.Abort()
		return
	}

	c.JSON(200, keys) // return all keys
}

// AddKey will add new global_access_key
func AddKey(c *gin.Context) {
	var key models.GlobalAccessKey

	if err := c.Bind(&key); err != nil {
		// Give user an informative error
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "status": "error"})
		c.Abort() // abort the request if JSON payload is invalid
		return
	}

	switch key.Type {
	// We do not currently support these connection types
	case "aws", "gcloud", "do", "ssh":
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dooesn't support aws, gcloud, do", "status": "error"})
		c.Abort()
		return
	case "credential":
		key.Secret = crypt.Encrypt(key.Secret)
		break
	default:
		// Give the user an informative error
		c.JSON(http.StatusBadRequest, gin.H{"message": "Unknown authentication scheme", "status": "error"})
		c.Abort()
		return
	}

	key.ID = bson.NewObjectId()

	if err := key.Insert(); err != nil {
		// Give user an informative error
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to create key", "status": "error"})
		c.Abort() // abort the request if insert fails
		return
	}

	if err := (models.Event{
		ID:          bson.NewObjectId(),
		ObjectType:  "key",
		ObjectID:    key.ID,
		Description: "Global Access Key " + key.Name + " created",
		Created:     time.Now(),
	}.Insert()); err != nil {
		// Log error but do not abort the request
		// Since information already in the database
		c.Error(err)
	}

	c.AbortWithStatus(204)
}

// UpdateKey will update a global_access_key
func UpdateKey(c *gin.Context) {
	var key models.GlobalAccessKey
	oldKey := c.MustGet("globalAccessKey").(models.GlobalAccessKey)

	if err := c.Bind(&key); err != nil {
		// Give user an informative error
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "status": "error"})
		c.Abort() // abort the request if JSON payload is invalid
		return
	}

	switch key.Type {
	// We do not currently support these
	case "aws", "gcloud", "do", "ssh":
		break
	case "credential":
		key.Secret = crypt.Encrypt(key.Secret)
		break
	default:
		// Give the user an informative error
		c.JSON(http.StatusBadRequest, gin.H{"message": "Unknown authentication scheme", "status": "error"})
		c.Abort()
		return
	}

	oldKey.Name = key.Name
	oldKey.Type = key.Type
	oldKey.Key = key.Key
	oldKey.Secret = key.Secret

	if err := oldKey.Update(); err != nil {
		// Give user an informative error
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to update key", "status": "error"})
		c.Abort() // abort the request if update fails
		return
	}

	if err := (models.Event{
		ID:          bson.NewObjectId(),
		Description: "Global Access Key " + key.Name + " updated",
		ObjectID:    oldKey.ID,
		ObjectType:  "key",
		Created:     time.Now(),
	}.Insert()); err != nil {
		// Log error but do not abort the request
		// Since information already updated
		c.Error(err)
		return
	}

	c.AbortWithStatus(204)
}

// RemoveKey will remove a key from the database
// key information will be gathered from gin context globalAccessKey
func RemoveKey(c *gin.Context) {
	key := c.MustGet("globalAccessKey").(models.GlobalAccessKey)

	if err := key.Remove(); err != nil {
		// Give user an informative error
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request", "status": "error"})
		c.Abort() // abort the request if remove failed
		return
	}

	if err := (models.Event{
		ID:          bson.NewObjectId(),
		ObjectID:    key.ID,
		ObjectType:  "key",
		Description: "Global Access Key " + key.Name + " deleted",
		Created:     time.Now(),
	}.Insert()); err != nil {
		// Log error but do not abort the request
		// Since information removed from the database
		c.Error(err)
		return
	}

	c.AbortWithStatus(204)
}
