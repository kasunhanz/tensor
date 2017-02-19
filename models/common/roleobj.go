package common

import "gopkg.in/mgo.v2/bson"

type RoleObj struct {
	Disassociate bool          `json:"disassociate"`
	Role         string        `json:"role" binding:"required"`
	ResourceID   bson.ObjectId `json:"resource" binding:"required"`
	ResourceType string        `json:"resource_type" binding:"required,resource_type"`
}
