package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
)

const DBC_USERS = "users"

// User is model for user collection
type User struct {
	ID              bson.ObjectId `bson:"_id" json:"id"`
	Type            string        `bson:"-" json:"type"`
	Url             string        `bson:"-" json:"url"`
	Related         gin.H         `bson:"-" json:"related"`
	Created         time.Time     `bson:"created" json:"created"`
	Username        string        `bson:"username" json:"username" binding:"required"`
	FirstName       string        `bson:"first_name" json:"first_name"`
	LastName        string        `bson:"last_name" json:"last_name"`
	Email           string        `bson:"email" json:"email" binding:"required"`
	IsSuperUser     bool          `bson:"is_superuser" json:"is_superuser"`
	IsSystemAuditor bool          `bson:"is_system_auditor" json:"is_system_auditor"`
	Password        string        `bson:"password" json:"-"`
}

func (u User) CreateIndexes() {

	// Collection People
	c := db.C(DBC_USERS)

	// Unique index username
	if err := c.EnsureIndex(mgo.Index{
		Key:        []string{"username"},
		Unique:     true,
		Background: true,
	}) err != nil {
		log.Println("Failed to create Unique Index for username of ", DBC_USERS, "Collection")
	}

	// Unique index email
	if err := c.EnsureIndex(mgo.Index{
		Key:        []string{"email"},
		Unique:     true,
		Background: true,
	}) err != nil {
		log.Println("Failed to create Unique Index for username of ", DBC_USERS, "Collection")
	}

}