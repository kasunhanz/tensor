package credential

import "bitbucket.pearson.com/apseng/tensor/models"

// hideEncrypted is replace encrypted fields by $encrypted$
func hideEncrypted(c *models.Credential) {
	c.Password = "$encrypted$"
	c.SshKeyData = "$encrypted$"
	c.SshKeyUnlock = "$encrypted$"
	c.BecomePassword = "$encrypted$"
	c.VaultPassword = "$encrypted$"
	c.AuthorizePassword = "$encrypted$"
}