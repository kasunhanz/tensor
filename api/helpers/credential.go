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
				common.CredentialKindSSH,
				common.CredentialKindWIN,
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
	count, err := db.Credentials().Find(bson.M{"_id": ID, "kind": common.CredentialKindNET}).Count()
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
				common.CredentialKindAWS,
				common.CredentialKindAZURE,
				common.CredentialKindCLOUDFORMS,
				common.CredentialKindGCE,
				common.CredentialKindOPENSTACK,
				common.CredentialKindSATELLITE6,
				common.CredentialKindVMWARE,
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
	count, err := db.Credentials().Find(bson.M{"_id": ID, "kind": common.CredentialKindSCM}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}
