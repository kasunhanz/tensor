package helpers

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
)

func IsUniqueOrganization(name string) bool {
	count, err := db.Organizations().Find(bson.M{"name": name}).Count();
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueOrganization(name string) bool {
	count, err := db.Organizations().Find(bson.M{"name": name}).Count();
	if err == nil && count > 0 {
		return true
	}

	return false
}

func OrganizationExist(ID bson.ObjectId) bool {
	count, err := db.Organizations().FindId(ID).Count();
	if err == nil && count > 0 {
		return true
	}
	return false
}

func OrganizationNotExist(ID bson.ObjectId) bool {
	count, err := db.Organizations().FindId(ID).Count();
	if err == nil && count > 0 {
		return false
	}
	return true
}