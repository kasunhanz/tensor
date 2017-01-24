package credentials

import (
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/roles"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/mgo.v2/bson"
)

// AccessList is a Gin handler function, returns access list
// for specifed credential object
func AccessList(c *gin.Context) {
	credential := c.MustGet(CTXCredential).(common.Credential)

	var organization common.Organization
	err := db.Organizations().FindId(credential.OrganizationID).One(&organization)
	if err != nil {
		log.Errorln("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting Access List"},
		})
		return
	}

	var allaccess map[bson.ObjectId]*common.AccessType

	// indirect access from organization
	for _, v := range organization.Roles {
		if v.Type == "user" {
			// if an organization admin
			switch v.Role {
			case roles.ORGANIZATION_ADMIN:
				{
					access := gin.H{
						"descendant_roles": []string{
							"admin",
							"use",
							"read",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "Can manage all aspects of the organization",
							"related": gin.H{
								"organization": "/v1/organizations/" + organization.ID.Hex() + "/",
							},
							"resource_type": "organization",
							"name":          roles.ORGANIZATION_ADMIN,
						},
					}

					allaccess[v.UserID].IndirectAccess = append(allaccess[v.UserID].IndirectAccess, access)
				}
			// if an organization auditor or member
			case roles.ORGANIZATION_MEMBER:
				{
					access := gin.H{
						"descendant_roles": []string{
							"read",
							"use",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "User is a member of the Organization",
							"related": gin.H{
								"organization": "/v1/organizations/" + organization.ID.Hex() + "/",
							},
							"resource_type": "organization",
							"name":          roles.ORGANIZATION_MEMBER,
						},
					}

					allaccess[v.UserID].IndirectAccess = append(allaccess[v.UserID].IndirectAccess, access)
				}
			// if an organization auditor
			case roles.ORGANIZATION_AUDITOR:
				{
					access := gin.H{
						"descendant_roles": []string{
							"read",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "Can view all aspects of the organization",
							"related": gin.H{
								"organization": "/v1/organizations/" + organization.ID.Hex() + "/",
							},
							"resource_type": "organization",
							"name":          roles.ORGANIZATION_AUDITOR,
						},
					}
					allaccess[v.UserID].IndirectAccess = append(allaccess[v.UserID].IndirectAccess, access)
				}
			}
		}
	}

	// direct access

	for _, v := range credential.Roles {
		if v.Type == "user" {
			// if an inventory admin
			switch v.Role {
			case roles.CREDENTIAL_ADMIN:
				{
					access := gin.H{
						"descendant_roles": []string{
							"admin",
							"use",
							"read",
						},
						"role": gin.H{
							"resource_name": credential.Name,
							"description":   "Can manage all aspects of the credential",
							"related": gin.H{
								"inventory": "/v1/credentials/" + credential.ID.Hex() + "/",
							},
							"resource_type": "credential",
							"name":          roles.INVENTORY_ADMIN,
						},
					}

					allaccess[v.UserID].DirectAccess = append(allaccess[v.UserID].DirectAccess, access)
				}
			// if an inventory
			case roles.INVENTORY_USE:
				{
					access := gin.H{
						"descendant_roles": []string{
							"use",
							"read",
						},
						"role": gin.H{
							"resource_name": credential.Name,
							"description":   "Can use the credential in a job template",
							"related": gin.H{
								"inventory": "/v1/credentials/" + credential.ID.Hex() + "/",
							},
							"resource_type": "credential",
							"name":          roles.INVENTORY_USE,
						},
					}
					allaccess[v.UserID].DirectAccess = append(allaccess[v.UserID].DirectAccess, access)
				}
			}
		}

	}

	var usrs []common.AccessUser

	for k, v := range allaccess {
		var user common.AccessUser
		err := db.Users().FindId(k).One(&user)
		if err != nil {
			log.Errorln("Error while retriving user data:", err)
			c.JSON(http.StatusInternalServerError, common.Error{
				Code:     http.StatusInternalServerError,
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
	c.JSON(http.StatusOK, common.Response{
		Count:    count,
		Next:     pgi.NextPage(),
		Previous: pgi.PreviousPage(),
		Results:  usrs[pgi.Skip():pgi.End()],
	})

}
