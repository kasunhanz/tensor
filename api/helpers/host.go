package helpers

import (
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
)

func IsUniqueHost(name string, IID bson.ObjectId) bool {
	count, err := db.Hosts().Find(bson.M{"name": name, "inventory_id": IID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueHost(name string, IID bson.ObjectId) bool {
	count, err := db.Hosts().Find(bson.M{"name": name, "inventory_id": IID}).Count()
	if err == nil && count > 0 {
		return true
	}

	return false
}
