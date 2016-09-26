package roles

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"gopkg.in/mgo.v2/bson"
)

//Important: if you are adding roles to team which means you are adding user to that team
const (
	// organization
	ORGANIZATION_ADMIN = "admin"
	ORGANIZATION_AUDITOR = "auditor"
	ORGANIZATION_MEMBER = "member"
	ORGANIZATION_READ = "read"

	// credential
	CREDENTIAL_ADMIN = "admin"
	CREDENTIAL_READ = "read"
	CREDENTIAL_USE = "use"

	// project
	PROJECT_ADMIN = "admin"
	PROJECT_USE = "use"
	PROJECT_UPDATE = "update"

	// inventory
	INVENTORY_ADMIN = "admin"
	INVENTORY_USE = "use"
	INVENTORY_ADD_HOC = "add_hoc"
	INVENTORY_UPDATE = "update"

	//job template
	JOB_TEMPLATE_ADMIN = "admin"
	JOB_TEMPLATE_EXECUTE = "execute"

	//job
	JOB_ADMIN = "admin"
	JOB_EXECUTE = "execute"

	//Teams
	TEAM_ADMIN = "admin"
	TEAM_MEMBER = "member"
	TEAM_READ = "read"
)

// steps
// check super permissins
// check specific permissions
// check organization permissions

func AddUserRole(object bson.ObjectId, user bson.ObjectId, role string) error {
	dbacl := db.C(db.ACl)
	err := dbacl.Insert(models.AccessControl{Type:"user", UserID:user, Role: role});
	if err != nil {
		log.Println("Error while creating ACL:", err)
		return err
	}
	return nil
}