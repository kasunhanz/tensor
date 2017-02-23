package rbac

import (
	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

const (
	OrganizationAdmin = "admin"
	OrganizationAuditor = "auditor"
	OrganizationMember = "member"
)

type Organization struct{}

func (Organization) Read(user common.User, organization common.Organization) bool {
	// Allow access if the user is super user or
	// a system auditor
	if HasGlobalRead(user) {
		return true
	}

	for _, v := range organization.Roles {
		if v.Type == RoleTypeUser && v.GranteeID == user.ID {
			// Any Organization Role could read
			return true
		}
	}

	return false
}

func (Organization) Write(user common.User, organization common.Organization) bool {
	// Allow access if the user is super user or
	// a system auditor
	if HasGlobalWrite(user) {
		return true
	}

	for _, v := range organization.Roles {
		if v.Type == RoleTypeUser && v.GranteeID == user.ID && v.Role == OrganizationAdmin {
			return true
		}
	}

	return false
}

func (o Organization) WriteByID(user common.User, organizationID bson.ObjectId) bool {
	var organization common.Organization
	if err := db.Organizations().FindId(organizationID).One(&organization); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		})
		return false
	}
	return o.Write(user, organization)
}

func (Organization) Associate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$addToSet": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Organizations().UpdateId(resourceID, access); err != nil {
		logrus.WithFields(logrus.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to assign the role, an error occured")
	}

	return
}

func (Organization) Disassociate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$pull": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Organizations().UpdateId(resourceID, access); err != nil {
		logrus.WithFields(logrus.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to disassociate role")
	}

	return
}
