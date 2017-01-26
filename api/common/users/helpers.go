package users

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"

	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

func addActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string) {

	a := common.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     userID,
		Type:        "user",
		ObjectID:    crdID,
		Description: desc,
		Created:     time.Now(),
	}
	if err := db.ActivityStream().Insert(a); err != nil {
		log.Errorln("Failed to add new Activity", err)
	}
}
