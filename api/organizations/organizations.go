package organizations

import (
	database "pearson.com/tensor/db"
	"pearson.com/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"time"
	mdlorg "pearson.com/tensor/models/organization"
	mdlusr "pearson.com/tensor/models/user"
)

// GetOrganizations returns a JSON array of projects
func GetOrganizations(c *gin.Context) {

	col := database.MongoDb.C("organizations")

	var orgs []mdlorg.Organization

	if err := col.Find(nil).Sort("name").All(&orgs); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	olen := len(orgs)

	resp := make(map[string]interface{})
	resp["count"] = olen
	resp["results"] = orgs

	for i := 0; i < olen; i++ {
		(&orgs[i]).IncludeMetadata()
	}

	c.JSON(200, resp)
}

// AddOrganization creates a new project
func AddOrganization(c *gin.Context) {
	var org mdlorg.Organization
	user := c.MustGet("user").(mdlusr.User)

	if err := c.Bind(&org); err != nil {
		// Return 400 if request has bad JSON format
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	org.ID = bson.NewObjectId()
	org.Created = time.Now()
	org.Modified = time.Now()
	org.CreatedBy = user.ID
	org.ModifiedBy = user.ID

	if err := org.Insert(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if err := (models.Event{
		ID:          bson.NewObjectId(),
		ProjectID:   org.ID,
		ObjectType:  "organization",
		Description: "Organization Created",
		Created:     org.Created,
	}.Insert()); err != nil {
		// We don't inform client about this error
		// do not ever panic :D
		c.Error(err)
		return
	}

	c.JSON(201, org)
}

func UpdateOrganization(c *gin.Context) {
	oldTemplate := c.MustGet("template").(models.Template)

	var template models.Template
	if err := c.Bind(&template); err != nil {
		return
	}

	template.ID = oldTemplate.ID

	if err := template.Update(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   oldTemplate.ProjectID,
		Description: "Template ID " + template.ID.String() + " updated",
		ObjectID:    oldTemplate.ID,
		ObjectType:  "template",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
