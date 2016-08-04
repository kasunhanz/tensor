package projects

import (
	"github.com/gamunu/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func KeyMiddleware(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	keyID := c.Params.ByName("key_id")

	key, err := project.GetAccessKey(bson.ObjectIdHex(keyID))

	if err != nil {
		panic(err)
	}

	c.Set("accessKey", key)
	c.Next()
}

func GetKeys(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	if len(c.Query("type")) > 0 {
		keys, err := project.GetAccessKeysByType(c.Query("type"))

		if err != nil {
			panic(err)
		}
		c.JSON(200, keys)
		return
	}

	keys, err := project.GetAccessKeys()
	if err != nil {
		panic(err)
	}

	c.JSON(200, keys)
}

func AddKey(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	var key models.AccessKey

	if err := c.Bind(&key); err != nil {
		return
	}

	switch key.Type {
	case "aws", "gcloud", "do", "ssh":
		break
	default:
		c.AbortWithStatus(400)
		return
	}

	key.ID = bson.NewObjectId()
	key.ProjectID = project.ID

	if err := key.Insert(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   project.ID,
		ObjectType:  "key",
		ObjectID:    key.ID,
		Description: "Access Key " + key.Name + " created",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func UpdateKey(c *gin.Context) {
	var key models.AccessKey
	oldKey := c.MustGet("accessKey").(models.AccessKey)

	if err := c.Bind(&key); err != nil {
		return
	}

	switch key.Type {
	case "aws", "gcloud", "do", "ssh":
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
		ProjectID:   oldKey.ProjectID,
		Description: "Access Key " + key.Name + " updated",
		ObjectID:    oldKey.ID,
		ObjectType:  "key",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveKey(c *gin.Context) {
	key := c.MustGet("accessKey").(models.AccessKey)

	if err := key.Remove(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   key.ProjectID,
		Description: "Access Key " + key.Name + " deleted",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
