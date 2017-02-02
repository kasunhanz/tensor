package metadata

import (
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/models/common"
)

// ActivityMetadata attach metadata to ActivityOrganization
func ActivityMetadata(ao *common.ActivityOrganization, typ string) {
	ID := ao.ID.Hex()
	ao.Type = "activity_stream"
	ao.URL = "/v1/" + typ + "/" + ID + "/activity_stream/"
	ao.Related = gin.H{}

	activitySummary(ao)
}

func activitySummary(ao *common.ActivityOrganization) {
	ao.Summary = gin.H{}
}
