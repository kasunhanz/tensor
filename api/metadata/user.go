package metadata

import (
	"bitbucket.pearson.com/apseng/tensor/models"
	"github.com/gin-gonic/gin"
)

// Create a new organization
func UserMetadata(u *models.User) {
	u.Type = "user"
	u.Url = "/v1/users/" + u.ID.Hex() + "/"
	u.Related = gin.H{
		"admin_of_organizations": "/api/v1/users/" + u.ID.Hex() + "/admin_of_organizations/",
		"organizations":          "/api/v1/users/" + u.ID.Hex() + "/organizations/",
		"roles":                  "/api/v1/users/" + u.ID.Hex() + "/roles/",
		"access_list":            "/api/v1/users/" + u.ID.Hex() + "/access_list/",
		"teams":                  "/api/v1/users/" + u.ID.Hex() + "/teams/",
		"credentials":            "/api/v1/users/" + u.ID.Hex() + "/credentials/",
		"activity_stream":        "/api/v1/users/" + u.ID.Hex() + "/activity_stream/",
		"projects":               "/api/v1/users/" + u.ID.Hex() + "/projects/",
	}
}

func AccessUserMetadata(u *models.AccessUser) {
	u.Type = "user"
	u.Url = "/v1/users/" + u.ID.Hex() + "/"
	u.Related = gin.H{
		"admin_of_organizations": "/api/v1/users/" + u.ID.Hex() + "/admin_of_organizations/",
		"organizations":          "/api/v1/users/" + u.ID.Hex() + "/organizations/",
		"roles":                  "/api/v1/users/" + u.ID.Hex() + "/roles/",
		"access_list":            "/api/v1/users/" + u.ID.Hex() + "/access_list/",
		"teams":                  "/api/v1/users/" + u.ID.Hex() + "/teams/",
		"credentials":            "/api/v1/users/" + u.ID.Hex() + "/credentials/",
		"activity_stream":        "/api/v1/users/" + u.ID.Hex() + "/activity_stream/",
		"projects":               "/api/v1/users/" + u.ID.Hex() + "/projects/",
	}
}
