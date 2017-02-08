package common

import (
	"time"

	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
	"github.com/pearsonappeng/tensor/db"
)

const (
	CredentialKindSSH = "ssh"
	CredentialKindNET = "net"
	CredentialKindWIN = "windows"
	CredentialKindSCM = "scm"
	CredentialKindAWS = "aws"
	CredentialKindRAX = "rax"
	CredentialKindVMWARE = "vmware"
	CredentialKindSATELLITE6 = "satellite6"
	CredentialKindCLOUDFORMS = "cloudforms"
	CredentialKindGCE = "gce"
	CredentialKindAZURE = "azure"
	CredentialKindOPENSTACK = "openstack"
)

// Credential is the model for Credential collection
type Credential struct {
	ID                bson.ObjectId `bson:"_id" json:"id"`
	// required fields
	Name              string `bson:"name" json:"name" binding:"required,min=1,max=500"`
	Kind              string `bson:"kind" json:"kind" binding:"required,credential_kind"`

	//optional fields
	Cloud             bool           `bson:"cloud,omitempty" json:"cloud"`
	Description       string         `bson:"description,omitempty" json:"description"`
	Host              string         `bson:"host,omitempty" json:"host"`
	Username          string         `bson:"username,omitempty" json:"username"`
	Password          string         `bson:"password,omitempty" json:"password"`
	SecurityToken     string         `bson:"security_token,omitempty" json:"security_token"`
	Project           string         `bson:"project,omitempty" json:"project"`
	Email             string         `bson:"email,omitempty" json:"email" binding:"omitempty,email"`
	Domain            string         `bson:"domain,omitempty" json:"domain"`
	SSHKeyData        string         `bson:"ssh_key_data,omitempty" json:"ssh_key_data"`
	SSHKeyUnlock      string         `bson:"ssh_key_unlock,omitempty" json:"ssh_key_unlock"`
	BecomeMethod      string         `bson:"become_method,omitempty" json:"become_method" binding:"omitempty,become_method"`
	BecomeUsername    string         `bson:"become_username,omitempty" json:"become_username"`
	BecomePassword    string         `bson:"become_password,omitempty" json:"become_password"`
	VaultPassword     string         `bson:"vault_password,omitempty" json:"vault_password"`
	Subscription      string         `bson:"subscription,omitempty" json:"subscription"`
	Tenant            string         `bson:"tenant,omitempty" json:"tenant"`
	Secret            string         `bson:"secret,omitempty" json:"secret"`
	Client            string         `bson:"client,omitempty" json:"client"`
	Authorize         bool           `bson:"authorize,omitempty" json:"authorize"`
	AuthorizePassword string         `bson:"authorize_password,omitempty" json:"authorize_password"`
	OrganizationID    *bson.ObjectId `bson:"organization_id,omitempty" json:"organization"`

	Created           time.Time `bson:"created" json:"created"`
	Modified          time.Time `bson:"modified" json:"modified"`

	CreatedByID       bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID      bson.ObjectId `bson:"modified_by_id" json:"-"`

	Type              string `bson:"-" json:"type"`
	URL               string `bson:"-" json:"url"`
	Related           gin.H  `bson:"-" json:"related"`
	Summary           gin.H  `bson:"-" json:"summary_fields"`

	Roles             []AccessControl `bson:"roles" json:"-"`
}

func (*Credential) GetType() string {
	return "credential"
}

func (crd *Credential) IsUnique() bool {
	count, err := db.Credentials().Find(bson.M{"name": crd.Name}).Count()
	if err == nil && count > 0 {
		return false
	}

	return true
}

func (crd *Credential) MachineCredentialExist() bool {
	query := bson.M{
		"_id": crd.ID,
		"kind": bson.M{
			"$in": []string{
				CredentialKindSSH,
				CredentialKindWIN,
			},
		},
	}
	count, err := db.Credentials().Find(query).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (crd *Credential) NetworkCredentialExist() bool {
	count, err := db.Credentials().Find(bson.M{"_id": crd.ID, "kind": CredentialKindNET}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (crd *Credential) CloudCredentialExist() bool {
	query := bson.M{
		"_id": crd.ID,
		"kind": bson.M{
			"$in": []string{
				CredentialKindAWS,
				CredentialKindAZURE,
				CredentialKindCLOUDFORMS,
				CredentialKindGCE,
				CredentialKindOPENSTACK,
				CredentialKindSATELLITE6,
				CredentialKindVMWARE,
			},
		},
	}
	count, err := db.Credentials().Find(query).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (crd *Credential) SCMCredentialExist() bool {
	count, err := db.Credentials().Find(bson.M{"_id": crd.ID, "kind": CredentialKindSCM}).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

func (crd *Credential) OrganizationExist() bool {
	count, err := db.Organizations().FindId(crd.OrganizationID).Count()
	if err == nil && count > 0 {
		return true
	}
	return false
}

type PatchCredential struct {
	Name              *string        `json:"name" binding:"omitempty,min=1,max=500"`
	Kind              *string        `json:"kind" binding:"omitempty,credential_kind"`
	Cloud             *bool          `json:"cloud"`
	Description       *string        `json:"description"`
	Host              *string        `json:"host"`
	Username          *string        `json:"username"`
	Password          *string        `json:"password"`
	SecurityToken     *string        `json:"security_token"`
	Project           *string        `json:"project"`
	Domain            *string        `json:"domain"`
	SSHKeyData        *string        `json:"ssh_key_data"`
	SSHKeyUnlock      *string        `json:"ssh_key_unlock"`
	BecomeMethod      *string        `json:"become_method" binding:"omitempty,become_method"`
	BecomeUsername    *string        `json:"become_username"`
	BecomePassword    *string        `json:"become_password"`
	VaultPassword     *string        `json:"vault_password"`
	Subscription      *string        `json:"subscription"`
	Tenant            *string        `json:"tenant"`
	Secret            *string        `json:"secret"`
	Client            *string        `json:"client"`
	Authorize         *bool          `json:"authorize"`
	AuthorizePassword *string        `json:"authorize_password"`
	OrganizationID    *bson.ObjectId `json:"organization"`
}
