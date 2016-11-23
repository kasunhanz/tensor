package credentials

import (
	"bitbucket.pearson.com/apseng/tensor/db"
	"bitbucket.pearson.com/apseng/tensor/models"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"time"
)

// hideEncrypted is replace encrypted fields by $encrypted$
func hideEncrypted(c *models.Credential) {
	encrypted := "$encrypted$"
	c.Password = encrypted
	c.SshKeyData = encrypted
	c.SshKeyUnlock = encrypted
	c.BecomePassword = encrypted
	c.VaultPassword = encrypted
	c.AuthorizePassword = encrypted
	c.Secret = encrypted
}

func addActivity(crdID bson.ObjectId, userID bson.ObjectId, desc string) {

	err := db.ActivityStream().Insert(models.Activity{
		ID:          bson.NewObjectId(),
		ActorID:     userID,
		Type:        _CTX_CREDENTIAL,
		ObjectID:    crdID,
		Description: desc,
		Created:     time.Now(),
	})

	if err != nil {
		log.Errorln("Failed to add new Activity", err)
	}
}
