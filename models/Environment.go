package models

import (
	database "github.com/gamunu/hilbertspace/db"
)
// Environment is the model for
// project_environment collection
type Environment struct {
	ID        int     `bson:"_id" json:"id"`
	Name      string  `bson:"name" json:"name" binding:"required"`
	ProjectID int     `bson:"project_id" json:"project_id"`
	Password  string `bson:"password" json:"password"`
	JSON      string  `bson:"json" json:"json" binding:"required"`
}

func (env Environment) Insert() error {
	c := database.MongoDb.C("project_environment")
	return c.Insert(env)
}

func (env Environment) Update() error {
	c := database.MongoDb.C("project_environment")
	return c.UpdateId(env.ID, env)
}

func (env Environment) Remove() error {
	c := database.MongoDb.C("project_environment")
	return c.RemoveId(env.ID)
}