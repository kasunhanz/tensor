package rbac

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"github.com/pearsonappeng/tensor/models"
)

func userRead(user common.User, object models.RootModel) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor || user.ID == object.GetID() {
		return true
	}

	if len(object.GetOrganizationID()) > 0 {
		// check whether the user is an member of the objects' organization
		// since this is read it doesn't matter what permission assigned to the user
		count, err := db.Organizations().Find(bson.M{
			"roles.user_id": user.ID,
			"_id": object.GetOrganizationID(),
			"roles.role": OrganizationAdmin,
		}).Count()
		if err != nil {
			log.Errorln("Error while checking the user and organizational memeber:", err)
		}
		if count > 0 {
			return true
		}
	}

	return false
}

func userWrite(user common.User, object models.RootModel) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	if len(object.GetOrganizationID()) > 0 {
		// check whether the user is an member of the objects' organization
		// since this is read it doesn't matter what permission assigned to the user
		count, err := db.Organizations().Find(bson.M{
			"roles.user_id": user.ID,
			"_id": object.GetOrganizationID(),
			"roles.role": OrganizationAdmin,
		}).Count()
		if err != nil {
			log.Errorln("Error while checking the user and organizational memeber:", err)
		}
		if count > 0 {
			return true
		}
	}

	return false
}
