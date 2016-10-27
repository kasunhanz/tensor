package helpers

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
)

func IsUniqueHost(name string, IID bson.ObjectId) bool {
	count, err := db.Hosts().Find(bson.M{"name": name, "inventory_id": IID}).Count();
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueHost(name string, IID bson.ObjectId) bool {
	count, err := db.Hosts().Find(bson.M{"name": name, "inventory_id": IID}).Count();
	if err == nil && count > 0 {
		return true
	}

	return false
}