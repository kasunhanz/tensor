package util

import (
	"errors"
	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const _EXP_DOMAIN_USER = `^[a-z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,4}$`

func isXHR(c *gin.Context) bool {
	accept := c.Request.Header.Get("Accept")
	if strings.Contains(accept, "text/html") {
		return false
	}

	return true
}

func AuthFailed(c *gin.Context) {
	if isXHR(c) == false {
		c.Redirect(302, "/?hai")
	} else {
		c.Writer.WriteHeader(401)
	}

	c.Abort()

	return
}

func ValidateEmail(email string) bool {
	exp := regexp.MustCompile(_EXP_DOMAIN_USER)

	if exp.MatchString(email) {
		return true
	}

	return false
}

func GetIntParam(name string, c *gin.Context) (int, error) {
	intParam, err := strconv.Atoi(c.Params.ByName(name))
	if err != nil {
		if isXHR(c) == false {
			c.Redirect(302, "/404")
		} else {
			c.AbortWithStatus(400)
		}

		return 0, err
	}

	return intParam, nil
}

func GetU64IntParam(name string, c *gin.Context) (uint64, error) {
	intParam, err := strconv.ParseUint(c.Params.ByName(name), 20, 64)
	if err != nil {
		if isXHR(c) == false {
			c.Redirect(302, "/404")
		} else {
			c.AbortWithStatus(400)
		}

		return 0, err
	}

	return intParam, nil
}

// GetIdParam is to Get ObjectID url parameter
// If the parameter is not an ObjectId it will terminate the request
func GetIdParam(name string, c *gin.Context) (string, error) {
	param := c.Params.ByName(name)

	if !bson.IsObjectIdHex(param) {
		return "", errors.New("Invalid ObjectId")
	}
	return param, nil
}

func FindTensor() string {
	cmdPath, _ := exec.LookPath("tensor")

	if len(cmdPath) == 0 {
		cmdPath, _ = filepath.Abs(os.Args[0])
	}

	return cmdPath
}

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
