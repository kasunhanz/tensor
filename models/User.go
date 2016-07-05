package models

import (
	"time"

	database "pearson.com/hilbert-space/db"
	"gopkg.in/mgo.v2/bson"
)

// User is model for user collection
type User struct {
	ID       bson.ObjectId       `bson:"_id" json:"id"`
	Created  time.Time `bson:"created" json:"created"`
	Username string    `bson:"username" json:"username" binding:"required"`
	Name     string    `bson:"name" json:"name" binding:"required"`
	Email    string    `bson:"email" json:"email" binding:"required"`
	Password string    `bson:"password" json:"-"`
}

// FetchUser is for retrieve user by userID
// userID is a bson.ObjectId
// return the User interface if found otherwise returns an error
func FetchUser(userID bson.ObjectId) (*User, error) {
	var user User

	c := database.MongoDb.C("user")
	err := c.FindId(userID).One(&user)

	return &user, err
}
