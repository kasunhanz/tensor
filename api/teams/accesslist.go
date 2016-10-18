package teams

import (
	"bitbucket.pearson.com/apseng/tensor/roles"
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"strconv"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/api/metadata"
)

func AccessList(c *gin.Context) {
	team := c.MustGet(_CTX_TEAM).(models.Team)

	var organization models.Organization
	err := db.Organizations().FindId(team.OrganizationID).One(&organization)
	if err != nil {
		log.Println("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:http.StatusInternalServerError,
			Messages: []string{"Error while getting Access List"},
		})
		return
	}

	var allaccess map[bson.ObjectId]*models.AccessType

	// indirect access from organization
	for _, v := range organization.Roles {
		if v.Type == "user" {
			// if an organization admin
			switch v.Role {
			case roles.ORGANIZATION_ADMIN :{
				access := gin.H{
					"descendant_roles": []string{
						"admin",
						"execute",
						"read",
					},
					"role": gin.H{
						"resource_name": organization.Name,
						"description": "Can manage all aspects of the organization",
						"related": gin.H{
							"organization": "/v1/organizations/" + organization.ID.Hex() + "/",
						},
						"resource_type": "organization",
						"name": roles.ORGANIZATION_ADMIN,
					},
				}

				allaccess[v.UserID].IndirectAccess = append(allaccess[v.UserID].IndirectAccess, access)
			}
			// if an organization auditor or member
			case roles.ORGANIZATION_MEMBER: {
				access := gin.H{
					"descendant_roles": []string{
						"execute",
						"read",
					},
					"role": gin.H{
						"resource_name": organization.Name,
						"description": "Can manage all aspects of the organization",
						"related": gin.H{
							"organization": "/v1/organizations/" + organization.ID.Hex() + "/",
						},
						"resource_type": "organization",
						"name": roles.ORGANIZATION_MEMBER,
					},
				}

				allaccess[v.UserID].IndirectAccess = append(allaccess[v.UserID].IndirectAccess, access)
			}
			// if an organization auditor
			case roles.ORGANIZATION_AUDITOR: {
				access := gin.H{
					"descendant_roles": []string{
						"read",
					},
					"role": gin.H{
						"resource_name": organization.Name,
						"description": "Can manage all aspects of the organization",
						"related": gin.H{
							"organization": "/v1/organizations/" + organization.ID.Hex() + "/",
						},
						"resource_type": "organization",
						"name": roles.ORGANIZATION_AUDITOR,
					},
				}
				allaccess[v.UserID].IndirectAccess = append(allaccess[v.UserID].IndirectAccess, access)
			}
			}
		}
	}

	// direct access

	for _, v := range team.Roles {
		if v.Type == "user" {
			// if an job template admin
			switch v.Role {
			case roles.TEAM_ADMIN :{
				access := gin.H{
					"descendant_roles": []string{
						roles.TEAM_ADMIN,
						roles.TEAM_MEMBER,
						roles.TEAM_READ,
					},
					"role": gin.H{
						"resource_name": team.Name,
						"description": "Can manage all aspects of the team",
						"related":gin.H{
							"team": "/v1/teams/" + team.ID.Hex() + "/",
						},
						"resource_type": "team",
						"name": roles.TEAM_ADMIN,
					},
				}

				allaccess[v.UserID].DirectAccess = append(allaccess[v.UserID].DirectAccess, access)
			}
			// if an job template execute
			case roles.TEAM_MEMBER: {
				access := gin.H{
					"descendant_roles": []string{
						"member",
						"read",
					},
					"role": gin.H{
						"resource_name":  team.Name,
						"description": "User is a member of the team",
						"related": gin.H{
							"team": "/api/v1/teams/" + team.ID.Hex() + "/",
						},
						"resource_type": "team",
						"name": roles.TEAM_MEMBER,
					},
				}
				allaccess[v.UserID].DirectAccess = append(allaccess[v.UserID].DirectAccess, access)
			}
			}
		}

	}

	var usrs []models.AccessUser

	for k, v := range allaccess {
		var user models.AccessUser
		err := db.Users().FindId(k).One(&user)
		if err != nil {
			log.Println("Error while retriving user data:", err)
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:http.StatusInternalServerError,
				Messages: []string{"Error while getting Access List"},
			})
			return
		}

		metadata.AccessUserMetadata(&user)
		user.Summary = v
		usrs = append(usrs, user)
	}

	count := len(usrs)
	pgi := util.NewPagination(c, count)
	//if page is incorrect return 404
	if pgi.HasPage() {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Invalid page " + strconv.Itoa(pgi.Page()) + ": That page contains no results."})
		return
	}
	// send response with JSON rendered data
	c.JSON(http.StatusOK, models.Response{
		Count:count,
		Next: pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results: usrs[pgi.Skip():pgi.End()],
	})

}
