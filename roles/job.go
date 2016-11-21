package roles

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
	log "github.com/Sirupsen/logrus"
)

func JobRead(user models.User, jtemplate models.Job) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	//need to get project for organization
	var project models.Project
	err := db.Projects().FindId(jtemplate.ProjectID).One(&project)
	if err != nil {
		log.Errorln("Error while getting project")
		return false
	}

	// check whether the user is an member of the objects' organization
	// since this is read it doesn't matter what permission assined to the user
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "_id": project.OrganizationID}).Count()
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
	for _, v := range jtemplate.Roles {
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

func JoWrite(user models.User, jtemplate models.Job) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	//need to get project for organization
	var project models.Project
	err := db.Projects().FindId(jtemplate.ProjectID).One(&project)
	if err != nil {
		log.Errorln("Error while getting project")
		return false
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "_id": project.OrganizationID, "roles.role": ORGANIZATION_ADMIN}).Count()
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
	for _, v := range jtemplate.Roles {
		if v.Type == "team" && (v.Role == JOB_ADMIN) {
			teams = append(teams, v.TeamID)
		}

		if v.Type == "user" && v.UserID == user.ID && (v.Role == JOB_ADMIN) {
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

func JobExecute(user models.User, jtemplate models.Job) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	//need to get project for organization id
	var project models.Project
	err := db.Projects().FindId(jtemplate.ProjectID).One(&project)
	if err != nil {
		log.Errorln("Error while getting project")
		return false
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	count, err := db.Organizations().Find(bson.M{"roles.user_id": user.ID, "_id": project.OrganizationID, "roles.role": ORGANIZATION_ADMIN}).Count()
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
	for _, v := range project.Roles {
		if v.Type == "team" && (v.Role == JOB_ADMIN || v.Role == JOB_EXECUTE) {
			teams = append(teams, v.TeamID)
		}

		if v.Type == "user" && v.UserID == user.ID && (v.Role == JOB_ADMIN || v.Role == JOB_EXECUTE) {
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