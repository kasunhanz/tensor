package metadata

import (
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/db"
	"gopkg.in/mgo.v2/bson"
	log "github.com/Sirupsen/logrus"
)

// Create a new organization
func OrganizationMetadata(o *models.Organization) error {

	ID := o.ID.Hex()
	o.Type = "organization"
	o.Url = "/v1/organizations/" + ID + "/"
	o.Related = gin.H{
		"created_by": "/v1/users/" + o.CreatedBy.Hex() + "/",
		"modified_by": "/v1/users/" + o.ModifiedBy.Hex() + "/",
		"notification_templates_error": "/v1/organizations/" + ID + "/notification_templates_error/",
		"notification_templates_success": "/v1/organizations/" + ID + "/notification_templates_success/",
		"users": "/v1/organizations/" + ID + "/users/",
		"object_roles": "/v1/organizations/" + ID + "/object_roles/",
		"notification_templates_any":  "/v1/organizations/" + ID + "/notification_templates_any/",
		"teams": "/v1/organizations/" + ID + "/teams/",
		"access_list": "/v1/organizations/" + ID + "/access_list/",
		"notification_templates": "/v1/organizations/" + ID + "/notification_templates/",
		"admins": "/v1/organizations/" + ID + "/admins/",
		"credentials": "/v1/organizations/" + ID + "/credentials/",
		"inventories":  "/v1/organizations/" + ID + "/inventories/",
		"activity_stream": "/v1/organizations/" + ID + "/activity_stream/",
		"projects": "/v1/organizations/" + ID + "/projects/",
	}

	if err := organizationSummary(o); err != nil {
		return err
	}

	return nil
}

func organizationSummary(o *models.Organization) error {

	var modified models.User
	var created models.User
	var owners []models.User

	if err := db.Users().FindId(o.CreatedBy).One(&created); err != nil {
		return err
	}

	if err := db.Users().FindId(o.ModifiedBy).One(&modified); err != nil {
		return err
	}

	jcount, err := db.JobTemplates().Find(bson.M{"organization_id": o.ID}).Count();
	if err != nil {
		log.Println("Erro wile getting JobTemplates count")
	}

	ucount, err := db.Users().Find(bson.M{"organization_id": o.ID}).Count();
	if err != nil {
		log.Println("Erro wile getting Users count")
	}

	tcount, err := db.Teams().Find(bson.M{"organization_id": o.ID}).Count();
	if err != nil {
		log.Println("Erro wile getting Team count")
	}

	icount, err := db.Inventories().Find(bson.M{"organization_id": o.ID}).Count();
	if err != nil {
		log.Println("Erro wile getting Inventories count")
	}

	pcount, err := db.Projects().Find(bson.M{"organization_id": o.ID}).Count();
	if err != nil {
		log.Println("Erro wile getting Project count")
	}

	//TODO: include teams to owners list

	summary := gin.H{
		"object_roles": []gin.H{
			{
				"Description": "Can view all settings for the organization",
				"Name":"auditor",
			},
			{
				"Description":"Can manage all aspects of the organization",
				"Name":"admin",
			},
			{
				"Description":"User is a member of the organization",
				"Name":"member",
			},
			{
				"Description":"May view settings for the organization",
				"Name":"read",
			},
		},

		"related_field_counts": map[string]int{
			"job_templates":jcount,
			"users":ucount,
			"teams":tcount,
			"admins":0,
			"inventories":icount,
			"projects":pcount,
		},
		"created_by": gin.H{
			"id":         created.ID,
			"username":   created.Username,
			"first_name": created.FirstName,
			"last_name":  created.LastName,
		},
		"modified_by": gin.H{
			"id":         modified.ID,
			"username":   modified.Username,
			"first_name": modified.FirstName,
			"last_name":  modified.LastName,
		},
		"owners": owners,
	}

	o.SummaryFields = summary

	return nil
}
