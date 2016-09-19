package runners

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
)

func (aj *AnsibleJob) updateStatus() {

	c := db.C(models.DBC_JOBS)

	if err := c.UpdateId(aj.job.ID, bson.M{"$set": bson.M{
		"status": aj.job.Status,
		"start":  aj.job.Started,
		"end":    aj.job.Finished,
	},
	}); err != nil {
		log.Println("Failed to update task status", err)
		panic(err)
	}
}