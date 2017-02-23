package db

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/mgo.v2"
)

// MongoDb store the name an session to mongodb
// database or database cluster
var MongoDb *mgo.Database

// MongoDB collection names
const (
	CAdHocCommands = "ad_hoc_commands"
	CCredentials = "credentials"
	CGroups = "groups"
	CHosts = "hosts"
	CInventories = "inventories"
	CInventoryScripts = "inventory_scripts"
	CInventorySources = "inventory_sources"
	CJobs = "jobs"
	CJobTemplates = "job_templates"
	CTerraformJobTemplates = "terrafrom_job_templates"
	CTerraformJobs = "terraform_jobs"
	CNotifications = "notifications"
	CNotificationTemplates = "notification_templates"
	COrganizations = "organizations"
	CProjects = "projects"
	CTeams = "teams"
	CUsers = "users"
	CActivityStream = "ativity_stream"
)

// Connect will create a session to Mongodb database given in the Config file or env
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
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Println("Unable to ping mongodb session")
		return err
	}

	MongoDb = session.DB(cfg.DbName)

	//Create indexes for each collection
	createIndexes()

	return nil
}

// C returns the mongodb collection for the given name
func C(c string) *mgo.Collection {
	return MongoDb.C(c)
}

// createIndexes optimzes the performance of database writes and read by
// ensuring correct collection indexes are created
func createIndexes() {
	// Collection People
	c := MongoDb.C(CUsers)

	// Unique index username
	if err := c.EnsureIndex(mgo.Index{
		Key:        []string{"username"},
		Unique:     true,
		Background: true,
	}); err != nil {
		logrus.Errorln("Failed to create Unique Index for username of ", CUsers, "Collection")
	}

	// Unique index email
	if err := c.EnsureIndex(mgo.Index{
		Key:        []string{"email"},
		Unique:     true,
		Background: true,
	}); err != nil {
		logrus.Errorln("Failed to create Unique Index for username of ", CUsers, "Collection")
	}

}

// Organizations returns a mgo.Collection for organizations
func Organizations() *mgo.Collection {
	return MongoDb.C(COrganizations)
}

// Credentials returns a mgo.Collection for credentials
func Credentials() *mgo.Collection {
	return MongoDb.C(CCredentials)
}

// Users returns a mgo.Collection for users
func Users() *mgo.Collection {
	return MongoDb.C(CUsers)
}

// Teams returns a mgo.Collection for teams
func Teams() *mgo.Collection {
	return MongoDb.C(CTeams)
}

// Jobs returns a mgo.Collection for jobs
func Jobs() *mgo.Collection {
	return MongoDb.C(CJobs)
}

// JobTemplates returns mgo.Collection for job_templates
func JobTemplates() *mgo.Collection {
	return MongoDb.C(CJobTemplates)
}

// TerrafromJobs returns a mgo.Collection for terraform_jobs
func TerrafromJobs() *mgo.Collection {
	return MongoDb.C(CTerraformJobs)
}

// TerrafromJobTemplates returns mgo.Collection for terraform_job_templates
func TerrafromJobTemplates() *mgo.Collection {
	return MongoDb.C(CTerraformJobTemplates)
}

// Hosts returns mgo.Collection for hosts
func Hosts() *mgo.Collection {
	return MongoDb.C(CHosts)
}

// Inventories returns mgo.Collection for inventories
func Inventories() *mgo.Collection {
	return MongoDb.C(CInventories)
}

// Groups returns mgo.Collection for groups
func Groups() *mgo.Collection {
	return MongoDb.C(CGroups)
}

// Projects returns mgo.Collection for projects
func Projects() *mgo.Collection {
	return MongoDb.C(CProjects)
}

// ActivityStream returns mgo.Collection for activity_stream
func ActivityStream() *mgo.Collection {
	return MongoDb.C(CActivityStream)
}
