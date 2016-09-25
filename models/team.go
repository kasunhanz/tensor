package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)

// team is the model for organization
// collection
type Team struct {
	ID             bson.ObjectId      `bson:"_id" json:"id"`

	Type           string             `bson:"-" json:"type"`
	Url            string             `bson:"-" json:"url"`
	Related        gin.H              `bson:"-" json:"related"`
	SummaryFields  gin.H              `bson:"-" json:"summary_fields"`

	Name           string             `bson:"name" json:"name" binding:"required"`
	Description    string             `bson:"description" json:"description"`
	OrganizationID bson.ObjectId      `bson:"organization_id" json:"organization"`
	CreatedBy      bson.ObjectId      `bson:"created_by" json:"created_by"`
	ModifiedBy     bson.ObjectId      `bson:"modified_by" json:"modified_by"`
	Created        time.Time          `bson:"created" json:"created"`
	Modified       time.Time          `bson:"modified" json:"modified"`

	Roles          []AccessControl    `bson:"roles" json:"-"`
}