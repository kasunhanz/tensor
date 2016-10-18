package helpers

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
)

func IsUniqueCredential(name string) bool {
	count, err := db.Hosts().FindId(bson.M{"name": name}).Count();
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueCredential(name string) bool {
	count, err := db.Hosts().FindId(bson.M{"name": name}).Count();
	if err == nil && count > 0 {
		return true
	}
	return false
}

func _credentialExist(ID bson.ObjectId) bool {
	count, err := db.Credentials().FindId(ID).Count();
	if err == nil && count > 0 {
		return true
	}
	return false
}

func MachineCredentialExist(ID bson.ObjectId) bool {
	if _credentialExist(ID) {
		return true
	}
	return false
}

func NetworkCredentialExist(ID bson.ObjectId) bool {
	if _credentialExist(ID) {
		return true
	}
	return false
}

func CloudCredentialExist(ID bson.ObjectId) bool {
	if _credentialExist(ID) {
		return true
	}
	return false
}

func SCMCredentialExist(ID bson.ObjectId) bool {
	if _credentialExist(ID) {
		return true
	}
	return false
}
