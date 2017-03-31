package metadata

import (
	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/db"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// Create a new organization
func OrganizationMetadata(o *common.Organization) {

	ID := o.ID.Hex()
	o.Type = "organization"
	o.Links = gin.H{
		"self":                           "/v1/organizations/" + ID,
		"created_by":                     "/v1/users/" + o.CreatedByID.Hex(),
		"modified_by":                    "/v1/users/" + o.ModifiedByID.Hex(),
		"notification_templates_error":   "/v1/organizations/" + ID + "/notification_templates_error",
		"notification_templates_success": "/v1/organizations/" + ID + "/notification_templates_success",
		"users":                      "/v1/organizations/" + ID + "/users",
		"object_roles":               "/v1/organizations/" + ID + "/object_roles",
		"notification_templates_any": "/v1/organizations/" + ID + "/notification_templates_any",
		"teams":                  "/v1/organizations/" + ID + "/teams",
		"access_list":            "/v1/organizations/" + ID + "/access_list",
		"notification_templates": "/v1/organizations/" + ID + "/notification_templates",
		"admins":                 "/v1/organizations/" + ID + "/admins",
		"credentials":            "/v1/organizations/" + ID + "/credentials",
		"inventories":            "/v1/organizations/" + ID + "/inventories",
		"activity_stream":        "/v1/organizations/" + ID + "/activity_stream",
		"projects":               "/v1/organizations/" + ID + "/projects",
	}

	organizationSummary(o)
}

func organizationSummary(o *common.Organization) {

	var modified common.User
	var created common.User

	jcount, err := db.JobTemplates().Find(bson.M{"organization_id": o.ID}).Count()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Organization":    o.Name,
			"Organization ID": o.ID.Hex(),
		}).Errorln("Error while getting Job Template count")
	}

	ucount, err := db.Users().Find(bson.M{"organization_id": o.ID}).Count()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Organization":    o.Name,
			"Organization ID": o.ID.Hex(),
		}).Errorln("Error while getting Users count")
	}

	tcount, err := db.Teams().Find(bson.M{"organization_id": o.ID}).Count()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Organization":    o.Name,
			"Organization ID": o.ID.Hex(),
		}).Errorln("Error while getting Teams count")
	}

	icount, err := db.Inventories().Find(bson.M{"organization_id": o.ID}).Count()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Organization":    o.Name,
			"Organization ID": o.ID.Hex(),
		}).Errorln("Error while getting Inventories count")
	}

	pcount, err := db.Projects().Find(bson.M{"organization_id": o.ID}).Count()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Organization":    o.Name,
			"Organization ID": o.ID.Hex(),
		}).Errorln("Error while getting Projects count")
	}

	summary := gin.H{
		"object_roles": []gin.H{
			{
				"description": "Can view all settings for the organization",
				"name":        "auditor",
			},
			{
				"description": "Can manage all aspects of the organization",
				"name":        "admin",
			},
			{
				"description": "User is a member of the organization",
				"name":        "member",
			},
			{
				"description": "May view settings for the organization",
				"name":        "read",
			},
		},
		"related_field_counts": gin.H{
			"job_templates": jcount,
			"users":         ucount,
			"teams":         tcount,
			"admins":        0, //TODO: get admins count
			"inventories":   icount,
			"projects":      pcount,
		},
		"created_by":  nil,
		"modified_by": nil,
		"owners":      nil,
	}

	if err := db.Users().FindId(o.CreatedByID).One(&created); err != nil {
		logrus.WithFields(logrus.Fields{
			"User ID":         o.CreatedByID.Hex(),
			"Organization":    o.Name,
			"Organization ID": o.ID.Hex(),
		}).Errorln("Error while getting created by User")
	} else {
		summary["created_by"] = gin.H{
			"id":         created.ID.Hex(),
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		}
	}

	if err := db.Users().FindId(o.ModifiedByID).One(&modified); err != nil {
		logrus.WithFields(logrus.Fields{
			"User ID":         o.ModifiedByID.Hex(),
			"Organization":    o.Name,
			"Organization ID": o.ID.Hex(),
		}).Errorln("Error while getting modified by User")
	} else {
		summary["modified_by"] = gin.H{
			"id":         modified.ID.Hex(),
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		}
	}

	//TODO: include teams to owners list
	o.Meta = summary
}
