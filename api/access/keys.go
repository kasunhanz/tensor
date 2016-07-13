package access

import (
	database "pearson.com/hilbert-space/db"
	"pearson.com/hilbert-space/models"
	"github.com/gin-gonic/gin"
	"pearson.com/hilbert-space/crypt"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func KeyMiddleware(c *gin.Context) {
	keyID := c.Params.ByName("key_id")

	var key models.GlobalAccessKey

	col := database.MongoDb.C("global_access_key")

	if err := col.FindId(bson.ObjectIdHex(keyID)).One(&key); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Set("globalAccessKey", key)
	c.Next()
}

func GetKeys(c *gin.Context) {
	var keys []models.GlobalAccessKey

	col := database.MongoDb.C("global_access_key")

	var query bson.M

	if len(c.Query("type")) > 0 {
		query = bson.M{"type": c.Query("type")}
	}

	if err := col.Find(query).Select(bson.M{"_id":1, "name":1, "type":1, "key":1}).All(&keys); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, keys)
}

func AddKey(c *gin.Context) {
	var key models.GlobalAccessKey

	if err := c.Bind(&key); err != nil {
		return
	}

	switch key.Type {
	case "aws", "gcloud", "do", "ssh":
		break
	case "credential":
		key.Secret = crypt.Encrypt(key.Secret);
		break
	default:
		c.AbortWithStatus(400)
		return
	}

	key.ID = bson.NewObjectId()

	if err := key.Insert(); err != nil {
		c.AbortWithError(500, err)
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectType:  "key",
		ObjectID:    key.ID,
		Description: "Global Access Key " + key.Name + " created",
		Created: time.Now(),
	}.Insert()); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.AbortWithStatus(204)
}

func UpdateKey(c *gin.Context) {
	var key models.GlobalAccessKey
	oldKey := c.MustGet("globalAccessKey").(models.GlobalAccessKey)

	if err := c.Bind(&key); err != nil {
		c.AbortWithError(500, err)
		return
	}

	switch key.Type {
	case "aws", "gcloud", "do", "ssh":
		break
	case "credential":
		secret := crypt.Encrypt(key.Secret)
		key.Secret = secret;
		break
	default:
		c.AbortWithStatus(400)
		return
	}

	oldKey.Name = key.Name
	oldKey.Type = key.Type
	oldKey.Key = key.Key
	oldKey.Secret = key.Secret

	if err := oldKey.Update(); err != nil {
		c.AbortWithError(500, err)
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		Description: "Global Access Key " + key.Name + " updated",
		ObjectID:    oldKey.ID,
		ObjectType:  "key",
		Created: time.Now(),
	}.Insert()); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.AbortWithStatus(204)
}

func RemoveKey(c *gin.Context) {
	key := c.MustGet("globalAccessKey").(models.GlobalAccessKey)

	if err := key.Remove(); err != nil {
		c.AbortWithError(500, err)
		return
	}

	if err := (models.Event{
		ID: bson.NewObjectId(),
		ObjectID: key.ID,
		ObjectType:  "key",
		Description: "Global Access Key " + key.Name + " deleted",
		Created: time.Now(),
	}.Insert()); err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.AbortWithStatus(204)
}
