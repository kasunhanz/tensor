package users

import (
	"github.com/gamunu/tensor/db"
	"github.com/gamunu/tensor/models"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func addActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string) {

	a := models.Activity{
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
