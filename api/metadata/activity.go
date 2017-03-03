package metadata

import (
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/models/common"
)

// ActivityOrganizationMetadata attach metadata to ActivityOrganization
func ActivityOrganizationMetadata(ao *common.Activity) {
	ID := ao.ID.Hex()
	ao.Type = "activity"
	ao.Links = gin.H{
		"self": "/v1/organizations/" + ID + "/activity_stream",
	}
	ao.Meta = gin.H{}
}

// ActivityUserMetadata attach metadata to ActivityUser
func ActivityUserMetadata(au *common.Activity) {
	ID := au.ID.Hex()
	au.Type = "activity"
	au.Links = gin.H{
		"self": "/v1/users/" + ID + "/activity_stream",
	}
	au.Meta = gin.H{}
}

// ActivityProjectMetadata attach metadata to ActivityProject
func ActivityProjectMetadata(ap *common.Activity) {
	ID := ap.ID.Hex()
	ap.Type = "activity"
	ap.Links = gin.H{
		"self": "/v1/projects/" + ID + "/activity_stream",
	}
	ap.Meta = gin.H{}
}

// ActivityCredentialMetadata attach metadata to ActivityCredential
func ActivityCredentialMetadata(ac *common.Activity) {
	ID := ac.ID.Hex()
	ac.Type = "activity"
	ac.Links = gin.H{
		"self": "/v1/credentials/" + ID + "/activity_stream",
	}
	ac.Links = gin.H{}
	ac.Meta = gin.H{}
}

// ActivityTeamMetadata attach metadata to ActivityTeam
func ActivityTeamMetadata(at *common.Activity) {
	ID := at.ID.Hex()
	at.Type = "activity"
	at.Links = gin.H{
		"self": "/v1/teams/" + ID + "/activity_stream",
	}
	at.Links = gin.H{}
	at.Meta = gin.H{}
}

// ActivityInventoryMetadata attach metadata to ActivityInventory
func ActivityInventoryMetadata(ai *common.Activity) {
	ID := ai.ID.Hex()
	ai.Type = "activity"
	ai.Links = gin.H{
		"self": "/v1/inventories/" + ID + "/activity_stream",
	}
	ai.Links = gin.H{}
	ai.Meta = gin.H{}
}

// ActivityHostMetadata attach metadata to ActivityHost
func ActivityHostMetadata(ah *common.Activity) {
	ID := ah.ID.Hex()
	ah.Type = "activity"
	ah.Links = gin.H{
		"self": "/v1/hosts/" + ID + "/activity_stream",
	}
	ah.Links = gin.H{}
	ah.Meta = gin.H{}
}

// ActivityGroupMetadata attach metadata to ActivityGroup
func ActivityGroupMetadata(ah *common.Activity) {
	ID := ah.ID.Hex()
	ah.Type = "activity"
	ah.Links = gin.H{
		"self": "/v1/groups/" + ID + "/activity_stream",
	}
	ah.Links = gin.H{}
	ah.Meta = gin.H{}
}

// ActivityJobTemplateMetadata attach metadata to ActivityJobTemplate
func ActivityJobTemplateMetadata(ah *common.Activity) {
	ID := ah.ID.Hex()
	ah.Type = "activity"
	ah.Links = gin.H{
		"self": "/v1/job_templates/" + ID + "/activity_stream",
	}
	ah.Links = gin.H{}
	ah.Meta = gin.H{}
}
