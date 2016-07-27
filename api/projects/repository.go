package projects

import (
	"github.com/gamunu/hilbert-space/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func RepositoryMiddleware(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	repositoryID := c.Params.ByName("repository_id")

	repository, err := project.GetRepository(bson.ObjectIdHex(repositoryID));

	if err != nil {
		panic(err)
	}

	c.Set("repository", repository)
	c.Next()
}

func GetRepositories(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	repos, err := project.GetRepositories();

	if err != nil {
		panic(err)
	}

	c.JSON(200, repos)
}

func AddRepository(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	var repository models.Repository

	if err := c.Bind(&repository); err != nil {
		return
	}

	repository.ID = bson.NewObjectId()
	repository.ProjectID = project.ID

	if err := repository.Insert(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   project.ID,
		ObjectType:  "repository",
		ObjectID:    repository.ID,
		Description: "Repository (" + repository.GitUrl + ") created",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func UpdateRepository(c *gin.Context) {
	oldRepo := c.MustGet("repository").(models.Repository)
	var repository models.Repository
	if err := c.Bind(&repository); err != nil {
		return
	}

	oldRepo.GitUrl = repository.GitUrl
	oldRepo.SshKeyID = repository.SshKeyID

	if err := oldRepo.Update(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   oldRepo.ProjectID,
		Description: "Repository (" + repository.GitUrl + ") updated",
		ObjectID:    oldRepo.ID,
		ObjectType:   "inventory",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveRepository(c *gin.Context) {
	repository := c.MustGet("repository").(models.Repository)

	if err := repository.Remove(); err != nil {
		panic(err)
	}

	if err := (models.Event{
		ProjectID:   repository.ProjectID,
		Description: "Repository (" + repository.GitUrl + ") deleted",
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
