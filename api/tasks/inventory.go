package tasks

import (
	"io/ioutil"
	"github.com/gamunu/hilbert-space/util"
	"gopkg.in/mgo.v2/bson"
	"encoding/json"
)

func (t *task) installInventory() error {
	if bson.IsObjectIdHex(t.inventory.SshKeyID.String()) {
		// write inventory key
		err := t.installKey(t.inventory.SshKey)
		if err != nil {
			return err
		}
	}

	switch t.inventory.Type {
	case "static":
		return t.installStaticInventory()
	}

	return nil
}

func (t *task) installStaticInventory() error {
	t.log("installing static inventory")

	bytes, err := json.Marshal(t.inventory.Inventory)
	if err != nil {
		return err
	}
	// create inventory file
	return ioutil.WriteFile(util.Config.TmpPath + "/inventory_" + t.task.ID.String(), bytes, 0664)
}
