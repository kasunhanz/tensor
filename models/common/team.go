package common

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// team is the model for organization
// collection
type Team struct {
	ID             bson.ObjectId `bson:"_id" json:"id"`

	Type           string `bson:"-" json:"type"`
	Links          gin.H  `bson:"-" json:"links"`
	Meta           gin.H  `bson:"-" json:"meta"`

	Name           string        `bson:"name" json:"name" binding:"required,min=1,max=500"`
	OrganizationID bson.ObjectId `bson:"organization_id" json:"organization" binding:"required"`

	Description    string `bson:"description,omitempty" json:"description"`

	CreatedByID    bson.ObjectId `bson:"created_by" json:"-"`
	ModifiedByID   bson.ObjectId `bson:"modified_by" json:"-"`

	Created        time.Time `bson:"created" json:"created"`
	Modified       time.Time `bson:"modified" json:"modified"`

	Roles          []AccessControl `bson:"roles" json:"-"`
}

func (Team) GetType() string {
	return "team"
}

func (tm Team) GetRoles() []AccessControl {
	return tm.Roles
}

func (tm Team) GetOrganizationID() (bson.ObjectId, error) {
	var org Organization
	err := db.Organizations().FindId(tm.OrganizationID).One(&org)
	return org.ID, err
}

func (team Team) IsUnique() bool {
	count, err := db.Teams().Find(bson.M{"name": team.Name, "organization_id": team.OrganizationID}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (team Team) OrganizationExist() bool {
	count, err := db.Organizations().FindId(team.OrganizationID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

type PatchTeam struct {
	Name           *string        `json:"name" binding:"omitempty,min=1,max=500"`
	OrganizationID *bson.ObjectId `json:"organization"`
	Description    *string        `json:"description"`
}
