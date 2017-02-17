package rbac

import (
	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

//Important: if you are adding roles to team which means you are adding user to that team
const (
	RoleTypeTeam = "team"
	RoleTypeUser = "user"
)

func IsOrganizationAdmin(orgID bson.ObjectId, userID bson.ObjectId) bool {
	// Organization can be empty in credential objects
	if len(orgID) > 0 {
		// check whether the user is an member of the objects' organization
		// since this is read it doesn't matter what permission assigned to the user
		if count, err := db.Organizations().Find(bson.M{
			"roles.grantee_id": userID,
			"organization_id":  orgID,
			"roles.role":       OrganizationAdmin,
		}).Count(); count > 0 && err == nil {
			return true
		} else {
			logrus.Warnln("Error while checking the user and organizational admin")
		}

	}
	return false
}

func IsInTeams(userID bson.ObjectId, teams []bson.ObjectId) bool {
	if count, err := db.Teams().Find(bson.M{
		"_id:":             bson.M{"$in": teams},
		"roles.grantee_id": userID,
	}).Count(); count > 0 && err == nil {
		return true
	} else {
		logrus.Warnln("Error while checking the user is granted teams' memeber")
	}
	return false
}

func HasGlobalRead(user common.User) bool {
	return user.IsSuperUser || user.IsSystemAuditor
}

func HasGlobalWrite(user common.User) bool {
	return user.IsSuperUser
}
