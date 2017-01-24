package helpers

import (
	"github.com/pearsonappeng/tensor/db"
	"gopkg.in/mgo.v2/bson"
)

func IsUniqueOrganization(name string) bool {
	count, err := db.Organizations().Find(bson.M{"name": name}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueOrganization(name string) bool {
	count, err := db.Organizations().Find(bson.M{"name": name}).Count()
	if err == nil && count > 0 {
		return true
	}

	return false
}

func OrganizationExist(ID bson.ObjectId) bool {
	count, err := db.Organizations().FindId(ID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func OrganizationNotExist(ID bson.ObjectId) bool {
	count, err := db.Organizations().FindId(ID).Count()
	if err == nil && count > 0 {
		return false
	}
	return true
}
