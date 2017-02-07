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
