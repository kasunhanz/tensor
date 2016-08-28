package models

import (
	database "pearson.com/tensor/db"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Event struct {
	ID          bson.ObjectId `bson:"_id" json:"id"`
	ProjectID   bson.ObjectId `bson:"project_id,omitempty" json:"project_id,omitempty"`
	ObjectID    bson.ObjectId `bson:"object_id" json:"object_id"`
	ObjectType  string        `bson:"object_type" json:"object_type"`
	Description string        `bson:"description" json:"description"`
	Created     time.Time     `bson:"created" json:"created"`

	ObjectName string `bson:"-" json:"object_name"`
}

func (evt Event) Insert() error {
	c := database.MongoDb.C("events")
	err := c.Insert(evt)

	return err
}
