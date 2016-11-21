package roles

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
	log "github.com/Sirupsen/logrus"
)

func TeamRead(user models.User, team models.Team) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is read it doesn't matter what permission assined to the user
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "_id": team.OrganizationID}).Count()
	if err != nil {
		log.Errorln("Error while checking the user and organizational memeber:", err)
		return false
	}
	if count > 0 {
		return true
	}

	for _, v := range team.Roles {
		if v.Type == "user" && v.UserID == user.ID {
			return true
		}
	}

	return false
}

func TeamWrite(user models.User, team models.Team) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "roles.role": ORGANIZATION_ADMIN, "_id": team.OrganizationID}).Count()
	if err != nil {
		log.Errorln("Error while checking the user and organizational admin:", err)
		return false
	}
	if count > 0 {
		return true
	}

	for _, v := range team.Roles {
		if v.Type == "user" && v.UserID == user.ID && v.Role == TEAM_ADMIN {
			return true
		}
	}

	return false
}