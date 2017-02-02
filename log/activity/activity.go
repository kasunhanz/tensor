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
func AddOrganizationActivity(req common.Organization, action string, user common.User) {
	if err := db.ActivityStream().Insert(common.ActivityOrganization{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		ActorID:   user.ID,
		Operation: action,
		Object1:   req,
	}); err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}
