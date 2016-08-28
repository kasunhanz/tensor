package projects

import (
	"pearson.com/tensor/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

func InventoryMiddleware(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	inventoryID := c.Params.ByName("inventory_id")

	inventory, err := project.GetInventory(bson.ObjectIdHex(inventoryID))
	if err != nil {
		panic(err)
	}

	c.Set("inventory", inventory)
	c.Next()
}

func GetInventory(c *gin.Context) {
	project := c.MustGet("project").(models.Project)

	inv, err := project.GetInventories()
	if err != nil {
		panic(err)
	}

	c.JSON(200, inv)
}

func AddInventory(c *gin.Context) {
	project := c.MustGet("project").(models.Project)
	var inventory models.Inventory

	if err := c.Bind(&inventory); err != nil {
		return
	}

	switch inventory.Type {
	case "static", "aws", "do", "gcloud":
		break
	default:
		c.AbortWithStatus(400)
		return
	}

	inventory.ID = bson.NewObjectId()

	if err := inventory.Insert(); err != nil {
		panic(err)
	}

	objType := "inventory"

	desc := "Inventory " + inventory.Name + " created"
	if err := (models.Event{
		ProjectID:   project.ID,
		ObjectType:  objType,
		ObjectID:    inventory.ID,
		Description: desc,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func UpdateInventory(c *gin.Context) {
	oldInventory := c.MustGet("inventory").(models.Inventory)

	var inventory models.Inventory

	if err := c.Bind(&inventory); err != nil {
		return
	}

	switch inventory.Type {
	case "static", "aws", "do", "gcloud":
		break
	default:
		c.AbortWithStatus(400)
		return
	}

	oldInventory.Name = inventory.Name
	oldInventory.Name = inventory.Type
	oldInventory.KeyID = inventory.KeyID
	oldInventory.SshKeyID = inventory.SshKeyID
	oldInventory.Inventory = inventory.Inventory

	if err := oldInventory.Update(); err != nil {
		panic(err)
	}

	desc := "Inventory " + inventory.Name + " updated"
	objType := "inventory"
	if err := (models.Event{
		ProjectID:   oldInventory.ProjectID,
		Description: desc,
		ObjectID:    oldInventory.ID,
		ObjectType:  objType,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}

func RemoveInventory(c *gin.Context) {
	inventory := c.MustGet("inventory").(models.Inventory)

	if err := inventory.Remove(); err != nil {
		panic(err)
	}

	desc := "Inventory " + inventory.Name + " deleted"
	if err := (models.Event{
		ProjectID:   inventory.ProjectID,
		Description: desc,
	}.Insert()); err != nil {
		panic(err)
	}

	c.AbortWithStatus(204)
}
