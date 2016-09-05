package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	database "bitbucket.pearson.com/apseng/tensor/db"
)


// Project is the model for project
// collection
type Project struct {
	ID                    bson.ObjectId     `bson:"_id" json:"id"`
	Type                  string            `bson:"-" json:"type"`
	Url                   string            `bson:"-" json:"url"`
	Related               map[string]string `bson:"-" json:"related"`
	SummaryFields         map[string]interface{} `bson:"-" json:"summary_fields"`
	Name                  string            `bson:"name" json:"name" binding:"required"`
	Description           string            `bson:"description" json:"description"`
	LocalPath             string            `bson:"local_path" json:"local_path"`
	ScmType               string            `bson:"scm_type" json:"scm_type" binding:"required"`
	ScmUrl                string            `bson:"scm_url" json:"scm_url" binding:"required"`
	ScmBranch             string            `bson:"scm_branch" json:"scm_branch"`
	ScmClean              bool              `bson:"scm_clean" json:"scm_clean"`
	ScmDeleteOnUpdate     bool              `bson:"scm_delete_on_update" json:"scm_delete_on_update"`
	ScmCredential         bson.ObjectId     `bson:"credentail" json:"credential"`
	LastJob               bson.ObjectId     `bson:"last_job" json:"last_job"`
	LastJobRun            time.Time         `bson:"last_job_run" json:"last_job_run"`
	LastJobFailed         bool              `bson:"last_job_failed" json:"last_job_failed"`
	HasSchedules          bool              `bson:"has_schedules" json:"has_schedules"`
	NextJobRun            time.Time         `bson:"next_job_run" json:"next_job_run"`
	Status                string            `bson:"status" json:"status"`
	Organization          bson.ObjectId     `bson:"organization" json:"organization" binding:"required"`
	ScmDeleteOnNextUpdate bool              `bson:"scm_delete_on_next_update" json:"scm_delete_on_next_update"`
	ScmUpdateOnLaunch     bool              `bson:"scm_update_on_launch" json:"scm_update_on_launch"`
	ScmUpdateCacheTimeout int               `bson:"scm_update_cache_timeout" json:"scm_update_cache_timeout"`
	LastUpdateFailed      bool              `bson:"last_update_failed" json:"last_update_failed"`
	LastUpdated           time.Time         `bson:"last_updated" json:"last_updated"`
	CreatedBy             bson.ObjectId     `bson:"created_by" json:"created_by"`
	Created               time.Time         `bson:"created" json:"created"`
	Modified              time.Time         `bson:"modified" json:"modified"`
}

type ProjectUser struct {
	UserID bson.ObjectId `bson:"user_id" json:"user_id"`
	Admin  bool          `bson:"admin" json:"admin"`
}

// Create a new organization
func (p *Project) IncludeMetadata() {

	p.Type = "project"
	p.Url = "/v1/projects/" + p.ID.Hex() + "/"
	p.Related = map[string]string{
		"created_by": "/api/v1/users/" + p.CreatedBy.Hex() + "/",
		"last_job": "/api/v1/project_updates/" + p.LastJob.Hex() + "/",
		"notification_templates_error": "/api/v1/projects/" + p.ID.Hex() + "/notification_templates_error/",
		"notification_templates_success": "/api/v1/projects/" + p.ID.Hex() + "/notification_templates_success/",
		"object_roles": "/api/v1/projects/" + p.ID.Hex() + "/object_roles/",
		"notification_templates_any": "/api/v1/projects/" + p.ID.Hex() + "/notification_templates_any/",
		"project_updates": "/api/v1/projects/" + p.ID.Hex() + "/project_updates/",
		"update": "/api/v1/projects/" + p.ID.Hex() + "/update/",
		"access_list": "/api/v1/projects/" + p.ID.Hex() + "/access_list/",
		"playbooks": "/api/v1/projects/" + p.ID.Hex() + "/playbooks/",
		"schedules": "/api/v1/projects/" + p.ID.Hex() + "/schedules/",
		"teams": "/api/v1/projects/" + p.ID.Hex() + "/teams/",
		"activity_stream": "/api/v1/projects/" + p.ID.Hex() + "/activity_stream/",
		"organization": "/api/v1/organizations/" + p.Organization.Hex() + "/",
		"last_update": "/api/v1/project_updates/" + p.LastJob.Hex() + "/",
	}
	p.setSummaryFields()
}

func (p *Project) setSummaryFields() {
	s := map[string]interface{}{
		"object_roles": []map[string]string{
			{
				"Description": "Can manage all aspects of the project",
				"Name":"Admin",
			},
			{
				"Description":"Can use the project in a job template",
				"Name":"Use",
			},
			{
				"Description":"May update project or inventory or group using the configured source update system",
				"Name":"Update",
			},
			{
				"Description":"May view settings for the project",
				"Name":"Read",
			},
		},
		"related_field_counts": map[string]int{
			"job_templates":1,
			"users":2,
			"teams":2,
			"admins":2,
			"inventories":1,
			"projects":1,
		},
	}

	var u User

	c := database.MongoDb.C("users")

	if err := c.FindId(p.CreatedBy).One(&u); err != nil {
		panic(err)
	}

	s["last_job"] = map[string]interface{}{
		"id": 3,
		"name": "Demo Project",
		"description": "",
		"finished": "2016-08-16T19:27:43.416Z",
		"status": "successful",
		"failed": false,
	}

	s["last_update"] = map[string]interface{}{
		"id": 3,
		"name": "Demo Project",
		"description": "",
		"status": "successful",
		"failed": false,
	}

	s["organization"] = map[string]interface{}{
		"id": 1,
		"name": "Default",
		"description": "",
	}

	s["created_by"] = map[string]interface{}{
		"id":u.ID,
		"username":u.Username,
		"first_name":u.FirstName,
		"last_name":u.LastName,
	}

	p.SummaryFields = s
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
