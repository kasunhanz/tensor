package common

import "gopkg.in/mgo.v2/bson"

// AccessControl type for storing roles in a object
// ResourceID ObjectIds highly likely unique
type AccessControl struct {
	GranteeID bson.ObjectId `bson:"grantee_id,omitempty" json:"grantee_id,omitempty"`
	Type      string        `bson:"type" json:"type"` // Team or a User
	Role      string        `bson:"role" json:"role"`
}
