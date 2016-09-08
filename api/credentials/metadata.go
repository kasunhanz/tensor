package credentials

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
	r := gin.H{
		"created_by": "/v1/users/" + cred.CreatedByID + "/",
		"modified_by": "/v1/users/" + cred.ModifiedByID + "/",
		"owner_teams": "/v1/organizations/" + cred.ID + "/users/",
		"owner_users": "/v1/organizations/" + cred.ID + "/object_roles/",
		"activity_stream": "/v1/organizations/" + cred.ID + "/activity_stream/",
		"access_list": "/v1/organizations/" + cred.ID + "/access_list/",
		"object_roles": "/api/v1/credentials/" + cred.ID + "/object_roles/",
		"user": "/api/v1/users/" + cred.CreatedByID + "/",
	}

	if cred.OrganizationID != "" {
		r["organization"] = "/api/v1/organizations/" + cred.OrganizationID + "/"
	}

	cred.Related = r

	if err := setSummaryFields(cred); err != nil {
		return err
	}

	return nil
}

func setSummaryFields(cred *models.Credential) error {
	dbu := database.MongoDb.C(models.DBC_USERS)
	dbacl := database.MongoDb.C(models.DBC_ACl)

	var modified models.User
	var created models.User
	var org models.Organization
	var owners []models.User

	if err := dbu.FindId(cred.CreatedByID).One(&created); err != nil {
		return err
	}

	if err := dbu.FindId(cred.ModifiedByID).One(&modified); err != nil {
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

	//TODO: include teams to owners list

	o := gin.H{
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

	if cred.OrganizationID != "" {

		if err := dbu.FindId(cred.OrganizationID).One(&org); err != nil {
			return err
		}

		o["organization"] = gin.H{
			"id": org.ID,
			"name": org.Name,
			"description": org.Description,
		}
	}

	return nil
}