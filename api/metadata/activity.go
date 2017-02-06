package metadata

import (
	"github.com/gin-gonic/gin"
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
func ActivityCredentialMetadata(ap *common.ActivityCredential) {
	ID := ap.ID.Hex()
	ap.Type = "activity_stream"
	ap.URL = "/v1/credentials/" + ID + "/activity_stream/"
	ap.Related = gin.H{}
	ap.Summary = gin.H{}
}
