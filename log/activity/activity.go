package activity

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	"time"

	log "github.com/Sirupsen/logrus"

	"gopkg.in/mgo.v2/bson"
)

// AddOrganizationActivity is resposible of creating new activity stream
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
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddUserActivity is resposible of creating new activity stream
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
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddProjectActivity is resposible of creating new activity stream
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
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddCredentialActivity is resposible of creating new activity stream
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
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}

// AddTeamActivity is resposible of creating new activity stream
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
		log.WithFields(log.Fields{
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
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}
