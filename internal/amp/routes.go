package amp

import (
	"net/http/httputil"

	"github.com/gin-gonic/gin"
)

func RegisterProxyRoutes(engine *gin.Engine, proxy *httputil.ReverseProxy) {
	proxyHandler := ProxyHandler(proxy)
	channelHandler := ChannelProxyHandler()
	modelsHandler := ModelsHandler()

	// Register amp proxy routes at root level
	registerAmpProxyAPI(engine, proxyHandler, channelHandler, modelsHandler)

	// Register management routes (user, auth, threads, etc.) - proxied to ampcode.com
	registerManagementRoutes(engine, proxyHandler)
}

// ThreadRedirectHandler redirects /threads/T-xxx to official Amp threads
func ThreadRedirectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		threadID := c.Param("threadID")
		officialURL := "https://ampcode.com/threads/" + threadID
		c.Redirect(302, officialURL)
	}
}

func createRoutingHandler(upstreamHandler, channelHandler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelCfg := GetChannelConfig(c)
		if channelCfg != nil && channelCfg.Channel != nil {
			channelHandler(c)
			return
		}
		upstreamHandler(c)
	}
}

// createProviderHandler routes provider requests, with special handling for /models endpoints
func createProviderHandler(upstreamHandler, channelHandler, modelsHandler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Param("path")

		// Check if this is a models request - return local metadata
		if c.Request.Method == "GET" && isModelsEndpoint(path) {
			modelsHandler(c)
			return
		}

		// Otherwise use normal routing
		channelCfg := GetChannelConfig(c)
		if channelCfg != nil && channelCfg.Channel != nil {
			channelHandler(c)
			return
		}
		upstreamHandler(c)
	}
}

// isModelsEndpoint checks if path is a models list endpoint
func isModelsEndpoint(path string) bool {
	// /models, /v1/models, /v1beta/models
	return path == "/models" || path == "/v1/models" || path == "/v1beta/models"
}

// registerManagementRoutes registers Amp management proxy routes
// These routes proxy through to ampcode.com for OAuth, user management, threads, etc.
func registerManagementRoutes(engine *gin.Engine, proxyHandler gin.HandlerFunc) {
	// Management routes under /api/* - proxied to ampcode.com
	api := engine.Group("/api")
	api.Use(APIKeyAuthMiddleware())

	// User and auth management
	api.Any("/user", proxyHandler)
	api.Any("/user/*path", proxyHandler)
	api.Any("/auth", proxyHandler)
	api.Any("/auth/*path", proxyHandler)

	// Metadata and telemetry
	api.Any("/meta", proxyHandler)
	api.Any("/meta/*path", proxyHandler)
	api.Any("/ads", proxyHandler)
	api.Any("/telemetry", proxyHandler)
	api.Any("/telemetry/*path", proxyHandler)

	// Thread management
	api.Any("/threads", proxyHandler)
	api.Any("/threads/*path", proxyHandler)

	// OpenTelemetry and tab
	api.Any("/otel", proxyHandler)
	api.Any("/otel/*path", proxyHandler)
	api.Any("/tab", proxyHandler)
	api.Any("/tab/*path", proxyHandler)

	// Root-level routes that AMP CLI expects without /api prefix
	engine.GET("/threads", proxyHandler)
	engine.GET("/threads/*path", proxyHandler)
	engine.GET("/docs", proxyHandler)
	engine.GET("/docs/*path", proxyHandler)
	engine.GET("/settings", proxyHandler)
	engine.GET("/settings/*path", proxyHandler)
	engine.GET("/threads.rss", proxyHandler)
	engine.GET("/news.rss", proxyHandler)

	// Root-level auth routes for CLI login flow
	engine.Any("/auth", proxyHandler)
	engine.Any("/auth/*path", proxyHandler)
}

// registerAmpProxyAPI registers amp proxy routes at root /api/* level
// This is needed because amp CLI ignores URL path and sends requests directly to /api/*
func registerAmpProxyAPI(engine *gin.Engine, proxyHandler, channelHandler, modelsHandler gin.HandlerFunc) {
	api := engine.Group("/api")
	api.Use(APIKeyAuthMiddleware())
	api.Use(ApplyModelMappingMiddleware())
	api.Use(ChannelRouterMiddleware())

	api.Any("/internal", LocalWebSearchMiddleware(), DebugInternalAPIMiddleware(), proxyHandler)
	api.Any("/internal/*path", LocalWebSearchMiddleware(), DebugInternalAPIMiddleware(), proxyHandler)

	api.Any("/provider/:provider/*path", createProviderHandler(proxyHandler, channelHandler, modelsHandler))

	// Root level v1/v1beta routes for OpenAI/Anthropic/Gemini compatible endpoints
	v1 := engine.Group("/v1")
	v1.Use(APIKeyAuthMiddleware())
	v1.Use(ApplyModelMappingMiddleware())
	v1.Use(ChannelRouterMiddleware())

	v1.POST("/chat/completions", createRoutingHandler(proxyHandler, channelHandler))
	v1.POST("/completions", createRoutingHandler(proxyHandler, channelHandler))
	v1.POST("/messages", createRoutingHandler(proxyHandler, channelHandler))
	v1.POST("/responses", createRoutingHandler(proxyHandler, channelHandler))
	v1.GET("/models", proxyHandler)

	v1beta := engine.Group("/v1beta")
	v1beta.Use(APIKeyAuthMiddleware())
	v1beta.Use(ApplyModelMappingMiddleware())
	v1beta.Use(ChannelRouterMiddleware())

	v1beta.POST("/models/*action", createRoutingHandler(proxyHandler, channelHandler))
	v1beta.GET("/models", proxyHandler)
	v1beta.GET("/models/*action", proxyHandler)
}
