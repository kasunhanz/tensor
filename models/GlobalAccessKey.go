package models

import (
	"strconv"

	"github.com/gamunu/hilbertspace/util"
)

type GlobalAccessKey struct {
	ID     int    `db:"id" json:"id"`
	Name   string `db:"name" json:"name" binding:"required"`
	// 'aws/do/gcloud/ssh/credential',
	Type   string `db:"type" json:"type" binding:"required"`

	// username
	Key    *string `db:"key" json:"key"`
	// password
	Secret *string `db:"secret" json:"secret"`
}

type GlobalAccessKeyResponse struct {
	ID     int    `db:"id" json:"id"`
	Name   string `db:"name" json:"name"`
	// 'aws/do/gcloud/ssh/credential',
	Type   string `db:"type" json:"type"`

	// username
	Key    *string `db:"key" json:"key"`
}


func (key GlobalAccessKey) GetPath() string {
	return util.Config.TmpPath + "/global_access_key_" + strconv.Itoa(key.ID)
}
