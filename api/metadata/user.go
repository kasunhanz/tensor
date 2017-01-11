package metadata

import (
	"github.com/gamunu/tensor/models"
	"github.com/gin-gonic/gin"
)

// Create a new organization
func UserMetadata(u *models.User) {
	u.Type = "user"
	u.Url = "/v1/users/" + u.ID.Hex() + "/"
	u.Related = gin.H{
		"admin_of_organizations": "/v1/users/" + u.ID.Hex() + "/admin_of_organizations/",
		"organizations":          "/v1/users/" + u.ID.Hex() + "/organizations/",
		"roles":                  "/v1/users/" + u.ID.Hex() + "/roles/",
		"access_list":            "/v1/users/" + u.ID.Hex() + "/access_list/",
		"teams":                  "/v1/users/" + u.ID.Hex() + "/teams/",
		"credentials":            "/v1/users/" + u.ID.Hex() + "/credentials/",
		"activity_stream":        "/v1/users/" + u.ID.Hex() + "/activity_stream/",
		"projects":               "/v1/users/" + u.ID.Hex() + "/projects/",
	}
}

func AccessUserMetadata(u *models.AccessUser) {
	u.Type = "user"
	u.Url = "/v1/users/" + u.ID.Hex() + "/"
	u.Related = gin.H{
		"admin_of_organizations": "/v1/users/" + u.ID.Hex() + "/admin_of_organizations/",
		"organizations":          "/v1/users/" + u.ID.Hex() + "/organizations/",
		"roles":                  "/v1/users/" + u.ID.Hex() + "/roles/",
		"access_list":            "/v1/users/" + u.ID.Hex() + "/access_list/",
		"teams":                  "/v1/users/" + u.ID.Hex() + "/teams/",
		"credentials":            "/v1/users/" + u.ID.Hex() + "/credentials/",
		"activity_stream":        "/v1/users/" + u.ID.Hex() + "/activity_stream/",
		"projects":               "/v1/users/" + u.ID.Hex() + "/projects/",
	}
}
