package api

import (
	"time"
	"github.com/gin-gonic/gin"
	"strings"
	"bitbucket.pearson.com/apseng/tensor/api/cors"
	"bitbucket.pearson.com/apseng/tensor/api/projects"
	"bitbucket.pearson.com/apseng/tensor/api/sockets"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/api/organizations"
	"bitbucket.pearson.com/apseng/tensor/api/credentials"
	"bitbucket.pearson.com/apseng/tensor/api/users"
	"bitbucket.pearson.com/apseng/tensor/api/teams"
	"bitbucket.pearson.com/apseng/tensor/api/dashboard"
	"bitbucket.pearson.com/apseng/tensor/api/inventories"
	"bitbucket.pearson.com/apseng/tensor/api/hosts"
	"bitbucket.pearson.com/apseng/tensor/api/groups"
	"bitbucket.pearson.com/apseng/tensor/api/jtemplates"
	"bitbucket.pearson.com/apseng/tensor/api/jobs"
	"bitbucket.pearson.com/apseng/tensor/api/gzip"
	"bitbucket.pearson.com/apseng/tensor/api/jwt"
	"net/http"
)

// Route declare all routes
func Route(r *gin.Engine) {

	// Apply the middleware to the router (works with groups too)
	r.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          50 * time.Second,
		Credentials:     true,
		ValidateHeaders: false,
	}))
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	r.GET("", util.GetAPIVersion)
	// set up the namespace
	// future reference: api := r.Group("/api")
	v1 := r.Group("/v1")
	{
		v1.GET("/", util.GetAPIInfo)
		v1.GET("/ping", util.GetPing)
		v1.POST("/authtoken", jwt.HeaderAuthMiddleware.LoginHandler)

		// from here user must authenticated to perforce requests
		v1.Use(jwt.HeaderAuthMiddleware.MiddlewareFunc())
		v1.GET("/refresh_token", jwt.HeaderAuthMiddleware.RefreshHandler)
		v1.GET("/config", getSystemInfo)
		v1.GET("/dashboard", dashboard.GetInfo)
		v1.GET("/ws", sockets.Handler)
		v1.GET("/me", users.GetUser)
		v1.GET("/events", getEvents)

		// organizations
		v1.GET("/organizations", organizations.GetOrganizations)
		v1.POST("/organizations", organizations.AddOrganization)
		v1.GET("/organizations/:organization_id", organizations.OrganizationMiddleware, organizations.GetOrganization)
		v1.PUT("/organizations/:organization_id", organizations.OrganizationMiddleware, organizations.UpdateOrganization)

		// users
		v1.GET("/users", users.GetUsers)
		v1.POST("/users", users.AddUser)
		v1.GET("/users/:user_id", users.GetUserMiddleware, users.GetUser)
		v1.PUT("/users/:user_id", users.GetUserMiddleware, users.UpdateUser)

		v1.POST("/users/:user_id/password", users.GetUserMiddleware, users.UpdateUserPassword)
		v1.DELETE("/users/:user_id", users.GetUserMiddleware, users.DeleteUser)

		// projects
		v1.GET("/projects", projects.GetProjects)
		v1.POST("/projects", projects.AddProject)
		v1.GET("/projects/:credential_id", projects.ProjectMiddleware, projects.GetProject)
		v1.PUT("/projects/:credential_id", projects.ProjectMiddleware, projects.UpdateProject)
		v1.DELETE("/projects/:credential_id", projects.ProjectMiddleware, projects.RemoveProject)

		// credentials
		v1.GET("/credentials", credentials.GetCredentials)
		v1.POST("/credentials", credentials.AddCredential)
		v1.GET("/credentials/:credential_id", credentials.CredentialMiddleware, credentials.GetCredential)
		v1.PUT("/credentials/:credential_id", credentials.CredentialMiddleware, credentials.UpdateCredential)
		v1.DELETE("/credentials/:credential_id", credentials.CredentialMiddleware, credentials.RemoveCredential)

		// teams
		v1.GET("/teams", teams.GetTeams)
		v1.POST("/teams", teams.AddTeam)
		v1.GET("/teams/:team_id", teams.TeamMiddleware, teams.GetTeam)
		v1.PUT("/teams/:team_id", teams.TeamMiddleware, teams.UpdateTeam)
		v1.DELETE("/teams/:team_id", teams.TeamMiddleware, teams.RemoveTeam)

		// inventories
		v1.GET("/inventories", inventories.GetInventories)
		v1.POST("/inventories", inventories.AddInventory)
		v1.GET("/inventories/:inventory_id", inventories.InventoryMiddleware, inventories.GetInventory)
		v1.PUT("/inventories/:inventory_id", inventories.InventoryMiddleware, inventories.UpdateInventory)
		v1.DELETE("/inventories/:inventory_id", inventories.InventoryMiddleware, inventories.RemoveInventory)

		// hosts
		v1.GET("/hosts", hosts.GetHosts)
		v1.POST("/hosts", hosts.AddHost)
		v1.GET("/hosts/:host_id", hosts.HostMiddleware, hosts.GetHost)
		v1.PUT("/hosts/:host_id", hosts.HostMiddleware, hosts.UpdateHost)
		v1.DELETE("/hosts/:host_id", hosts.HostMiddleware, hosts.RemoveHost)

		// groups
		v1.GET("/groups", groups.GetGroups)
		v1.POST("/groups", groups.AddGroup)
		v1.GET("/groups/:group_id", groups.GroupMiddleware, groups.GetGroup)
		v1.PUT("/groups/:group_id", groups.GroupMiddleware, groups.UpdateGroup)
		v1.DELETE("/groups/:group_id", groups.GroupMiddleware, groups.RemoveGroup)

		// job_templates
		v1.GET("/job_templates", jtemplate.GetJTemplates)
		v1.POST("/job_templates", jtemplate.AddJTemplate)
		v1.GET("/job_templates/:job_template_id", jtemplate.JTemplateM, jtemplate.GetJTemplate)
		v1.PUT("/job_templates/:job_template_id", jtemplate.JTemplateM, jtemplate.UpdateJTemplate)
		v1.DELETE("/job_templates/:job_template_id", jtemplate.JTemplateM, jtemplate.RemoveJTemplate)

		// job
		v1.GET("/jobs", jobs.GetJobs)
		v1.POST("/jobs", jobs.GetJob)
		v1.GET("/jobs/:job_id", jobs.JobMiddleware, jobs.GetJob)
		v1.DELETE("/jobs/:job_id", jobs.JobMiddleware, jobs.GetJob)
	}
}

func getSystemInfo(c *gin.Context) {
	body := map[string]interface{}{
		"version": util.Version,
		"config": map[string]string{
			"dbHost":  strings.Join(util.Config.MongoDB.Hosts, ","),
			"dbName":  util.Config.MongoDB.DbName,
			"dbUser":  util.Config.MongoDB.Username,
			"path":    util.Config.TmpPath,
			"cmdPath": util.FindTensor(),
		},
	}

	c.JSON(http.StatusOK, body)
}
