package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/cors"
	"github.com/pearsonappeng/tensor/jwt"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/gin-gonic/gin.v1"
)

// Route defines all API endpoints
func Route(r *gin.Engine) {

	// Include CORS middleware to accept cross origin requests
	// Apply the middleware to the router (works with groups too)
	r.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "POST, GET, OPTIONS, PUT, PATCH, DELETE",
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

	r.GET("", util.GetAPIVersion)

	v1 := r.Group("v1")
	{
		v1.GET("/", util.GetAPIInfo)
		v1.GET("/ping", util.GetPing)
		v1.GET("/queue", QueueStats)
		v1.POST("/authtoken", jwt.HeaderAuthMiddleware.LoginHandler)

		// Include jwt authentication middleware
		v1.Use(jwt.HeaderAuthMiddleware.MiddlewareFunc())
		{
			dashboard := new(DashBoardController)
			users := new(UserController)

			v1.GET("/refresh_token", jwt.HeaderAuthMiddleware.RefreshHandler)
			v1.GET("/config", getSystemInfo)
			v1.GET("/dashboard", dashboard.GetInfo)
			v1.GET("/me", users.One)

			organizations := new(OrganizationController)
			// Organizations endpoints
			orgz := v1.Group("/organizations")
			{
				orgz.GET("", organizations.All)
				orgz.POST("", organizations.Create)

				org := orgz.Group("/:organization_id", organizations.Middleware)
				{
					org.GET("", organizations.One)
					org.PUT("", organizations.Update)
					org.PATCH("", organizations.Patch)
					org.DELETE("", organizations.Delete)

					// 'Organization' related endpoints
					org.GET("/users", organizations.GetUsers)
					org.GET("/inventories", organizations.GetInventories)
					org.GET("/activity_stream", organizations.ActivityStream)
					org.GET("/projects", organizations.GetProjects)
					org.GET("/admins", organizations.GetAdmins)
					org.GET("/teams", organizations.GetTeams)
					org.GET("/credentials", organizations.GetCredentials)

					org.GET("/notification_templates_error", notImplemented)   //TODO: implement
					org.GET("/notification_templates_success", notImplemented) //TODO: implement
					org.GET("/object_roles", notImplemented)                   //TODO: implement
					org.GET("/notification_templates", notImplemented)         //TODO: implement
					org.GET("/notification_templates_any", notImplemented)     //TODO: implement
					org.GET("/access_list", notImplemented)                    //TODO: implement
				}
			}

			usrs := v1.Group("/users")
			{
				// Users endpoints
				usrs.GET("", users.All)
				usrs.POST("", users.Create)

				usr := usrs.Group("/:user_id", users.Middleware)
				{
					usr.GET("", users.One)
					usr.PUT("", users.Update)
					usr.PATCH("", users.Patch)
					usr.DELETE("", users.Delete)
					// 'User' related endpoints
					usr.GET("/admin_of_organizations", users.AdminsOfOrganizations)
					usr.GET("/organizations", users.Organizations)
					usr.GET("/teams", users.Teams)
					usr.GET("/credentials", users.Credentials)
					usr.GET("/activity_stream", users.ActivityStream)
					usr.GET("/projects", users.Projects)
					usr.GET("/access_list", notImplemented) //TODO: implement

					usr.GET("/roles", users.GetRoles)
					usr.POST("/roles", users.AssignRole)
				}
			}

			prjcts := v1.Group("/projects")
			{
				projects := new(ProjectController)
				// Projects endpoints
				prjcts.GET("", projects.All)
				prjcts.POST("", projects.Create)

				prjct := prjcts.Group("/:project_id", projects.Middleware)
				{
					prjct.GET("", projects.One)
					prjct.PUT("", projects.Update)
					prjct.PATCH("", projects.Patch)
					prjct.DELETE("", projects.Delete)

					// 'Project' related endpoints
					prjct.GET("/activity_stream", projects.ActivityStream)
					prjct.GET("/teams", projects.OwnerTeams)
					prjct.GET("/playbooks", projects.Playbooks)
					prjct.GET("/access_list", projects.AccessList)
					prjct.GET("/update", projects.SCMUpdateInfo)
					prjct.POST("/update", projects.SCMUpdate)
					prjct.GET("/project_updates", projects.ProjectUpdates)

					prjct.GET("/schedules", notImplemented) //TODO: implement
				}
			}

			crdntls := v1.Group("/credentials")
			{
				credentials := new(CredentialController)
				// Credentials endpoints
				crdntls.GET("", credentials.All)
				crdntls.POST("", credentials.Create)

				crdntl := crdntls.Group("/:credential_id", credentials.Middleware)
				{
					crdntl.GET("", credentials.One)
					crdntl.PUT("", credentials.Update)
					crdntl.PATCH("", credentials.Patch)
					crdntl.DELETE("", credentials.Delete)

					// 'Credential' related endpoints
					crdntl.GET("/owner_teams", credentials.OwnerTeams)
					crdntl.GET("/owner_users", credentials.OwnerUsers)
					crdntl.GET("/activity_stream", credentials.ActivityStream)
					crdntl.GET("/access_list", notImplemented)  //TODO: implement
					crdntl.GET("/object_roles", notImplemented) //TODO: implement
				}
			}

			tms := v1.Group("/teams")
			{
				teams := new(TeamController)
				// Teams endpoints
				tms.GET("", teams.All)
				tms.POST("", teams.Create)

				tm := tms.Group("/:team_id", teams.Middleware)
				{
					tm.GET("", teams.One)
					tm.PUT("", teams.Update)
					tm.PATCH("", teams.Patch)
					tm.DELETE("", teams.Delete)

					// 'Team' related endpoints
					tm.GET("/users", teams.Users)
					tm.GET("/credentials", teams.Credentials)
					tm.GET("/projects", teams.Projects)
					tm.GET("/activity_stream", teams.ActivityStream)
					tm.GET("/access_list", teams.AccessList)

					tm.GET("/roles", teams.GetRoles)
					tm.POST("/roles", teams.AssignRole)
				}
			}

			invtrs := v1.Group("/inventories")
			{
				inventories := new(InventoryController)
				// Inventories endpoints
				invtrs.GET("", inventories.All)
				invtrs.POST("", inventories.Create)

				intr := invtrs.Group("/:inventory_id", inventories.Middleware)
				{
					intr.GET("", inventories.One)
					intr.PUT("", inventories.Update)
					intr.PATCH("", inventories.Patch)
					intr.DELETE("", inventories.Delete)
					intr.GET("/script", inventories.Script)

					// 'Inventory' related endpoints
					intr.GET("/job_templates", inventories.JobTemplates)
					intr.GET("/variable_data", inventories.VariableData)
					intr.GET("/root_groups", inventories.RootGroups)
					intr.GET("/ad_hoc_commands", notImplemented) //TODO: implement
					intr.GET("/tree", inventories.Tree)          //TODO: implement
					intr.GET("/access_list", inventories.AccessList)
					intr.GET("/hosts", inventories.Hosts)
					intr.GET("/groups", inventories.Groups)
					intr.GET("/activity_stream", inventories.ActivityStream)
					intr.GET("/inventory_sources", notImplemented) //TODO: implement
				}
			}

			hsts := v1.Group("/hosts")
			{
				hosts := new(HostController)
				// Hosts endpoints
				hsts.GET("", hosts.All)
				hsts.POST("", hosts.Create)

				hst := hsts.Group("/:host_id", hosts.Middleware)
				{
					hst.GET("", hosts.One)
					hst.PUT("", hosts.Update)
					hst.PATCH("", hosts.Patch)
					hst.DELETE("", hosts.Delete)

					// 'Host' related endpoints
					hst.GET("/job_host_summaries", notImplemented) //TODO: implement
					hst.GET("/job_events", notImplemented)         //TODO: implement
					hst.GET("/ad_hoc_commands", notImplemented)    //TODO: implement
					hst.GET("/inventory_sources", notImplemented)  //TODO: implement
					hst.GET("/activity_stream", hosts.ActivityStream)
					hst.GET("/ad_hoc_command_events", notImplemented) //TODO: implement
					hst.GET("/variable_data", hosts.VariableData)
					hst.GET("/groups", hosts.Groups)
					hst.GET("/all_groups", hosts.AllGroups)
				}
			}

			grps := v1.Group("/groups")
			{
				groups := new(GroupController)
				// Groups endpoints
				grps.GET("", groups.All)
				grps.POST("", groups.Create)

				grp := grps.Group("/:group_id", groups.Middleware)
				{
					grp.GET("", groups.One)
					grp.PUT("", groups.Update)
					grp.PATCH("", groups.Patch)
					grp.DELETE("", groups.Delete)

					// 'Group' related endpoints
					grp.GET("/variable_data", groups.VariableData)
					grp.GET("/job_events", notImplemented)         //TODO: implement
					grp.GET("/potential_children", notImplemented) //TODO: implement
					grp.GET("/ad_hoc_commands", notImplemented)    //TODO: implement
					grp.GET("/all_hosts", notImplemented)          //TODO: implement
					grp.GET("/activity_stream", groups.ActivityStream)
					grp.GET("/hosts", notImplemented)              //TODO: implement
					grp.GET("/children", notImplemented)           //TODO: implement
					grp.GET("/job_host_summaries", notImplemented) //TODO: implement
				}
			}

			ajtmps := v1.Group("/job_templates")
			{
				jtemplate := new(JobTemplateController)
				// Job Templates endpoints for Ansible
				ajtmps.GET("", jtemplate.All)
				ajtmps.POST("", jtemplate.Create)

				jtmp := ajtmps.Group("/:job_template_id", jtemplate.Middleware)
				{
					jtmp.GET("", jtemplate.One)
					jtmp.PUT("", jtemplate.Update)
					jtmp.PATCH("", jtemplate.Patch)
					jtmp.DELETE("", jtemplate.Delete)

					// 'Job Template' related endpoints
					jtmp.GET("/jobs", jtemplate.Jobs)
					jtmp.GET("/object_roles", jtemplate.ObjectRoles)
					jtmp.GET("/access_list", jtemplate.AccessList)
					jtmp.GET("/launch", jtemplate.LaunchInfo)
					jtmp.POST("/launch", jtemplate.Launch)
					jtmp.GET("/activity_stream", jtemplate.ActivityStream)

					jtmp.GET("/schedules", notImplemented)                      //TODO: implement
					jtmp.GET("/notification_templates_error", notImplemented)   //TODO: implement
					jtmp.GET("/notification_templates_success", notImplemented) //TODO: implement
					jtmp.GET("/notification_templates_any", notImplemented)     //TODO: implement
				}
			}

			ajbs := v1.Group("/jobs")
			{
				jobs := new(JobController)
				// Jobs endpoints for Ansible
				ajbs.GET("", jobs.All)

				jb := ajbs.Group("/:job_id", jobs.Middleware)
				{
					jb.GET("", jobs.One)
					jb.GET("/cancel", jobs.CancelInfo)
					jb.POST("/cancel", jobs.Cancel)
					jb.GET("/stdout", jobs.StdOut)

					// 'Job' related endpoints
					jb.GET("/job_tasks", notImplemented)       //TODO: implement
					jb.GET("/job_plays", notImplemented)       //TODO: implement
					jb.GET("/job_events", notImplemented)      //TODO: implement
					jb.GET("/notifications", notImplemented)   //TODO: implement
					jb.GET("/activity_stream", notImplemented) //TODO: implement
					jb.GET("/start", notImplemented)           //TODO: implement
					jb.GET("/relaunch", notImplemented)        //TODO: implement
				}
			}

			terraform := v1.Group("/terraform")
			{
				tjtmps := terraform.Group("/job_templates")
				{
					tjtemplate := new(TJobTmplController)
					// Job Templates endpoints for Terraform
					tjtmps.GET("", tjtemplate.All)
					tjtmps.POST("", tjtemplate.Create)

					jtmp := tjtmps.Group("/:job_template_id", tjtemplate.Middleware)
					{
						jtmp.GET("", tjtemplate.One)
						jtmp.PUT("", tjtemplate.Update)
						jtmp.PATCH("", tjtemplate.Patch)
						jtmp.DELETE("", tjtemplate.Delete)

						// 'Job Template' endpoints
						jtmp.GET("/jobs", tjtemplate.Jobs)
						jtmp.GET("/object_roles", tjtemplate.ObjectRoles)
						jtmp.GET("/access_list", tjtemplate.AccessList)
						jtmp.GET("/launch", tjtemplate.LaunchInfo)
						jtmp.POST("/launch", tjtemplate.Launch)
						jtmp.GET("/activity_stream", tjtemplate.ActivityStream)

						jtmp.GET("/schedules", notImplemented)                      //TODO: implement
						jtmp.GET("/notification_templates_error", notImplemented)   //TODO: implement
						jtmp.GET("/notification_templates_success", notImplemented) //TODO: implement
						jtmp.GET("/notification_templates_any", notImplemented)     //TODO: implement
					}
				}

				tjbs := terraform.Group("/jobs")
				{
					tjobs := new(TerraformJobController)
					// Jobs endpoints for Terraform
					tjbs.GET("", tjobs.All)

					jb := tjbs.Group("/:job_id", tjobs.Middleware)
					{
						jb.GET("", tjobs.One)
						jb.GET("/cancel", tjobs.CancelInfo)
						jb.POST("/cancel", tjobs.Cancel)
						jb.GET("/stdout", tjobs.StdOut)

						// 'TerraformJob' endpoints
						jb.GET("/notifications", notImplemented)   //TODO: implement
						jb.GET("/activity_stream", notImplemented) //TODO: implement
						jb.GET("/start", notImplemented)           //TODO: implement
						jb.GET("/relaunch", notImplemented)        //TODO: implement
					}
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
