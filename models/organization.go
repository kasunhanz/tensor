package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)

// Organization is the model for organization
// collection
type Organization struct {
	ID           bson.ObjectId      `bson:"_id" json:"id"`
	Type         string             `bson:"-" json:"type"`
	Url          string             `bson:"-" json:"url"`
	Related      gin.H              `bson:"-" json:"related"`
	Summary      gin.H              `bson:"-" json:"summary_fields"`

	Name         string             `bson:"name" json:"name" binding:"required,min=1,max=500"`
	Description  string             `bson:"description" json:"description"`

	CreatedByID  bson.ObjectId      `bson:"created_by_id" json:"-"`
	ModifiedByID bson.ObjectId      `bson:"modified_by_id" json:"-"`

	Created      time.Time          `bson:"created" json:"created"`
	Modified     time.Time          `bson:"modified" json:"modified"`

	Roles        []AccessControl    `bson:"roles" json:"-"`
}

type PatchOrganization struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=500"`
	Description *string `json:"description"`
}