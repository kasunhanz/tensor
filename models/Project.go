package models

import (
	"time"

	database "pearson.com/hilbert-space/db"
	"gopkg.in/mgo.v2/bson"
	"github.com/ansible-semaphore/semaphore/models"
)

// Project is the model for project
// collection
type Project struct {
	ID      bson.ObjectId       `bson:"_id" json:"id"`
	Name    string    `bson:"name" json:"name" binding:"required"`
	Created time.Time `bson:"created" json:"created"`
	Users   []ProjectUser `bson:"users" json:"users"`
}

type ProjectUser  struct {
	UserID bson.ObjectId `bson:"user_id" json:"user_id"`
	Admin  bool `bson:"admin" json:"admin"`
}

// Create a new project
func (project *Project) Insert() error {
	c := database.MongoDb.C("project")
	return c.Insert(project)
}

// GetEnvironments is to get all environments associated with a project
// envs prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetEnvironments(envs *[]models.Environment) error {
	c := database.MongoDb.C("project_environment")
	return c.Find(bson.M{"project_id": proj.ID, }).All(envs)
}

// GetEnvironment returns the environment associated with the project
// envId parameter required
// env parameter need to be reference
func (proj Project) GetEnvironment(envId bson.ObjectId, env *models.Environment) error {
	c := database.MongoDb.C("project_environment")
	return c.Find(bson.M{"project_id": proj.ID, "_id": envId}).One(env)
}

// GetInventories is to get all inventories associated with a project
// invs prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetInventories(invs *[]models.Inventory) error {
	c := database.MongoDb.C("project_inventory")
	return c.Find(bson.M{"project_id": proj.ID, }).All(invs)
}

// GetInventory returns the inventory associated with the project
// invId parameter required
// inv parameter need to be reference
func (proj Project) GetInventory(invId bson.ObjectId, inv *models.Environment) error {
	c := database.MongoDb.C("project_inventory")
	return c.Find(bson.M{"project_id": proj.ID, "_id": invId}).One(inv)
}

// GetAccessKeysByType is to get all access keys by type associated with a project
// keys prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetAccessKeysByType(keys *[]models.AccessKey, keyType string) error {
	c := database.MongoDb.C("access_key")
	m := bson.M{"project_id": proj.ID, }
	if len(keyType) > 0 {
		m["type"] = keyType
	}

	return c.Find(m).All(keys)
}

// GetAccessKeys is to get all access keys associated with a project
// keys prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetAccessKeys(keys *[]models.AccessKey) error {
	c := database.MongoDb.C("access_key")
	return c.Find(bson.M{"project_id": proj.ID, }).All(keys)
}

// GetAccessKey returns the inventory associated with the project
// keyId parameter required
// key parameter need to be reference
func (proj Project) GetAccessKey(keyId bson.ObjectId, key *models.AccessKey) error {
	c := database.MongoDb.C("access_key")
	return c.Find(bson.M{"project_id": proj.ID, "_id": keyId}).One(key)
}

// GetRepositories is to get all repositories associated with a project
// repos prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetRepositories() ([]models.AccessKey, error) {
	c := database.MongoDb.C("project_repository")

	var repos []models.Repository
	err := c.Find(bson.M{"project_id": proj.ID, }).All(repos)

	return repos, err
}

// GetInventory returns the inventory associated with the project
// invId parameter required
// inv parameter need to be reference
func (proj Project) GetRepository(repoId bson.ObjectId) (models.Repository, error) {
	c := database.MongoDb.C("project_repository")

	var repo models.Repository
	err := c.Find(bson.M{"project_id": proj.ID, "_id": repoId}).One(repo)

	return repo, err
}

// GetTemplates is to get all repositories associated with a project
// repos prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetTemplates() ([]models.Template, error) {
	c := database.MongoDb.C("project_template")

	var repos []models.Template
	err := c.Find(bson.M{"project_id": proj.ID, }).All(repos)

	return repos, err
}

// GetTemplate returns the inventory associated with the project
// invId parameter required
// inv parameter need to be reference
func (proj Project) GetTemplate(tempId bson.ObjectId) (models.Template, error) {
	c := database.MongoDb.C("project_template")

	var tpl models.Template
	err := c.Find(bson.M{"project_id": proj.ID, "_id": tempId}).One(tpl)

	return tpl, err
}