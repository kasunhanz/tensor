package models

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"time"
)

type NotificationTemplate struct {
	ID                        bson.ObjectId  `bson:"_id" json:"id"`

	Description               string         `bson:"description" json:"description"`
	Name                      string         `bson:"name" json:"name"`
	NotificationsType         string         `bson:"notification_type" json:"notification_type"`
	NotificationConfiguration string         `bson:"notification_configuration" json:"notification_configuration"`
	Subject                   string         `bson:"subject" json:"subject"`

	CreatedByID               bson.ObjectId  `bson:"created_by_id" json:"-"`
	ModifiedByID              bson.ObjectId  `bson:"modified_by_id" json:"-"`

	Created                   time.Time      `bson:"created" json:"created"`
	Modified                  time.Time      `bson:"modified" json:"modified"`

	Type                      string         `bson:"-" json:"type"`
	Url                       string         `bson:"-" json:"url"`
	Related                   gin.H          `bson:"-" json:"related"`
	SummaryFields             gin.H          `bson:"-" json:"summary_fields"`

	Access           []AccessControl    `bson:"access" json:"-"`
}