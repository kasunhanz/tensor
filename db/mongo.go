package db

import (
	"time"

	"bitbucket.pearson.com/apseng/tensor/util"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
)

var MongoDb *mgo.Database

const AD_HOC_COMMANDS = "ad_hoc_commands"
const CREDENTIALS = "credentials"
const GROUPS = "groups"
const HOSTS = "hosts"
const INVENTORIES = "inventories"
const INVENTORY_SCRIPT = "inventory_scripts"
const INVENTORY_SOURCES = "inventory_sources"
const JOBS = "jobs"
const JOB_TEMPLATES = "job_templates"
const NOTIFICATIONS = "notifications"
const NOTIFICATION_TEMPLATES = "notification_templates"
const ORGANIZATIONS = "organizations"
const PROJECTS = "projects"
const TEAMS = "teams"
const USERS = "users"
const ACTIVITY_STREAM = "ativity_stream"

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
	// session.SetMode(mgo.Monotonic, true)

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

func CreateIndexes() {
	// Collection People
	c := MongoDb.C(USERS)

	// Unique index username
	if err := c.EnsureIndex(mgo.Index{
		Key:        []string{"username"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Errorln("Failed to create Unique Index for username of ", USERS, "Collection")
	}

	// Unique index email
	if err := c.EnsureIndex(mgo.Index{
		Key:        []string{"email"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Errorln("Failed to create Unique Index for username of ", USERS, "Collection")
	}

}

// collection shortcut methods
func Organizations() *mgo.Collection {
	return MongoDb.C(ORGANIZATIONS)
}

func Credentials() *mgo.Collection {
	return MongoDb.C(CREDENTIALS)
}

func Users() *mgo.Collection {
	return MongoDb.C(USERS)
}

func Teams() *mgo.Collection {
	return MongoDb.C(TEAMS)
}

func Jobs() *mgo.Collection {
	return MongoDb.C(JOBS)
}

func JobTemplates() *mgo.Collection {
	return MongoDb.C(JOB_TEMPLATES)
}

func Hosts() *mgo.Collection {
	return MongoDb.C(HOSTS)
}

func Inventories() *mgo.Collection {
	return MongoDb.C(INVENTORIES)
}

func Groups() *mgo.Collection {
	return MongoDb.C(GROUPS)
}

func Projects() *mgo.Collection {
	return MongoDb.C(PROJECTS)
}

func ActivityStream() *mgo.Collection {
	return MongoDb.C(ACTIVITY_STREAM)
}
