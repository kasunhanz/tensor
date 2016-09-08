package api

import (
	"time"
	"github.com/gin-gonic/gin"
	"strings"
	"bitbucket.pearson.com/apseng/tensor/api/access"
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

	/*r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, "PONG")
	})
	*/
	r.GET("", util.GetAPIVersion)
	// set up the namespace
	//future refrence: api := r.Group("/api")
	v1 := r.Group("/v1")
	{
		v1.GET("/", util.GetAPIInfo)
		v1.GET("/ping", util.GetPing)

		auth := v1.Group("/auth")
		{
			auth.POST("/login", access.Login)
			auth.POST("/logout", access.Logout)
		}

		//authenticated user
		v1.Use(access.Authentication)

		v1.GET("/config", getSystemInfo)
		v1.GET("/dashboard", dashboard.GetInfo)

		v1.GET("/ws", sockets.Handler)

		v1.GET("/me", users.GetUser)

		user := v1.Group("/user")
		{
			user.GET("", users.GetUser)
			// api.PUT("/user", misc.UpdateUser)
			user.GET("/tokens", getAPITokens)
			user.POST("/tokens", createAPIToken)
			user.DELETE("/tokens/:token_id", expireAPIToken)
		}

		v1.GET("/events", getEvents)

		//organizations
		v1.GET("/organizations", organizations.GetOrganizations)
		v1.POST("/organizations", organizations.AddOrganization)
		v1.GET("/organizations/:organization_id", organizations.OrganizationMiddleware, organizations.GetOrganization)
		v1.PUT("/organizations/:organization_id", organizations.OrganizationMiddleware, organizations.UpdateOrganization)

		//users
		v1.GET("/users", users.GetUsers)
		v1.POST("/users", users.AddUser)
		v1.GET("/users/:user_id", users.GetUserMiddleware, users.GetUser)
		v1.PUT("/users/:user_id", users.GetUserMiddleware, users.UpdateUser)

		v1.POST("/users/:user_id/password", users.GetUserMiddleware, users.UpdateUserPassword)
		v1.DELETE("/users/:user_id", users.GetUserMiddleware, users.DeleteUser)

		//projects
		v1.GET("/projects", projects.GetProjects)
		v1.POST("/projects", projects.AddProject)
		v1.GET("/projects/:project_id", projects.ProjectMiddleware, projects.GetProject)
		v1.PUT("/projects/:project_id", projects.ProjectMiddleware, projects.GetProject)


		//credentials
		v1.GET("/credentials", credentials.GetCredentials)
		v1.POST("/credentials", credentials.AddCredential)
		v1.GET("/credentials/:credential_id", credentials.CredentialMiddleware, credentials.GetCredential)
		v1.PUT("/credentials/:credential_id", credentials.CredentialMiddleware, credentials.UpdateCredential)
		v1.DELETE("/credentials/:credential_id", credentials.CredentialMiddleware, credentials.RemoveCredential)

		//teams
		v1.GET("/teams", teams.GetTeams)
		v1.POST("/teams", teams.AddTeam)
		v1.GET("/teams/:team_id", teams.TeamMiddleware, teams.GetTeam)
		v1.PUT("/teams/:team_id", teams.TeamMiddleware, teams.UpdateTeam)
		v1.DELETE("/teams/:team_id", teams.TeamMiddleware, teams.RemoveTeam)

		//inventories
		v1.GET("/inventories", inventories.GetInventories)
		v1.POST("/inventories", inventories.AddInventory)
		v1.GET("/inventories/:inventory_id", inventories.InventoryMiddleware, inventories.GetInventory)
		v1.PUT("/inventories/:inventory_id", inventories.InventoryMiddleware, inventories.UpdateInventory)
		v1.DELETE("/inventories/:inventory_id", inventories.InventoryMiddleware, inventories.RemoveInventory)

		/*	p := v1.Group("/project/:project_id")
			{
				p.Use(projects.ProjectMiddleware)

				p.GET("", projects.GetProject)

				p.GET("/events", getEvents)
				p.GET("/tasks", tasks.GetAll)
				p.POST("/tasks", tasks.AddTask)
				p.GET("/tasks/:task_id/output", tasks.GetTaskMiddleware, tasks.GetTaskOutput)
			}

			at := v1.Group("/addhoc")
			{
				at.POST("/tasks", addhoctasks.AddTask)
				at.GET("/tasks/:task_id", addhoctasks.GetTaskWithoutLogMiddleware, addhoctasks.GetTaskWithoutLog)
				at.GET("/tasks/:task_id/log", addhoctasks.GetTaskMiddleware, addhoctasks.GetTaskOutput)
			}
	*/
		k := v1.Group("access")
		{
			k.GET("/keys", access.GetKeys)
			k.POST("/keys", access.AddKey)
			k.PUT("/keys/:key_id", access.KeyMiddleware, access.UpdateKey)
			k.DELETE("/keys/:key_id", access.KeyMiddleware, access.RemoveKey)
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

	c.JSON(200, body)
}
