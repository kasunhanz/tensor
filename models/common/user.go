package common

import (
	"time"

	"github.com/pearsonappeng/tensor/db"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// User is model for user collection
type User struct {
	ID    bson.ObjectId `bson:"_id" json:"id"`
	Type  string        `bson:"-" json:"type"`
	Links gin.H         `bson:"-" json:"links"`

	Username        string `bson:"username" json:"username" binding:"required,min=1,max=30"`
	FirstName       string `bson:"first_name" json:"first_name,min=1,max=30"`
	LastName        string `bson:"last_name" json:"last_name,min=1,max=30"`
	Email           string `bson:"email" json:"email" binding:"required,email"`
	IsSuperUser     bool   `bson:"is_superuser" json:"is_superuser"`
	IsSystemAuditor bool   `bson:"is_system_auditor" json:"is_system_auditor"`
	Password        string `bson:"password,omitempty" json:"password"`

	Created  time.Time `bson:"created" json:"created"`
	Modified time.Time `bson:"modified" json:"modified"`

	Roles []AccessControl `bson:"roles" json:"-"`
}

func (User) GetType() string {
	return "user"
}

func (u User) GetRoles() []AccessControl {
	return u.Roles
}

func (user User) IsUniqueUsername() bool {
	count, err := db.Users().Find(bson.M{"username": user.Username}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (user User) IsUniqueEmail() bool {
	count, err := db.Users().Find(bson.M{"email": user.Email}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

type AccessUser struct {
	ID      bson.ObjectId `bson:"_id" json:"id"`
	Type    string        `bson:"-" json:"type"`
	URL     string        `bson:"-" json:"url"`
	Related gin.H         `bson:"-" json:"related"`
	Summary *AccessType   `bson:"-" json:"summary_fields"`

	Created         time.Time     `bson:"created" json:"created"`
	Username        string        `bson:"username" json:"username" binding:"required"`
	FirstName       string        `bson:"first_name" json:"first_name"`
	LastName        string        `bson:"last_name" json:"last_name"`
	Email           string        `bson:"email" json:"email" binding:"required"`
	IsSuperUser     bool          `bson:"is_superuser" json:"is_superuser"`
	IsSystemAuditor bool          `bson:"is_system_auditor" json:"is_system_auditor"`
	Password        string        `bson:"password" json:"-"`
	OrganizationID  bson.ObjectId `bson:"organization_id" json:"organization"`
}

type AccessType struct {
	DirectAccess   []gin.H `json:"direct_access"`
	IndirectAccess []gin.H `json:"indirect_access"`
}
