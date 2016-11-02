package controllers

import (
	"time"
	"github.com/gin-gonic/gin"
	"bitbucket.pearson.com/apseng/tensor/controllers/projects"
	"bitbucket.pearson.com/apseng/tensor/controllers/sockets"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/controllers/organizations"
	"bitbucket.pearson.com/apseng/tensor/controllers/credentials"
	"bitbucket.pearson.com/apseng/tensor/controllers/users"
	"bitbucket.pearson.com/apseng/tensor/controllers/teams"
	"bitbucket.pearson.com/apseng/tensor/controllers/dashboard"
	"bitbucket.pearson.com/apseng/tensor/controllers/inventories"
	"net/http"
	"bitbucket.pearson.com/apseng/tensor/models"
	"bitbucket.pearson.com/apseng/tensor/cors"
	"bitbucket.pearson.com/apseng/tensor/jwt"
	"bitbucket.pearson.com/apseng/tensor/controllers/hosts"
	"bitbucket.pearson.com/apseng/tensor/controllers/groups"
	"bitbucket.pearson.com/apseng/tensor/controllers/jtemplates"
	"bitbucket.pearson.com/apseng/tensor/controllers/jobs"
	"strings"
)

// Route declare all routes
func Route(r *gin.Engine) {

	// Apply the middleware to the router (works with groups too)
	r.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE, PATCH",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          50 * time.Second,
		Credentials:     true,
		ValidateHeaders: false,
	}))

	// handle not found
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, models.Error{
			Code: http.StatusNotFound,
			Messages: []string{"Not found"},
		})
	})

	r.GET("", util.GetAPIVersion)
	// set up the namespace
	// future reference: api := r.Group("/api")
	r.GET("/v1/", util.GetAPIInfo)
	r.GET("/v1/ping", util.GetPing)
	r.POST("/v1/authtoken", jwt.HeaderAuthMiddleware.LoginHandler)

	// from here user must authenticated to perforce requests
	r.Use(jwt.HeaderAuthMiddleware.MiddlewareFunc())
	r.GET("/v1/refresh_token", jwt.HeaderAuthMiddleware.RefreshHandler)
	r.GET("/v1/config", getSystemInfo)
	r.GET("/v1/dashboard", dashboard.GetInfo)
	r.GET("/v1/ws", sockets.Handler)
	r.GET("/v1/me", users.GetUser)

	// organizations

	r.GET("/v1/organizations/", organizations.GetOrganizations)
	r.POST("/v1/organizations/", organizations.AddOrganization)
	r.GET("/v1/organizations/:organization_id/", organizations.Middleware, organizations.GetOrganization)
	r.PUT("/v1/organizations/:organization_id/", organizations.Middleware, organizations.UpdateOrganization)
	r.PATCH("/v1/organizations/:organization_id/", organizations.Middleware, organizations.PatchOrganization)
	r.DELETE("/v1/organizations/:organization_id/", organizations.Middleware, organizations.RemoveOrganization)

	//related
	r.GET("/v1/organizations/:organization_id/users/", organizations.Middleware, organizations.GetUsers)
	r.GET("/v1/organizations/:organization_id/inventories/", organizations.Middleware, organizations.GetInventories)
	r.GET("/v1/organizations/:organization_id/activity_stream/", organizations.Middleware, organizations.ActivityStream)
	r.GET("/v1/organizations/:organization_id/projects/", organizations.Middleware, organizations.GetProjects)
	r.GET("/v1/organizations/:organization_id/admins/", organizations.Middleware, organizations.GetAdmins)
	r.GET("/v1/organizations/:organization_id/teams/", organizations.Middleware, organizations.GetTeams)
	r.GET("/v1/organizations/:organization_id/credentials/", organizations.Middleware, organizations.GetCredentials)

	r.GET("/v1/organizations/:organization_id/notification_templates_error/", organizations.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/organizations/:organization_id/notification_templates_success/", organizations.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/organizations/:organization_id/object_roles/", organizations.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/organizations/:organization_id/notification_templates/", organizations.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/organizations/:organization_id/notification_templates_any/", organizations.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/organizations/:organization_id/access_list/", organizations.Middleware, notImplemented) //TODO: implement

	// users
	r.GET("/v1/users/", users.GetUsers)
	r.POST("/v1/users/", users.AddUser)
	r.GET("/v1/users/:user_id/", users.Middleware, users.GetUser)
	r.PUT("/v1/users/:user_id/", users.Middleware, users.UpdateUser)
	r.DELETE("/v1/users/:user_id/", users.Middleware, users.DeleteUser)

	//related
	r.GET("/v1/users/:user_id/admin_of_organizations/", users.Middleware, users.AdminsOfOrganizations)
	r.GET("/v1/users/:user_id/organizations/", users.Middleware, users.Organizations)
	r.GET("/v1/users/:user_id/teams/", users.Middleware, users.Teams)
	r.GET("/v1/users/:user_id/credentials/", users.Middleware, users.Credentials)
	r.GET("/v1/users/:user_id/activity_stream/", users.Middleware, users.ActivityStream)
	r.GET("/v1/users/:user_id/projects/", users.Middleware, users.Projects)

	r.GET("/v1/users/:user_id/roles/", users.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/users/:user_id/access_list/", users.Middleware, notImplemented) //TODO: implement

	// projects
	//main functions
	r.GET("/v1/projects/", projects.GetProjects)
	r.POST("/v1/projects/", projects.AddProject)
	r.GET("/v1/projects/:project_id/", projects.Middleware, projects.GetProject)
	r.PUT("/v1/projects/:project_id/", projects.Middleware, projects.UpdateProject)
	r.PATCH("/v1/projects/:project_id/", projects.Middleware, projects.PatchProject)
	r.DELETE("/v1/projects/:project_id/", projects.Middleware, projects.RemoveProject)

	//related
	r.GET("/v1/projects/:project_id/activity_stream/", projects.Middleware, projects.ActivityStream)
	r.GET("/v1/projects/:project_id/teams/", projects.Middleware, projects.Teams)
	r.GET("/v1/projects/:project_id/playbooks/", projects.Middleware, projects.Playbooks)
	r.GET("/v1/projects/:project_id/access_list/", projects.Middleware, projects.AccessList)
	r.GET("/v1/projects/:project_id/update/", projects.Middleware, projects.SCMUpdateInfo)
	r.POST("/v1/projects/:project_id/update/", projects.Middleware, projects.SCMUpdate)
	r.GET("/v1/projects/:project_id/project_updates/", projects.Middleware, projects.ProjectUpdates)

	r.GET("/v1/projects/:project_id/schedules/", projects.Middleware, notImplemented) //TODO: implement


	// credentials
	r.GET("/v1/credentials/", credentials.GetCredentials)
	r.POST("/v1/credentials/", credentials.AddCredential)
	r.GET("/v1/credentials/:credential_id/", credentials.Middleware, credentials.GetCredential)
	r.PUT("/v1/credentials/:credential_id/", credentials.Middleware, credentials.UpdateCredential)
	r.PATCH("/v1/credentials/:credential_id/", credentials.Middleware, credentials.PatchCredential)
	r.DELETE("/v1/credentials/:credential_id/", credentials.Middleware, credentials.RemoveCredential)

	//relatedd
	r.GET("/v1/credentials/:credential_id/owner_teams/", credentials.Middleware, credentials.OwnerTeams)
	r.GET("/v1/credentials/:credential_id/owner_users/", credentials.Middleware, credentials.OwnerUsers)
	r.GET("/v1/credentials/:credential_id/activity_stream/", credentials.Middleware, credentials.ActivityStream)
	r.GET("/v1/credentials/:credential_id/access_list/", credentials.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/credentials/:credential_id/object_roles/", credentials.Middleware, notImplemented) //TODO: implement

	// teams
	r.GET("/v1/teams/", teams.GetTeams)
	r.POST("/v1/teams/", teams.AddTeam)
	r.GET("/v1/teams/:team_id/", teams.Middleware, teams.GetTeam)
	r.PUT("/v1/teams/:team_id/", teams.Middleware, teams.UpdateTeam)
	r.PATCH("/v1/teams/:team_id/", teams.Middleware, teams.PatchTeam)
	r.DELETE("/v1/teams/:team_id/", teams.Middleware, teams.RemoveTeam)

	//related
	r.GET("/v1/teams/:team_id/users/", teams.Middleware, teams.Users)
	r.GET("/v1/teams/:team_id/credentials/", teams.Middleware, teams.Credentials)
	r.GET("/v1/teams/:team_id/projects/", teams.Middleware, teams.Projects)
	r.GET("/v1/teams/:team_id/activity_stream/", teams.Middleware, teams.ActivityStream)
	r.GET("/v1/teams/:team_id/access_list/", teams.Middleware, teams.AccessList)

	// inventories
	r.GET("/v1/inventories/", inventories.GetInventories)
	r.POST("/v1/inventories/", inventories.AddInventory)
	r.GET("/v1/inventories/:inventory_id/", inventories.Middleware, inventories.GetInventory)
	r.PUT("/v1/inventories/:inventory_id/", inventories.Middleware, inventories.UpdateInventory)
	r.PATCH("/v1/inventories/:inventory_id/", inventories.Middleware, inventories.PatchInventory)
	r.DELETE("/v1/inventories/:inventory_id/", inventories.Middleware, inventories.RemoveInventory)
	r.GET("/v1/inventories/:inventory_id/script/", inventories.Middleware, inventories.Script)

	//related
	r.GET("/v1/inventories/:inventory_id/job_templates/", inventories.Middleware, inventories.JobTemplates)
	r.GET("/v1/inventories/:inventory_id/variable_data/", inventories.Middleware, inventories.VariableData)
	r.GET("/v1/inventories/:inventory_id/root_groups/", inventories.Middleware, inventories.RootGroups)
	r.GET("/v1/inventories/:inventory_id/ad_hoc_commands/", inventories.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/inventories/:inventory_id/tree/", inventories.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/inventories/:inventory_id/access_list/", inventories.Middleware, inventories.AccessList)
	r.GET("/v1/inventories/:inventory_id/hosts/", inventories.Middleware, inventories.Hosts)
	r.GET("/v1/inventories/:inventory_id/groups/", inventories.Middleware, inventories.Groups)
	r.GET("/v1/inventories/:inventory_id/activity_stream/", inventories.Middleware, inventories.ActivityStream)
	r.GET("/v1/inventories/:inventory_id/inventory_sources/", inventories.Middleware, notImplemented) //TODO: implement

	// hosts
	r.GET("/v1/hosts/", hosts.GetHosts)
	r.POST("/v1/hosts/", hosts.AddHost)
	r.GET("/v1/hosts/:host_id/", hosts.Middleware, hosts.GetHost)
	r.PUT("/v1/hosts/:host_id/", hosts.Middleware, hosts.UpdateHost)
	r.PATCH("/v1/hosts/:host_id/", hosts.Middleware, hosts.PatchHost)
	r.DELETE("/v1/hosts/:host_id/", hosts.Middleware, hosts.RemoveHost)

	//related
	r.GET("/v1/hosts/:host_id/job_host_summaries/", hosts.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/hosts/:host_id/job_events/", hosts.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/hosts/:host_id/ad_hoc_commands/", hosts.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/hosts/:host_id/inventory_sources/", hosts.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/hosts/:host_id/activity_stream/", hosts.Middleware, hosts.ActivityStream)
	r.GET("/v1/hosts/:host_id/ad_hoc_command_events/", hosts.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/hosts/:host_id/variable_data/", hosts.Middleware, hosts.VariableData)
	r.GET("/v1/hosts/:host_id/groups/", hosts.Middleware, hosts.Groups)
	r.GET("/v1/hosts/:host_id/all_groups/", hosts.Middleware, hosts.AllGroups)

	// groups
	r.GET("/v1/groups/", groups.GetGroups)
	r.POST("/v1/groups/", groups.AddGroup)
	r.GET("/v1/groups/:group_id/", groups.Middleware, groups.GetGroup)
	r.PUT("/v1/groups/:group_id/", groups.Middleware, groups.UpdateGroup)
	r.PATCH("/v1/groups/:group_id/", groups.Middleware, groups.PatchGroup)
	r.DELETE("/v1/groups/:group_id/", groups.Middleware, groups.RemoveGroup)

	//related
	r.GET("/v1/groups/:group_id/variable_data/", groups.Middleware, groups.VariableData)
	r.GET("/v1/groups/:group_id/job_events/", groups.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/groups/:group_id/potential_children/", groups.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/groups/:group_id/ad_hoc_commands/", groups.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/groups/:group_id/all_hosts/", groups.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/groups/:group_id/activity_stream/", groups.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/groups/:group_id/hosts/", groups.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/groups/:group_id/children/", groups.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/groups/:group_id/job_host_summaries/", groups.Middleware, notImplemented) //TODO: implement

	// job_templates
	r.GET("/v1/job_templates/", jtemplate.GetJTemplates)
	r.POST("/v1/job_templates/", jtemplate.AddJTemplate)
	r.GET("/v1/job_templates/:job_template_id/", jtemplate.Middleware, jtemplate.GetJTemplate)
	r.PUT("/v1/job_templates/:job_template_id/", jtemplate.Middleware, jtemplate.UpdateJTemplate)
	r.PATCH("/v1/job_templates/:job_template_id/", jtemplate.Middleware, jtemplate.PatchJTemplate)
	r.DELETE("/v1/job_templates/:job_template_id/", jtemplate.Middleware, jtemplate.RemoveJTemplate)

	//related
	r.GET("/v1/job_templates/:job_template_id/jobs/", jtemplate.Middleware, jtemplate.Jobs)
	r.GET("/v1/job_templates/:job_template_id/object_roles/", jtemplate.Middleware, jtemplate.ObjectRoles)
	r.GET("/v1/job_templates/:job_template_id/access_list/", jtemplate.Middleware, jtemplate.AccessList)
	r.GET("/v1/job_templates/:job_template_id/launch/", jtemplate.Middleware, jtemplate.LaunchInfo)
	r.POST("/v1/job_templates/:job_template_id/launch/", jtemplate.Middleware, jtemplate.Launch)
	r.GET("/v1/job_templates/:job_template_id/activity_stream/", jtemplate.Middleware, jtemplate.ActivityStream)

	r.GET("/v1/job_templates/:job_template_id/schedules/", jtemplate.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/job_templates/:job_template_id/notification_templates_error/", jtemplate.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/job_templates/:job_template_id/notification_templates_success/", jtemplate.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/job_templates/:job_template_id/notification_templates_any/", jtemplate.Middleware, notImplemented) //TODO: implement

	// job
	r.GET("/v1/jobs/", jobs.GetJobs)
	r.GET("/v1/jobs/:job_id/", jobs.Middleware, jobs.GetJob)
	r.GET("/v1/jobs/:job_id/cancel/", jobs.Middleware, jobs.CancelInfo)
	r.POST("/v1/jobs/:job_id/cancel/", jobs.Middleware, jobs.Cancel)
	r.GET("/v1/jobs/:job_id/stdout/", jobs.Middleware, jobs.StdOut)

	r.GET("/v1/jobs/:job_id/job_tasks/", jobs.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/jobs/:job_id/job_plays/", jobs.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/jobs/:job_id/job_events/", jobs.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/jobs/:job_id/notifications/", jobs.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/jobs/:job_id/activity_stream/", jobs.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/jobs/:job_id/start/", jobs.Middleware, notImplemented) //TODO: implement
	r.GET("/v1/jobs/:job_id/relaunch/", jobs.Middleware, notImplemented) //TODO: implement
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
		Messages: []string{"Method not implemented"},
	})
}