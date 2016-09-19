package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

const DBC_NOTIFICATIONS = "notifications"

type Notification struct {
	ID                     bson.ObjectId  `bson:"_id" json:"id"`

	Status                 string         `bson:"status" json:"status"`
	Error                  string         `bson:"error" json:"error"`
	NotificationsSent      uint64          `bson:"notifications_sent" json:"notifications_sent"`
	NotificationsType      string         `bson:"notification_type" json:"notification_type"`
	Recipients             string         `bson:"recipients" json:"recipients"`
	Subject                string         `bson:"subject" json:"subject"`
	Body                   string         `bson:"body" json:"body"`
	NotificationTemplateId string         `bson:"notification_template_id" json:"notification_template_id"`

	CreatedByID            bson.ObjectId  `bson:"created_by_id" json:"created_by"`
	ModifiedByID           bson.ObjectId  `bson:"modified_by_id" json:"modified_by"`
	Created                time.Time      `bson:"created" json:"created"`
	Modified               time.Time      `bson:"modified" json:"modified"`

	Type                   string         `bson:"-" json:"type"`
	Url                    string         `bson:"-" json:"url"`
	Related                gin.H          `bson:"-" json:"related"`
	SummaryFields          gin.H          `bson:"-" json:"summary_fields"`
}

func (n Notification) CreateIndexes()  {

}