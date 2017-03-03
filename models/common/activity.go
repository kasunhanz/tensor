package common

import (
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// Activity is the model for Activity collection
type Activity struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	Type      string        `bson:"-" json:"type"`
	URL       string        `bson:"-" json:"url"`
	ActorID   bson.ObjectId `bson:"actor_id"`
	Object1ID bson.ObjectId `bson:"object1_id"`
	Object2ID bson.ObjectId   `bson:"object2_id,omitempty"`
	Links     gin.H         `bson:"-" json:"links"`
	Meta      gin.H         `bson:"-" json:"meta"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
	Operation string        `bson:"operation" json:"operation"`
	Changes   map[string]interface{}   `bson:"changes" json:"changes"`
	Object1   string   `bson:"object1" json:"object1"`
	Object2   string   `bson:"object2,omitempty" json:"object2"`
}