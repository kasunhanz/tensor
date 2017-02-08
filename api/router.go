package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/pearsonappeng/tensor/api/ansible/groups"
	"github.com/pearsonappeng/tensor/api/ansible/hosts"
	"github.com/pearsonappeng/tensor/api/ansible/inventories"
	"github.com/pearsonappeng/tensor/api/ansible/jobs"
	"github.com/pearsonappeng/tensor/api/ansible/jtemplates"
	"github.com/pearsonappeng/tensor/api/common/credentials"
	"github.com/pearsonappeng/tensor/api/common/dashboard"
	"github.com/pearsonappeng/tensor/api/common/organizations"
	"github.com/pearsonappeng/tensor/api/common/projects"
	"github.com/pearsonappeng/tensor/api/common/teams"
	"github.com/pearsonappeng/tensor/api/common/users"
	"github.com/pearsonappeng/tensor/api/sockets"
	tjobs "github.com/pearsonappeng/tensor/api/terraform/jobs"
	tjtemplate "github.com/pearsonappeng/tensor/api/terraform/jtemplates"
	"github.com/pearsonappeng/tensor/cors"
	"github.com/pearsonappeng/tensor/jwt"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/util"
	"gopkg.in/gin-gonic/gin.v1"
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
		v1.GET("/queue", QueueStats)
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
				usrs.GET("/", users.GetUsers)
				usrs.POST("/", users.AddUser)

				usr := usrs.Group("/:user_id", users.Middleware)
				{
					usr.GET("/", users.GetUser)
					usr.PUT("/", users.UpdateUser)
					usr.PATCH("/", users.PatchUser)
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

			prjcts := v1.Group("/projects")
			{
				// Projects endpoints
				prjcts.GET("/", projects.GetProjects)
				prjcts.POST("/", projects.AddProject)

				prjct := prjcts.Group("/:project_id", projects.Middleware)
				{
					prjct.GET("/", projects.GetProject)
					prjct.PUT("/", projects.UpdateProject)
					prjct.PATCH("/", projects.PatchProject)
					prjct.DELETE("/", projects.RemoveProject)

					// 'Project' releated endpoints
					prjct.GET("/activity_stream/", projects.ActivityStream)
					prjct.GET("/teams/", projects.Teams)
					prjct.GET("/playbooks/", projects.Playbooks)
					prjct.GET("/access_list/", projects.AccessList)
					prjct.GET("/update/", projects.SCMUpdateInfo)
					prjct.POST("/update/", projects.SCMUpdate)
					prjct.GET("/project_updates/", projects.ProjectUpdates)

					prjct.GET("/schedules/", notImplemented) //TODO: implement
				}
			}

			crdntls := v1.Group("/credentials")
			{
				// Credentials endpoints
				crdntls.GET("/", credentials.GetCredentials)
				crdntls.POST("/", credentials.AddCredential)

				crdntl := crdntls.Group(":credential_id", credentials.Middleware)
				{
					crdntl.GET("/", credentials.GetCredential)
					crdntl.PUT("/", credentials.UpdateCredential)
					crdntl.PATCH("/", credentials.PatchCredential)
					crdntl.DELETE("/", credentials.RemoveCredential)

					// 'Credential' releated endpoints
					crdntl.GET("/owner_teams/", credentials.OwnerTeams)
					crdntl.GET("/owner_users/", credentials.OwnerUsers)
					crdntl.GET("/activity_stream/", credentials.ActivityStream)
					crdntl.GET("/access_list/", notImplemented)  //TODO: implement
					crdntl.GET("/object_roles/", notImplemented) //TODO: implement
				}
			}

			tms := v1.Group("/teams")
			{
				// Teams endpoints
				tms.GET("/", teams.GetTeams)
				tms.POST("/", teams.AddTeam)

				tm := tms.Group("/:team_id", teams.Middleware)
				{
					tm.GET("/", teams.GetTeam)
					tm.PUT("/", teams.UpdateTeam)
					tm.PATCH("/", teams.PatchTeam)
					tm.DELETE("/", teams.RemoveTeam)

					// 'Team' releated endpoints
					tm.GET("/users/", teams.Users)
					tm.GET("/credentials/", teams.Credentials)
					tm.GET("/projects/", teams.Projects)
					tm.GET("/activity_stream/", teams.ActivityStream)
					tm.GET("/access_list/", teams.AccessList)
				}
			}

			invtrs := v1.Group("/inventories")
			{
				// Inventories endpoints
				invtrs.GET("/", inventories.GetInventories)
				invtrs.POST("/", inventories.AddInventory)

				intr := invtrs.Group("/:inventory_id", inventories.Middleware)
				{
					intr.GET("/", inventories.GetInventory)
					intr.PUT("/", inventories.UpdateInventory)
					intr.PATCH("/", inventories.PatchInventory)
					intr.DELETE("/", inventories.RemoveInventory)
					intr.GET("/script/", inventories.Script)

					// 'Inventory' releated endpoints
					intr.GET("/job_templates/", inventories.JobTemplates)
					intr.GET("/variable_data/", inventories.VariableData)
					intr.GET("/root_groups/", inventories.RootGroups)
					intr.GET("/ad_hoc_commands/", notImplemented) //TODO: implement
					intr.GET("/tree/", inventories.Tree)          //TODO: implement
					intr.GET("/access_list/", inventories.AccessList)
					intr.GET("/hosts/", inventories.Hosts)
					intr.GET("/groups/", inventories.Groups)
					intr.GET("/activity_stream/", inventories.ActivityStream)
					intr.GET("/inventory_sources/", notImplemented) //TODO: implement
				}
			}

			hsts := v1.Group("/hosts")
			{
				// Hosts endpoints
				hsts.GET("/", hosts.GetHosts)
				hsts.POST("/", hosts.AddHost)

				hst := hsts.Group("/:host_id", hosts.Middleware)
				{
					hst.GET("/", hosts.GetHost)
					hst.PUT("/", hosts.UpdateHost)
					hst.PATCH("/", hosts.PatchHost)
					hst.DELETE("/", hosts.RemoveHost)

					// 'Host' releated endpoints
					hst.GET("/job_host_summaries/", notImplemented) //TODO: implement
					hst.GET("/job_events/", notImplemented)         //TODO: implement
					hst.GET("/ad_hoc_commands/", notImplemented)    //TODO: implement
					hst.GET("/inventory_sources/", notImplemented)  //TODO: implement
					hst.GET("/activity_stream/", hosts.ActivityStream)
					hst.GET("/ad_hoc_command_events/", notImplemented) //TODO: implement
					hst.GET("/variable_data/", hosts.VariableData)
					hst.GET("/groups/", hosts.Groups)
					hst.GET("/all_groups/", hosts.AllGroups)
				}
			}

			grps := v1.Group("/groups")
			{
				// Groups endpoints
				grps.GET("/", groups.GetGroups)
				grps.POST("/", groups.AddGroup)

				grp := grps.Group("/:group_id", groups.Middleware)
				{
					grp.GET("/", groups.GetGroup)
					grp.PUT("/", groups.UpdateGroup)
					grp.PATCH("/", groups.PatchGroup)
					grp.DELETE("/", groups.RemoveGroup)

					// 'Group' related endpoints
					grp.GET("/variable_data/", groups.VariableData)
					grp.GET("/job_events/", notImplemented)         //TODO: implement
					grp.GET("/potential_children/", notImplemented) //TODO: implement
					grp.GET("/ad_hoc_commands/", notImplemented)    //TODO: implement
					grp.GET("/all_hosts/", notImplemented)          //TODO: implement
					grp.GET("/activity_stream/", groups.ActivityStream)
					grp.GET("/hosts/", notImplemented)              //TODO: implement
					grp.GET("/children/", notImplemented)           //TODO: implement
					grp.GET("/job_host_summaries/", notImplemented) //TODO: implement
				}
			}

			ajtmps := v1.Group("/job_templates")
			{
				// Job Templates endpoints for Ansible
				ajtmps.GET("/", jtemplate.GetJTemplates)
				ajtmps.POST("/", jtemplate.AddJTemplate)

				jtmp := ajtmps.Group("/:job_template_id", jtemplate.Middleware)
				{
					jtmp.GET("/", jtemplate.GetJTemplate)
					jtmp.PUT("/", jtemplate.UpdateJTemplate)
					jtmp.PATCH("/", jtemplate.PatchJTemplate)
					jtmp.DELETE("/", jtemplate.RemoveJTemplate)

					// 'Job Template' releated endpoints
					jtmp.GET("/jobs/", jtemplate.Jobs)
					jtmp.GET("/object_roles/", jtemplate.ObjectRoles)
					jtmp.GET("/access_list/", jtemplate.AccessList)
					jtmp.GET("/launch/", jtemplate.LaunchInfo)
					jtmp.POST("/launch/", jtemplate.Launch)
					jtmp.GET("/activity_stream/", jtemplate.ActivityStream)

					jtmp.GET("/schedules/", notImplemented)                      //TODO: implement
					jtmp.GET("/notification_templates_error/", notImplemented)   //TODO: implement
					jtmp.GET("/notification_templates_success/", notImplemented) //TODO: implement
					jtmp.GET("/notification_templates_any/", notImplemented)     //TODO: implement
				}
			}

			ajbs := v1.Group("/jobs")
			{
				// Jobs endpoints for Ansible
				ajbs.GET("/", jobs.GetJobs)

				jb := ajbs.Group("/:job_id", jobs.Middleware)
				{
					jb.GET("/", jobs.GetJob)
					jb.GET("/cancel/", jobs.CancelInfo)
					jb.POST("/cancel/", jobs.Cancel)
					jb.GET("/stdout/", jobs.StdOut)

					// 'Job' releated endpoints
					jb.GET("/job_tasks/", notImplemented)       //TODO: implement
					jb.GET("/job_plays/", notImplemented)       //TODO: implement
					jb.GET("/job_events/", notImplemented)      //TODO: implement
					jb.GET("/notifications/", notImplemented)   //TODO: implement
					jb.GET("/activity_stream/", notImplemented) //TODO: implement
					jb.GET("/start/", notImplemented)           //TODO: implement
					jb.GET("/relaunch/", notImplemented)        //TODO: implement
				}
			}

			terraform := v1.Group("/terraform")
			{
				tjtmps := terraform.Group("/job_templates/")
				{
					// Job Templates endpoints for Terraform
					tjtmps.GET("/", tjtemplate.GetJTemplates)
					tjtmps.POST("/", tjtemplate.AddJTemplate)

					jtmp := tjtmps.Group("/:job_template_id", tjtemplate.Middleware)
					{
						jtmp.GET("/", tjtemplate.GetJTemplate)
						jtmp.PUT("/", tjtemplate.UpdateJTemplate)
						jtmp.PATCH("/", tjtemplate.PatchJTemplate)
						jtmp.DELETE("/", tjtemplate.RemoveJTemplate)

						// 'Job Template' endpoints
						jtmp.GET("/jobs/", tjtemplate.Jobs)
						jtmp.GET("/object_roles/", tjtemplate.ObjectRoles)
						jtmp.GET("/access_list/", tjtemplate.AccessList)
						jtmp.GET("/launch/", tjtemplate.LaunchInfo)
						jtmp.POST("/launch/", tjtemplate.Launch)
						jtmp.GET("/activity_stream/", tjtemplate.ActivityStream)

						jtmp.GET("/schedules/", notImplemented)                      //TODO: implement
						jtmp.GET("/notification_templates_error/", notImplemented)   //TODO: implement
						jtmp.GET("/notification_templates_success/", notImplemented) //TODO: implement
						jtmp.GET("/notification_templates_any/", notImplemented)     //TODO: implement
					}
				}

				tjbs := terraform.Group("/jobs")
				{
					// Jobs endpoints for Terraform
					tjbs.GET("/", tjobs.GetJobs)

					jb := tjbs.Group("/:job_id", tjobs.Middleware)
					{
						jb.GET("/", tjobs.GetJob)
						jb.GET("/cancel/", tjobs.CancelInfo)
						jb.POST("/cancel/", tjobs.Cancel)
						jb.GET("/stdout/", tjobs.StdOut)

						// 'TerraformJob' endpoints
						jb.GET("/notifications/", notImplemented)   //TODO: implement
						jb.GET("/activity_stream/", notImplemented) //TODO: implement
						jb.GET("/start/", notImplemented)           //TODO: implement
						jb.GET("/relaunch/", notImplemented)        //TODO: implement
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
