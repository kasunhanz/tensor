package api

import (
	"time"
	"github.com/gin-gonic/gin"
	"strings"
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
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/api/cors"
	"bitbucket.pearson.com/apseng/tensor/api/jwt"
	"bitbucket.pearson.com/apseng/tensor/models"
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
	// causing some issues
	//r.Use(gzip.Gzip(gzip.DefaultCompression))

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

		// organizations
		gOrganizations := v1.Group("/organizations")
		{
			gOrganizations.GET("/", organizations.GetOrganizations)
			gOrganizations.POST("/", organizations.AddOrganization)
			gOrganizations.GET("/:organization_id", organizations.Middleware, organizations.GetOrganization)
			gOrganizations.PUT("/:organization_id", organizations.Middleware, organizations.UpdateOrganization)
			gOrganizations.DELETE("/:organization_id", organizations.Middleware, organizations.RemoveOrganization)

			//related
			gOrganizations.GET("/:organization_id/users/", organizations.Middleware, organizations.GetUsers)
			gOrganizations.GET("/:organization_id/inventories/", organizations.Middleware, organizations.GetInventories)
			gOrganizations.GET("/:organization_id/activity_stream/", organizations.Middleware, organizations.ActivityStream)
			gOrganizations.GET("/:organization_id/projects/", organizations.Middleware, organizations.GetProjects)
			gOrganizations.GET("/:organization_id/admins/", organizations.Middleware, organizations.GetAdmins)
			gOrganizations.GET("/:organization_id/teams/", organizations.Middleware, organizations.GetTeams)
			gOrganizations.PUT("/:organization_id/credentials/", organizations.Middleware, organizations.GetCredentials)

			gOrganizations.PUT("/:organization_id/notification_templates_error/", organizations.Middleware, notImplemented) //TODO: implement
			gOrganizations.PUT("/:organization_id/notification_templates_success/", organizations.Middleware, notImplemented) //TODO: implement
			gOrganizations.PUT("/:organization_id/object_roles/", organizations.Middleware, notImplemented) //TODO: implement
			gOrganizations.PUT("/:organization_id/notification_templates/", organizations.Middleware, notImplemented) //TODO: implement
			gOrganizations.PUT("/:organization_id/notification_templates_any/", organizations.Middleware, notImplemented) //TODO: implement
			gOrganizations.PUT("/:organization_id/access_list/", organizations.Middleware, notImplemented) //TODO: implement
		}

		// users
		gUsers := v1.Group("/users")
		{
			gUsers.GET("/", users.GetUsers)
			gUsers.POST("/", users.AddUser)
			gUsers.GET("/:user_id", users.Middleware, users.GetUser)
			gUsers.PUT("/:user_id", users.Middleware, users.UpdateUser)
			gUsers.POST("/:user_id/password", users.Middleware, users.UpdateUserPassword)
			gUsers.DELETE("/:user_id", users.Middleware, users.DeleteUser)

			//related
			gUsers.GET("/:user_id/admin_of_organizations/", users.Middleware, users.AdminsOfOrganizations)
			gUsers.GET("/:user_id/organizations/", users.Middleware, users.Organizations)
			gUsers.GET("/:user_id/teams/", users.Middleware, users.Teams)
			gUsers.GET("/:user_id/credentials/", users.Middleware, users.Credentials)
			gUsers.GET("/:user_id/activity_stream/", users.Middleware, users.ActivityStream)
			gUsers.GET("/:user_id/projects/", users.Middleware, users.Projects)

			gUsers.GET("/:user_id/roles/", users.Middleware, notImplemented) //TODO: implement
			gUsers.GET("/:user_id/access_list/", users.Middleware, notImplemented) //TODO: implement

		}

		// projects
		gProjects := v1.Group("/projects")
		{
			//main functions
			gProjects.GET("/", projects.GetProjects)
			gProjects.POST("/", projects.AddProject)
			gProjects.GET("/:project_id", projects.Middleware, projects.GetProject)
			gProjects.PUT("/:project_id", projects.Middleware, projects.UpdateProject)
			gProjects.DELETE("/:project_id", projects.Middleware, projects.RemoveProject)

			//related
			gProjects.GET("/:project_id/activity_stream", projects.Middleware, projects.ActivityStream)
			gProjects.GET("/:project_id/teams", projects.Middleware, projects.Teams)
			gProjects.GET("/:project_id/playbooks", projects.Middleware, projects.Playbooks)
			gProjects.GET("/:project_id/access_list", projects.Middleware, projects.AccessList)
			gProjects.GET("/:project_id/update", projects.Middleware, projects.SCMUpdate)
			gProjects.GET("/:project_id/project_updates", projects.Middleware, projects.ProjectUpdates)

			gProjects.GET("/:project_id/schedules", projects.Middleware, notImplemented) //TODO: implement
		}


		// credentials
		gCredentials := v1.Group("/credentials")
		{
			gCredentials.GET("/", credentials.GetCredentials)
			gCredentials.POST("/", credentials.AddCredential)
			gCredentials.GET("/:credential_id", credentials.Middleware, credentials.GetCredential)
			gCredentials.PUT("/:credential_id", credentials.Middleware, credentials.UpdateCredential)
			gCredentials.DELETE("/:credential_id", credentials.Middleware, credentials.RemoveCredential)

			//relatedd
			gCredentials.GET("/:credential_id/owner_teams/", credentials.Middleware, credentials.OwnerTeams)
			gCredentials.GET("/:credential_id/owner_users/", credentials.Middleware, credentials.OwnerUsers)
			gCredentials.GET("/:credential_id/activity_stream/", credentials.Middleware, credentials.ActivityStream)
			gCredentials.GET("/:credential_id/access_list/", credentials.Middleware, notImplemented) //TODO: implement
			gCredentials.GET("/:credential_id/object_roles/", credentials.Middleware, notImplemented) //TODO: implement
		}

		// teams
		gTeams := v1.Group("/teams")
		{
			gTeams.GET("/", teams.GetTeams)
			gTeams.POST("/", teams.AddTeam)
			gTeams.GET("/:team_id", teams.Middleware, teams.GetTeam)
			gTeams.PUT("/:team_id", teams.Middleware, teams.UpdateTeam)
			gTeams.DELETE("/:team_id", teams.Middleware, teams.RemoveTeam)

			//related
			gTeams.GET("/:team_id/users", teams.Middleware, teams.Users)
			gTeams.GET("/:team_id/credentials", teams.Middleware, teams.Credentials)
			gTeams.GET("/:team_id/projects", teams.Middleware, teams.Projects)
			gTeams.GET("/:team_id/activity_stream", teams.Middleware, teams.ActivityStream)
			gTeams.GET("/:team_id/access_list", teams.Middleware, teams.AccessList)
		}

		// inventories
		gInventories := v1.Group("/inventories")
		{
			gInventories.GET("/", inventories.GetInventories)
			gInventories.POST("/", inventories.AddInventory)
			gInventories.GET("/:inventory_id", inventories.Middleware, inventories.GetInventory)
			gInventories.PUT("/:inventory_id", inventories.Middleware, inventories.UpdateInventory)
			gInventories.DELETE("/:inventory_id", inventories.Middleware, inventories.RemoveInventory)
			gInventories.GET("/:inventory_id/script", inventories.Middleware, inventories.Script)

			//related
			gInventories.GET("/:inventory_id/job_templates", inventories.Middleware, inventories.JobTemplates)
			gInventories.GET("/:inventory_id/variable_data", inventories.Middleware, inventories.VariableData)
			gInventories.GET("/:inventory_id/root_groups", inventories.Middleware, inventories.RootGroups)
			gInventories.GET("/:inventory_id/ad_hoc_commands", inventories.Middleware, notImplemented) //TODO: implement
			gInventories.GET("/:inventory_id/tree", inventories.Middleware, notImplemented) //TODO: implement
			gInventories.GET("/:inventory_id/access_list", inventories.Middleware, inventories.AccessList)
			gInventories.GET("/:inventory_id/hosts", inventories.Middleware, inventories.Hosts)
			gInventories.GET("/:inventory_id/groups", inventories.Middleware, inventories.Groups)
			gInventories.GET("/:inventory_id/activity_stream", inventories.Middleware, inventories.ActivityStream)

			gInventories.GET("/:inventory_id/inventory_sources", inventories.Middleware, notImplemented) //TODO: implement
		}

		// hosts
		gHosts := v1.Group("/hosts")
		{
			gHosts.GET("/", hosts.GetHosts)
			gHosts.POST("/", hosts.AddHost)
			gHosts.GET("/:host_id", hosts.Middleware, hosts.GetHost)
			gHosts.PUT("/:host_id", hosts.Middleware, hosts.UpdateHost)
			gHosts.DELETE("/:host_id", hosts.Middleware, hosts.RemoveHost)

			//related
			gHosts.GET("/:host_id/job_host_summaries", hosts.Middleware, notImplemented) //TODO: implement
			gHosts.GET("/:host_id/job_events", hosts.Middleware, notImplemented) //TODO: implement
			gHosts.GET("/:host_id/ad_hoc_commands", hosts.Middleware, notImplemented) //TODO: implement
			gHosts.GET("/:host_id/inventory_sources", hosts.Middleware, notImplemented) //TODO: implement
			gHosts.GET("/:host_id/activity_stream", hosts.Middleware, hosts.ActivityStream)
			gHosts.GET("/:host_id/ad_hoc_command_events", hosts.Middleware, notImplemented) //TODO: implement
			gHosts.GET("/:host_id/variable_data", hosts.Middleware, hosts.VariableData)
			gHosts.GET("/:host_id/groups", hosts.Middleware, hosts.Groups)
			gHosts.GET("/:host_id/all_groups", hosts.Middleware, hosts.AllGroups)

		}

		// groups
		gGroups := v1.Group("/groups")
		{
			gGroups.GET("/", groups.GetGroups)
			gGroups.POST("/", groups.AddGroup)
			gGroups.GET("/:group_id", groups.Middleware, groups.GetGroup)
			gGroups.PUT("/:group_id", groups.Middleware, groups.UpdateGroup)
			gGroups.DELETE("/:group_id", groups.Middleware, groups.RemoveGroup)

			//related
			gGroups.GET("/:group_id/variable_data", groups.Middleware, groups.VariableData)
			gGroups.GET("/:group_id/job_events", groups.Middleware, notImplemented) //TODO: implement
			gGroups.GET("/:group_id/potential_children", groups.Middleware, notImplemented) //TODO: implement
			gGroups.GET("/:group_id/ad_hoc_commands", groups.Middleware, notImplemented) //TODO: implement
			gGroups.GET("/:group_id/all_hosts", groups.Middleware, notImplemented) //TODO: implement
			gGroups.GET("/:group_id/activity_stream", groups.Middleware, notImplemented) //TODO: implement
			gGroups.GET("/:group_id/hosts", groups.Middleware, notImplemented) //TODO: implement
			gGroups.GET("/:group_id/children", groups.Middleware, notImplemented) //TODO: implement

			gGroups.GET("/:group_id/job_host_summaries", groups.Middleware, notImplemented) //TODO: implement
		}

		// job_templates
		gJobTemplates := v1.Group("/job_templates")
		{
			gJobTemplates.GET("/", jtemplate.GetJTemplates)
			gJobTemplates.POST("/", jtemplate.AddJTemplate)
			gJobTemplates.GET("/:job_template_id", jtemplate.Middleware, jtemplate.GetJTemplate)
			gJobTemplates.PUT("/:job_template_id", jtemplate.Middleware, jtemplate.UpdateJTemplate)
			gJobTemplates.DELETE("/:job_template_id", jtemplate.Middleware, jtemplate.RemoveJTemplate)

			//related
			gJobTemplates.GET("/:job_template_id/jobs/", jtemplate.Middleware, jtemplate.Jobs)
			gJobTemplates.GET("/:job_template_id/object_roles/", jtemplate.Middleware, jtemplate.ObjectRoles)
			gJobTemplates.GET("/:job_template_id/access_list/", jtemplate.Middleware, jtemplate.AccessList)
			gJobTemplates.GET("/:job_template_id/launch/", jtemplate.Middleware, jtemplate.LaunchInfo)
			gJobTemplates.POST("/:job_template_id/launch/", jtemplate.Middleware, jtemplate.Launch)
			gJobTemplates.GET("/:job_template_id/activity_stream/", jtemplate.Middleware, jtemplate.ActivityStream)

			gJobTemplates.GET("/:job_template_id/schedules/", jtemplate.Middleware, notImplemented) //TODO: implement
			gJobTemplates.GET("/:job_template_id/notification_templates_error/", jtemplate.Middleware, notImplemented) //TODO: implement
			gJobTemplates.GET("/:job_template_id/notification_templates_success/", jtemplate.Middleware, notImplemented) //TODO: implement
			gJobTemplates.GET("/:job_template_id/notification_templates_any/", jtemplate.Middleware, notImplemented) //TODO: implement
		}

		// job
		gJobs := v1.Group("/jobs")
		{
			gJobs.GET("/", jobs.GetJobs)
			gJobs.GET("/:job_id", jobs.Middleware, jobs.GetJob)
			gJobs.DELETE("/:job_id", jobs.Middleware, jobs.GetJob)

			//related
			gJobs.GET("/:job_id/stdout", jobs.Middleware, notImplemented) //TODO: implement
			gJobs.GET("/:job_id/job_tasks", jobs.Middleware, notImplemented) //TODO: implement
			gJobs.GET("/:job_id/job_plays", jobs.Middleware, notImplemented) //TODO: implement
			gJobs.GET("/:job_id/job_events", jobs.Middleware, notImplemented) //TODO: implement
			gJobs.GET("/:job_id/notifications", jobs.Middleware, notImplemented) //TODO: implement
			gJobs.GET("/:job_id/activity_stream", jobs.Middleware, notImplemented) //TODO: implement
			gJobs.GET("/:job_id/start", jobs.Middleware, notImplemented) //TODO: implement
			gJobs.GET("/:job_id/cancel", jobs.Middleware, notImplemented) //TODO: implement
			gJobs.GET("/:job_id/relaunch", jobs.Middleware, notImplemented) //TODO: implement
		}

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

func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.Error{
		Code:http.StatusNotImplemented,
		Message: "Method not implemented",
	})
}