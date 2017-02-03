package activity

import (
	"github.com/pearsonappeng/tensor/db"
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
