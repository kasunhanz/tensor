package users

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
)



// Create a new organization
func SetMetadata(u *models.User) {
	u.Type = "user"
	u.Url = "/v1/users/" + u.ID.Hex() + "/"
	u.Related = map[string]string{
		"admin_of_organizations": "/api/v1/users/" + u.ID.Hex() + "/admin_of_organizations/",
		"organizations": "/api/v1/users/" + u.ID.Hex() + "/organizations/",
		"roles": "/api/v1/users/" + u.ID.Hex() + "/roles/",
		"access_list": "/api/v1/users/" + u.ID.Hex() + "/access_list/",
		"teams": "/api/v1/users/" + u.ID.Hex() + "/teams/",
		"credentials": "/api/v1/users/" + u.ID.Hex() + "/credentials/",
		"activity_stream": "/api/v1/users/" + u.ID.Hex() + "/activity_stream/",
		"projects": "/api/v1/users/" + u.ID.Hex() + "/projects/",
	}
}

func setSummaryFields(o *models.Organization) error {

	//	dbacl := database.MongoDb.C(models.DBC_ACl)


	/*q := []bson.M{
		{"$match": bson.M{"object": o.ID}},
		{
			"$lookup": bson.M{
				"from": "users",
				"localField": "user_id",
				"foreignField": "_id",
				"as": "user",
			},
		},
		{"$project": bson.M{"_id":0, "users":bson.M{"$arrayElemAt": []interface{}{"$user", 0} }, }},
		{"$project": bson.M{
			"_id":"$users._id",
			"created":"$users.created",
			"email":"$users.email",
			"name":"$users.name",
			"password":"$users.password",
			"username":"$users.username",
		}},
	}
*/
	/*if err := dbacl.Pipe(q).All(&owners); err != nil {
		return err
	}*/

	o.SummaryFields = gin.H{
		"direct_access": []gin.H{
			{
				"Description": "Can view all settings for the organization",
				"Name":"auditor",
			},
			{
				"Description":"Can manage all aspects of the organization",
				"Name":"admin",
			},
			{
				"Description":"User is a member of the organization",
				"Name":"member",
			},
			{
				"Description":"May view settings for the organization",
				"Name":"read",
			},
		},
		"indirect_access": []gin.H{
			{
				"Description": "Can view all settings for the organization",
				"Name":"auditor",
			},
			{
				"Description":"Can manage all aspects of the organization",
				"Name":"admin",
			},
			{
				"Description":"User is a member of the organization",
				"Name":"member",
			},
			{
				"Description":"May view settings for the organization",
				"Name":"read",
			},
		},
	}

	return nil
}
