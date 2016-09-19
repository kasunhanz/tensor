package models

import (
	"time"
	"gopkg.in/mgo.v2/bson"
	"github.com/gin-gonic/gin"
)

const DBC_INVENTORY_SCRIPT = "inventory_scripts"

// CustomInventoryScript is the model for organization
// collection
type InventoryScript struct {
	ID                bson.ObjectId  `bson:"_id" json:"id"`
	Name              string         `bson:"name" json:"name" binding:"required"`
	Description       string         `bson:"description" json:"description"`
	Script            string         `bson:"script" json:"script" binding:"required"`
	CreatedByID       bson.ObjectId  `bson:"created_by_id" json:"created_by"`
	ModifiedByID      bson.ObjectId  `bson:"modified_by_id" json:"modified_by"`
	Created           time.Time      `bson:"created" json:"created"`
	Modified          time.Time      `bson:"modified" json:"modified"`

	Type              string         `bson:"-" json:"type"`
	Url               string         `bson:"-" json:"url"`
	Related           gin.H          `bson:"-" json:"related"`
	SummaryFields     gin.H          `bson:"-" json:"summary_fields"`
}

func (c InventoryScript) CreateIndexes()  {

}