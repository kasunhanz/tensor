package metadata

import (
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/models/ansible"
	"github.com/pearsonappeng/tensor/models/common"
)

// ActivityOrganizationMetadata attach metadata to ActivityOrganization
func ActivityOrganizationMetadata(ao *common.ActivityOrganization) {
	ID := ao.ID.Hex()
	ao.Type = "activity_stream"
	ao.URL = "/v1/organizations/" + ID + "/activity_stream/"
	ao.Related = gin.H{}
	ao.Summary = gin.H{}
}

// ActivityUserMetadata attach metadata to ActivityUser
func ActivityUserMetadata(au *common.ActivityUser) {
	ID := au.ID.Hex()
	au.Type = "activity_stream"
	au.URL = "/v1/users/" + ID + "/activity_stream/"
	au.Related = gin.H{}
	au.Summary = gin.H{}
}

// ActivityProjectMetadata attach metadata to ActivityProject
func ActivityProjectMetadata(ap *common.ActivityProject) {
	ID := ap.ID.Hex()
	ap.Type = "activity_stream"
	ap.URL = "/v1/projects/" + ID + "/activity_stream/"
	ap.Related = gin.H{}
	ap.Summary = gin.H{}
}

// ActivityCredentialMetadata attach metadata to ActivityCredential
func ActivityCredentialMetadata(ac *common.ActivityCredential) {
	ID := ac.ID.Hex()
	ac.Type = "activity_stream"
	ac.URL = "/v1/credentials/" + ID + "/activity_stream/"
	ac.Related = gin.H{}
	ac.Summary = gin.H{}
}

// ActivityTeamMetadata attach metadata to ActivityTeam
func ActivityTeamMetadata(at *common.ActivityTeam) {
	ID := at.ID.Hex()
	at.Type = "activity_stream"
	at.URL = "/v1/teams/" + ID + "/activity_stream/"
	at.Related = gin.H{}
	at.Summary = gin.H{}
}

// ActivityInventoryMetadata attach metadata to ActivityInventory
func ActivityInventoryMetadata(ai *ansible.ActivityInventory) {
	ID := ai.ID.Hex()
	ai.Type = "activity_stream"
	ai.URL = "/v1/inventories/" + ID + "/activity_stream/"
	ai.Related = gin.H{}
	ai.Summary = gin.H{}
}

// ActivityHostMetadata attach metadata to ActivityHost
func ActivityHostMetadata(ah *ansible.ActivityHost) {
	ID := ah.ID.Hex()
	ah.Type = "activity_stream"
	ah.URL = "/v1/hosts/" + ID + "/activity_stream/"
	ah.Related = gin.H{}
	ah.Summary = gin.H{}
}
