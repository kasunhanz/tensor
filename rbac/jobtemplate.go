package rbac

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/models/ansible"
	"gopkg.in/mgo.v2/bson"
)

const (
	JobTemplateAdmin = "admin"
	JobTemplateExecute = "execute"
)

type JobTemplate struct{}

func (JobTemplate) Read(user common.User, jtemplate ansible.JobTemplate) bool {
	// Allow access if the user is super user or
	// a system auditor
	if HasGlobalRead(user) {
		return true
	}

	if jtemplate.MachineCredentialID != nil {
		return new(Credential).ReadByID(user, *jtemplate.MachineCredentialID)
	}

	if jtemplate.CloudCredentialID != nil {
		return new(Credential).ReadByID(user, *jtemplate.CloudCredentialID)
	}

	if jtemplate.NetworkCredentialID != nil {
		return new(Credential).ReadByID(user, *jtemplate.NetworkCredentialID)
	}

	if new(Inventory).ReadByID(user, jtemplate.InventoryID) {
		return true
	}

	if new(Project).ReadByID(user, jtemplate.ProjectID) {
		return true
	}

	if orgID, err := jtemplate.GetOrganizationID(); err != nil {
		// check whether the user is an member of the objects' organization
		// since this is write permission it is must user need to be an admin
		if IsOrganizationAdmin(orgID, user.ID) {
			return true
		}
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range jtemplate.GetRoles() {
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
	count, err := db.Teams().Find(query).Count()
	if err != nil {
		logrus.Errorln("Error while checking the user is granted teams' memeber:", err)
	}
	if count > 0 {
		return true
	}

	return false
}

func (JobTemplate) Write(user common.User, jtemplate ansible.JobTemplate) bool {
	// Allow access if the user is super user or
	// a system auditor
	if HasGlobalWrite(user) {
		return true
	}

	if orgID, err := jtemplate.GetOrganizationID(); err != nil {
		// check whether the user is an member of the objects' organization
		// since this is write permission it is must user need to be an admin
		if IsOrganizationAdmin(orgID, user.ID) {
			return true
		}
	}

	var teams []bson.ObjectId
	// check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range jtemplate.GetRoles() {
		if v.Type == RoleTypeTeam && (v.Role == JobTemplateAdmin || v.Role == JobTemplateExecute) {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == RoleTypeUser && v.GranteeID == user.ID && (v.Role == JobTemplateAdmin || v.Role == JobTemplateExecute) {
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

func (j JobTemplate) ReadByID(user common.User, templateID bson.ObjectId) bool {
	var template ansible.JobTemplate
	if err := db.JobTemplates().FindId(templateID).One(&template); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		})
		return false
	}
	return j.Read(user, template)
}

func (j JobTemplate) WriteByID(user common.User, templateID bson.ObjectId) bool {
	var template ansible.JobTemplate
	if err := db.JobTemplates().FindId(templateID).One(&template); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		})
		return false
	}
	return j.Write(user, template)
}

func (JobTemplate) Associate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$addToSet": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.JobTemplates().UpdateId(resourceID, access); err != nil {
		logrus.WithFields(logrus.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to assign the role, an error occured")
	}

	return
}

func (JobTemplate) Disassociate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$pull": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.JobTemplates().UpdateId(resourceID, access); err != nil {
		logrus.WithFields(logrus.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to disassociate role")
	}

	return
}
