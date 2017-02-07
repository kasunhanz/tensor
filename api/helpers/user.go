package helpers

import (
	"github.com/pearsonappeng/tensor/db"
	"gopkg.in/mgo.v2/bson"
)

func IsUniqueUsername(name string) bool {
	count, err := db.Users().Find(bson.M{"username": name}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueUsername(name string) bool {
	count, err := db.Users().Find(bson.M{"username": name}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func IsUniqueEmail(email string) bool {
	count, err := db.Users().Find(bson.M{"email": email}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueEmail(email string) bool {
	count, err := db.Users().Find(bson.M{"email": email}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}