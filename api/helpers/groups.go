package helpers

import (
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"bitbucket.pearson.com/apseng/tensor/models"
	"net/http"
)


func IsUniqueGroup(name string, IID bson.ObjectId) bool {
	count, err := db.Hosts().FindId(bson.M{"name": name, "inventory_id": IID}).Count();
	if err == nil && count == 1 {
		return false
	}

	return true
}

func _groupExist(ID bson.ObjectId) bool {
	count, err := db.Groups().FindId(ID).Count();
	if err == nil && count == 1 {
		return true
	}
	log.Println("Bad payload:", err)
	return false
}

func GroupExist(ID bson.ObjectId, c *gin.Context) bool {
	if _groupExist(ID) {
		return true
	}
	// Return 400 if request has bad JSON format
	c.JSON(http.StatusBadRequest, models.Error{
		Code:http.StatusBadRequest,
		Message: []string{"Group does not exist"},
	})
	return false
}

func ParentGroupExist(ID bson.ObjectId, c *gin.Context) bool {
	if _groupExist(ID) {
		return true
	}
	// Return 400 if request has bad JSON format
	c.JSON(http.StatusBadRequest, models.Error{
		Code:http.StatusBadRequest,
		Message: []string{"Parent Group does not exist"},
	})
	return false
}