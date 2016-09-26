package models

import "gopkg.in/mgo.v2/bson"

type AccessControl struct {
	UserID bson.ObjectId  `bson:"user_id,omitempty" json:"user_id,omitempty"`
	TeamID bson.ObjectId  `bson:"team_id,omitempty" json:"team_id,omitempty"`
	Type   string         `bson:"type" json:"type"` // Team or a User
	Role   string         `bson:"role" json:"role"`
}