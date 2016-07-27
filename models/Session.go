package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	database "github.com/gamunu/hilbert-space/db"
)

// Session is the model for session
// collection
type Session struct {
	ID         bson.ObjectId      `bson:"_id" json:"id"`
	UserID     bson.ObjectId       `bson:"user_id" json:"user_id"`
	Created    time.Time `bson:"created" json:"created"`
	LastActive time.Time `bson:"last_active" json:"last_active"`
	IP         string    `bson:"ip" json:"ip"`
	UserAgent  string    `bson:"user_agent" json:"user_agent"`
	Expired    bool      `bson:"expired" json:"expired"`
}

func (s Session) Insert() error {
	c := database.MongoDb.C("session")
	return c.Insert(s)
}
