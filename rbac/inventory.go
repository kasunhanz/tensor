package rbac

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/models/ansible"
	"gopkg.in/mgo.v2/bson"
)

const (
	InventoryAdmin  = "admin"
	InventoryUse    = "use"
	InventoryUpdate = "update"
)

type Inventory struct{}

func (Inventory) Read(user common.User, inventory ansible.Inventory) bool {
	// Allow access if the user is super user or
	// a system auditor
	if HasGlobalRead(user) {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	if IsOrganizationAdmin(inventory.OrganizationID, user.ID) {
		return true
	}

	var teams []bson.ObjectId
	// Check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range inventory.GetRoles() {
		if v.Type == "team" {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == "user" && v.GranteeID == user.ID {
			return true
		}
	}

	// Check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{"_id:": bson.M{"$in": teams}, "roles.grantee_id": user.ID}
	count, err := db.Teams().Find(query).Count()
	if err != nil {
		log.Errorln("Error while checking the user is granted teams' memeber:", err)
	}
	if count > 0 {
		return true
	}

	return false
}

func (Inventory) Write(user common.User, inventory ansible.Inventory) bool {
	// Allow access if the user is super user or
	// a system auditor
	if HasGlobalWrite(user) {
		return true
	}

	// check whether the user is an member of the objects' organization
	// since this is write permission it is must user need to be an admin
	if IsOrganizationAdmin(inventory.OrganizationID, user.ID) {
		return true
	}

	var teams []bson.ObjectId
	// Check whether the user has access to object
	// using roles list
	// if object has granted team get those teams to list
	for _, v := range inventory.Roles {
		if v.Type == "team" && (v.Role == InventoryAdmin || v.Role == InventoryUpdate) {
			teams = append(teams, v.GranteeID)
		}

		if v.Type == "user" && v.GranteeID == user.ID && (v.Role == InventoryAdmin || v.Role == InventoryUpdate) {
			return true
		}
	}

	// Check team permissions of the user,
	// and team has admin and update privileges
	query := bson.M{"_id:": bson.M{"$in": teams}, "roles.grantee_id": user.ID}
	count, err := db.Teams().Find(query).Count()
	if err != nil {
		log.Errorln("Error while checking the user is granted teams' memeber:", err)
	}
	if count > 0 {
		return true
	}

	return false
}

func (i Inventory) ReadByID(user common.User, inventoryID bson.ObjectId) bool {
	var inventory ansible.Inventory
	if err := db.Inventories().FindId(inventoryID).One(&inventory); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		})
		return false
	}
	return i.Read(user, inventory)
}

func (Inventory) Associate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$addToSet": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Inventories().UpdateId(resourceID, access); err != nil {
		log.WithFields(log.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to assign the role")
	}

	return
}

func (Inventory) Disassociate(resourceID bson.ObjectId, grantee bson.ObjectId, roleType string, role string) (err error) {
	access := bson.M{"$pull": bson.M{"roles": common.AccessControl{Type: roleType, GranteeID: grantee, Role: role}}}

	if err = db.Inventories().UpdateId(resourceID, access); err != nil {
		log.WithFields(log.Fields{
			"Resource ID": resourceID,
			"Role Type":   roleType,
			"Error":       err.Error(),
		}).Errorln("Unable to disassociate role")
	}

	return
}
