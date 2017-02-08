package rbac

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"github.com/pearsonappeng/tensor/models"
)

func projectRead(user common.User, project models.RootModel) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is read it doesn't matter what permission assigned to the user
	count, err := db.Organizations().Find(bson.M{
		"roles.user_id": user.ID,
		"organization_id": project.GetOrganizationID(),
		"roles.role": OrganizationAdmin,
	}).Count()

	if err != nil {
		log.Errorln("Error while checking the user and organizational memeber:", err)
	}
	if count > 0 {
		return true
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range project.GetRoles() {
		if v.Type == RoleTypeTeam {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == RoleTypeUser && v.GranteeID == user.ID {
			return true
		}
	}

	// Check team permissions && whether the user is in an team which has appropriate permissions
	count, err = db.Teams().Find(bson.M{
		"_id:": bson.M{"$in": teams},
		"roles.user_id": user.ID,
	}).Count()

	if err != nil {
		log.Errorln("Error while checking the user is granted teams' memeber:", err)
	}
	if count > 0 {
		return true
	}

	return false
}


func projectWrite(user common.User, project models.RootModel) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, error := db.Organizations().Find(bson.M{"" +
		"roles.user_id": user.ID,
		"organization_id": project.GetOrganizationID(),
		"roles.role": OrganizationAdmin,
	}).Count()

	if error != nil {
		log.Errorln("Error while checking the user and organizational admin:", error)
	}
	if count > 0 {
		return true
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range project.GetRoles() {
		if v.Type == RoleTypeTeam && v.Role == ProjectAdmin {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == RoleTypeUser && v.GranteeID == user.ID && v.Role == ProjectAdmin {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:":          bson.M{"$in": teams},
		"roles.user_id": user.ID,
	}
	count, error = db.Teams().Find(query).Count()
	if error != nil {
		log.Errorln("Error while checking the user is granted teams' memeber:", error)
	}
	if count > 0 {
		return true
	}

	return false
}

func projectUse(user common.User, project models.RootModel) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, error := db.Organizations().Find(bson.M{
		"roles.user_id": user.ID,
		"organization_id": project.GetOrganizationID(),
		"roles.role": OrganizationAdmin,
	}).Count()
	if error != nil {
		log.Errorln("Error while checking the user and organizational admin:", error)
	}
	if count > 0 {
		return true
	}

	//teams which has relevant permissions
	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range project.GetRoles() {
		if v.Type == RoleTypeTeam && (v.Role == ProjectAdmin || v.Role == ProjectUse) {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == RoleTypeUser && v.GranteeID == user.ID && (v.Role == ProjectAdmin || v.Role == ProjectUse) {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:":          bson.M{"$in": teams},
		"roles.user_id": user.ID,
	}
	count, error = db.Teams().Find(query).Count()
	if error != nil {
		log.Errorln("Error while checking the user is granted teams' memeber:", error)
	}
	if count > 0 {
		return true
	}

	return false
}
