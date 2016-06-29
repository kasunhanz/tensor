package access

import (
	"database/sql"

	database "github.com/gamunu/hilbertspace/db"
	"github.com/gamunu/hilbertspace/models"
	"github.com/gamunu/hilbertspace/util"
	"github.com/gin-gonic/gin"
	"github.com/masterminds/squirrel"
	"github.com/gamunu/hilbertspace/crypt"
)

func KeyMiddleware(c *gin.Context) {
	keyID, err := util.GetIntParam("key_id", c)
	if err != nil {
		return
	}

	var key models.AccessKey
	if err := database.Mysql.SelectOne(&key, "select * from global_access_key where id=?", keyID); err != nil {
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
	var keys []models.GlobalAccessKeyResponse

	q := squirrel.Select("id, name, type, `key`").
	From("global_access_key")

	if len(c.Query("type")) > 0 {
		q = q.Where("type=?", c.Query("type"))
	}

	query, args, _ := q.ToSql()
	if _, err := database.Mysql.Select(&keys, query, args...); err != nil {
		panic(err)
	}

	c.JSON(200, keys)
}

func AddKey(c *gin.Context) {
	var key models.AccessKey

	if err := c.Bind(&key); err != nil {
		return
	}

	switch key.Type {
	case "aws", "gcloud", "do", "ssh":
		break
	case "credential":
		secret := crypt.Encrypt(*key.Secret)
		key.Secret = &secret;
		break
	default:
		c.AbortWithStatus(400)
		return
	}

	res, err := database.Mysql.Exec("insert into global_access_key set name=?, type=?, `key`=?, secret=?", key.Name, key.Type, key.Key, key.Secret)
	if err != nil {
		panic(err)
	}

	insertID, _ := res.LastInsertId()
	insertIDInt := int(insertID)
	objType := "key"

	desc := "Global Access Key " + key.Name + " created"
	if err := (models.Event{
		ObjectType:  &objType,
		ObjectID:    &insertIDInt,
		Description: &desc,
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
		secret := crypt.Encrypt(*key.Secret)
		key.Secret = &secret;
		break
	default:
		c.AbortWithStatus(400)
		return
	}

	if _, err := database.Mysql.Exec("update global_access_key set name=?, type=?, `key`=?, secret=?", key.Name, key.Type, key.Key, key.Secret, oldKey.ID); err != nil {
		panic(err)
	}

	desc := "Global Access Key " + key.Name + " updated"
	objType := "key"
	if err := (models.Event{
		Description: &desc,
		ObjectID:    &oldKey.ID,
		ObjectType:  &objType,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveKey(c *gin.Context) {
	key := c.MustGet("globalAccessKey").(models.GlobalAccessKey)

	if _, err := database.Mysql.Exec("delete from global_access_key where id=?", key.ID); err != nil {
		panic(err)
	}

	desc := "Global Access Key " + key.Name + " deleted"
	if err := (models.Event{
		Description: &desc,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
