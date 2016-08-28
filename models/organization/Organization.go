package organization

import (
	"time"

	database "bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/models/user"
)

// Organization is the model for organization
// collection
type Organization struct {
	ID            bson.ObjectId `bson:"_id" json:"id"`
	Type          string `bson:"-" json:"type"`
	Url           string `bson:"-" json:"url"`
	Related       map[string]string `bson:"-" json:"related"`
	SummaryFields map[string]interface{} `bson:"-" json:"summary_fields"`
	Name          string        `bson:"name" json:"name" binding:"required"`
	Description   string        `bson:"description" json:"description"`
	CreatedBy     bson.ObjectId  `bson:"created_by" json:"created_by"`
	ModifiedBy    bson.ObjectId  `bson:"modified_by" json:"modified_by"`
	Created       time.Time     `bson:"created" json:"created"`
	Modified      time.Time     `bson:"modified" json:"modified"`
	Users         []OrganizationUser `bson:"users" json:"-"`
}

type OrganizationUser struct {
	ID     bson.ObjectId `bson:"_id" json:"id"`
	UserId bson.ObjectId `bson:"user_id" json:"user_id"`
}

// Create a new organization
func (organization Organization) Insert() error {
	c := database.MongoDb.C("organizations")
	return c.Insert(organization)
}


// Create a new organization
func (org *Organization) IncludeMetadata() {

	org.Type = "organization"
	org.Url = "/v1/organizations/" + org.ID.Hex() + "/"
	org.Related = map[string]string{
		"created_by": "/v1/users/" + org.CreatedBy.Hex() + "/",
		"modified_by": "/v1/users/" + org.ModifiedBy.Hex() + "/",
		"notification_templates_error": "/v1/organizations/" + org.ID.Hex() + "/notification_templates_error/",
		"notification_templates_success": "/v1/organizations/" + org.ID.Hex() + "/notification_templates_success/",
		"users": "/v1/organizations/" + org.ID.Hex() + "/users/",
		"object_roles": "/v1/organizations/" + org.ID.Hex() + "/object_roles/",
		"notification_templates_any":  "/v1/organizations/" + org.ID.Hex() + "/notification_templates_any/",
		"teams": "/v1/organizations/" + org.ID.Hex() + "/teams/",
		"access_list": "/v1/organizations/" + org.ID.Hex() + "/access_list/",
		"notification_templates": "/v1/organizations/" + org.ID.Hex() + "/notification_templates/",
		"admins": "/v1/organizations/" + org.ID.Hex() + "/admins/",
		"credentials": "/v1/organizations/" + org.ID.Hex() + "/credentials/",
		"inventories":  "/v1/organizations/" + org.ID.Hex() + "/inventories/",
		"activity_stream": "/v1/organizations/" + org.ID.Hex() + "/activity_stream/",
		"projects": "/v1/organizations/" + org.ID.Hex() + "/projects/",
	}
	org.setSummaryFields()
}

func (org *Organization) setSummaryFields() {
	s := map[string]interface{}{
		"object_roles": []map[string]string{
			{
				"Description": "Can view all settings for the organization",
				"Name":"Auditor",
			},
			{
				"Description":"Can manage all aspects of the organization",
				"Name":"Admin",
			},
			{
				"Description":"User is a member of the organization",
				"Name":"Member",
			},
			{
				"Description":"May view settings for the organization",
				"Name":"Read",
			},
		},
		"related_field_counts": map[string]int{
			"job_templates":1,
			"users":2,
			"teams":2,
			"admins":2,
			"inventories":1,
			"projects":1,
		},
	}

	var u user.User

	c := database.MongoDb.C("users")

	if err := c.FindId(org.CreatedBy).One(&u); err != nil {
		panic(err)
	}

	s["created_by"] = map[string]interface{}{
		"id":u.ID,
		"username":u.Username,
		"first_name":u.FirstName,
		"last_name":u.LastName,
	}

	if err := c.FindId(org.ModifiedBy).One(&u); err != nil {
		panic(err)
	}

	s["modified_by"] = map[string]interface{}{
		"id":u.ID,
		"username":u.Username,
		"first_name":u.FirstName,
		"last_name":u.LastName,
	}

	org.SummaryFields = s
}