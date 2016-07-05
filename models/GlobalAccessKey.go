package models

import (
	"pearson.com/hilbert-space/util"
	"gopkg.in/mgo.v2/bson"
	database "pearson.com/hilbert-space/db"
)

// GlobalAccessKey is the model for
// global_access_key collection
type GlobalAccessKey struct {
	ID     bson.ObjectId    `bson:"_id" json:"id"`
	Name   string `bson:"name" json:"name" binding:"required"`
	// 'aws/do/gcloud/ssh/credential',
	Type   string `bson:"type" json:"type" binding:"required"`

	// username
	Key    string `bson:"key" json:"key"`
	// password
	Secret string `bson:"secret" json:"-"`
}

// Get storage path for global access key
// for keys except credentials
func (key GlobalAccessKey) GetPath() string {
	return util.Config.TmpPath + "/global_access_key_" + key.ID.String()
}

func (key GlobalAccessKey) Insert() error {
	c := database.MongoDb.C("global_access_key")
	return c.Insert(key)
}

func (key GlobalAccessKey) Update() error {
	c := database.MongoDb.C("global_access_key")
	return c.UpdateId(key.ID, key)
}

func (key GlobalAccessKey) Remove() error {
	c := database.MongoDb.C("global_access_key")
	return c.RemoveId(key.ID)
}