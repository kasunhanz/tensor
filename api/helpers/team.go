package helpers

import (
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
)

func IsUniqueTeam(name string, OID bson.ObjectId) bool {
	count, err := db.Teams().Find(bson.M{"name": name, "organization_id": OID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueTeam(name string, OID bson.ObjectId) bool {
	count, err := db.Teams().Find(bson.M{"name": name, "organization_id": OID}).Count()
	if err == nil && count > 0 {
		return true
	}

	return false
}
