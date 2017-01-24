package helpers

import (
	"github.com/pearsonappeng/tensor/db"
	"gopkg.in/mgo.v2/bson"
)

func IsUniqueJTemplate(name string, pID bson.ObjectId) bool {
	count, err := db.JobTemplates().Find(bson.M{"name": name, "project_id": pID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func IsNotUniqueJTemplate(name string, pID bson.ObjectId) bool {
	count, err := db.JobTemplates().Find(bson.M{"name": name, "project_id": pID}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}
