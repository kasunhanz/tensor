package api

import (
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/util"
	"github.com/gin-gonic/gin"
	"net/http"
)

type LogFields struct {
	Context *gin.Context
	Status  int
	Message string
	Log     logrus.Fields
}

func AbortWithError(lg LogFields) {
	lg.Context.Error(&gin.Error{
		Type: gin.ErrorTypePrivate,
		Err:  errors.New(lg.Message),
	})

	if lg.Log != nil {
		logrus.WithFields(lg.Log).Errorln(lg.Message)
	}

	lg.Context.JSON(lg.Status, common.Error{
		Code:    lg.Status,
		Message: lg.Message,
	})
	lg.Context.Abort()
}

func AbortWithCode(c *gin.Context, status int, code int, message string) {
	c.JSON(status, common.Error{
		Code:    code,
		Message: message,
	})
	c.Abort()
}

func AbortWithErrors(c *gin.Context, status int, message string, emsgs ...string) {
	c.Error(&gin.Error{
		Type: gin.ErrorTypePrivate,
		Err:  errors.New(message),
	})
	c.JSON(status, common.Error{
		Code:    status,
		Message: message,
		Errors:  emsgs,
	})
	c.Abort()
}

// hideEncrypted is replaces encrypted fields by $encrypted$ string
func hideEncrypted(c *common.Credential) {
	encrypted := "$encrypted$"
	c.Password = encrypted
	c.SSHKeyData = encrypted
	c.SSHKeyUnlock = encrypted
	c.BecomePassword = encrypted
	c.VaultPassword = encrypted
	c.AuthorizePassword = encrypted
	c.Secret = encrypted
}

func GetAPIVersion(c *gin.Context) {
	version := gin.H{
		"available_versions": gin.H{"v1": "/v1"},
		"description":        "Tensor REST API",
		"current_version":    "/v1",
	}
	c.JSON(http.StatusOK, version)
}

func GetAPIInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"authtoken":               "/v1/authtoken",
		"ping":                    "/v1/ping",
		"config":                  "/v1/config",
		"queue":                   "/v1/queue",
		"me":                      "/v1/me",
		"dashboard":               "/v1/dashboard",
		"organizations":           "/v1/organizations",
		"users":                   "/v1/users",
		"projects":                "/v1/projects",
		"teams":                   "/v1/teams",
		"credentials":             "/v1/credentials",
		"inventory":               "/v1/inventories",
		"inventory_scripts":       "/v1/inventory_scripts",
		"inventory_sources":       "/v1/inventory_sources",
		"groups":                  "/v1/groups",
		"hosts":                   "/v1/hosts",
		"job_templates":           "/v1/job_templates",
		"jobs":                    "/v1/jobs",
		"job_events":              "/v1/job_events",
		"ad_hoc_commands":         "/v1/ad_hoc_commands",
		"system_job_templates":    "/v1/system_job_templates",
		"system_jobs":             "/v1/system_jobs",
		"terraform_jobs":          "/v1/terraform_jobs",
		"terraform_job_templates": "/v1/terraform_job_templates",
		"schedules":               "/v1/schedules",
		"roles":                   "/v1/roles",
		"notification_templates":  "/v1/notification_templates",
		"notifications":           "/v1/notifications",
		"labels":                  "/v1/labels",
		"unified_job_templates":   "/v1/unified_job_templates",
		"unified_jobs":            "/v1/unified_jobs",
		"activity_stream":         "/v1/activity_stream",
	})
}

func GetPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": util.Version,
	})
}
