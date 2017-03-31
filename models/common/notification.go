package common

import (
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type Notification struct {
	ID bson.ObjectId `bson:"_id" json:"id"`

	Status                 string `bson:"status" json:"status"`
	Error                  string `bson:"error" json:"error"`
	NotificationsSent      uint64 `bson:"notifications_sent" json:"notifications_sent"`
	NotificationsType      string `bson:"notification_type" json:"notification_type"`
	Recipients             string `bson:"recipients" json:"recipients"`
	Subject                string `bson:"subject" json:"subject"`
	Body                   string `bson:"body" json:"body"`
	NotificationTemplateID string `bson:"notification_template_id" json:"notification_template_id"`

	CreatedByID  bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created  time.Time `bson:"created" json:"created"`
	Modified time.Time `bson:"modified" json:"modified"`

	Type  string `bson:"-" json:"type"`
	Links gin.H  `bson:"-" json:"links"`
	Meta  gin.H  `bson:"-" json:"meta"`

	Access []AccessControl `bson:"access" json:"-"`
}

func (Notification) GetType() string {
	return "notification"
}
