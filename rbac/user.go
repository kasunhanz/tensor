package rbac

import (
	"github.com/pearsonappeng/tensor/models/common"
)

type User struct{}

func (User) Read(user common.User, object common.User) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor || user.ID == object.ID {
		return true
	}

	/*if orgID, err := object.GetOrganizationID(); err != nil {
		// check whether the user is an member of the objects' organization
		// since this is write permission it is must user need to be an admin
		if isOrganizationAdmin(orgID, user.ID) {
			return true
		}
	}*/

	return false
}

func (User) Write(user common.User, object common.User) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor || user.ID == object.ID {
		return true
	}

	//TODO: fix this
	/*if orgID, err := object.GetOrganizationID(); err != nil {
		// check whether the user is an member of the objects' organization
		// since this is write permission it is must user need to be an admin
		if isOrganizationAdmin(orgID, user.ID) {
			return true
		}
	}*/

	return false
}

func (User) WriteSpecial(user common.User, object common.User) bool {
	// allow access if the user is super user or
	// a system auditor
	if user.IsSuperUser || user.IsSystemAuditor {
		return true
	}

	//TODO: fix this
	/*if orgID, err := object.GetOrganizationID(); err != nil {
		// check whether the user is an member of the objects' organization
		// since this is write permission it is must user need to be an admin
		if isOrganizationAdmin(orgID, user.ID) {
			return true
		}
	}*/

	return false
}
