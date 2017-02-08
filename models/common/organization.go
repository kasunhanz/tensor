package common

import (
	"time"

	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
	"github.com/pearsonappeng/tensor/db"
)

// Organization is the model for organization collection
type Organization struct {
	ID           bson.ObjectId `bson:"_id" json:"id"`
	Type         string        `bson:"-" json:"type"`
	URL          string        `bson:"-" json:"url"`
	Related      gin.H         `bson:"-" json:"related"`
	Summary      gin.H         `bson:"-" json:"summary_fields"`

	Name         string `bson:"name" json:"name" binding:"required,min=1,max=500"`
	Description  string `bson:"description" json:"description"`

	CreatedByID  bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created      time.Time `bson:"created" json:"created"`
	Modified     time.Time `bson:"modified" json:"modified"`

	Roles        []AccessControl `bson:"roles" json:"-"`
}

func (*Organization) GetType() string {
	return "organization"
}

func (org *Organization) IsUnique() bool {
	count, err := db.Organizations().Find(bson.M{"name": org.Name}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (org *Organization) Exist() bool {
	count, err := db.Organizations().FindId(org.ID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}


type PatchOrganization struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=500"`
	Description *string `json:"description"`
}
