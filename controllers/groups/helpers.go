package groups

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/models"
	"time"
	"bitbucket.pearson.com/apseng/tensor/db"
	log "github.com/Sirupsen/logrus"
)

func addActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string) {

	a := models.Activity{
		ID: bson.NewObjectId(),
		ActorID: userID,
		Type: _CTX_GROUP,
		ObjectID: crdID,
		Description: desc,
		Created: time.Now(),
	}

	if err := db.ActivityStream().Insert(a); err != nil {
		log.Errorln("Failed to add new Activity", err)
	}
}
