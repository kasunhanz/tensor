package roles

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
	"log"
)

func ProjectRead(user models.User, project models.Project) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is read it doesn't matter what permission assined to the user
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "organization_id": project.OrganizationID}).Count()
	if err != nil {
		log.Println("Error while checking the user and organizational memeber:", err)
		return false
	}
	if count > 0 {
		return true
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range project.Roles {
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
		log.Println("Error while checking the user is granted teams' memeber:", err)
		return false
	}
	if count > 0 {
		return true
	}

	return false
}

func ProjectWrite(user models.User, project models.Project) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, error := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "organization_id": project.OrganizationID, "roles.role": ORGANIZATION_ADMIN}).Count()
	if error != nil {
		log.Println("Error while checking the user and organizational admin:", error)
		return false
	}
	if count > 0 {
		return true
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range project.Roles {
		if v.Type == "team" && (v.Role == PROJECT_ADMIN || v.Role == PROJECT_UPDATE) {
			teams = append(teams, v.TeamID)
		}

		if v.Type == "user" && v.UserID == user.ID && (v.Role == PROJECT_ADMIN || v.Role == PROJECT_UPDATE) {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:": bson.M{"$in": teams},
		"roles.user_id": user.ID,
	}
	count, error = db.Teams().Find(query).Count()
	if error != nil {
		log.Println("Error while checking the user is granted teams' memeber:", error)
		return false
	}
	if count > 0 {
		return true
	}

	return false
}

func ProjectUse(user models.User, project models.Project) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, error := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "organization_id": project.OrganizationID, "roles.role": ORGANIZATION_ADMIN}).Count()
	if error != nil {
		log.Println("Error while checking the user and organizational admin:", error)
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
	for _, v := range project.Roles {
		if v.Type == "team" && (v.Role == PROJECT_ADMIN || v.Role == PROJECT_USE) {
			teams = append(teams, v.TeamID)
		}

		if v.Type == "user" && v.UserID == user.ID && (v.Role == PROJECT_ADMIN || v.Role == PROJECT_USE) {
			return true
		}
	}

	// check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{
		"_id:": bson.M{"$in": teams},
		"roles.user_id": user.ID,
	}
	count, error = db.Teams().Find(query).Count()
	if error != nil {
		log.Println("Error while checking the user is granted teams' memeber:", error)
		return false
	}
	if count > 0 {
		return true
	}

	return false
}