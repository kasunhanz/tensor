package api

import "github.com/gin-gonic/gin"

type DashBoardController struct{}

// GetInfo is a Gin handler function which returns summary data for UI dashboard
func (ctrl DashBoardController) GetInfo(c *gin.Context) {
	info := gin.H{
		"related": gin.H{
			"jobs_graph": "/v1/dashboard/graphs/jobs/",
		},
		"inventories": gin.H{
			"url":                         "/v1/inventories/",
			"job_failed":                  0,
			"total":                       1,
			"inventory_failed":            0,
			"total_with_inventory_source": 0,
		},
		"inventory_sources": gin.H{
			"ec2": gin.H{
				"url":          "/v1/inventory_sources/?source=ec2",
				"total":        0,
				"failures_url": "/v1/inventory_sources/?source=ec2&status=failed",
				"failed":       0,
				"label":        "Amazon EC2",
			},
			"rax": gin.H{
				"url":          "/v1/inventory_sources/?source=rax",
				"total":        0,
				"failures_url": "/v1/inventory_sources/?source=rax&status=failed",
				"failed":       0,
				"label":        "Rackspace",
			},
		},
		"groups": gin.H{
			"url":              "/v1/groups/",
			"total":            1,
			"failures_url":     "/v1/groups/?has_active_failures=True",
			"inventory_failed": 0,
			"job_failed":       0,
		},
		"hosts": gin.H{
			"url":          "/v1/hosts/",
			"total":        3,
			"failures_url": "/v1/hosts/?has_active_failures=True",
			"failed":       0,
		},
		"projects": gin.H{
			"url":          "/v1/projects/",
			"total":        3,
			"failures_url": "/v1/projects/?last_job_failed=True",
			"failed":       2,
		},
		"scm_types": gin.H{
			"svn": gin.H{
				"url":          "/v1/projects/?scm_type=svn",
				"total":        0,
				"failures_url": "/v1/projects/?scm_type=svn&last_job_failed=True",
				"failed":       0,
				"label":        "Subversion",
			},
			"git": gin.H{
				"url":          "/v1/projects/?scm_type=git",
				"total":        3,
				"failures_url": "/v1/projects/?scm_type=git&last_job_failed=True",
				"failed":       2,
				"label":        "Git",
			},
			"hg": gin.H{
				"url":          "/v1/projects/?scm_type=hg",
				"total":        0,
				"failures_url": "/v1/projects/?scm_type=hg&last_job_failed=True",
				"failed":       0,
				"label":        "Mercurial",
			},
		},
		"jobs": gin.H{
			"url":         "/v1/jobs/",
			"total":       0,
			"failure_url": "/v1/jobs/?failed=True",
			"failed":      0,
		},
		"users": gin.H{
			"url":   "/v1/users/",
			"total": 6,
		},
		"organizations": gin.H{
			"url":   "/v1/organizations/",
			"total": 1,
		},
		"teams": gin.H{
			"url":   "/v1/teams/",
			"total": 3,
		},
		"credentials": gin.H{
			"url":   "/v1/credentials/",
			"total": 3,
		},
		"job_templates": gin.H{
			"url":   "/v1/job_templates/",
			"total": 2,
		},
	}

	c.JSON(201, info)
}
