package models

import "gopkg.in/mgo.v2/bson"

type ACL struct {
	ID     bson.ObjectId  `bson:"_id" json:"id"`
	Object bson.ObjectId  `bson:"object" json:"object"`
	Type   string         `bson:"type" json:"type"`
	UserID bson.ObjectId  `bson:"user_id,omitempty" json:"user_id,omitempty"`
	TeamID bson.ObjectId  `bson:"team_id,omitempty" json:"team_id,omitempty"`
	Role   string         `bson:"role" json:"role"`
}