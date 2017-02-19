package rbac

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

const (
	TeamAdmin  = "admin"
	TeamMember = "member"
)

type Team struct{}

func (Team) Read(user common.User, team common.Team) bool {
	// allow access if the user is super user or
	// a system auditor
	if HasGlobalRead(user) {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is read it doesn't matter what permission assigned to the user
	count, err := db.Organizations().Find(bson.M{
		"roles.grantee_id": user.ID,
		"_id":              team.OrganizationID,
	}).Count()
	if err != nil {
		log.Errorln("Error while checking the user and organizational memeber:", err)
	}
	if count > 0 {
		return true
	}

	for _, v := range team.Roles {
		if v.Type == RoleTypeUser && v.GranteeID == user.ID {
			return true
		}
	}

	return false
}

func (Team) Write(user common.User, team common.Team) bool {
	// allow access if the user is super user or
	// a system auditor
	if HasGlobalWrite(user) {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	if IsOrganizationAdmin(team.OrganizationID, user.ID) {
		return true
	}

	for _, v := range team.Roles {
		if v.Type == RoleTypeUser && v.GranteeID == user.ID && v.Role == TeamAdmin {
			return true
		}
	}

	return false
}

func (Team) Associate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$addToSet": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Teams().UpdateId(resourceID, access); err != nil {
		log.WithFields(log.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to associate role")
	}

	return
}

func (Team) Disassociate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$pull": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Teams().UpdateId(resourceID, access); err != nil {
		log.WithFields(log.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to disassociate role")
	}

	return
}
