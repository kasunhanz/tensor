package common

import "github.com/gin-gonic/gin"

type SCMUpdate struct {
	ExtraVars gin.H `bson:"extra_vars,omitempty" json:"extra_vars,omitempty"`
}
