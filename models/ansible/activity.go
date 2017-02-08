package ansible

import (
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// ActivityInventory is the model for Inventories collection
type ActivityInventory struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	Type      string        `bson:"-" json:"type"`
	URL       string        `bson:"-" json:"url"`
	ActorID   bson.ObjectId `bson:"actor_id" json:"actor_id"`
	Related   gin.H         `bson:"-" json:"related"`
	Summary   gin.H         `bson:"-" json:"summary_fields"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
	Operation string        `bson:"operation" json:"operation"`
	Object1   Inventory     `bson:"object1" json:"object1"`
	Object2   *Inventory    `bson:"object2" json:"object2"`
}

// ActivityHost is the model for Host collection
type ActivityHost struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	Type      string        `bson:"-" json:"type"`
	URL       string        `bson:"-" json:"url"`
	ActorID   bson.ObjectId `bson:"actor_id" json:"actor_id"`
	Related   gin.H         `bson:"-" json:"related"`
	Summary   gin.H         `bson:"-" json:"summary_fields"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
	Operation string        `bson:"operation" json:"operation"`
	Object1   Host          `bson:"object1" json:"object1"`
	Object2   *Host         `bson:"object2" json:"object2"`
}

// ActivityGroup is the model for Group collection
type ActivityGroup struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	Type      string        `bson:"-" json:"type"`
	URL       string        `bson:"-" json:"url"`
	ActorID   bson.ObjectId `bson:"actor_id" json:"actor_id"`
	Related   gin.H         `bson:"-" json:"related"`
	Summary   gin.H         `bson:"-" json:"summary_fields"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
	Operation string        `bson:"operation" json:"operation"`
	Object1   Group         `bson:"object1" json:"object1"`
	Object2   *Group        `bson:"object2" json:"object2"`
}

// ActivityJobTemplate is the model for JobTemplate collection
type ActivityJobTemplate struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	Type      string        `bson:"-" json:"type"`
	URL       string        `bson:"-" json:"url"`
	ActorID   bson.ObjectId `bson:"actor_id" json:"actor_id"`
	Related   gin.H         `bson:"-" json:"related"`
	Summary   gin.H         `bson:"-" json:"summary_fields"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
	Operation string        `bson:"operation" json:"operation"`
	Object1   JobTemplate   `bson:"object1" json:"object1"`
	Object2   *JobTemplate  `bson:"object2" json:"object2"`
}
