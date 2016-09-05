package credential

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	database "bitbucket.pearson.com/apseng/tensor/db"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// Create a new organization
func setMetadata(cred *models.Credential) error {

	cred.Type = "credential"
	cred.Url = "/v1/credentials/" + cred.ID.Hex() + "/"
	cred.Related = gin.H{
		"created_by": "/v1/users/" + cred.CreatedBy.Hex() + "/",
		"modified_by": "/v1/users/" + cred.ModifiedBy.Hex() + "/",
		"owner_teams": "/v1/organizations/" + cred.ID.Hex() + "/users/",
		"owner_users": "/v1/organizations/" + cred.ID.Hex() + "/object_roles/",
		"activity_stream": "/v1/organizations/" + cred.ID.Hex() + "/activity_stream/",
		"access_list": "/v1/organizations/" + cred.ID.Hex() + "/access_list/",
		"object_roles": "/api/v1/credentials/" + cred.ID.Hex() + "/object_roles/",
		"user": "/api/v1/users/" + cred.CreatedBy.Hex() + "/",
	}
	if err := setSummaryFields(cred); err != nil {
		return err
	}

	return nil
}

func setSummaryFields(cred *models.Credential) error {
	dbu := database.MongoDb.C(models.DBC_USER)
	dbacl := database.MongoDb.C(models.DBC_ACl)

	var modified models.User
	var created models.User
	var owners []models.User

	if err := dbu.FindId(cred.CreatedBy).One(&created); err != nil {
		return err
	}

	if err := dbu.FindId(cred.ModifiedBy).One(&modified); err != nil {
		return err
	}

	q := []bson.M{
		{"$match": bson.M{"object": cred.ID, "role": "admin", }},
		{
			"$lookup": bson.M{
				"from": "users",
				"localField": "user_id",
				"foreignField": "_id",
				"as": "owner",
			},
		},
		{"$project": bson.M{"_id":0, "users":bson.M{"$arrayElemAt": []interface{}{"$owner", 0} }, }},
		{"$project": bson.M{
			"_id":"$users._id",
			"created":"$users.created",
			"email":"$users.email",
			"name":"$users.name",
			"password":"$users.password",
			"username":"$users.username",
		}},
	}

	if err := dbacl.Pipe(q).All(&owners); err != nil {
		return err
	}

	//TODO: include teams to woners list

	cred.SummaryFields = gin.H{
		"object_roles": []gin.H{
			{
				"Description": "Can manage all aspects of the credential",
				"Name":"admin",
			},
			{
				"Description":"Can use the credential in a job template",
				"Name":"use",
			},
			{
				"Description":"May view settings for the credential",
				"Name":"read",
			},
		},
		"created_by": gin.H{
			"id":         created.ID,
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		},
		"modified_by": gin.H{
			"id":         modified.ID,
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		},
		"owners": owners,
	}

	return nil
}