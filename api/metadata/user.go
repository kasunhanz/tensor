package metadata

import (
	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/models/common"
)

// Create a new organization
func UserMetadata(u *common.User) {
	u.Type = "user"
	u.URL = "/v1/users/" + u.ID.Hex() + "/"
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

func AccessUserMetadata(u *common.AccessUser) {
	u.Type = "user"
	u.URL = "/v1/users/" + u.ID.Hex() + "/"
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
