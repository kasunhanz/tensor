package rbac

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"github.com/pearsonappeng/tensor/models"
)

func teamRead(user common.User, team models.RootModel) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is read it doesn't matter what permission assined to the user
	count, err := db.Organizations().Find(bson.M{
		"roles.user_id": user.ID,
		"_id": team.GetOrganizationID(),
	}).Count()
	if err != nil {
		log.Errorln("Error while checking the user and organizational memeber:", err)
	}
	if count > 0 {
		return true
	}

	for _, v := range team.GetRoles() {
		if v.Type == RoleTypeUser && v.GranteeID == user.ID {
			return true
		}
	}

	return false
}

func teamWrite(user common.User, team  models.RootModel) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, err := db.Organizations().Find(bson.M{
		"roles.user_id": user.ID,
		"roles.role": OrganizationAdmin,
		"_id": team.GetOrganizationID(),
	}).Count()
	if err != nil {
		log.Errorln("Error while checking the user and organizational admin:", err)
	}
	if count > 0 {
		return true
	}

	for _, v := range team.GetRoles() {
		if v.Type == RoleTypeUser && v.GranteeID == user.ID && v.Role == TeamAdmin {
			return true
		}
	}

	return false
}
