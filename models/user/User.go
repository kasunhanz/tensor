package user

import (
	"time"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
)

// User is model for user collection
type User struct {
	ID              bson.ObjectId `bson:"_id" json:"id"`
	Type            string        `bson:"-" json:"user"`
	Url             string        `bson:"-" json:"url"`
	Related         map[string]string   `bson:"-" json:"related"`
	Created         time.Time     `bson:"created" json:"created"`
	Username        string        `bson:"username" json:"username" binding:"required"`
	FirstName       string        `bson:"first_name" json:"first_name"`
	LastName        string        `bson:"last_name" json:"last_name"`
	Email           string        `bson:"email" json:"email" binding:"required"`
	IsSuperUser     bool          `bson:"is_superuser" json:"is_superuser"`
	IsSystemAuditor bool          `bson:"is_system_auditor" json:"is_system_auditor"`
	Password        string        `bson:"password" json:"-"`
}

// FetchUser is for retrieve user by userID
// userID is a bson.ObjectId
// return the User interface if found otherwise returns an error
func FetchUser(userID bson.ObjectId) (*User, error) {
	var user User

	c := database.MongoDb.C("users")
	err := c.FindId(userID).One(&user)

	return &user, err
}

func (usr User) Insert() error {
	c := database.MongoDb.C("users")
	return c.Insert(usr)
}


// Create a new organization
func (usr *User) IncludeMetadata() {
	usr.Type = "user"
	usr.Url = "/v1/users/" + usr.ID.Hex() + "/"
	usr.Related = map[string]string{
		"admin_of_organizations": "/api/v1/users/" + usr.ID.Hex() + "/admin_of_organizations/",
		"organizations": "/api/v1/users/" + usr.ID.Hex() + "/organizations/",
		"roles": "/api/v1/users/" + usr.ID.Hex() + "/roles/",
		"access_list": "/api/v1/users/" + usr.ID.Hex() + "/access_list/",
		"teams": "/api/v1/users/" + usr.ID.Hex() + "/teams/",
		"credentials": "/api/v1/users/" + usr.ID.Hex() + "/credentials/",
		"activity_stream": "/api/v1/users/" + usr.ID.Hex() + "/activity_stream/",
		"projects": "/api/v1/users/" + usr.ID.Hex() + "/projects/",
	}
}