package metadata

import (
	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/gin-gonic/gin.v1"
)

func CredentialMetadata(c *common.Credential) {

	ID := c.ID.Hex()
	c.Type = "credential"
	c.URL = "/v1/credentials/" + ID + "/"
	related := gin.H{
		"created_by":      "/v1/users/" + c.CreatedByID.Hex() + "/",
		"modified_by":     "/v1/users/" + c.ModifiedByID.Hex() + "/",
		"owner_teams":     "/v1/credentials/" + ID + "/owner_teams/",
		"owner_users":     "/v1/credentials/" + ID + "/owner_users/",
		"activity_stream": "/v1/credentials/" + ID + "/activity_stream/",
		"access_list":     "/v1/credentials/" + ID + "/access_list/",
		"object_roles":    "/api/v1/credentials/" + ID + "/object_roles/",
		"user":            "/v1/users/" + c.CreatedByID.Hex() + "/",
	}

	if c.OrganizationID != nil {
		related["organization"] = "/api/v1/organizations/" + (*c.OrganizationID).Hex() + "/"
	}

	c.Related = related
	credentialSummary(c)
}

func credentialSummary(c *common.Credential) {

	var modified common.User
	var created common.User
	var org common.Organization
	var owners []gin.H

	summary := gin.H{
		"object_roles": []gin.H{
			{
				"Description": "Can manage all aspects of the credential",
				"Name":        "admin",
			},
			{
				"Description": "Can use the credential in a job template",
				"Name":        "use",
			},
			{
				"Description": "May view settings for the credential",
				"Name":        "read",
			},
		},
		"created_by":  nil,
		"modified_by": nil,
		"owners":      nil,
	}

	if err := db.Users().FindId(c.CreatedByID).One(&created); err != nil {
		log.WithFields(log.Fields{
			"User ID":       c.CreatedByID.Hex(),
			"Credential":    c.Name,
			"Credential ID": c.ID.Hex(),
		}).Errorln("Error while getting created by User")
	} else {
		summary["created_by"] = gin.H{
			"id":         modified.ID,
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	if err := db.Users().FindId(c.ModifiedByID).One(&modified); err != nil {
		log.WithFields(log.Fields{
			"User ID":       c.ModifiedByID.Hex(),
			"Credential":    c.Name,
			"Credential ID": c.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	} else {
		summary["modified_by"] = gin.H{
			"id":         modified.ID,
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	// owners
	// teams and users
	for _, v := range c.Roles {
		switch v.Type {
		case "user":
			{
				var user common.User
				if err := db.Users().FindId(v.UserID).One(&user); err != nil {
					log.WithFields(log.Fields{
						"User ID":       v.UserID.Hex(),
						"Credential":    c.Name,
						"Credential ID": c.ID.Hex(),
					}).Warnln("Error while getting owner user")
					continue
				}
				owners = append(owners, gin.H{
					"url":         "/v1/users/" + v.UserID + "/",
					"description": "",
					"type":        "user",
					"id":          v.UserID,
					"name":        user.Username,
				})
			}
		case "team":
			{
				var team common.Team
				if err := db.Teams().FindId(v.TeamID).One(&team); err != nil {
					log.WithFields(log.Fields{
						"Team ID":       v.TeamID.Hex(),
						"Credential":    c.Name,
						"Credential ID": c.ID.Hex(),
					}).Warnln("Error while getting owner team")
					continue
				}
				owners = append(owners, gin.H{
					"url":         "/v1/teams/" + v.TeamID + "/",
					"description": team.Description,
					"type":        "team",
					"id":          v.TeamID,
					"name":        team.Name,
				})
			}
		}
	}

	if c.OrganizationID != nil {
		if err := db.Organizations().FindId(*c.OrganizationID).One(&org); err != nil {
			log.WithFields(log.Fields{
				"Organization ID": (*c.OrganizationID).Hex(),
				"Credential":      c.Name,
				"Credential ID":   c.ID.Hex(),
			}).Warnln("Error while getting Organization")
		} else {
			owners = append(owners, gin.H{
				"url":         "/v1/organizations/" + *c.OrganizationID + "/",
				"description": org.Description,
				"type":        "organization",
				"id":          org.ID,
				"name":        org.Name,
			})

			summary["organization"] = gin.H{
				"id":          org.ID,
				"name":        org.Name,
				"description": org.Description,
			}
		}
	}

	summary["owners"] = owners

	c.Summary = summary
}
