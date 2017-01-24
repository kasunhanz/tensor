package helpers

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

func IsUniqueCredential(name string) bool {
	count, err := db.Credentials().Find(bson.M{"name": name}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueCredential(name string) bool {
	count, err := db.Credentials().Find(bson.M{"name": name}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func MachineCredentialExist(ID bson.ObjectId) bool {
	query := bson.M{
		"_id": ID,
		"kind": bson.M{
			"$in": []string{
				common.CREDENTIAL_KIND_SSH,
				common.CREDENTIAL_KIND_WIN,
			},
		},
	}
	count, err := db.Credentials().Find(query).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func NetworkCredentialExist(ID bson.ObjectId) bool {
	count, err := db.Credentials().Find(bson.M{"_id": ID, "kind": common.CREDENTIAL_KIND_NET}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func CloudCredentialExist(ID bson.ObjectId) bool {
	query := bson.M{
		"_id": ID,
		"kind": bson.M{
			"$in": []string{
				common.CREDENTIAL_KIND_AWS,
				common.CREDENTIAL_KIND_AZURE,
				common.CREDENTIAL_KIND_CLOUDFORMS,
				common.CREDENTIAL_KIND_GCE,
				common.CREDENTIAL_KIND_OPENSTACK,
				common.CREDENTIAL_KIND_SATELLITE6,
				common.CREDENTIAL_KIND_VMWARE,
			},
		},
	}
	count, err := db.Credentials().Find(query).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func SCMCredentialExist(ID bson.ObjectId) bool {
	count, err := db.Credentials().Find(bson.M{"_id": ID, "kind": common.CREDENTIAL_KIND_SCM}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}
