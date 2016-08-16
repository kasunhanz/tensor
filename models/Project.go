package models

import (
	"time"

	database "github.com/gamunu/tensor/db"
	"gopkg.in/mgo.v2/bson"
)

// Project is the model for project
// collection
type Project struct {
	ID      bson.ObjectId `bson:"_id" json:"id"`
	Name    string        `bson:"name" json:"name" binding:"required"`
	Created time.Time     `bson:"created" json:"created"`
	Users   []ProjectUser `bson:"users" json:"users"`
}

type ProjectUser struct {
	UserID bson.ObjectId `bson:"user_id" json:"user_id"`
	Admin  bool          `bson:"admin" json:"admin"`
}

// Create a new project
func (project Project) Insert() error {
	c := database.MongoDb.C("projects")
	return c.Insert(project)
}

// GetEnvironments is to get all environments associated with a project
// envs prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetEnvironments() ([]Environment, error) {
	c := database.MongoDb.C("project_environments")

	var envs []Environment
	err := c.Find(bson.M{"project_id": proj.ID}).All(envs)

	return envs, err
}

// GetEnvironment returns the environment associated with the project
// envId parameter required
// env parameter need to be reference
func (proj Project) GetEnvironment(envId bson.ObjectId) (Environment, error) {
	c := database.MongoDb.C("project_environments")

	var env Environment

	err := c.Find(bson.M{"project_id": proj.ID, "_id": envId}).One(env)

	return env, err
}

// GetInventories is to get all inventories associated with a project
// invs prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetInventories() ([]Inventory, error) {
	c := database.MongoDb.C("project_inventories")

	var invs []Inventory
	err := c.Find(bson.M{"project_id": proj.ID}).All(invs)

	return invs, err
}

// GetInventory returns the inventory associated with the project
// invId parameter required
// inv parameter need to be reference
func (proj Project) GetInventory(invId bson.ObjectId) (Environment, error) {
	c := database.MongoDb.C("project_inventories")

	var inv Environment

	err := c.Find(bson.M{"project_id": proj.ID, "_id": invId}).One(inv)

	return inv, err
}

// GetAccessKeysByType is to get all access keys by type associated with a project
// keys prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetAccessKeysByType(keyType string) ([]AccessKey, error) {
	c := database.MongoDb.C("access_keys")
	m := bson.M{"project_id": proj.ID}
	if len(keyType) > 0 {
		m["type"] = keyType
	}
	var keys []AccessKey
	err := c.Find(m).All(&keys)

	return keys, err
}

// GetAccessKeys is to get all access keys associated with a project
// keys prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetAccessKeys() ([]AccessKey, error) {
	c := database.MongoDb.C("access_keys")

	var keys []AccessKey

	err := c.Find(bson.M{"project_id": proj.ID}).All(&keys)

	return keys, err
}

// GetAccessKey returns the inventory associated with the project
// keyId parameter required
// key parameter need to be reference
func (proj Project) GetAccessKey(keyId bson.ObjectId) (AccessKey, error) {
	c := database.MongoDb.C("access_keys")

	var key AccessKey
	err := c.Find(bson.M{"project_id": proj.ID, "_id": keyId}).One(&key)

	return key, err
}

// GetRepositories is to get all repositories associated with a project
// repos prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetRepositories() ([]Repository, error) {
	c := database.MongoDb.C("project_repositories")

	var repos []Repository
	err := c.Find(bson.M{"project_id": proj.ID}).All(&repos)

	return repos, err
}

// GetInventory returns the inventory associated with the project
// invId parameter required
// inv parameter need to be reference
func (proj Project) GetRepository(repoId bson.ObjectId) (Repository, error) {
	c := database.MongoDb.C("project_repositories")

	var repo Repository
	err := c.Find(bson.M{"project_id": proj.ID, "_id": repoId}).One(&repo)

	return repo, err
}

// GetTemplates is to get all repositories associated with a project
// repos prameter need to be a reference
// returns the error returned by mongo driver
func (proj Project) GetTemplates() ([]Template, error) {
	c := database.MongoDb.C("project_templates")

	var repos []Template
	err := c.Find(bson.M{"project_id": proj.ID}).All(&repos)

	return repos, err
}

// GetTemplate returns the inventory associated with the project
// invId parameter required
// inv parameter need to be reference
func (proj Project) GetTemplate(tempId bson.ObjectId) (Template, error) {
	c := database.MongoDb.C("project_templates")

	var tpl Template
	err := c.Find(bson.M{"project_id": proj.ID, "_id": tempId}).One(&tpl)

	return tpl, err
}
