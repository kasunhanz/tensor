package models

import (
	"pearson.com/hilbert-space/util"
	"gopkg.in/mgo.v2/bson"
	database "pearson.com/hilbert-space/db"
)

// AccessKey is the model for access_key
// collection
type AccessKey struct {
	ID        bson.ObjectId    `bson:"_id" json:"id"`
	Name      string `bson:"name" json:"name" binding:"required"`
	// 'aws/do/gcloud/ssh',
	Type      string `bson:"type" json:"type" binding:"required"`

	ProjectID bson.ObjectId    `bson:"project_id" json:"project_id"`
	Key       string `bson:"key" json:"key"`
	Secret    string `bson:"secret" json:"secret"`
}

// get access key path
func (key AccessKey) GetPath() string {
	return util.Config.TmpPath + "/access_key_" + key.ID.String()
}

func (key AccessKey) Insert() error {
	c := database.MongoDb.C("access_key")
	return c.Insert(key)
}

func (key AccessKey) Remove() error {
	c := database.MongoDb.C("access_key")
	return c.RemoveId(key.ID)
}

func (key AccessKey) Update() error {
	c := database.MongoDb.C("access_key")
	return c.UpdateId(key.ID, key)
}