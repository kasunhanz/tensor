package metadata

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
)



// Create a new organization
func TeamMetadata(o *models.Team) error {

	o.Type = "team"
	o.Url = "/v1/teams/" + o.ID.Hex() + "/"
	o.Related = gin.H{
		"created_by": "/v1/users/" + o.CreatedBy.Hex() + "/",
		"modified_by": "/v1/users/" + o.ModifiedBy.Hex() + "/",
		"users": "/v1/teams/" + o.ID.Hex() + "/users/",
		"roles": "/v1/teams/" + o.ID.Hex() + "/roles/",
		"object_roles": "/v1/teams/" + o.ID.Hex() + "/object_roles/",
		"credentials": "/v1/teams/" + o.ID.Hex() + "/credentials/",
		"projects": "/v1/teams/" + o.ID.Hex() + "/projects/",
		"activity_stream": "/v1/teams/" + o.ID.Hex() + "/activity_stream/",
		"access_list": "/v1/teams/" + o.ID.Hex() + "/access_list/",
		"organization": "/v1/organizations/" + o.OrganizationID.Hex() + "/",
	}

	if err := teamSummary(o); err != nil {
		return err
	}

	return nil
}

func teamSummary(o *models.Team) error {

	var modified models.User
	var created models.User
	var org models.Organization

	if err := db.Users().FindId(o.CreatedBy).One(&created); err != nil {
		return err
	}

	if err := db.Users().FindId(o.ModifiedBy).One(&modified); err != nil {
		return err
	}

	if err := db.Organizations().FindId(o.OrganizationID).One(&org); err != nil {
		return err
	}

	o.SummaryFields = gin.H{
		"organization": gin.H{
			"id": org.ID,
			"name": org.Name,
			"description": org.Description,
		},
		"object_roles": gin.H{
			"admin_role": gin.H{
				"description": "Can manage all aspects of the team",
				"name": "admin",
			},
			"member_role": gin.H{
				"description": "User is a member of the team",
				"name": "member",
			},
			"read_role": gin.H{
				"description": "May view settings for the team",
				"name": "read",
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

	return nil
}
