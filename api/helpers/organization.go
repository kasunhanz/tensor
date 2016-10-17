package helpers

import (
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/db"
	"log"
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func OrganizationExist(ID bson.ObjectId, c *gin.Context) bool {
	count, err := db.Organizations().FindId(ID).Count();
	if err == nil && count == 1 {
		return true
	}
	log.Println("Bad payload:", err)
	// Return 400 if request has bad JSON format
	c.JSON(http.StatusBadRequest, models.Error{
		Code:http.StatusBadRequest,
		Message: []string{"Organization does not exist"},
	})
	return false
}