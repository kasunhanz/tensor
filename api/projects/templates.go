package projects

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
	"bitbucket.pearson.com/apseng/tensor/models"
)

func TemplatesMiddleware(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	templateID := c.Params.ByName("template_id")

	template, err := project.GetTemplate(bson.ObjectIdHex(templateID))

	if err != nil {
		panic(err)
	}

	c.Set("template", template)
	c.Next()
}

func GetTemplates(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	templates, err := project.GetTemplates()

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
		Description: "Template ID " + template.ID.String() + " created",
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
		Description: "Template ID " + template.ID.String() + " updated",
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
		Description: "Template ID " + tpl.ID.String() + " deleted",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
