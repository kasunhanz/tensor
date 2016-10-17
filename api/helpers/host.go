package helpers

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
)

func IsUniqueHost(name string, IID bson.ObjectId) bool {
	count, err := db.Hosts().FindId(bson.M{"name": name, "inventory_id": IID}).Count();
	if err == nil && count == 1 {
		return true
	}

	return false
}