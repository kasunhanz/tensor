package common

import (
	"time"

	"github.com/gin-gonic/gin"

	"gopkg.in/mgo.v2/bson"
)

// Activity constants
const (
	Create       = "create"
	Update       = "update"
	Delete       = "delete"
	Associate    = "associate"
	Disassociate = "disassociate"
)

// Activity is the model for Activity collection
type Activity struct {
	ID          bson.ObjectId `bson:"_id" json:"id"`
	ObjectID    bson.ObjectId `bson:"object_id" json:"object_id"`
	ActorID     bson.ObjectId `bson:"actor_id" json:"actor_id"`
	Type        string        `bson:"type" json:"type"`
	Description string        `bson:"description" json:"description"`
	Created     time.Time     `bson:"created" json:"created"`
}

// ActivityOrganization is the model for Organization collection
type ActivityOrganization struct {
	ID        bson.ObjectId `bson:"_id" json:"id"`
	Type      string        `bson:"-" json:"type"`
	URL       string        `bson:"-" json:"url"`
	ActorID   bson.ObjectId `bson:"actor_id" json:"actor_id"`
	Related   gin.H         `bson:"-" json:"related"`
	Summary   gin.H         `bson:"-" json:"summary_fields"`
	Timestamp time.Time     `bson:"timestamp" json:"timestamp"`
	Operation string        `bson:"operation" json:"operation"`
	Object1   Organization  `bson:"object1" json:"object1"`
	Object2   *Organization `bson:"object2,omitempty" json:"object2,omitempty"`
}
