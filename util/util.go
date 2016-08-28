package util

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"regexp"
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

// GetObjectIdParam is to Get ObjectID url parameter
// If the parameter is not an ObjectId it will terminate the request
func GetObjectIdParam(name string, c *gin.Context) (string, error) {
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
