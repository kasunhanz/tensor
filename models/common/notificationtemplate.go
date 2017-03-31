package common

import (
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type NotificationTemplate struct {
	ID bson.ObjectId `bson:"_id" json:"id"`

	Description               string `bson:"description" json:"description"`
	Name                      string `bson:"name" json:"name"`
	NotificationsType         string `bson:"notification_type" json:"notification_type"`
	NotificationConfiguration string `bson:"notification_configuration" json:"notification_configuration"`
	Subject                   string `bson:"subject" json:"subject"`

	CreatedByID  bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created  time.Time `bson:"created" json:"created"`
	Modified time.Time `bson:"modified" json:"modified"`

	Type  string `bson:"-" json:"type"`
	Links gin.H  `bson:"-" json:"links"`
	Meta  gin.H  `bson:"-" json:"meta"`

	Roles []AccessControl `bson:"access" json:"-"`
}

func (NotificationTemplate) GetType() string {
	return "notification_template"
}

func (n NotificationTemplate) GetRoles() []AccessControl {
	return n.Roles
}
