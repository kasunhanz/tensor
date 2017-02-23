package rbac

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

const (
	ProjectAdmin  = "admin"
	ProjectUse    = "use"
	ProjectUpdate = "update"
)

type Project struct{}

func (Project) Read(user common.User, project common.Project) bool {
	// allow access if the user is super user or
	// a system auditor
	if HasGlobalRead(user) {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	if IsOrganizationAdmin(project.OrganizationID, user.ID) {
		return true
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range project.Roles {
		if v.Type == RoleTypeTeam {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == RoleTypeUser && v.GranteeID == user.ID {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:":             bson.M{"$in": teams},
		"roles.grantee_id": user.ID,
	}
	count, error := db.Teams().Find(query).Count()
	if error != nil {
		logrus.Errorln("Error while checking the user is granted teams' memeber:", error)
	}
	if count > 0 {
		return true
	}

	return false
}

func (Project) Write(user common.User, project common.Project) bool {
	// allow access if the user is super user or
	// a system auditor
	if HasGlobalWrite(user) {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	if IsOrganizationAdmin(project.OrganizationID, user.ID) {
		return true
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range project.Roles {
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
		"_id:":             bson.M{"$in": teams},
		"roles.grantee_id": user.ID,
	}
	count, error := db.Teams().Find(query).Count()
	if error != nil {
		logrus.Errorln("Error while checking the user is granted teams' memeber:", error)
	}
	if count > 0 {
		return true
	}

	return false
}

func (Project) Update(user common.User, project common.Project) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	if IsOrganizationAdmin(project.OrganizationID, user.ID) {
		return true
	}

	//teams which has relevant permissions
	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range project.GetRoles() {
		if v.Type == RoleTypeTeam && (v.Role == ProjectAdmin || v.Role == ProjectUpdate) {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == RoleTypeUser && v.GranteeID == user.ID && (v.Role == ProjectAdmin || v.Role == ProjectUpdate) {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	if IsInTeams(user.ID, teams) {
		return true
	}

	return false
}

func (p Project) ReadByID(user common.User, projectID bson.ObjectId) bool {
	var project common.Project
	if err := db.Projects().FindId(projectID).One(&project); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		})
		return false
	}
	return p.Read(user, project)
}

func (Project) Associate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$addToSet": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Projects().UpdateId(resourceID, access); err != nil {
		logrus.WithFields(logrus.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to associate the role")
	}

	return
}

func (Project) Disassociate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$pull": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Projects().UpdateId(resourceID, access); err != nil {
		logrus.WithFields(logrus.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to disassociate the role")
	}

	return
}
