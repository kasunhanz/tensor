package helpers

import (
	"gopkg.in/mgo.v2/bson"
	"log"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"net/http"
	"github.com/gin-gonic/gin"
)

func _credentialExist(ID bson.ObjectId) bool {
	count, err := db.Credentials().FindId(ID).Count();
	if err == nil && count == 1 {
		return true
	}
	log.Println("Bad payload:", err)
	return false
}

func MachineCredentialExist(ID bson.ObjectId, c *gin.Context) bool {
	if _credentialExist(ID) {
		return true
	}
	// Return 400 if request has bad JSON format
	c.JSON(http.StatusBadRequest, models.Error{
		Code:http.StatusBadRequest,
		Message: []string{"Machine Credential does not exist"},
	})
	return false
}

func NetworkCredentialExist(ID bson.ObjectId, c *gin.Context) bool {
	if _credentialExist(ID) {
		return true
	}
	// Return 400 if request has bad JSON format
	c.JSON(http.StatusBadRequest, models.Error{
		Code:http.StatusBadRequest,
		Message: []string{"Network Credential does not exist"},
	})
	return false
}

func CloudCredentialExist(ID bson.ObjectId, c *gin.Context) bool {
	if _credentialExist(ID) {
		return true
	}
	// Return 400 if request has bad JSON format
	c.JSON(http.StatusBadRequest, models.Error{
		Code:http.StatusBadRequest,
		Message: []string{"Network Credential does not exist"},
	})
	return false
}

func SCMCredentialExist(ID bson.ObjectId, c *gin.Context) bool {
	if _credentialExist(ID) {
		return true
	}
	// Return 400 if request has bad JSON format
	c.JSON(http.StatusBadRequest, models.Error{
		Code:http.StatusBadRequest,
		Message: []string{"SCM Credential does not exist"},
	})
	return false
}
