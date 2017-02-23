package api

import (
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"
	"github.com/pearsonappeng/tensor/rbac"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// AccessList is a Gin handler function, returns access list
// for specified credential object
func (ctrl CredentialController) AccessList(c *gin.Context) {
	credential := c.MustGet(cCredential).(common.Credential)

	var organization common.Organization
	if err := db.Organizations().FindId(credential.OrganizationID).One(&organization); err != nil {
		log.Errorln("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Access List"},
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
							"use",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "User is a member of the Organization",
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

	for _, v := range credential.Roles {
		if v.Type == "user" {
			// if an inventory admin
			switch v.Role {
			case rbac.CredentialAdmin:
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
							"name":          rbac.InventoryAdmin,
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
							"resource_name": credential.Name,
							"description":   "Can use the credential in a job template",
							"related": gin.H{
								"inventory": "/v1/credentials/" + credential.ID.Hex() + "/",
							},
							"resource_type": "credential",
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
				Code:   http.StatusInternalServerError,
				Errors: []string{"Error while getting Access List"},
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
		Data:  usrs[pgi.Skip():pgi.End()],
	})

}

// AccessList returns the list of teams and users that is able to access
// current project object in the gin context
func (ctrl ProjectController) AccessList(c *gin.Context) {
	project := c.MustGet(cProject).(common.Project)

	var organization common.Organization
	err := db.Organizations().FindId(project.OrganizationID).One(&organization)
	if err != nil {
		log.Errorln("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Access List"},
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
							"admin",
							"execute",
							"read",
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
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "Can manage all aspects of the organization",
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
							"description":   "Can manage all aspects of the organization",
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

	for _, v := range project.Roles {
		if v.Type == "user" {
			// if an job template admin
			switch v.Role {
			case rbac.JobTemplateAdmin:
				{
					access := gin.H{
						"descendant_roles": []string{
							"admin",
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": project.Name,
							"description":   "May run the job template",
							"related": gin.H{
								"job_template": "/v1/job_templates/" + project.ID.Hex() + "/",
							},
							"resource_type": "job_template",
							"name":          rbac.JobTemplateAdmin,
						},
					}

					allaccess[v.GranteeID].DirectAccess = append(allaccess[v.GranteeID].DirectAccess, access)
				}
			// if an job template execute
			case rbac.JobTemplateExecute:
				{
					access := gin.H{
						"descendant_roles": []string{
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": project.Name,
							"description":   "Can manage all aspects of the job template",
							"related": gin.H{
								"job_template": "/v1/job_templates/" + project.ID.Hex() + "/",
							},
							"resource_type": "job_template",
							"name":          rbac.JobTemplateExecute,
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
				Code:   http.StatusInternalServerError,
				Errors: []string{"Error while getting Access List"},
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
		Data:  usrs[pgi.Skip():pgi.End()],
	})

}

func (ctrl TeamController) AccessList(c *gin.Context) {
	team := c.MustGet(cTeam).(common.Team)

	var organization common.Organization
	err := db.Organizations().FindId(team.OrganizationID).One(&organization)
	if err != nil {
		log.Errorln("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Access List"},
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
							"admin",
							"execute",
							"read",
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
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "Can manage all aspects of the organization",
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
							"description":   "Can manage all aspects of the organization",
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

	for _, v := range team.Roles {
		if v.Type == "user" {
			// if an job template admin
			switch v.Role {
			case rbac.TeamAdmin:
				{
					access := gin.H{
						"descendant_roles": []string{
							rbac.TeamAdmin,
							rbac.TeamMember,
						},
						"role": gin.H{
							"resource_name": team.Name,
							"description":   "Can manage all aspects of the team",
							"related": gin.H{
								"team": "/v1/teams/" + team.ID.Hex() + "/",
							},
							"resource_type": "team",
							"name":          rbac.TeamAdmin,
						},
					}

					allaccess[v.GranteeID].DirectAccess = append(allaccess[v.GranteeID].DirectAccess, access)
				}
			// if an job template execute
			case rbac.TeamMember:
				{
					access := gin.H{
						"descendant_roles": []string{
							"member",
							"read",
						},
						"role": gin.H{
							"resource_name": team.Name,
							"description":   "User is a member of the team",
							"related": gin.H{
								"team": "/api/v1/teams/" + team.ID.Hex() + "/",
							},
							"resource_type": "team",
							"name":          rbac.TeamMember,
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
				Code:   http.StatusInternalServerError,
				Errors: []string{"Error while getting Access List"},
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
		Data:  usrs[pgi.Skip():pgi.End()],
	})

}

func (ctrl InventoryController) AccessList(c *gin.Context) {
	inventory := c.MustGet(cInventory).(ansible.Inventory)

	var organization common.Organization
	err := db.Organizations().FindId(inventory.OrganizationID).One(&organization)
	if err != nil {
		log.Errorln("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting Access List"},
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
				Code:   http.StatusInternalServerError,
				Errors: []string{"Error while getting Access List"},
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
		Data:  usrs[pgi.Skip():pgi.End()],
	})

}

// AccessList is Gin Handler function
func (ctrl JobTemplateController) AccessList(c *gin.Context) {
	jobTemplate := c.MustGet(cJobTemplate).(ansible.JobTemplate)

	var project common.Project
	err := db.Projects().Find(bson.M{"project_id": jobTemplate.ProjectID}).One(&project)
	if err != nil {
		log.Errorln("Error while retriving Project:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting AccessList"},
		})
		return
	}

	var organization common.Organization
	err = db.Organizations().FindId(project.OrganizationID).One(&organization)
	if err != nil {
		log.Errorln("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting AccessList"},
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
							"admin",
							"execute",
							"read",
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
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "Can manage all aspects of the organization",
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
							"description":   "Can manage all aspects of the organization",
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

	for _, v := range jobTemplate.Roles {
		if v.Type == "user" {
			// if an job template admin
			switch v.Role {
			case rbac.JobTemplateAdmin:
				{
					access := gin.H{
						"descendant_roles": []string{
							"admin",
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": jobTemplate.Name,
							"description":   "May run the job template",
							"related": gin.H{
								"job_template": "/v1/job_templates/" + jobTemplate.ID.Hex() + "/",
							},
							"resource_type": "job_template",
							"name":          rbac.JobTemplateAdmin,
						},
					}

					allaccess[v.GranteeID].DirectAccess = append(allaccess[v.GranteeID].DirectAccess, access)
				}
			// if an job template execute
			case rbac.JobTemplateExecute:
				{
					access := gin.H{
						"descendant_roles": []string{
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": jobTemplate.Name,
							"description":   "Can manage all aspects of the job template",
							"related": gin.H{
								"job_template": "/api/v1/job_templates/" + jobTemplate.ID.Hex() + "/",
							},
							"resource_type": "job_template",
							"name":          rbac.JobTemplateExecute,
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
				Code:   http.StatusInternalServerError,
				Errors: []string{"Error while getting Access List"},
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
		Data:  usrs[pgi.Skip():pgi.End()],
	})

}

// AccessList is Gin Handler function
func (ctrl TJobTmplController) AccessList(c *gin.Context) {
	jobTemplate := c.MustGet(cJobTemplate).(terraform.JobTemplate)

	var project common.Project
	err := db.Projects().Find(bson.M{"project_id": jobTemplate.ProjectID}).One(&project)
	if err != nil {
		log.Errorln("Error while retriving Project:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting AccessList"},
		})
		return
	}

	var organization common.Organization
	err = db.Organizations().FindId(project.OrganizationID).One(&organization)
	if err != nil {
		log.Errorln("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:   http.StatusInternalServerError,
			Errors: []string{"Error while getting AccessList"},
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
							"admin",
							"execute",
							"read",
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
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": organization.Name,
							"description":   "Can manage all aspects of the organization",
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
							"description":   "Can manage all aspects of the organization",
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

	for _, v := range jobTemplate.Roles {
		if v.Type == "user" {
			// if an job template admin
			switch v.Role {
			case rbac.JobTemplateAdmin:
				{
					access := gin.H{
						"descendant_roles": []string{
							"admin",
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": jobTemplate.Name,
							"description":   "May run the job template",
							"related": gin.H{
								"job_template": "/v1/job_templates/" + jobTemplate.ID.Hex() + "/",
							},
							"resource_type": "job_template",
							"name":          rbac.JobTemplateAdmin,
						},
					}

					allaccess[v.GranteeID].DirectAccess = append(allaccess[v.GranteeID].DirectAccess, access)
				}
			// if an job template execute
			case rbac.JobTemplateExecute:
				{
					access := gin.H{
						"descendant_roles": []string{
							"execute",
							"read",
						},
						"role": gin.H{
							"resource_name": jobTemplate.Name,
							"description":   "Can manage all aspects of the job template",
							"related": gin.H{
								"job_template": "/api/v1/job_templates/" + jobTemplate.ID.Hex() + "/",
							},
							"resource_type": "job_template",
							"name":          rbac.JobTemplateExecute,
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
				Code:   http.StatusInternalServerError,
				Errors: []string{"Error while getting Access List"},
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
		Data:  usrs[pgi.Skip():pgi.End()],
	})

}
