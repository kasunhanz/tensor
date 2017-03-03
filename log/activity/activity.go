package activity

import (
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"time"

	"github.com/Sirupsen/logrus"

	"gopkg.in/mgo.v2/bson"
	"reflect"
)

// Activity constants
const (
	Create = "create"
	Update = "update"
	Delete = "delete"
	Associate = "associate"
	Disassociate = "disassociate"
)


// AddOrganizationActivity is responsible of creating new activity stream
// for Organization related activities
func AddActivity(operation string, userID bson.ObjectId, object1, object2 interface{}) {
	stream := common.Activity{
		ID:        bson.NewObjectId(),
		Timestamp: time.Now(),
		Operation: operation,
		ActorID:userID,
	}

	changes := map[string]interface{}{}
	v1 := reflect.ValueOf(object1)
	stream.Object1ID = v1.FieldByName("ID").Interface().(bson.ObjectId)
	stream.Object1 = v1.MethodByName("GetType").Call([]reflect.Value{})[0].String()

	if object2 != nil {
		v2 := reflect.ValueOf(object2)
		stream.Object2ID = v2.FieldByName("ID").Interface().(bson.ObjectId)
		stream.Object2 = v2.MethodByName("GetType").Call([]reflect.Value{})[0].String()
		if operation == Update {
			if v1.Type() == v2.Type() && v1.Kind() == reflect.Struct {
				for i, n := 0, v1.NumField(); i < n; i++ {
					if !reflect.DeepEqual(v1.Field(i).Interface(), v2.Field(i).Interface()) {
						changes[v1.Type().Field(i).Name] = v2.Field(i).Interface()
					}
				}
			}
			stream.Changes = changes
		}
	}


	if err := db.ActivityStream().Insert(stream); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err.Error(),
		}).Errorln("Failed to add new Activity")
	}
}