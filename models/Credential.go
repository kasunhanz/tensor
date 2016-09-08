package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)

const DBC_CREDENTIALS = "credentials"

// Organization is the model for organization
// collection
type Credential struct {
	ID                bson.ObjectId  `bson:"_id" json:"id"`
	Name              string         `bson:"name" json:"name" binding:"required"`
	Description       string         `bson:"description" json:"description"`
	Kind              string         `bson:"kind" json:"kind" binding:"required"`
	Cloud             bool           `bson:"cloud" json:"cloud"`
	Host              string         `bson:"host" json:"host"`
	Username          string         `bson:"username" json:"username"`
	Password          string         `bson:"password" json:"password"`
	SecurityToken     string         `bson:"security_token" json:"security_token"`
	Project           string         `bson:"project" json:"project"`
	Domain            string         `bson:"domain" json:"domain"`
	SshKeyData        string         `bson:"ssh_key_data" json:"ssh_key_data"`
	SshKeyUnlock      string         `bson:"ssh_key_unlock" json:"ssh_key_unlock"`
	BecomeMethod      string         `bson:"become_method" json:"become_method"`
	BecomeUsername    string         `bson:"become_username" json:"become_username"`
	BecomePassword    string         `bson:"become_password" json:"become_password"`
	VaultPassword     string         `bson:"vault_password" json:"vault_password"`
	Subscription      string         `bson:"subscription" json:"subscription"`
	Tenant            string         `bson:"tenant" json:"tenant"`
	Secret            string         `bson:"secret" json:"secret"`
	Client            string         `bson:"client" json:"client"`
	Authorize         bool           `bson:"authorize" json:"authorize"`
	AuthorizePassword string         `bson:"authorize_password" json:"authorize_password"`
	OrganizationID    bson.ObjectId  `bson:"organization_id,omitempty" json:"organization_id"`
	CreatedByID       bson.ObjectId  `bson:"created_by_id" json:"created_by"`
	ModifiedByID      bson.ObjectId  `bson:"modified_by_id" json:"modified_by"`
	Created           time.Time      `bson:"created" json:"created"`
	Modified          time.Time      `bson:"modified" json:"modified"`

	Type              string         `bson:"-" json:"type"`
	Url               string         `bson:"-" json:"url"`
	Related           gin.H          `bson:"-" json:"related"`
	SummaryFields     gin.H          `bson:"-" json:"summary_fields"`
}
