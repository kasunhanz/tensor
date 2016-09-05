package api

import (
	"time"
	"github.com/gin-gonic/gin"
	"strings"
	"bitbucket.pearson.com/apseng/tensor/api/access"
	"bitbucket.pearson.com/apseng/tensor/api/addhoctasks"
	"bitbucket.pearson.com/apseng/tensor/api/cors"
	"bitbucket.pearson.com/apseng/tensor/api/projects"
	"bitbucket.pearson.com/apseng/tensor/api/sockets"
	"bitbucket.pearson.com/apseng/tensor/util"
	"bitbucket.pearson.com/apseng/tensor/api/organizations"
	"bitbucket.pearson.com/apseng/tensor/api/tasks"
	"bitbucket.pearson.com/apseng/tensor/api/credential"
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

		auth := v1.Group("/auth")
		{
			auth.POST("/login", access.Login)
			auth.POST("/logout", access.Logout)
		}

		//authenticated user
		v1.Use(access.Authentication)

		v1.GET("/ws", sockets.Handler)

		v1.GET("/info", getSystemInfo)

		v1.GET("/me", getUser)

		user := v1.Group("/user")
		{
			user.GET("", getUser)
			// api.PUT("/user", misc.UpdateUser)
			user.GET("/tokens", getAPITokens)
			user.POST("/tokens", createAPIToken)
			user.DELETE("/tokens/:token_id", expireAPIToken)
		}

		v1.GET("/events", getEvents)

		//users
		v1.GET("/users", getUsers)
		v1.POST("/users", addUser)
		v1.GET("/users/:user_id", getUserMiddleware, getUser)
		v1.PUT("/users/:user_id", getUserMiddleware, updateUser)
		v1.POST("/users/:user_id/password", getUserMiddleware, updateUserPassword)
		v1.DELETE("/users/:user_id", getUserMiddleware, deleteUser)

		//organizations
		v1.GET("/organizations", organizations.GetOrganizations)
		v1.POST("/organizations", organizations.AddOrganization)
		v1.GET("/organizations/:organization_id", organizations.OrganizationMiddleware, organizations.GetOrganization)
		v1.PUT("/organizations/:organization_id", organizations.OrganizationMiddleware, organizations.UpdateOrganization)

		//projects
		v1.GET("/projects", projects.GetProjects)
		v1.POST("/projects", projects.AddProject)
		v1.GET("/projects/:project_id", projects.ProjectMiddleware, projects.GetProject)
		v1.PUT("/projects/:project_id", projects.ProjectMiddleware, projects.GetProject)


		//credentials
		v1.GET("/credentials", credential.GetCredentials)
		v1.POST("/credentials", credential.AddCredential)
		v1.GET("/credentials/:credential_id", credential.CredentialMiddleware, credential.GetCredential)
		v1.PUT("/credentials/:credential_id", credential.CredentialMiddleware, credential.UpdateCredential)
		v1.DELETE("/credentials/:credential_id", credential.CredentialMiddleware, credential.RemoveCredential)

		p := v1.Group("/project/:project_id")
		{
			p.Use(projects.ProjectMiddleware)

			p.GET("", projects.GetProject)

			p.GET("/events", getEvents)

			/*p.GET("/users", projects.GetUsers)
			p.POST("/users", projects.AddUser)
			p.POST("/users/:user_id/admin", projects.UserMiddleware, projects.MakeUserAdmin)
			p.DELETE("/users/:user_id/admin", projects.UserMiddleware, projects.MakeUserAdmin)
			p.DELETE("/users/:user_id", projects.UserMiddleware, projects.RemoveUser)*/

			p.GET("/keys", projects.GetKeys)
			p.POST("/keys", projects.AddKey)
			p.PUT("/keys/:key_id", projects.KeyMiddleware, projects.UpdateKey)
			p.DELETE("/keys/:key_id", projects.KeyMiddleware, projects.RemoveKey)

			p.GET("/repositories", projects.GetRepositories)
			p.POST("/repositories", projects.AddRepository)
			p.DELETE("/repositories/:repository_id", projects.RepositoryMiddleware, projects.RemoveRepository)

			p.GET("/inventory", projects.GetInventory)
			p.POST("/inventory", projects.AddInventory)
			p.PUT("/inventory/:inventory_id", projects.InventoryMiddleware, projects.UpdateInventory)
			p.DELETE("/inventory/:inventory_id", projects.InventoryMiddleware, projects.RemoveInventory)

			p.GET("/environment", projects.GetEnvironment)
			p.POST("/environment", projects.AddEnvironment)
			p.PUT("/environment/:environment_id", projects.EnvironmentMiddleware, projects.UpdateEnvironment)
			p.DELETE("/environment/:environment_id", projects.EnvironmentMiddleware, projects.RemoveEnvironment)

			p.GET("/templates", projects.GetTemplates)
			p.POST("/templates", projects.AddTemplate)
			p.PUT("/templates/:template_id", projects.TemplatesMiddleware, projects.UpdateTemplate)
			p.DELETE("/templates/:template_id", projects.TemplatesMiddleware, projects.RemoveTemplate)

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
