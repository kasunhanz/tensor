package rbac

import (
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models"
)

func organizationRead(user common.User, organization models.RootModel) bool {
	// Allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	for _, v := range organization.GetRoles() {
		if v.Type == RoleTypeUser && v.GranteeID == user.ID {
			return true
		}
	}

	return false
}

func organizationWrite(user common.User, organization models.RootModel) bool {
	// Allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	for _, v := range organization.GetRoles() {
		if v.Type == RoleTypeUser && v.GranteeID == user.ID && v.Role == OrganizationAdmin {
			return true
		}
	}

	return false
}
