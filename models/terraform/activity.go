package terraform

import (
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// ActivityJobTemplate is the model for JobTemplate collection
type ActivityJobTemplate struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	Type      string        `bson:"-" json:"type"`
	ActorID   bson.ObjectId `bson:"actor_id" json:"actor_id"`
	Links     gin.H         `bson:"-" json:"links"`
	Meta      gin.H         `bson:"-" json:"meta"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
	Operation string        `bson:"operation" json:"operation"`
	Object1   JobTemplate   `bson:"object1" json:"object1"`
	Object2   *JobTemplate  `bson:"object2" json:"object2"`
}
