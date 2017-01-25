package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/ansible/groups"
	"github.com/pearsonappeng/tensor/api/ansible/hosts"
	"github.com/pearsonappeng/tensor/api/ansible/inventories"
	"github.com/pearsonappeng/tensor/api/ansible/jobs"
	jtemplate "github.com/pearsonappeng/tensor/api/ansible/jtemplates"
	"github.com/pearsonappeng/tensor/api/common/credentials"
	"github.com/pearsonappeng/tensor/api/common/dashboard"
	"github.com/pearsonappeng/tensor/api/common/organizations"
	"github.com/pearsonappeng/tensor/api/common/projects"
	"github.com/pearsonappeng/tensor/api/common/teams"
	"github.com/pearsonappeng/tensor/api/common/users"
	"github.com/pearsonappeng/tensor/api/sockets"
	"github.com/pearsonappeng/tensor/cors"
	"github.com/pearsonappeng/tensor/jwt"
	"github.com/pearsonappeng/tensor/models/common"

	"github.com/gin-gonic/gin"
	"github.com/pearsonappeng/tensor/util"
)

// Route defines all API endpoints
func Route(r *gin.Engine) {

	// Include cors middleware to accept cross origin requests
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

	// Handle 404
	// this creates a response with standard error body
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, common.Error{
			Code:     http.StatusNotFound,
			Messages: []string{"Not found"},
		})
	})

	r.GET("", util.GetAPIVersion)

	v1 := r.Group("v1")
	{
		v1.GET("/", util.GetAPIInfo)
		v1.GET("/ping", util.GetPing)
		v1.POST("/authtoken", jwt.HeaderAuthMiddleware.LoginHandler)

		// Include jwt authentication middleware
		v1.Use(jwt.HeaderAuthMiddleware.MiddlewareFunc())
		{
			v1.GET("/refresh_token", jwt.HeaderAuthMiddleware.RefreshHandler)
			v1.GET("/config", getSystemInfo)
			v1.GET("/dashboard", dashboard.GetInfo)
			v1.GET("/ws", sockets.Handler)
			v1.GET("/me", users.GetUser)

			// Organizations endpoints
			orgz := v1.Group("/organizations")
			{
				orgz.GET("/", organizations.GetOrganizations)
				orgz.POST("/", organizations.AddOrganization)

				org := orgz.Group("/:organization_id", organizations.Middleware)
				{
					org.GET("/", organizations.GetOrganization)
					org.PUT("/", organizations.UpdateOrganization)
					org.PATCH("/", organizations.PatchOrganization)
					org.DELETE("/", organizations.RemoveOrganization)

					// 'Organization' related endpoints
					org.GET("/users/", organizations.GetUsers)
					org.GET("/inventories/", organizations.GetInventories)
					org.GET("/activity_stream/", organizations.ActivityStream)
					org.GET("/projects/", organizations.GetProjects)
					org.GET("/admins/", organizations.GetAdmins)
					org.GET("/teams/", organizations.GetTeams)
					org.GET("/credentials/", organizations.GetCredentials)

					org.GET("/notification_templates_error/", notImplemented)   //TODO: implement
					org.GET("/notification_templates_success/", notImplemented) //TODO: implement
					org.GET("/object_roles/", notImplemented)                   //TODO: implement
					org.GET("/notification_templates/", notImplemented)         //TODO: implement
					org.GET("/notification_templates_any/", notImplemented)     //TODO: implement
					org.GET("/access_list/", notImplemented)                    //TODO: implement
				}
			}

			usrs := v1.Group("/users")
			{
				// Users endpoints
				v1.GET("/users/", users.GetUsers)
				v1.POST("/users/", users.AddUser)

				usr := usrs.Group("/:user_id", users.Middleware)
				{
					usr.GET("/", users.GetUser)
					usr.PUT("/", users.UpdateUser)
					usr.DELETE("/", users.DeleteUser)
					// 'User' related endpoints
					usr.GET("/admin_of_organizations/", users.AdminsOfOrganizations)
					usr.GET("/organizations/", users.Organizations)
					usr.GET("/teams/", users.Teams)
					usr.GET("/credentials/", users.Credentials)
					usr.GET("/activity_stream/", users.ActivityStream)
					usr.GET("/projects/", users.Projects)

					usr.GET("/roles/", notImplemented)       //TODO: implement
					usr.GET("/access_list/", notImplemented) //TODO: implement
				}
			}
			// Projects endpoints
			r.GET("/projects/", projects.GetProjects)
			r.POST("/projects/", projects.AddProject)
			r.GET("/projects/:project_id/", projects.Middleware, projects.GetProject)
			r.PUT("/projects/:project_id/", projects.Middleware, projects.UpdateProject)
			r.PATCH("/projects/:project_id/", projects.Middleware, projects.PatchProject)
			r.DELETE("/projects/:project_id/", projects.Middleware, projects.RemoveProject)

			// 'Project' releated endpoints
			r.GET("/projects/:project_id/activity_stream/", projects.Middleware, projects.ActivityStream)
			r.GET("/projects/:project_id/teams/", projects.Middleware, projects.Teams)
			r.GET("/projects/:project_id/playbooks/", projects.Middleware, projects.Playbooks)
			r.GET("/projects/:project_id/access_list/", projects.Middleware, projects.AccessList)
			r.GET("/projects/:project_id/update/", projects.Middleware, projects.SCMUpdateInfo)
			r.POST("/projects/:project_id/update/", projects.Middleware, projects.SCMUpdate)
			r.GET("/projects/:project_id/project_updates/", projects.Middleware, projects.ProjectUpdates)

			r.GET("/projects/:project_id/schedules/", projects.Middleware, notImplemented) //TODO: implement

			// Credentials endpoints
			r.GET("/credentials/", credentials.GetCredentials)
			r.POST("/credentials/", credentials.AddCredential)
			r.GET("/credentials/:credential_id/", credentials.Middleware, credentials.GetCredential)
			r.PUT("/credentials/:credential_id/", credentials.Middleware, credentials.UpdateCredential)
			r.PATCH("/credentials/:credential_id/", credentials.Middleware, credentials.PatchCredential)
			r.DELETE("/credentials/:credential_id/", credentials.Middleware, credentials.RemoveCredential)

			// 'Credential' releated endpoints
			r.GET("/credentials/:credential_id/owner_teams/", credentials.Middleware, credentials.OwnerTeams)
			r.GET("/credentials/:credential_id/owner_users/", credentials.Middleware, credentials.OwnerUsers)
			r.GET("/credentials/:credential_id/activity_stream/", credentials.Middleware, credentials.ActivityStream)
			r.GET("/credentials/:credential_id/access_list/", credentials.Middleware, notImplemented)  //TODO: implement
			r.GET("/credentials/:credential_id/object_roles/", credentials.Middleware, notImplemented) //TODO: implement

			// Teams endpoints
			r.GET("/teams/", teams.GetTeams)
			r.POST("/teams/", teams.AddTeam)
			r.GET("/teams/:team_id/", teams.Middleware, teams.GetTeam)
			r.PUT("/teams/:team_id/", teams.Middleware, teams.UpdateTeam)
			r.PATCH("/teams/:team_id/", teams.Middleware, teams.PatchTeam)
			r.DELETE("/teams/:team_id/", teams.Middleware, teams.RemoveTeam)

			// 'Team' releated endpoints
			r.GET("/teams/:team_id/users/", teams.Middleware, teams.Users)
			r.GET("/teams/:team_id/credentials/", teams.Middleware, teams.Credentials)
			r.GET("/teams/:team_id/projects/", teams.Middleware, teams.Projects)
			r.GET("/teams/:team_id/activity_stream/", teams.Middleware, teams.ActivityStream)
			r.GET("/teams/:team_id/access_list/", teams.Middleware, teams.AccessList)

			// Inventories endpoints
			r.GET("/inventories/", inventories.GetInventories)
			r.POST("/inventories/", inventories.AddInventory)
			r.GET("/inventories/:inventory_id/", inventories.Middleware, inventories.GetInventory)
			r.PUT("/inventories/:inventory_id/", inventories.Middleware, inventories.UpdateInventory)
			r.PATCH("/inventories/:inventory_id/", inventories.Middleware, inventories.PatchInventory)
			r.DELETE("/inventories/:inventory_id/", inventories.Middleware, inventories.RemoveInventory)
			r.GET("/inventories/:inventory_id/script/", inventories.Middleware, inventories.Script)

			// 'Inventory' releated endpoints
			r.GET("/inventories/:inventory_id/job_templates/", inventories.Middleware, inventories.JobTemplates)
			r.GET("/inventories/:inventory_id/variable_data/", inventories.Middleware, inventories.VariableData)
			r.GET("/inventories/:inventory_id/root_groups/", inventories.Middleware, inventories.RootGroups)
			r.GET("/inventories/:inventory_id/ad_hoc_commands/", inventories.Middleware, notImplemented) //TODO: implement
			r.GET("/inventories/:inventory_id/tree/", inventories.Middleware, notImplemented)            //TODO: implement
			r.GET("/inventories/:inventory_id/access_list/", inventories.Middleware, inventories.AccessList)
			r.GET("/inventories/:inventory_id/hosts/", inventories.Middleware, inventories.Hosts)
			r.GET("/inventories/:inventory_id/groups/", inventories.Middleware, inventories.Groups)
			r.GET("/inventories/:inventory_id/activity_stream/", inventories.Middleware, inventories.ActivityStream)
			r.GET("/inventories/:inventory_id/inventory_sources/", inventories.Middleware, notImplemented) //TODO: implement

			// Hosts endpoints
			r.GET("/hosts/", hosts.GetHosts)
			r.POST("/hosts/", hosts.AddHost)
			r.GET("/hosts/:host_id/", hosts.Middleware, hosts.GetHost)
			r.PUT("/hosts/:host_id/", hosts.Middleware, hosts.UpdateHost)
			r.PATCH("/hosts/:host_id/", hosts.Middleware, hosts.PatchHost)
			r.DELETE("/hosts/:host_id/", hosts.Middleware, hosts.RemoveHost)

			// 'Host' releated endpoints
			r.GET("/hosts/:host_id/job_host_summaries/", hosts.Middleware, notImplemented) //TODO: implement
			r.GET("/hosts/:host_id/job_events/", hosts.Middleware, notImplemented)         //TODO: implement
			r.GET("/hosts/:host_id/ad_hoc_commands/", hosts.Middleware, notImplemented)    //TODO: implement
			r.GET("/hosts/:host_id/inventory_sources/", hosts.Middleware, notImplemented)  //TODO: implement
			r.GET("/hosts/:host_id/activity_stream/", hosts.Middleware, hosts.ActivityStream)
			r.GET("/hosts/:host_id/ad_hoc_command_events/", hosts.Middleware, notImplemented) //TODO: implement
			r.GET("/hosts/:host_id/variable_data/", hosts.Middleware, hosts.VariableData)
			r.GET("/hosts/:host_id/groups/", hosts.Middleware, hosts.Groups)
			r.GET("/hosts/:host_id/all_groups/", hosts.Middleware, hosts.AllGroups)

			// Groups endpoints
			r.GET("/groups/", groups.GetGroups)
			r.POST("/groups/", groups.AddGroup)
			r.GET("/groups/:group_id/", groups.Middleware, groups.GetGroup)
			r.PUT("/groups/:group_id/", groups.Middleware, groups.UpdateGroup)
			r.PATCH("/groups/:group_id/", groups.Middleware, groups.PatchGroup)
			r.DELETE("/groups/:group_id/", groups.Middleware, groups.RemoveGroup)

			// 'Group' related endpoints
			r.GET("/groups/:group_id/variable_data/", groups.Middleware, groups.VariableData)
			r.GET("/groups/:group_id/job_events/", groups.Middleware, notImplemented)         //TODO: implement
			r.GET("/groups/:group_id/potential_children/", groups.Middleware, notImplemented) //TODO: implement
			r.GET("/groups/:group_id/ad_hoc_commands/", groups.Middleware, notImplemented)    //TODO: implement
			r.GET("/groups/:group_id/all_hosts/", groups.Middleware, notImplemented)          //TODO: implement
			r.GET("/groups/:group_id/activity_stream/", groups.Middleware, notImplemented)    //TODO: implement
			r.GET("/groups/:group_id/hosts/", groups.Middleware, notImplemented)              //TODO: implement
			r.GET("/groups/:group_id/children/", groups.Middleware, notImplemented)           //TODO: implement
			r.GET("/groups/:group_id/job_host_summaries/", groups.Middleware, notImplemented) //TODO: implement

			// Job Templates endpoints
			r.GET("/job_templates/", jtemplate.GetJTemplates)
			r.POST("/job_templates/", jtemplate.AddJTemplate)
			r.GET("/job_templates/:job_template_id/", jtemplate.Middleware, jtemplate.GetJTemplate)
			r.PUT("/job_templates/:job_template_id/", jtemplate.Middleware, jtemplate.UpdateJTemplate)
			r.PATCH("/job_templates/:job_template_id/", jtemplate.Middleware, jtemplate.PatchJTemplate)
			r.DELETE("/job_templates/:job_template_id/", jtemplate.Middleware, jtemplate.RemoveJTemplate)

			// 'Job Template' releated endpoints
			r.GET("/job_templates/:job_template_id/jobs/", jtemplate.Middleware, jtemplate.Jobs)
			r.GET("/job_templates/:job_template_id/object_roles/", jtemplate.Middleware, jtemplate.ObjectRoles)
			r.GET("/job_templates/:job_template_id/access_list/", jtemplate.Middleware, jtemplate.AccessList)
			r.GET("/job_templates/:job_template_id/launch/", jtemplate.Middleware, jtemplate.LaunchInfo)
			r.POST("/job_templates/:job_template_id/launch/", jtemplate.Middleware, jtemplate.Launch)
			r.GET("/job_templates/:job_template_id/activity_stream/", jtemplate.Middleware, jtemplate.ActivityStream)

			r.GET("/job_templates/:job_template_id/schedules/", jtemplate.Middleware, notImplemented)                      //TODO: implement
			r.GET("/job_templates/:job_template_id/notification_templates_error/", jtemplate.Middleware, notImplemented)   //TODO: implement
			r.GET("/job_templates/:job_template_id/notification_templates_success/", jtemplate.Middleware, notImplemented) //TODO: implement
			r.GET("/job_templates/:job_template_id/notification_templates_any/", jtemplate.Middleware, notImplemented)     //TODO: implement

			// Jobs endpoints
			r.GET("/jobs/", jobs.GetJobs)
			r.GET("/jobs/:job_id/", jobs.Middleware, jobs.GetJob)
			r.GET("/jobs/:job_id/cancel/", jobs.Middleware, jobs.CancelInfo)
			r.POST("/jobs/:job_id/cancel/", jobs.Middleware, jobs.Cancel)
			r.GET("/jobs/:job_id/stdout/", jobs.Middleware, jobs.StdOut)

			// 'Job' releated endpoints
			r.GET("/jobs/:job_id/job_tasks/", jobs.Middleware, notImplemented)       //TODO: implement
			r.GET("/jobs/:job_id/job_plays/", jobs.Middleware, notImplemented)       //TODO: implement
			r.GET("/jobs/:job_id/job_events/", jobs.Middleware, notImplemented)      //TODO: implement
			r.GET("/jobs/:job_id/notifications/", jobs.Middleware, notImplemented)   //TODO: implement
			r.GET("/jobs/:job_id/activity_stream/", jobs.Middleware, notImplemented) //TODO: implement
			r.GET("/jobs/:job_id/start/", jobs.Middleware, notImplemented)           //TODO: implement
			r.GET("/jobs/:job_id/relaunch/", jobs.Middleware, notImplemented)        //TODO: implement
		}
	}
}

// getSystemInfo returns version and configuration information
// response includes,
// 	System version
// 	Configuration : database host, database name, database user,
// 				    project path, tensor executable path
func getSystemInfo(c *gin.Context) {
	body := map[string]interface{}{
		"version": util.Version,
		"config": map[string]string{
			"dbHost":       strings.Join(util.Config.MongoDB.Hosts, ","),
			"dbName":       util.Config.MongoDB.DbName,
			"dbUser":       util.Config.MongoDB.Username,
			"projectsPath": util.Config.ProjectsHome,
			"cmdPath":      util.FindTensor(),
		},
	}

	c.JSON(http.StatusOK, body)
}

// notImplemented create a response with Status Not Implemented (501)
// with standard error response body
func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, common.Error{
		Code:     http.StatusNotImplemented,
		Messages: []string{"Method not implemented"},
	})
}
