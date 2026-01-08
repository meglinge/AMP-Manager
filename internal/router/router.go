package router

import (
	"strings"

	"ampmanager/internal/amp"
	"ampmanager/internal/config"
	"ampmanager/internal/handler"
	"ampmanager/internal/middleware"
	"ampmanager/internal/web"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.Default()

	cfg := config.Get()

	allowedOrigins := strings.Split(cfg.CORSAllowedOrigins, ",")
	if len(allowedOrigins) == 0 || allowedOrigins[0] == "" {
		allowedOrigins = []string{"*"}
	}

	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowed := false
		for _, o := range allowedOrigins {
			o = strings.TrimSpace(o)
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if allowedOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Vary", "Origin")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	authLimiter := middleware.NewRateLimiter(cfg.RateLimitAuthRPS, 10)

	userHandler := handler.NewUserHandler()
	ampHandler := handler.NewAmpHandler()
	requestLogHandler := handler.NewRequestLogHandler()
	channelHandler := handler.NewChannelHandler()
	modelHandler := handler.NewModelHandler()
	modelMetadataHandler := handler.NewModelMetadataHandler()
	systemHandler := handler.NewSystemHandler()

	api := r.Group("/api")
	{
		// Local management auth (using /manage/auth to avoid conflict with proxy /api/auth/*)
		manageAuth := api.Group("/manage/auth")
		manageAuth.Use(authLimiter.RateLimitByIP())
		{
			manageAuth.POST("/register", userHandler.Register)
			manageAuth.POST("/login", userHandler.Login)
		}

		me := api.Group("/me")
		me.Use(middleware.JWTAuthMiddleware())
		{
			me.PUT("/password", userHandler.ChangePassword)
			me.PUT("/username", userHandler.ChangeUsername)

			ampGroup := me.Group("/amp")
			{
				ampGroup.GET("/settings", ampHandler.GetSettings)
				ampGroup.PUT("/settings", ampHandler.UpdateSettings)
				ampGroup.POST("/settings/test", ampHandler.TestConnection)

				ampGroup.GET("/api-keys", ampHandler.ListAPIKeys)
				ampGroup.POST("/api-keys", ampHandler.CreateAPIKey)
				ampGroup.GET("/api-keys/:id", ampHandler.GetAPIKey)
				ampGroup.DELETE("/api-keys/:id", ampHandler.DeleteAPIKey)

				ampGroup.GET("/bootstrap", ampHandler.GetBootstrap)

				// 请求日志
				ampGroup.GET("/request-logs", requestLogHandler.ListRequestLogs)
				ampGroup.GET("/request-logs/:id", requestLogHandler.GetRequestLog)
				ampGroup.GET("/usage/summary", requestLogHandler.GetUsageSummary)
			}
		}

		models := api.Group("/models")
		models.Use(middleware.JWTAuthMiddleware())
		{
			models.GET("", modelHandler.ListAvailableModels)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuthMiddleware())
		admin.Use(middleware.AdminMiddleware())
		{
			channels := admin.Group("/channels")
			{
				channels.GET("", channelHandler.List)
				channels.POST("", channelHandler.Create)
				channels.GET("/:id", channelHandler.Get)
				channels.PUT("/:id", channelHandler.Update)
				channels.DELETE("/:id", channelHandler.Delete)
				channels.PATCH("/:id/enabled", channelHandler.SetEnabled)
				channels.POST("/:id/test", channelHandler.TestConnection)
				channels.POST("/:id/fetch-models", modelHandler.FetchChannelModels)
				channels.GET("/:id/models", modelHandler.GetChannelModels)
			}

			adminModels := admin.Group("/models")
			{
				adminModels.POST("/fetch-all", modelHandler.FetchAllModels)
			}

			modelMetadata := admin.Group("/model-metadata")
			{
				modelMetadata.GET("", modelMetadataHandler.List)
				modelMetadata.GET("/:id", modelMetadataHandler.Get)
				modelMetadata.POST("", modelMetadataHandler.Create)
				modelMetadata.PUT("/:id", modelMetadataHandler.Update)
				modelMetadata.DELETE("/:id", modelMetadataHandler.Delete)
			}

			system := admin.Group("/system")
			{
				system.POST("/database/upload", systemHandler.UploadDatabase)
				system.GET("/database/download", systemHandler.DownloadDatabase)
				system.GET("/database/backups", systemHandler.ListBackups)
				system.POST("/database/restore", systemHandler.RestoreBackup)
				system.DELETE("/database/backups/:filename", systemHandler.DeleteBackup)

				// 重试配置
				system.GET("/retry-config", systemHandler.GetRetryConfig)
				system.PUT("/retry-config", systemHandler.UpdateRetryConfig)
			}

			users := admin.Group("/users")
			{
				users.GET("", userHandler.ListUsers)
				users.PATCH("/:id/admin", userHandler.SetAdmin)
				users.POST("/:id/reset-password", userHandler.ResetPassword)
				users.DELETE("/:id", userHandler.DeleteUser)
			}

			// 管理员日志和使用统计
			admin.GET("/request-logs", requestLogHandler.AdminListRequestLogs)
			admin.GET("/request-logs/models", requestLogHandler.AdminGetDistinctModels)
			admin.GET("/usage/summary", requestLogHandler.AdminGetUsageSummary)
		}
	}

	proxy := amp.CreateDynamicReverseProxy()
	amp.RegisterProxyRoutes(r, proxy)

	// Serve embedded frontend static files
	web.RegisterStaticRoutes(r)

	return r
}
