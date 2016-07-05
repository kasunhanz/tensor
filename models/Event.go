package models

import (
	"time"
	database "github.com/gamunu/hilbertspace/db"
	"gopkg.in/mgo.v2/bson"
)

type Event struct {
	ID          bson.ObjectId `bson:"_id" json:"id"`
	ProjectID   bson.ObjectId      `bson:"project_id" json:"project_id"`
	ObjectID    bson.ObjectId      `bson:"object_id" json:"object_id"`
	ObjectType  string   `bson:"object_type" json:"object_type"`
	Description string   `bson:"description" json:"description"`
	Created     time.Time `bson:"created" json:"created"`

	ObjectName  string  `bson:"-" json:"object_name"`
	ProjectName string `bson:"project_name" json:"project_name"`
}

func (evt Event) Insert() error {
	c := database.MongoDb.C("event")
	err := c.Insert(evt)

	return err
}
