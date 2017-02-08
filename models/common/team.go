package common

import (
	"time"

	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// team is the model for organization
// collection
type Team struct {
	ID             bson.ObjectId `bson:"_id" json:"id"`

	Type           string `bson:"-" json:"type"`
	URL            string `bson:"-" json:"url"`
	Related        gin.H  `bson:"-" json:"related"`
	Summary        gin.H  `bson:"-" json:"summary_fields"`

	Name           string        `bson:"name" json:"name" binding:"required,min=1,max=500"`
	OrganizationID bson.ObjectId `bson:"organization_id" json:"organization" binding:"required"`

	Description    string `bson:"description,omitempty" json:"description"`

	CreatedByID    bson.ObjectId `bson:"created_by" json:"-"`
	ModifiedByID   bson.ObjectId `bson:"modified_by" json:"-"`

	Created        time.Time `bson:"created" json:"created"`
	Modified       time.Time `bson:"modified" json:"modified"`

	Roles          []AccessControl `bson:"roles" json:"-"`
}

func (*Team) GetType() string {
	return "team"
}

type PatchTeam struct {
	Name           *string        `json:"name" binding:"omitempty,min=1,max=500"`
	OrganizationID *bson.ObjectId `json:"organization"`
	Description    *string        `json:"description"`
}
