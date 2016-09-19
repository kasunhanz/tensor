package db

import (
	"gopkg.in/mgo.v2"
	"time"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/models"
	"fmt"
)

var MongoDb *mgo.Database

// Mongodb database
func Connect() error {

	cfg := util.Config.MongoDB

	// We need this object to establish a session to our MongoDB.
	info := &mgo.DialInfo{
		Addrs:    cfg.Hosts,
		Timeout:  60 * time.Second,
		Database: cfg.DbName,
		Username: cfg.Username,
		Password: cfg.Password,
	}

	if len(cfg.ReplicaSet) > 0 {
		info.ReplicaSetName = cfg.ReplicaSet
		info.Mechanism = "SCRAM-SHA-1"
	}
	// Create a session which maintains a pool of socket connections
	// to our MongoDB.
	session, err := mgo.DialWithInfo(info)
	if err != nil {
		return err
	}

	// Switch the session to a monotonic behavior.
	//session.SetMode(mgo.Monotonic, true)

	if err := session.Ping(); err != nil {
		return err
	}

	MongoDb = session.DB(cfg.DbName)

	//Create indexes for each collection
	CreateIndexes()

	return nil
}

func C(c string) *mgo.Collection {
	return MongoDb.C(c)
}

//
// PowerShell Script to generate the list
// get-childitem | select-object -Property Name | % { Write-Output "models.$(($_.Name).TrimEnd(".go")){}.CreateIndexes()" }
func CreateIndexes()  {
	//TODO: call models CreateIndex methods
	models.ACL{}.CreateIndexes()
	models.AdHocCommand{}.CreateIndexes()
	models.Credential{}.CreateIndexes()
	models.InventoryScript{}.CreateIndexes()
	models.Event{}.CreateIndexes()
	models.Group{}.CreateIndexes()
	models.Host{}.CreateIndexes()
	models.Inventory{}.CreateIndexes()
	models.InventorySource{}.CreateIndexes()
	models.Job{}.CreateIndexes()
	models.JobTemplate{}.CreateIndexes()
	models.Notification{}.CreateIndexes()
	models.NotificationTemplate{}.CreateIndexes()
	models.Organization{}.CreateIndexes()
	models.Project{}.CreateIndexes()
	models.Session{}.CreateIndexes()
	models.Team{}.CreateIndexes()
	models.User{}.CreateIndexes()
}