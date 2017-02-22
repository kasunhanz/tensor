package util

import (
	"gopkg.in/gin-gonic/gin.v1"
	"net/http"
)

func GetAPIVersion(c *gin.Context) {
	version := gin.H{
		"available_versions": gin.H{"v1": "/v1/"},
		"description":        "Tensor REST API",
		"current_version":    "/v1/",
	}
	c.JSON(http.StatusOK, version)
}

func GetAPIInfo(c *gin.Context) {
	info := gin.H{
		"authtoken":              "/v1/authtoken/",
		"ping":                   "/v1/ping/",
		"config":                 "/v1/config/",
		"queue":                  "/v1/queue/",
		"me":                     "/v1/me/",
		"dashboard":              "/v1/dashboard/",
		"organizations":          "/v1/organizations/",
		"users":                  "/v1/users/",
		"projects":               "/v1/projects/",
		"teams":                  "/v1/teams/",
		"credentials":            "/v1/credentials/",
		"inventory":              "/v1/inventories/",
		"inventory_scripts":      "/v1/inventory_scripts/",
		"inventory_sources":      "/v1/inventory_sources/",
		"groups":                 "/v1/groups/",
		"hosts":                  "/v1/hosts/",
		"job_templates":          "/v1/job_templates/",
		"jobs":                   "/v1/jobs/",
		"job_events":             "/v1/job_events/",
		"ad_hoc_commands":        "/v1/ad_hoc_commands/",
		"system_job_templates":   "/v1/system_job_templates/",
		"system_jobs":            "/v1/system_jobs/",
		"schedules":              "/v1/schedules/",
		"roles":                  "/v1/roles/",
		"notification_templates": "/v1/notification_templates/",
		"notifications":          "/v1/notifications/",
		"labels":                 "/v1/labels/",
		"unified_job_templates":  "/v1/unified_job_templates/",
		"unified_jobs":           "/v1/unified_jobs/",
		"activity_stream":        "/v1/activity_stream/",
	}
	c.JSON(http.StatusOK, info)
}

func GetPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": Version,
	})
}
