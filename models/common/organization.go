package common

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// Organization is the model for organization collection
type Organization struct {
	ID    bson.ObjectId `bson:"_id" json:"id"`
	Type  string        `bson:"-" json:"type"`
	Links gin.H         `bson:"-" json:"links"`
	Meta  gin.H         `bson:"-" json:"meta"`

	Name        string `bson:"name" json:"name" binding:"required,min=1,max=500"`
	Description string `bson:"description" json:"description"`

	CreatedByID  bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created  time.Time `bson:"created" json:"created"`
	Modified time.Time `bson:"modified" json:"modified"`

	Roles []AccessControl `bson:"roles" json:"-"`
}

func (Organization) GetType() string {
	return "organization"
}

func (org Organization) GetRoles() []AccessControl {
	return org.Roles
}

func (org Organization) GetOrganizationID() (bson.ObjectId, error) {
	return org.ID, nil
}

func (org Organization) IsUnique() bool {
	count, err := db.Organizations().Find(bson.M{"name": org.Name}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (org Organization) Exist() bool {
	count, err := db.Organizations().FindId(org.ID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}
