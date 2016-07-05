package projects

import (
	"pearson.com/hilbert-space/models"
	"pearson.com/hilbert-space/util"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func TemplatesMiddleware(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	templateID, err := util.GetIntParam("template_id", c)
	if err != nil {
		return
	}
	template, err := project.GetTemplate(templateID);
	if err != nil {
		panic(err)
	}

	c.Set("template", template)
	c.Next()
}

func GetTemplates(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	templates, err := project.GetTemplates();

	if err != nil {
		panic(err)
	}

	c.JSON(200, templates)
}

func AddTemplate(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	var template models.Template
	if err := c.Bind(&template); err != nil {
		return
	}

	template.ID = bson.NewObjectId()
	template.ProjectID = project.ID

	if err := template.Insert(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   project.ID,
		ObjectType:  "template",
		ObjectID:    template.ID,
		Description: "Template ID " + template.ID + " created",
	}.Insert()); err != nil {
		panic(err)
	}

	c.JSON(201, template)
}

func UpdateTemplate(c *gin.Context) {
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
		Description: "Template ID " + template.ID + " updated",
		ObjectID:    oldTemplate.ID,
		ObjectType:  "template",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveTemplate(c *gin.Context) {
	tpl := c.MustGet("template").(models.Template)

	if err := tpl.Remove(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   tpl.ProjectID,
		Description: "Template ID " + tpl.ID + " deleted",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
