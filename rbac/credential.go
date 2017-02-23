package rbac

import (
	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/mgo.v2/bson"
)

const (
	CredentialAdmin = "admin"
	CredentialUse   = "use"
)

type Credential struct{}

func (Credential) Read(user common.User, credential common.Credential) bool {
	// allow access if the user is super user or
	// a system auditor
	if HasGlobalRead(user) {
		return true
	}

	if credential.OrganizationID != nil {
		// Organization can be empty in credential objects
		if IsOrganizationAdmin(*credential.OrganizationID, user.ID) {
			return true
		}

	}

	var teams []bson.ObjectId
	// Check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range credential.Roles {
		if v.Type == "team" {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == "user" && v.GranteeID == user.ID {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:":             bson.M{"$in": teams},
		"roles.grantee_id": user.ID,
	}
	count, err := db.Teams().Find(query).Count()

	if err != nil {
		logrus.Errorln("Error while checking the user is granted teams' memeber:", err)
	}
	if count > 0 {
		return true
	}

	return false
}

func (Credential) Write(user common.User, credential common.Credential) bool {
	// allow access if the user is super user or
	// a system auditor
	if HasGlobalWrite(user) {
		return true
	}

	if credential.OrganizationID != nil {
		// Organization can be empty in credential objects
		if IsOrganizationAdmin(*credential.OrganizationID, user.ID) {
			return true
		}
	}

	var teams []bson.ObjectId
	// Check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range credential.Roles {
		if v.Type == "team" && v.Role == CredentialAdmin {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == "user" && v.GranteeID == user.ID && v.Role == CredentialAdmin {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:":             bson.M{"$in": teams},
		"roles.grantee_id": user.ID,
	}
	count, err := db.Teams().Find(query).Count()

	if err != nil {
		logrus.Errorln("Error while checking the user is granted teams' memeber:", err)
	}
	if count > 0 {
		return true
	}

	return false
}

func (c Credential) ReadByID(user common.User, credentialID bson.ObjectId) bool {
	var credential common.Credential
	if err := db.Credentials().FindId(credentialID).One(&credential); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		})
		return false
	}
	return c.Read(user, credential)
}

func (Credential) Associate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$addToSet": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Credentials().UpdateId(resourceID, access); err != nil {
		logrus.WithFields(logrus.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to associate role")
	}

	return
}

func (Credential) Disassociate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$pull": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Credentials().UpdateId(resourceID, access); err != nil {
		logrus.WithFields(logrus.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to disassociate role")
	}

	return
}
