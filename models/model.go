package models

import (
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

type RootModel interface {
	GetType() string
	GetRoles() []common.AccessControl
	GetOrganizationID() bson.ObjectId
	GetID() bson.ObjectId
}