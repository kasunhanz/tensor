package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/cors"
	"github.com/pearsonappeng/tensor/jwt"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/util"
	"github.com/gin-gonic/gin"
)

// Route defines all API endpoints
func Route(r *gin.Engine) {
	// Include CORS middleware to accept cross origin requests
	// Apply the middleware to the router (works with groups too)
	r.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "POST, GET, OPTIONS, PUT, DELETE",
		RequestHeaders:  "X-Requested-With, Content-Type, Origin, Authorization, Accept, Client-Security-Token, Accept-Encoding, x-access-token",
		ExposedHeaders:  "Content-Length",
		MaxAge:          86400 * time.Second,
		Credentials:     true,
		ValidateHeaders: false,
	}))
	// Handle 404
	// this creates a response with standard error body
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, common.Error{
			Code:   http.StatusNotFound,
			Errors: []string{"Not found"},
		})
	})
	r.GET("", GetAPIVersion)
	v1 := r.Group("v1")
	{
		v1.GET("/", GetAPIInfo)
		v1.GET("/ping", GetPing)
		v1.GET("/queue", QueueStats)
		v1.POST("/authtoken", jwt.HeaderAuthMiddleware.LoginHandler)

		v1.Use(jwt.HeaderAuthMiddleware.MiddlewareFunc())
		{
			dashboard := new(DashBoardController)
			users := new(UserController)
			v1.GET("/refresh_token", jwt.HeaderAuthMiddleware.RefreshHandler)
			v1.GET("/config", getSystemInfo)
			v1.GET("/dashboard", dashboard.GetInfo)
			v1.GET("/me", users.One)

			organizations := v1.Group("/organizations")
			{
				ctrl := new(OrganizationController)
				organizations.GET("", ctrl.All)
				organizations.POST("", ctrl.Create)
				organization := organizations.Group("/:organization_id", ctrl.Middleware)
				{
					organization.GET("", ctrl.One)
					organization.PUT("", ctrl.Update)
					organization.DELETE("", ctrl.Delete)
					organization.GET("/users", ctrl.GetUsers)
					organization.GET("/inventories", ctrl.GetInventories)
					organization.GET("/activity_stream", ctrl.ActivityStream)
					organization.GET("/projects", ctrl.GetProjects)
					organization.GET("/admins", ctrl.GetAdmins)
					organization.GET("/teams", ctrl.GetTeams)
					organization.GET("/credentials", ctrl.GetCredentials)
					organization.GET("/object_roles", ctrl.ObjectRoles)
					organization.GET("/access_list", notImplemented)                    //TODO: implement
					organization.GET("/notification_templates_error", notImplemented)   //TODO: implement
					organization.GET("/notification_templates_success", notImplemented) //TODO: implement
					organization.GET("/notification_templates", notImplemented)         //TODO: implement
					organization.GET("/notification_templates_any", notImplemented)     //TODO: implement
				}
			}

			groupUsers := v1.Group("/users")
			{
				groupUsers.GET("", users.All)
				groupUsers.POST("", users.Create)
				user := groupUsers.Group("/:user_id", users.Middleware)
				{
					user.GET("", users.One)
					user.PUT("", users.Update)
					user.DELETE("", users.Delete)
					user.GET("/admin_of_organizations", users.AdminsOfOrganizations)
					user.GET("/organizations", users.Organizations)
					user.GET("/teams", users.Teams)
					user.GET("/credentials", users.Credentials)
					user.GET("/activity_stream", users.ActivityStream)
					user.GET("/projects", users.Projects)
					user.GET("/roles", users.GetRoles)
					user.POST("/roles", users.AssignRole)
					user.GET("/access_list", notImplemented) //TODO: implement
				}
			}

			projects := v1.Group("/projects")
			{
				ctrl := new(ProjectController)
				projects.GET("", ctrl.All)
				projects.POST("", ctrl.Create)
				project := projects.Group("/:project_id", ctrl.Middleware)
				{
					project.GET("", ctrl.One)
					project.PUT("", ctrl.Update)
					project.DELETE("", ctrl.Delete)
					project.GET("/activity_stream", ctrl.ActivityStream)
					project.GET("/teams", ctrl.OwnerTeams)
					project.GET("/playbooks", ctrl.Playbooks)
					project.GET("/access_list", ctrl.AccessList)
					project.GET("/update", ctrl.SCMUpdateInfo)
					project.POST("/update", ctrl.SCMUpdate)
					project.GET("/project_updates", ctrl.ProjectUpdates)
					project.GET("/object_roles", ctrl.ObjectRoles)
					project.GET("/schedules", notImplemented) //TODO: implement
				}
			}

			credentials := v1.Group("/credentials")
			{
				ctrl := new(CredentialController)
				credentials.GET("", ctrl.All)
				credentials.POST("", ctrl.Create)
				credential := credentials.Group("/:credential_id", ctrl.Middleware)
				{
					credential.GET("", ctrl.One)
					credential.PUT("", ctrl.Update)
					credential.DELETE("", ctrl.Delete)
					credential.GET("/owner_teams", ctrl.OwnerTeams)
					credential.GET("/owner_users", ctrl.OwnerUsers)
					credential.GET("/activity_stream", ctrl.ActivityStream)
					credential.GET("/object_roles", ctrl.ObjectRoles)
					credential.GET("/access_list", notImplemented) //TODO: implement
				}
			}

			teams := v1.Group("/teams")
			{
				ctrl := new(TeamController)
				teams.GET("", ctrl.All)
				teams.POST("", ctrl.Create)
				team := teams.Group("/:team_id", ctrl.Middleware)
				{
					team.GET("", ctrl.One)
					team.PUT("", ctrl.Update)
					team.DELETE("", ctrl.Delete)
					team.GET("/users", ctrl.Users)
					team.GET("/credentials", ctrl.Credentials)
					team.GET("/projects", ctrl.Projects)
					team.GET("/activity_stream", ctrl.ActivityStream)
					team.GET("/access_list", ctrl.AccessList)
					team.GET("/roles", ctrl.GetRoles)
					team.POST("/roles", ctrl.AssignRole)
				}
			}

			inventories := v1.Group("/inventories")
			{
				ctrl := new(InventoryController)
				inventories.GET("", ctrl.All)
				inventories.POST("", ctrl.Create)
				inventory := inventories.Group("/:inventory_id", ctrl.Middleware)
				{
					inventory.GET("", ctrl.One)
					inventory.PUT("", ctrl.Update)
					inventory.DELETE("", ctrl.Delete)
					inventory.GET("/script", ctrl.Script)
					inventory.GET("/job_templates", ctrl.JobTemplates)
					inventory.GET("/variable_data", ctrl.VariableData)
					inventory.GET("/root_groups", ctrl.RootGroups)
					inventory.GET("/access_list", ctrl.AccessList)
					inventory.GET("/hosts", ctrl.Hosts)
					inventory.GET("/groups", ctrl.Groups)
					inventory.GET("/activity_stream", ctrl.ActivityStream)
					inventory.GET("/object_roles", ctrl.ObjectRoles)
					inventory.GET("/tree", ctrl.Tree)                   //TODO: implement
					inventory.GET("/inventory_sources", notImplemented) //TODO: implement
				}
			}

			hosts := v1.Group("/hosts")
			{
				ctrl := new(HostController)
				hosts.GET("", ctrl.All)
				hosts.POST("", ctrl.Create)
				host := hosts.Group("/:host_id", ctrl.Middleware)
				{
					host.GET("", ctrl.One)
					host.PUT("", ctrl.Update)
					host.DELETE("", ctrl.Delete)
					host.GET("/activity_stream", ctrl.ActivityStream)
					host.GET("/variable_data", ctrl.VariableData)
					host.GET("/groups", ctrl.Groups)
					host.GET("/all_groups", ctrl.AllGroups)
					host.GET("/job_host_summaries", notImplemented) //TODO: implement
					host.GET("/job_events", notImplemented)         //TODO: implement
					host.GET("/inventory_sources", notImplemented)  //TODO: implement
				}
			}

			groups := v1.Group("/groups")
			{
				ctrl := new(GroupController)
				groups.GET("", ctrl.All)
				groups.POST("", ctrl.Create)
				group := groups.Group("/:group_id", ctrl.Middleware)
				{
					group.GET("", ctrl.One)
					group.PUT("", ctrl.Update)
					group.DELETE("", ctrl.Delete)
					group.GET("/variable_data", ctrl.VariableData)
					group.GET("/activity_stream", ctrl.ActivityStream)
					group.GET("/job_events", notImplemented)         //TODO: implement
					group.GET("/potential_children", notImplemented) //TODO: implement
					group.GET("/all_hosts", notImplemented)          //TODO: implement
					group.GET("/hosts", notImplemented)              //TODO: implement
					group.GET("/children", notImplemented)           //TODO: implement
					group.GET("/job_host_summaries", notImplemented) //TODO: implement
				}
			}

			ansibleTemplates := v1.Group("/job_templates")
			{
				ctrl := new(JobTemplateController)
				ansibleTemplates.GET("", ctrl.All)
				ansibleTemplates.POST("", ctrl.Create)
				template := ansibleTemplates.Group("/:job_template_id", ctrl.Middleware)
				{
					template.GET("", ctrl.One)
					template.PUT("", ctrl.Update)
					template.DELETE("", ctrl.Delete)
					template.GET("/jobs", ctrl.Jobs)
					template.GET("/object_roles", ctrl.ObjectRoles)
					template.GET("/access_list", ctrl.AccessList)
					template.GET("/launch", ctrl.LaunchInfo)
					template.POST("/launch", ctrl.Launch)
					template.GET("/activity_stream", ctrl.ActivityStream)
					template.GET("/schedules", notImplemented)                      //TODO: implement
					template.GET("/notification_templates_error", notImplemented)   //TODO: implement
					template.GET("/notification_templates_success", notImplemented) //TODO: implement
					template.GET("/notification_templates_any", notImplemented)     //TODO: implement
				}
			}

			ansibleJobs := v1.Group("/jobs")
			{
				ctrl := new(JobController)
				ansibleJobs.GET("", ctrl.All)
				job := ansibleJobs.Group("/:job_id", ctrl.Middleware)
				{
					job.GET("", ctrl.One)
					job.GET("/cancel", ctrl.CancelInfo) //TODO: implement
					job.POST("/cancel", ctrl.Cancel)    //TODO: implement
					job.GET("/stdout", ctrl.StdOut)
					job.GET("/job_tasks", notImplemented)       //TODO: implement
					job.GET("/job_plays", notImplemented)       //TODO: implement
					job.GET("/job_events", notImplemented)      //TODO: implement
					job.GET("/notifications", notImplemented)   //TODO: implement
					job.GET("/activity_stream", notImplemented) //TODO: implement
					job.GET("/start", notImplemented)           //TODO: implement
					job.GET("/relaunch", notImplemented)        //TODO: implement
				}
			}

			terraformTemplate := v1.Group("/terraform_job_templates")
			{
				ctrl := new(TJobTmplController)
				terraformTemplate.GET("", ctrl.All)
				terraformTemplate.POST("", ctrl.Create)
				template := terraformTemplate.Group("/:terraform_job_template_id", ctrl.Middleware)
				{
					template.GET("", ctrl.One)
					template.PUT("", ctrl.Update)
					template.DELETE("", ctrl.Delete)
					template.GET("/jobs", ctrl.Jobs)
					template.GET("/access_list", ctrl.AccessList)
					template.GET("/launch", ctrl.LaunchInfo)
					template.POST("/launch", ctrl.Launch)
					template.GET("/activity_stream", ctrl.ActivityStream)
					template.GET("/object_roles", ctrl.ObjectRoles)
					template.GET("/schedules", notImplemented)                      //TODO: implement
					template.GET("/notification_templates_error", notImplemented)   //TODO: implement
					template.GET("/notification_templates_success", notImplemented) //TODO: implement
					template.GET("/notification_templates_any", notImplemented)     //TODO: implement
				}
			}

			terraformJobs := v1.Group("/terraform_jobs")
			{
				ctrl := new(TerraformJobController)
				terraformJobs.GET("", ctrl.All)
				job := terraformJobs.Group("/:terraform_job_id", ctrl.Middleware)
				{
					job.GET("", ctrl.One)
					job.GET("/cancel", ctrl.CancelInfo)
					job.POST("/cancel", ctrl.Cancel)
					job.GET("/stdout", ctrl.StdOut)
					job.GET("/notifications", notImplemented)   //TODO: implement
					job.GET("/activity_stream", notImplemented) //TODO: implement
					job.GET("/start", notImplemented)           //TODO: implement
					job.GET("/relaunch", notImplemented)        //TODO: implement
				}
			}
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
		},
	}

	c.JSON(http.StatusOK, body)
}

// notImplemented create a response with Status Not Implemented (501)
// with standard error response body
func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, common.Error{
		Code:   http.StatusNotImplemented,
		Errors: []string{"Method not implemented"},
	})
}
