package api

import (
	"errors"
	"github.com/pearsonappeng/tensor/models/common"
	"gopkg.in/gin-gonic/gin.v1"
)

func AbortWithError(c *gin.Context, status int, message string) {
	c.Error(&gin.Error{
		Type: gin.ErrorTypePrivate,
		Err:  errors.New(message),
	})
	c.JSON(status, common.Error{
		Code:    status,
		Message: message,
	})
	c.Abort()
}

func AbortWithCode(c *gin.Context, status int, code int, message string) {
	c.JSON(status, common.Error{
		Code:    code,
		Message: message,
	})
	c.Abort()
}

func AbortWithErrors(c *gin.Context, status int, message string, emsgs ...string) {
	c.Error(&gin.Error{
		Type: gin.ErrorTypePrivate,
		Err:  errors.New(message),
	})
	c.JSON(status, common.Error{
		Code:    status,
		Message: message,
		Errors:  emsgs,
	})
	c.Abort()
}

// hideEncrypted is replaces encrypted fields by $encrypted$ string
func hideEncrypted(c *common.Credential) {
	encrypted := "$encrypted$"
	c.Password = encrypted
	c.SSHKeyData = encrypted
	c.SSHKeyUnlock = encrypted
	c.BecomePassword = encrypted
	c.VaultPassword = encrypted
	c.AuthorizePassword = encrypted
	c.Secret = encrypted
}
