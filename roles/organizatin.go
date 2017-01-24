package roles

import (
	"github.com/pearsonappeng/tensor/models"
)

func OrganizationRead(user models.User, organization models.Organization) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	for _, v := range organization.Roles {
		if v.Type == "user" && v.UserID == user.ID {
			return true
		}
	}

	return false
}

func OrganizationWrite(user models.User, organization models.Organization) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser {
		return true
	}

	for _, v := range organization.Roles {
		if v.Type == "user" && v.UserID == user.ID && v.Role == ORGANIZATION_ADMIN {
			return true
		}
	}

	return false
}
