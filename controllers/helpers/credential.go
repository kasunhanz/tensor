package helpers

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
)

func IsUniqueCredential(name string) bool {
	count, err := db.Credentials().Find(bson.M{"name": name}).Count();
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueCredential(name string) bool {
	count, err := db.Credentials().Find(bson.M{"name": name}).Count();
	if err == nil && count > 0 {
		return true
	}
	return false
}

func MachineCredentialExist(ID bson.ObjectId) bool {
	query := bson.M{
		"_id":ID,
		"kind": bson.M{
			"$in": []string{
				models.CREDENTIAL_KIND_SSH,
				models.CREDENTIAL_KIND_NET,
			},
		},

	}
	count, err := db.Credentials().Find(query).Count();
	if err == nil && count > 0 {
		return true
	}
	return false
}

func NetworkCredentialExist(ID bson.ObjectId) bool {
	count, err := db.Credentials().Find(bson.M{"_id":ID, "kind":models.CREDENTIAL_KIND_NET}).Count();
	if err == nil && count > 0 {
		return true
	}
	return false
}

func CloudCredentialExist(ID bson.ObjectId) bool {
	query := bson.M{
		"_id":ID,
		"kind": bson.M{
			"$in": []string{
				models.CREDENTIAL_KIND_AWS,
				models.CREDENTIAL_KIND_AZURE,
				models.CREDENTIAL_KIND_CLOUDFORMS,
				models.CREDENTIAL_KIND_GCE,
				models.CREDENTIAL_KIND_OPENSTACK,
				models.CREDENTIAL_KIND_SATELLITE6,
				models.CREDENTIAL_KIND_VMWARE,
			},
		},

	}
	count, err := db.Credentials().Find(query).Count();
	if err == nil && count > 0 {
		return true
	}
	return false
}

func SCMCredentialExist(ID bson.ObjectId) bool {
	count, err := db.Credentials().Find(bson.M{"_id":ID, "kind":models.CREDENTIAL_KIND_SCM}).Count();
	if err == nil && count > 0 {
		return true
	}
	return false
}
