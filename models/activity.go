package models

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Activity struct {
	ID          bson.ObjectId `bson:"_id" json:"id"`
	ObjectID    bson.ObjectId `bson:"object_id" json:"object_id"`
	ActorID     bson.ObjectId `bson:"actor_id" json:"actor_id"`
	Type        string        `bson:"type" json:"type"`
	Description string        `bson:"description" json:"description"`
	Created     time.Time     `bson:"created" json:"created"`
}
