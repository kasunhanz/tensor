package models

import (
	database "github.com/gamunu/hilbertspace/db"
	"gopkg.in/mgo.v2/bson"
)

// Repository is the model for project_repository
// collection
type Repository struct {
	ID        bson.ObjectId    `bson:"_id" json:"id"`
	Name      string `bson:"name" json:"name" binding:"required"`
	ProjectID bson.ObjectId    `bson:"project_id" json:"project_id"`
	GitUrl    string `bson:"git_url" json:"git_url" binding:"required"`
	SshKeyID  bson.ObjectId    `bson:"ssh_key_id" json:"ssh_key_id" binding:"required"`

	SshKey    AccessKey `bson:"-" json:"-"`
}

func (repo Repository) Remove() error {
	c := database.MongoDb.C("project_repository")
	return c.RemoveId(repo.ID)
}

func (repo Repository) Update() error {
	c := database.MongoDb.C("project_repository")
	return c.UpdateId(repo.ID, repo)
}

func (repo Repository) Insert() error {
	c := database.MongoDb.C("project_repository")
	return c.Insert(repo)
}
