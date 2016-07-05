package access

import (
	"database/sql"

	database "pearson.com/hilbert-space/db"
	"pearson.com/hilbert-space/models"
	"pearson.com/hilbert-space/util"
	"github.com/gin-gonic/gin"
	"pearson.com/hilbert-space/crypt"
	"gopkg.in/mgo.v2/bson"
)

func KeyMiddleware(c *gin.Context) {
	keyID, err := util.GetIntParam("key_id", c)
	if err != nil {
		return
	}

	var key models.AccessKey

	col := database.MongoDb.C("global_access_key")

	if err := col.FindId(keyID).One(&key); err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatus(404)
			return
		}

		panic(err)
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
		panic(err)
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
		secret := crypt.Encrypt(key.Secret)
		key.Secret = secret;
		break
	default:
		c.AbortWithStatus(400)
		return
	}

	key.ID = bson.NewObjectId()

	if err := key.Insert(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ObjectType:  "key",
		ObjectID:    key.ID,
		Description: "Global Access Key " + key.Name + " created",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func UpdateKey(c *gin.Context) {
	var key models.GlobalAccessKey
	oldKey := c.MustGet("globalAccessKey").(models.GlobalAccessKey)

	if err := c.Bind(&key); err != nil {
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
		panic(err)
	}

	if err := (models.Event{
		Description: "Global Access Key " + key.Name + " updated",
		ObjectID:    oldKey.ID,
		ObjectType:  "key",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveKey(c *gin.Context) {
	key := c.MustGet("globalAccessKey").(models.GlobalAccessKey)

	if err := key.Remove(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		Description: "Global Access Key " + key.Name + " deleted",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
