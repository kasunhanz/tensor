package metadata

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"github.com/gin-gonic/gin"
)

func CredentialMetadata(cred *models.Credential) error {

	ID := cred.ID.Hex()
	cred.Type = "credential"
	cred.Url = "/v1/credentials/" + ID + "/"
	related := gin.H{
		"created_by": "/v1/users/" + cred.CreatedByID.Hex() + "/",
		"modified_by": "/v1/users/" + cred.ModifiedByID.Hex() + "/",
		"owner_teams": "/v1/credentials/" + ID + "/owner_teams/",
		"owner_users": "/v1/credentials/" + ID + "/owner_users/",
		"activity_stream": "/v1/credentials/" + ID + "/activity_stream/",
		"access_list": "/v1/credentials/" + ID + "/access_list/",
		"object_roles": "/api/v1/credentials/" + ID + "/object_roles/",
		"user": "/v1/users/" + cred.CreatedByID.Hex() + "/",
	}

	if cred.OrganizationID != nil {
		related["organization"] = "/api/v1/organizations/" + *cred.OrganizationID + "/"
	}

	cred.Related = related

	if err := credentialSummary(cred); err != nil {
		return err
	}

	return nil
}

func credentialSummary(cred *models.Credential) error {

	var modified models.User
	var created models.User
	var org models.Organization
	var owners []gin.H

	if err := db.Users().FindId(cred.CreatedByID).One(&created); err != nil {
		return err
	}

	if err := db.Users().FindId(cred.ModifiedByID).One(&modified); err != nil {
		return err
	}

	summary := gin.H{
		"host": gin.H{}, // TODO: implement
		"project": gin.H{}, // TODO: implement
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
	}


	// owners
	// teams and users
	for _, v := range cred.Roles {
		switch v.Type {
		case "user": {
			var user models.User
			if err := db.Users().FindId(v.UserID).One(&user); err != nil {
				return err
			}
			owners = append(owners, gin.H{
				"url": "/v1/users/" + v.UserID + "/",
				"description": "",
				"type": "user",
				"id": v.UserID,
				"name": user.Username,
			})
		}
		case "team": {
			var team models.Team
			if err := db.Teams().FindId(v.TeamID).One(&team); err != nil {
				return err
			}
			owners = append(owners, gin.H{
				"url": "/v1/teams/" + v.TeamID + "/",
				"description": team.Description,
				"type": "team",
				"id": v.TeamID,
				"name": team.Name,
			})
		}
		}
	}

	if cred.OrganizationID != nil {
		if err := db.Organizations().FindId(*cred.OrganizationID).One(&org); err != nil {
			return err
		}
		owners = append(owners, gin.H{
			"url": "/v1/organizations/" + *cred.OrganizationID + "/",
			"description": org.Description,
			"type": "organization",
			"id": org.ID,
			"name": org.Name,
		})

		summary["organization"] = gin.H{
			"id": org.ID,
			"name": org.Name,
			"description": org.Description,
		}
	}

	summary["owners"] = owners

	cred.Summary = summary;

	return nil
}