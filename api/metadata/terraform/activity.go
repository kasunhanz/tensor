package terraform

import (
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/models/common"
)

// ActivityJobTemplateMetadata attach metadata to ActivityJobTemplate
func ActivityJobTemplateMetadata(ah *common.Activity) {
	ID := ah.ID.Hex()
	ah.Type = "activity"
	ah.Links = gin.H{
		"self": "/v1/terraform_job_templates/" + ID + "/activity_stream",
	}
	ah.Meta = gin.H{}
}
