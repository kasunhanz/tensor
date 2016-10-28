package teams

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/models"
	"time"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
)

func addActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string) {

	a := models.Activity{
		ID: bson.NewObjectId(),
		ActorID: userID,
		Type: _CTX_TEAM,
		ObjectID: crdID,
		Description: desc,
		Created: time.Now(),
	}

	if err := db.ActivityStream().Insert(a); err != nil {
		log.Println("Failed to add new Activity", err)
	}
}

// hideEncrypted is replace encrypted fields by $encrypted$
func hideEncrypted(c *models.Credential) {
	encrypted := "$encrypted$"
	c.Password = encrypted
	c.SshKeyData = encrypted
	c.SshKeyUnlock = encrypted
	c.BecomePassword = encrypted
	c.VaultPassword = encrypted
	c.AuthorizePassword = encrypted
}
