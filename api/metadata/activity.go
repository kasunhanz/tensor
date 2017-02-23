package metadata

import (
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
)

// ActivityOrganizationMetadata attach metadata to ActivityOrganization
func ActivityOrganizationMetadata(ao *common.ActivityOrganization) {
	ID := ao.ID.Hex()
	ao.Type = "activity"
	ao.Links = gin.H{
		"self":  "/v1/organizations/" + ID + "/activity_stream",
	}
	ao.Meta = gin.H{}
}

// ActivityUserMetadata attach metadata to ActivityUser
func ActivityUserMetadata(au *common.ActivityUser) {
	ID := au.ID.Hex()
	au.Type = "activity"
	au.Links = gin.H{
		"self": "/v1/users/" + ID + "/activity_stream",
	}
	au.Meta = gin.H{}
}

// ActivityProjectMetadata attach metadata to ActivityProject
func ActivityProjectMetadata(ap *common.ActivityProject) {
	ID := ap.ID.Hex()
	ap.Type = "activity"
	ap.Links = gin.H{
		"self": "/v1/projects/" + ID + "/activity_stream",
	}
	ap.Meta = gin.H{}
}

// ActivityCredentialMetadata attach metadata to ActivityCredential
func ActivityCredentialMetadata(ac *common.ActivityCredential) {
	ID := ac.ID.Hex()
	ac.Type = "activity"
	ac.Links = gin.H{
		"self": "/v1/credentials/" + ID + "/activity_stream",
	}
	ac.Links = gin.H{}
	ac.Meta = gin.H{}
}

// ActivityTeamMetadata attach metadata to ActivityTeam
func ActivityTeamMetadata(at *common.ActivityTeam) {
	ID := at.ID.Hex()
	at.Type = "activity"
	at.Links = gin.H{
		"self": "/v1/teams/" + ID + "/activity_stream",
	}
	at.Links = gin.H{}
	at.Meta = gin.H{}
}

// ActivityInventoryMetadata attach metadata to ActivityInventory
func ActivityInventoryMetadata(ai *ansible.ActivityInventory) {
	ID := ai.ID.Hex()
	ai.Type = "activity"
	ai.Links = gin.H{
		"self": "/v1/inventories/" + ID + "/activity_stream",
	}
	ai.Links = gin.H{}
	ai.Meta = gin.H{}
}

// ActivityHostMetadata attach metadata to ActivityHost
func ActivityHostMetadata(ah *ansible.ActivityHost) {
	ID := ah.ID.Hex()
	ah.Type = "activity"
	ah.Links = gin.H{
		"self": "/v1/hosts/" + ID + "/activity_stream",
	}
	ah.Links = gin.H{}
	ah.Meta = gin.H{}
}

// ActivityGroupMetadata attach metadata to ActivityGroup
func ActivityGroupMetadata(ah *ansible.ActivityGroup) {
	ID := ah.ID.Hex()
	ah.Type = "activity"
	ah.Links = gin.H{
		"self": "/v1/groups/" + ID + "/activity_stream",
	}
	ah.Links = gin.H{}
	ah.Meta = gin.H{}
}

// ActivityJobTemplateMetadata attach metadata to ActivityJobTemplate
func ActivityJobTemplateMetadata(ah *ansible.ActivityJobTemplate) {
	ID := ah.ID.Hex()
	ah.Type = "activity"
	ah.Links = gin.H{
		"self": "/v1/job_templates/" + ID + "/activity_stream",
	}
	ah.Links = gin.H{}
	ah.Meta = gin.H{}
}
