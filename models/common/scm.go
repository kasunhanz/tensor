package common

import "gopkg.in/gin-gonic/gin.v1"

type SCMUpdate struct {
	ExtraVars gin.H `bson:"extra_vars,omitempty" json:"extra_vars,omitempty"`
}
