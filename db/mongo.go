package db

import (
	"gopkg.in/mgo.v2"
	"pearson.com/hilbert-space/util"
)

var MongoDb *mgo.Database


// Mongodb database
func MdbConnect() error {

	cfg := util.Config.MySQL

	session, err := mgo.Dial("localhost")
	if err != nil {
		return err
	}

	// Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	if err := session.Ping(); err != nil {
		return err
	}

	MongoDb = session.DB(cfg.DbName)
	return nil
}