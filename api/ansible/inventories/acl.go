package inventories

import (
	"net/http"
	"strconv"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/gin-gonic/gin.v1"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/mgo.v2/bson"
)

func AccessList(c *gin.Context) {
	inventory := c.MustGet(CTXInventory).(ansible.Inventory)

	var organization common.Organization
	err := db.Organizations().FindId(inventory.OrganizationID).One(&organization)
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
			case rbac.OrganizationAdmin:
				{
					access := gin.H{
						"descendant_roles": []string{
							"adhoc",
							"use",
							"read",
							"admin",
							"update",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "Can manage all aspects of the organization",
							"related": gin.H{
								"organization": "/v1/organizations/" + organization.ID.Hex() + "/",
							},
							"resource_type": "organization",
							"name":          rbac.OrganizationAdmin,
						},
					}

					allaccess[v.GranteeID].IndirectAccess = append(allaccess[v.GranteeID].IndirectAccess, access)
				}
			// if an organization auditor or member
			case rbac.OrganizationMember:
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
							"name":          rbac.OrganizationMember,
						},
					}

					allaccess[v.GranteeID].IndirectAccess = append(allaccess[v.GranteeID].IndirectAccess, access)
				}
			// if an organization auditor
			case rbac.OrganizationAuditor:
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
							"name":          rbac.OrganizationAuditor,
						},
					}
					allaccess[v.GranteeID].IndirectAccess = append(allaccess[v.GranteeID].IndirectAccess, access)
				}
			}
		}
	}

	// direct access

	for _, v := range inventory.Roles {
		if v.Type == "user" {
			// if an inventory admin
			switch v.Role {
			case rbac.InventoryAdmin:
				{
					access := gin.H{
						"descendant_roles": []string{
							"adhoc",
							"use",
							"read",
							"admin",
							"update",
						},
						"role": gin.H{
							"resource_name": inventory.Name,
							"description":   "Can manage all aspects of the Inventory",
							"related": gin.H{
								"inventory": "/v1/inventories/" + inventory.ID.Hex() + "/",
							},
							"resource_type": "inventory",
							"name":          rbac.InventoryAdmin,
						},
					}

					allaccess[v.GranteeID].DirectAccess = append(allaccess[v.GranteeID].DirectAccess, access)
				}
			// if an inventory execute
			case rbac.InventoryUpdate:
				{
					access := gin.H{
						"descendant_roles": []string{
							"read",
							"update",
						},
						"role": gin.H{
							"resource_name": inventory.Name,
							"description":   "Can update the Inventory",
							"related": gin.H{
								"inventory": "/v1/inventories/" + inventory.ID.Hex() + "/",
							},
							"resource_type": "inventory",
							"name":          rbac.InventoryUpdate,
						},
					}
					allaccess[v.GranteeID].DirectAccess = append(allaccess[v.GranteeID].DirectAccess, access)
				}
			// if an inventory
			case rbac.InventoryUse:
				{
					access := gin.H{
						"descendant_roles": []string{
							"use",
							"read",
						},
						"role": gin.H{
							"resource_name": inventory.Name,
							"description":   "Can use the inventory in a job template",
							"related": gin.H{
								"inventory": "/v1/inventories/" + inventory.ID.Hex() + "/",
							},
							"resource_type": "inventory",
							"name":          rbac.InventoryUse,
						},
					}
					allaccess[v.GranteeID].DirectAccess = append(allaccess[v.GranteeID].DirectAccess, access)
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
