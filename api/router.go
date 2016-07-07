package api

import (
	"time"

	"pearson.com/hilbert-space/api/cors"
	"pearson.com/hilbert-space/api/projects"
	"pearson.com/hilbert-space/api/sockets"
	"pearson.com/hilbert-space/api/tasks"
	"pearson.com/hilbert-space/util"
	"github.com/gin-gonic/gin"
	"pearson.com/hilbert-space/api/addhoctasks"
	"pearson.com/hilbert-space/api/access"
	"strings"
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

	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "PONG")
	})

	// set up the namespace
	//future refrence: api := r.Group("/api")

	auth := r.Group("/auth")
	{
		auth.POST("/login", access.Login)
		auth.POST("/logout", access.Logout)
	}

	r.Use(access.Authentication)

	r.GET("/ws", sockets.Handler)

	r.GET("/info", getSystemInfo)

	user := r.Group("/user")
	{
		user.GET("", getUser)
		// api.PUT("/user", misc.UpdateUser)
		user.GET("/tokens", getAPITokens)
		user.POST("/tokens", createAPIToken)
		user.DELETE("/tokens/:token_id", expireAPIToken)
	}

	r.GET("/projects", projects.GetProjects)
	r.POST("/projects", projects.AddProject)
	r.GET("/events", getEvents)

	r.GET("/users", getUsers)
	r.POST("/users", addUser)
	r.GET("/users/:user_id", getUserMiddleware, getUser)
	r.PUT("/users/:user_id", getUserMiddleware, updateUser)
	r.POST("/users/:user_id/password", getUserMiddleware, updateUserPassword)
	r.DELETE("/users/:user_id", getUserMiddleware, deleteUser)

	apiProject := r.Group("/project/:project_id")
	{
		apiProject.Use(projects.ProjectMiddleware)

		apiProject.GET("", projects.GetProject)

		apiProject.GET("/events", getEvents)

		apiProject.GET("/users", projects.GetUsers)
		apiProject.POST("/users", projects.AddUser)
		apiProject.POST("/users/:user_id/admin", projects.UserMiddleware, projects.MakeUserAdmin)
		apiProject.DELETE("/users/:user_id/admin", projects.UserMiddleware, projects.MakeUserAdmin)
		apiProject.DELETE("/users/:user_id", projects.UserMiddleware, projects.RemoveUser)

		apiProject.GET("/keys", projects.GetKeys)
		apiProject.POST("/keys", projects.AddKey)
		apiProject.PUT("/keys/:key_id", projects.KeyMiddleware, projects.UpdateKey)
		apiProject.DELETE("/keys/:key_id", projects.KeyMiddleware, projects.RemoveKey)

		apiProject.GET("/repositories", projects.GetRepositories)
		apiProject.POST("/repositories", projects.AddRepository)
		apiProject.DELETE("/repositories/:repository_id", projects.RepositoryMiddleware, projects.RemoveRepository)

		apiProject.GET("/inventory", projects.GetInventory)
		apiProject.POST("/inventory", projects.AddInventory)
		apiProject.PUT("/inventory/:inventory_id", projects.InventoryMiddleware, projects.UpdateInventory)
		apiProject.DELETE("/inventory/:inventory_id", projects.InventoryMiddleware, projects.RemoveInventory)

		apiProject.GET("/environment", projects.GetEnvironment)
		apiProject.POST("/environment", projects.AddEnvironment)
		apiProject.PUT("/environment/:environment_id", projects.EnvironmentMiddleware, projects.UpdateEnvironment)
		apiProject.DELETE("/environment/:environment_id", projects.EnvironmentMiddleware, projects.RemoveEnvironment)

		apiProject.GET("/templates", projects.GetTemplates)
		apiProject.POST("/templates", projects.AddTemplate)
		apiProject.PUT("/templates/:template_id", projects.TemplatesMiddleware, projects.UpdateTemplate)
		apiProject.DELETE("/templates/:template_id", projects.TemplatesMiddleware, projects.RemoveTemplate)

		apiProject.GET("/tasks", tasks.GetAll)
		apiProject.POST("/tasks", tasks.AddTask)
		apiProject.GET("/tasks/:task_id/output", tasks.GetTaskMiddleware, tasks.GetTaskOutput)
	}

	addHocTask := r.Group("/addhoc")
	{
		addHocTask.POST("/tasks", addhoctasks.AddTask)
		addHocTask.GET("/tasks/:task_id/output", addhoctasks.GetTaskMiddleware, addhoctasks.GetTaskOutput)
	}

	globalAccessKeys := r.Group("access")
	{
		globalAccessKeys.GET("/keys", access.GetKeys)
		globalAccessKeys.POST("/keys", access.AddKey)
		globalAccessKeys.PUT("/keys/:key_id", access.KeyMiddleware, access.UpdateKey)
		globalAccessKeys.DELETE("/keys/:key_id", access.KeyMiddleware, access.RemoveKey)
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
			"cmdPath": util.FindHilbertspace(),
		},
	}

	c.JSON(200, body)
}