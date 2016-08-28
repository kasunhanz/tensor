package models

import (
	database "bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/util"
	"gopkg.in/mgo.v2/bson"
)

// GlobalAccessKey is the model for
// global_access_key collection
type GlobalAccessKey struct {
	ID     bson.ObjectId `bson:"_id" json:"id"`
	Name   string        `bson:"name" json:"name" binding:"required"`
	// 'aws/do/gcloud/ssh/credential',
	Type   string `bson:"type" json:"type" binding:"required"`

	// username
	Key    string `bson:"key" json:"key"`
	// password
	Secret string `bson:"secret" json:"secret"`
}

// Get storage path for global access key
// for keys except credentials
func (key GlobalAccessKey) GetPath() string {
	return util.Config.TmpPath + "/global_access_key_" + key.ID.Hex()
}

func (key GlobalAccessKey) Insert() error {
	c := database.MongoDb.C("global_access_keys")
	return c.Insert(key)
}

func (key GlobalAccessKey) Update() error {
	c := database.MongoDb.C("global_access_keys")
	return c.UpdateId(key.ID, key)
}

func (key GlobalAccessKey) Remove() error {
	c := database.MongoDb.C("global_access_keys")
	return c.RemoveId(key.ID)
}
