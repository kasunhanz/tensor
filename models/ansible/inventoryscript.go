package ansible

import (
	"time"

	"gopkg.in/gin-gonic/gin.v1"
	"gopkg.in/mgo.v2/bson"
)

// InventoryScript is the model for organization collection
type InventoryScript struct {
	ID            bson.ObjectId `bson:"_id" json:"id"`
	Name          string        `bson:"name" json:"name" binding:"required"`
	Description   string        `bson:"description" json:"description"`
	Script        string        `bson:"script" json:"script" binding:"required"`

	CreatedByID   bson.ObjectId `bson:"created_by_id" json:"-"`
	ModifiedByID  bson.ObjectId `bson:"modified_by_id" json:"-"`

	Created       time.Time `bson:"created" json:"created"`
	Modified      time.Time `bson:"modified" json:"modified"`

	Type          string `bson:"-" json:"type"`
	URL           string `bson:"-" json:"url"`
	Related       gin.H  `bson:"-" json:"related"`
	SummaryFields gin.H  `bson:"-" json:"summary_fields"`
}

func (*InventoryScript) GetType() string {
	return "inventory_script"
}