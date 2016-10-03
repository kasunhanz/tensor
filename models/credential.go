package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)

const (
	CREDENTIAL_KIND_SSH = "ssh"
	CREDENTIAL_KIND_NET = "net"
	CREDENTIAL_KIND_SCM = "scm"
	CREDENTIAL_KIND_AWS = "aws"
	CREDENTIAL_KIND_RAX = "rax"
	CREDENTIAL_KIND_VMWARE = "vmware"
	CREDENTIAL_KIND_SATELLITE6 = "satellite6"
	CREDENTIAL_KIND_CLOUDFORMS = "cloudforms"
	CREDENTIAL_KIND_GCE = "gce"
	CREDENTIAL_KIND_AZURE = "azure"
	CREDENTIAL_KIND_OPENSTACK = "openstack"
)

// Organization is the model for organization
// collection
type Credential struct {
	ID                bson.ObjectId   `bson:"_id" json:"id"`
	// required feilds
	Name              string          `bson:"name" json:"name" binding:"required"`
	Kind              string          `bson:"kind" json:"kind" binding:"required"`

	//optional feilds
	Cloud             bool            `bson:"cloud,omitempty" json:"cloud"`
	Description       string         `bson:"description,omitempty" json:"description"`
	Host              string         `bson:"host,omitempty" json:"host"`
	Username          string         `bson:"username,omitempty" json:"username"`
	Password          string         `bson:"password,omitempty" json:"password"`
	SecurityToken     string         `bson:"security_token,omitempty" json:"security_token"`
	Project           string         `bson:"project,omitempty" json:"project"`
	Domain            string         `bson:"domain,omitempty" json:"domain"`
	SshKeyData        string         `bson:"ssh_key_data,omitempty" json:"ssh_key_data"`
	SshKeyUnlock      string         `bson:"ssh_key_unlock,omitempty" json:"ssh_key_unlock"`
	BecomeMethod      string         `bson:"become_method,omitempty" json:"become_method"`
	BecomeUsername    string         `bson:"become_username,omitempty" json:"become_username"`
	BecomePassword    string         `bson:"become_password,omitempty" json:"become_password"`
	VaultPassword     string         `bson:"vault_password,omitempty" json:"vault_password"`
	Subscription      string         `bson:"subscription,omitempty" json:"subscription"`
	Tenant            string         `bson:"tenant,omitempty" json:"tenant"`
	Secret            string         `bson:"secret,omitempty" json:"secret"`
	Client            string         `bson:"client,omitempty" json:"client"`
	Authorize         bool            `bson:"authorize,omitempty" json:"authorize"`
	AuthorizePassword string         `bson:"authorize_password,omitempty" json:"authorize_password"`
	OrganizationID    *bson.ObjectId  `bson:"organization_id,omitempty" json:"organization"`

	Created           time.Time       `bson:"created" json:"created"`
	Modified          time.Time       `bson:"modified" json:"modified"`

	CreatedByID       bson.ObjectId   `bson:"created_by_id" json:"-"`
	ModifiedByID      bson.ObjectId   `bson:"modified_by_id" json:"-"`

	Type              string          `bson:"-" json:"type"`
	Url               string          `bson:"-" json:"url"`
	Related           gin.H           `bson:"-" json:"related"`
	Summary           gin.H           `bson:"-" json:"summary_fields"`

	Roles             []AccessControl `bson:"roles" json:"-"`
}