package models

import (
	database "github.com/gamunu/hilbert-space/db"
	"time"
	"gopkg.in/mgo.v2/bson"
)

// APIToken is the model for token
// collection
type APIToken struct {
	ID      bson.ObjectId    `bson:"_id" json:"id"`
	Created time.Time `bson:"created" json:"created"`
	Expired bool      `bson:"expired" json:"expired"`
	UserID  bson.ObjectId       `bson:"user_id" json:"user_id"`
}

// Create a new
func (apiToken APIToken) Insert() error {
	c := database.MongoDb.C("user_token")
	return c.Insert(apiToken)
}
