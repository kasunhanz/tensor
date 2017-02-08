package rbac

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/db"
	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/models"
	"errors"
)

//Important: if you are adding roles to team which means you are adding user to that team
const (
	// organization
	OrganizationAdmin = "admin"
	OrganizationAuditor = "auditor"
	OrganizationMember = "member"
	OrganizationRead = "read"

	// credential
	CredentialAdmin = "admin"
	CredentialRead = "read"
	CredentialUse = "use"

	// project
	ProjectAdmin = "admin"
	ProjectUse = "use"
	ProjectUpdate = "update"

	// inventory
	InventoryAdmin = "admin"
	InventoryUse = "use"
	InventoryUpdate = "update"

	//job template
	JobTemplateAdmin = "admin"
	JobTemplateExecute = "execute"

	//job
	JobAdmin = "admin"
	JobExecute = "execute"

	//Teams
	TeamAdmin = "admin"
	TeamMember = "member"
	TeamRead = "read"

	RoleTypeTeam = "team"
	RoleTypeUser = "user"
)

// resource object
// role OrganizationAdmin, CredentialAdmin
// grantee user id or team id
// roletype RoleTypeTeam, RoleTypeUser
func AssignRole(resource models.RootModel, grantee bson.ObjectId, roleType string, role string) (err error) {

	access := bson.M{"$addToSet": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	// switch resource type
	switch resource.GetType() {
	case "credential": {
		err = db.Credentials().UpdateId(resource.GetID(), access);
	}
	case "inventory":{
		err = db.Inventories().UpdateId(resource.GetID(), access);
	}
	case "organization": {
		err = db.Organizations().UpdateId(resource.GetID(), access);
	}
	case "project": {
		err = db.Projects().UpdateId(resource.GetID(), access);
	}
	case "terraform_job_template": {
		err = db.TerrafromJobTemplates().UpdateId(resource.GetID(), access);
	}
	case "job_template": {
		err = db.JobTemplates().UpdateId(resource.GetID(), access);
	}
	case "team": {
		err = db.Teams().UpdateId(resource.GetID(), access);
	}
	}

	if err != nil {
		log.WithFields(log.Fields{
			"Resource ID": resource.GetID(),
			"Role Type": roleType,
			"Error":         err.Error(),
		}).Errorln("Unable to assign the role, an error occured")
	}

	return
}

func HasRole(user common.User, resource models.RootModel, role string) bool {
	switch resource.GetType() {
	case "organization": {
		switch role {
		case OrganizationAdmin: {
			return organizationWrite(user, resource)
		}
		case OrganizationAuditor, OrganizationRead: {
			return organizationRead(user, resource)
		}
		}
	}
	case "credential": {
		switch role {
		case CredentialUse: {
			return credentialUse(user, resource)
		}
		case CredentialAdmin: {
			return credentialWrite(user, resource)
		}
		case CredentialRead: {
			return credentialRead(user, resource)
		}
		}
	}
	case "project": {
		switch role {
		case ProjectUse: {
			return projectUse(user, resource)
		}
		case ProjectAdmin: {
			return projectWrite(user, resource)
		}
		}
	}
	case "inventory":{
		switch role {
		case InventoryAdmin, InventoryUpdate: {
			return inventoryWrite(user, resource)
		}
		case InventoryUse: {
			return inventoryUse(user, resource)
		}
		}
	}
	case "team": {
		switch role {
		case TeamAdmin: {
			return teamWrite(user, resource)
		}
		case TeamRead: {
			return teamRead(user, resource)
		}
		}
	}
	case "job_template": {
		switch role {
		case JobTemplateAdmin: {
			return jobTemplateWrite(user, resource)
		}
		case JobTemplateExecute: {
			return jobTemplateRead(user, resource)
		}
		}
	}
	case "terraform_job_template": {
		switch role {
		case JobTemplateAdmin: {
			return jobTemplateWrite(user, resource)
		}
		case JobTemplateExecute: {
			return jobTemplateRead(user, resource)
		}
		}
	}
	}

	return false
}

func RevokeRole(revokee string, revoked ...string) error {
	return errors.New("return error if fail")
}