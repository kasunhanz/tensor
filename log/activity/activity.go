package activity

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"

	"time"

	"github.com/Sirupsen/logrus"

	"github.com/pearsonappeng/tensor/rbac"
	"gopkg.in/mgo.v2/bson"
)

// AddOrganizationActivity is responsible of creating new activity stream
// for Organization related activities
func AddOrganizationActivity(action string, user common.User, req ...common.Organization) {
	activity := common.ActivityOrganization{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddUserActivity is responsible of creating new activity stream
// for User related activities
func AddUserActivity(action string, user common.User, req ...common.User) {
	activity := common.ActivityUser{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddProjectActivity is responsible of creating new activity stream
// for Project related activities
func AddProjectActivity(action string, user common.User, req ...common.Project) {
	activity := common.ActivityProject{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddCredentialActivity is responsible of creating new activity stream
// for Credential related activities
func AddCredentialActivity(action string, user common.User, req ...common.Credential) {
	activity := common.ActivityCredential{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddTeamActivity is responsible of creating new activity stream
// for Team related activities
func AddTeamActivity(action string, user common.User, req ...common.Team) {
	activity := common.ActivityTeam{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddInventoryActivity is resposible of creating new activity stream
// for Inventory related activities
func AddInventoryActivity(action string, user common.User, req ...ansible.Inventory) {
	activity := ansible.ActivityInventory{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddHostActivity is responsible of creating new activity stream
// for Host related activities
func AddHostActivity(action string, user common.User, req ...ansible.Host) {
	activity := ansible.ActivityHost{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddGroupActivity is responsible of creating new activity stream
// for Group related activities
func AddGroupActivity(action string, user common.User, req ...ansible.Group) {
	activity := ansible.ActivityGroup{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddJobTemplateActivity is responsible of creating new activity stream
// for JobTemplate related activities
func AddJobTemplateActivity(action string, user common.User, req ...ansible.JobTemplate) {
	activity := ansible.ActivityJobTemplate{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddTJobTemplateActivity is responsible of creating new activity stream
// for terraform JobTemplate related activities
func AddTJobTemplateActivity(action string, user common.User, req ...terraform.JobTemplate) {
	activity := terraform.ActivityJobTemplate{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req[0],
	}
	// Set Object2 for PUT and PATCH requests
	if len(req) > 1 {
		activity.Object2 = &req[1]
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

//AddRBACActivity is responsible for creating new activity stream
//for RBAC related Activity
func AddRBACActivity(user common.User, req common.RoleObj) {
	activity := common.ActivityAssociation{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Object:    req,
		Operation: common.Associate,
	}

	if req.Disassociate {
		activity.Operation = common.Disassociate
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

//AddProjectAssociationActivity is responsible for creating new activity stream
//for project association related Activity
func AddProjectAssociationActivity(user common.User, req common.Project) {
	role := common.RoleObj{
		Disassociate: false,
		ResourceID:   req.ID,
		ResourceType: "project",
		Role:         rbac.ProjectAdmin,
	}
	activity := common.ActivityAssociation{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Object:    role,
		Operation: common.Associate,
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

//AddCredentialAssociationActivity is responsible for creating new activity stream
//for credential association related Activity
func AddCredentialAssociationActivity(user common.User, req common.Credential) {
	role := common.RoleObj{
		Disassociate: false,
		ResourceID:   req.ID,
		ResourceType: "credential",
		Role:         rbac.ProjectAdmin,
	}
	activity := common.ActivityAssociation{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Object:    role,
		Operation: common.Associate,
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

//AddTeamAssociationActivity is responsible for creating new activity stream
//for team association related Activity
func AddTeamAssociationActivity(user common.User, req common.Team) {
	role := common.RoleObj{
		Disassociate: false,
		ResourceID:   req.ID,
		ResourceType: "team",
		Role:         rbac.ProjectAdmin,
	}
	activity := common.ActivityAssociation{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Object:    role,
		Operation: common.Associate,
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

//AddInventoryAssociationActivity is responsible for creating new activity stream
//for Inventory association related Activity
func AddInventoryAssociationActivity(user common.User, req ansible.Inventory) {
	role := common.RoleObj{
		Disassociate: false,
		ResourceID:   req.ID,
		ResourceType: "inventory",
		Role:         rbac.ProjectAdmin,
	}
	activity := common.ActivityAssociation{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Object:    role,
		Operation: common.Associate,
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

//AddJobTemplateAssociationActivity is responsible for creating new activity stream
//for JobTemplate association related Activity
func AddJobTemplateAssociationActivity(user common.User, req terraform.JobTemplate) {
	role := common.RoleObj{
		Disassociate: false,
		ResourceID:   req.ID,
		ResourceType: "jobTemplate",
		Role:         rbac.ProjectAdmin,
	}
	activity := common.ActivityAssociation{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Object:    role,
		Operation: common.Associate,
	}

	if err := db.ActivityStream().Insert(activity); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}
