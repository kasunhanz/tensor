package helpers

import (
	"github.com/gamunu/tensor/db"
	"gopkg.in/mgo.v2/bson"
)

func IsUniqueGroup(name string, IID bson.ObjectId) bool {
	count, err := db.Hosts().Find(bson.M{"name": name, "inventory_id": IID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueGroup(name string, IID bson.ObjectId) bool {
	count, err := db.Hosts().Find(bson.M{"name": name, "inventory_id": IID}).Count()
	if err == nil && count > 0 {
		return true
	}

	return false
}

func _groupExist(ID bson.ObjectId) bool {
	count, err := db.Groups().FindId(ID).Count()
	if err == nil && count == 1 {
		return true
	}
	return false
}

func GroupExist(ID bson.ObjectId) bool {
	if _groupExist(ID) {
		return true
	}
	return false
}

func ParentGroupExist(ID bson.ObjectId) bool {
	if _groupExist(ID) {
		return true
	}
	return false
}
