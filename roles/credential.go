package roles

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
	log "github.com/Sirupsen/logrus"
)

func CredentialRead(user models.User, credential models.Credential) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is read it doesn't matter what permission assigned to the user
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "organization_id": credential.OrganizationID}).Count()
	if err != nil {
		log.Errorln("Error while checking the user and organizational memeber:", err)
		return false
	}
	if count > 0 {
		return true
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range credential.Roles {
		if v.Type == "team" {
			teams = append(teams, v.TeamID)
		}

		if v.Type == "user" && v.UserID == user.ID {
			return true
		}
	}

	//check team permissions if, the user is in a team assign indirect permissions
	count, err = db.Teams().Find(bson.M{"_id:": bson.M{"$in": teams}, "roles.user_id": user.ID, }).Count()
	if err != nil {
		log.Errorln("Error while checking the user is granted teams' memeber:", err)
		return false
	}
	if count > 0 {
		return true
	}

	return false
}

func CredentialWrite(user models.User, credential models.Credential) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "organization_id": credential.OrganizationID, "roles.role": ORGANIZATION_ADMIN}).Count()
	if err != nil {
		log.Errorln("Error while checking the user and organizational admin:", err)
		return false
	}
	if count > 0 {
		return true
	}

	var teams []bson.ObjectId

	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range credential.Roles {
		if v.Type == "team" && v.Role == CREDENTIAL_ADMIN {
			teams = append(teams, v.TeamID)
		}

		if v.Type == "user" && v.UserID == user.ID && v.Role == CREDENTIAL_ADMIN {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:": bson.M{"$in": teams},
		"roles.user_id": user.ID,
	}
	count, err = db.Teams().Find(query).Count()

	if err != nil {
		log.Errorln("Error while checking the user is granted teams' memeber:", err)
		return false
	}
	if count > 0 {
		return true
	}

	return false
}

func CredentialUse(user models.User, credential models.Credential) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "organization_id": credential.OrganizationID, "roles.role": ORGANIZATION_ADMIN}).Count()
	if err != nil {
		log.Errorln("Error while checking the user and organizational admin:", err)
		return false
	}

	if count > 0 {
		return true
	}

	//teams which has relevant permissions
	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range credential.Roles {
		if v.Type == "team" && (v.Role == CREDENTIAL_ADMIN || v.Role == CREDENTIAL_USE) {
			teams = append(teams, v.TeamID)
		}

		if v.Type == "user" && v.UserID == user.ID && (v.Role == CREDENTIAL_ADMIN || v.Role == CREDENTIAL_USE) {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:": bson.M{"$in": teams},
		"roles.user_id": user.ID,
	}
	count, err = db.Teams().Find(query).Count()
	if err != nil {
		log.Errorln("Error while checking the user is granted teams' memeber:", err)
		return false
	}
	if count > 0 {
		return true
	}

	return false
}

func AddCredentialUser(credential models.Credential, user bson.ObjectId, role string) {
	access := bson.M{"$addToSet": bson.M{"roles": models.AccessControl{Type:"user", UserID:user, Role: role}}}
	err := db.Credentials().UpdateId(credential.ID, access);
	if err != nil {
		log.WithFields(log.Fields{
			"User ID": user,
			"Credential ID": credential.ID.Hex(),
			"Error": err.Error(),
		}).Errorln("Error while adding the user to roles")
	}
}

func AddCredentialTeam(credential models.Credential, team bson.ObjectId, role string) {
	access := bson.M{"$addToSet": bson.M{"roles": models.AccessControl{Type:"team", TeamID:team, Role: role}}}
	err := db.Credentials().UpdateId(credential.ID, access);
	if err != nil {
		log.WithFields(log.Fields{
			"Team ID": team,
			"Credential ID": credential.ID.Hex(),
			"Error": err.Error(),
		}).Errorln("Error while adding the Team to roles")
	}
}