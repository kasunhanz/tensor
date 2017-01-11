package metadata

import (
	"github.com/gamunu/tensor/db"
	"github.com/gamunu/tensor/models"
	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

// Create a new organization
func TeamMetadata(tm *models.Team) {

	tm.Type = "team"
	tm.Url = "/v1/teams/" + tm.ID.Hex() + "/"
	tm.Related = gin.H{
		"created_by":      "/v1/users/" + tm.CreatedByID.Hex() + "/",
		"modified_by":     "/v1/users/" + tm.ModifiedByID.Hex() + "/",
		"users":           "/v1/teams/" + tm.ID.Hex() + "/users/",
		"roles":           "/v1/teams/" + tm.ID.Hex() + "/roles/",
		"object_roles":    "/v1/teams/" + tm.ID.Hex() + "/object_roles/",
		"credentials":     "/v1/teams/" + tm.ID.Hex() + "/credentials/",
		"projects":        "/v1/teams/" + tm.ID.Hex() + "/projects/",
		"activity_stream": "/v1/teams/" + tm.ID.Hex() + "/activity_stream/",
		"access_list":     "/v1/teams/" + tm.ID.Hex() + "/access_list/",
		"organization":    "/v1/organizations/" + tm.OrganizationID.Hex() + "/",
	}

	teamSummary(tm)
}

func teamSummary(tm *models.Team) {

	var modified models.User
	var created models.User
	var org models.Organization

	summary := gin.H{
		"organization": nil,
		"object_roles": gin.H{
			"admin_role": gin.H{
				"description": "Can manage all aspects of the team",
				"name":        "admin",
			},
			"member_role": gin.H{
				"description": "User is a member of the team",
				"name":        "member",
			},
			"read_role": gin.H{
				"description": "May view settings for the team",
				"name":        "read",
			},
		},
		"created_by":  nil,
		"modified_by": nil,
	}

	if err := db.Users().FindId(tm.CreatedByID).One(&created); err != nil {
		log.WithFields(log.Fields{
			"User ID": tm.CreatedByID.Hex(),
			"Team":    tm.Name,
			"Team ID": tm.ID.Hex(),
		}).Errorln("Error while getting created by User")
	} else {
		summary["created_by"] = gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	if err := db.Users().FindId(tm.ModifiedByID).One(&modified); err != nil {
		log.WithFields(log.Fields{
			"User ID": tm.ModifiedByID.Hex(),
			"Team":    tm.Name,
			"Team ID": tm.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	} else {
		summary["modified_by"] = gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	if err := db.Organizations().FindId(tm.OrganizationID).One(&org); err != nil {
		log.WithFields(log.Fields{
			"Organization ID": tm.OrganizationID.Hex(),
			"Team":            tm.Name,
			"Team ID":         tm.ID.Hex(),
		}).Errorln("Error while getting Organization")
	} else {
		summary["organization"] = gin.H{
			"id":          org.ID,
			"name":        org.Name,
			"description": org.Description,
		}
	}

	tm.Summary = summary
}
