package jtemplate

import (
	"net/http"
	"strconv"

	"github.com/pearsonappeng/tensor/api/metadata"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/models/terraform"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/gin-gonic/gin.v1"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/mgo.v2/bson"
	"github.com/pearsonappeng/tensor/rbac"
)

// AccessList is Gin Handler function
func AccessList(c *gin.Context) {
	jobTemplate := c.MustGet(CTXJobTemplate).(terraform.JobTemplate)

	var project common.Project
	err := db.Projects().Find(bson.M{"project_id": jobTemplate.ProjectID}).One(&project)
	if err != nil {
		log.Errorln("Error while retriving Project:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting AccessList"},
		})
		return
	}

	var organization common.Organization
	err = db.Organizations().FindId(project.OrganizationID).One(&organization)
	if err != nil {
		log.Errorln("Error while retriving Organization:", err)
		c.JSON(http.StatusInternalServerError, common.Error{
			Code:     http.StatusInternalServerError,
			Messages: []string{"Error while getting AccessList"},
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
