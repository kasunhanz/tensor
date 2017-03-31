package ansible

import (
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type AdHocCommand struct {
	ID            bson.ObjectId `bson:"_id" json:"id"`
	ModuleName    string        `bson:"module_name" json:"module_name" binding:"required"`
	Limit         string        `bson:"limit" json:"limit"`
	ModuleArgs    string        `bson:"module_args" json:"module_args"`
	JobType       string        `bson:"job_type" json:"job_type"`
	Forks         uint8         `bson:"forks" json:"forks"`
	Verbosity     uint8         `bson:"verbosity" json:"verbosity"`
	BecomeEnabled bool          `bson:"become_enabled" json:"become_enabled"`
	CredentialID  bson.ObjectId `bson:"credential_id" json:"credential"`
	InventoryID   bson.ObjectId `bson:"inventory_id" json:"inventory"`
	ExtraVars     gin.H         `bson:"extra_vars" json:"extra_vars"`
	CreatedByID   bson.ObjectId `bson:"created_by_id" json:"created_by"`
	ModifiedByID  bson.ObjectId `bson:"modified_by_id" json:"modified_by"`
	Created       time.Time     `bson:"created" json:"created"`
	Modified      time.Time     `bson:"modified" json:"modified"`

	Type  string `bson:"-" json:"type"`
	Links gin.H  `bson:"-" json:"links"`
	Meta  gin.H  `bson:"-" json:"meta"`
}

type AdHocCommandEvent struct {
	ID             bson.ObjectId `bson:"_id" json:"id"`
	HostName       string        `bson:"host_name" json:"host_name" binding:"required"`
	Event          string        `bson:"event" json:"event"`
	EventData      string        `bson:"event_data" json:"event_data"`
	Failed         bool          `bson:"failed" json:"failed"`
	Changed        bool          `bson:"changed" json:"changed"`
	Counter        uint8         `bson:"counter" json:"counter"`
	HostID         bson.ObjectId `bson:"host_id" json:"host"`
	AdHocCommandID bson.ObjectId `bson:"ad_hoc_command_id" json:"ad_hoc_command_id"`

	Type  string `bson:"-" json:"type"`
	Links gin.H  `bson:"-" json:"links"`
	Meta  gin.H  `bson:"-" json:"meta"`
}
