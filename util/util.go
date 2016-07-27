package util

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"os/exec"
	"path/filepath"
	"os"
	"gopkg.in/mgo.v2/bson"
	"errors"
)

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

// GetObjectIdParam is to Get ObjectID url parameter
// If the parameter is not an ObjectId it will terminate the request
func GetObjectIdParam(name string, c *gin.Context) (string, error) {
	param := c.Params.ByName(name)

	if !bson.IsObjectIdHex(param) {
		return "", errors.New("Invalid ObjectId")
	}
	return param, nil;
}

func FindHilbertspace() string {
	cmdPath, _ := exec.LookPath("hilbertspace")

	if len(cmdPath) == 0 {
		cmdPath, _ = filepath.Abs(os.Args[0])
	}

	return cmdPath
}