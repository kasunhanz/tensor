package terraform

import (
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/models/terraform"
)

// ActivityJobTemplateMetadata attach metadata to ActivityJobTemplate
func ActivityJobTemplateMetadata(ah *terraform.ActivityJobTemplate) {
	ID := ah.ID.Hex()
	ah.Type = "activity"
	ah.Links = gin.H{
		"self": "/v1/job_templates/" + ID + "/activity_stream",
	}
	ah.Meta = gin.H{}
}
