package http

import (
	"github.com/gin-gonic/gin"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/internal/infrastructure/proxy"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/handler"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
	"github.com/zhaojiewen/open-station/internal/version"
)

type Router struct {
	engine          *gin.Engine
	proxyHandler    *handler.ProxyHandler
	anthropicHandler *handler.AnthropicHandler
	mcpHandler      *handler.MCPHandler
	billingHandler  *handler.BillingHandler
	apiKeyHandler   *handler.APIKeyHandler
	userHandler     *handler.UserHandler
	pluginHandler   *handler.PluginHandler
	authMiddleware  gin.HandlerFunc
	adminMiddleware gin.HandlerFunc
	rateLimitMiddleware gin.HandlerFunc
	safeMiddleware  gin.HandlerFunc
	loggingMiddleware gin.HandlerFunc
	recoveryMiddleware gin.HandlerFunc
}

func NewRouter(
	proxyService *proxy.ProxyService,
	billingService *service.BillingService,
	asyncBilling *service.AsyncBillingQueue,
	authService *auth.AuthService,
	mcpService *service.MCPService,
	pluginService *service.PluginService,
	authMiddleware gin.HandlerFunc,
	adminMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	safeMiddleware gin.HandlerFunc,
) *Router {
	r := &Router{
		engine: gin.New(),
		proxyHandler: handler.NewProxyHandler(proxyService, billingService, asyncBilling, authService),
		anthropicHandler: handler.NewAnthropicHandler(proxyService, authService, billingService),
		mcpHandler: handler.NewMCPHandler(mcpService),
		billingHandler: handler.NewBillingHandler(billingService),
		apiKeyHandler: handler.NewAPIKeyHandler(authService),
		userHandler: handler.NewUserHandler(),
		authMiddleware: authMiddleware,
		adminMiddleware: adminMiddleware,
		rateLimitMiddleware: rateLimitMiddleware,
		safeMiddleware: safeMiddleware,
		loggingMiddleware: middleware.LoggingMiddleware(),
		recoveryMiddleware: middleware.RecoveryMiddleware(),
	}

	// Plugin handler is optional - only set if plugin service is provided
	if pluginService != nil {
		r.pluginHandler = handler.NewPluginHandler(pluginService)
	}

	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	r.engine.Use(r.safeMiddleware)
	r.engine.Use(r.recoveryMiddleware)
	r.engine.Use(r.loggingMiddleware)

	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.engine.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	r.engine.GET("/version", func(c *gin.Context) {
		c.JSON(200, version.GetVersionInfo())
	})

	v1 := r.engine.Group("/v1")
	{
		// Anthropic Messages API 兼容端点 (供 Claude Code CLI 使用)
		anthropicGroup := v1.Group("")
		anthropicGroup.Use(r.authMiddleware)
		anthropicGroup.Use(r.rateLimitMiddleware)
		{
			anthropicGroup.POST("/messages", r.anthropicHandler.Messages)
			anthropicGroup.GET("/models", r.anthropicHandler.Models)
		}

		proxyGroup := v1.Group("/proxy")
		proxyGroup.Use(r.authMiddleware)
		proxyGroup.Use(r.rateLimitMiddleware)
		{
			proxyGroup.POST("/chat/completions", r.proxyHandler.ChatCompletions)
			proxyGroup.POST("/embeddings", r.proxyHandler.Embeddings)
		}

		directGroup := v1.Group("/:provider")
		directGroup.Use(r.authMiddleware)
		directGroup.Use(r.rateLimitMiddleware)
		{
			directGroup.POST("/chat/completions", r.proxyHandler.ChatCompletions)
			directGroup.POST("/messages", r.proxyHandler.ChatCompletions)
			directGroup.POST("/embeddings", r.proxyHandler.Embeddings)
		}
	}

	admin := r.engine.Group("/admin")
	admin.Use(r.authMiddleware)
	admin.Use(r.adminMiddleware)
	{
		admin.GET("/billing/balance/:tenant_id", r.billingHandler.GetBalance)
		admin.POST("/billing/recharge", r.billingHandler.Recharge)
		admin.GET("/billing/usage", r.billingHandler.GetUsage)
		admin.GET("/billing/bills", r.billingHandler.GetBills)

		admin.GET("/api-keys", r.apiKeyHandler.ListAPIKeys)
		admin.POST("/api-keys", r.apiKeyHandler.CreateAPIKey)
		admin.POST("/api-keys/:id/revoke", r.apiKeyHandler.RevokeAPIKey)

		admin.GET("/models", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "models endpoint - to be implemented"})
		})
		admin.POST("/models", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "models endpoint - to be implemented"})
		})

		admin.GET("/users", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "users endpoint - to be implemented"})
		})
		admin.POST("/users", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "users endpoint - to be implemented"})
		})
		admin.GET("/users/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "user endpoint - to be implemented"})
		})
		admin.PUT("/users/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "user endpoint - to be implemented"})
		})
		admin.DELETE("/users/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "user endpoint - to be implemented"})
		})

		admin.GET("/tenants", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "tenants endpoint - to be implemented"})
		})
		admin.POST("/tenants", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "tenants endpoint - to be implemented"})
		})
		admin.PUT("/tenants/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "tenant endpoint - to be implemented"})
		})

			// Plugin management routes
			admin.GET("/plugins", r.pluginHandler.List)
			admin.GET("/plugins/available", r.pluginHandler.ListAvailable)
			admin.GET("/plugins/search", r.pluginHandler.Search)
			admin.GET("/plugins/providers", r.pluginHandler.GetProviders)
			admin.GET("/plugins/stats", r.pluginHandler.GetAllStats)
			admin.GET("/plugins/capability/:capability", r.pluginHandler.ByCapability)
			admin.GET("/plugins/:id", r.pluginHandler.Get)
			admin.POST("/plugins/:id/install", r.pluginHandler.Install)
			admin.PUT("/plugins/:id/configure", r.pluginHandler.Configure)
			admin.POST("/plugins/:id/activate", r.pluginHandler.Activate)
			admin.POST("/plugins/:id/deactivate", r.pluginHandler.Deactivate)
			admin.DELETE("/plugins/:id", r.pluginHandler.Uninstall)
			admin.GET("/plugins/:id/health", r.pluginHandler.HealthCheck)
			admin.GET("/plugins/:id/stats", r.pluginHandler.GetStats)
	}

	user := r.engine.Group("/user")
	user.Use(r.authMiddleware)
	{
		user.GET("/profile", r.userHandler.GetProfile)
		user.GET("/api-keys", r.apiKeyHandler.ListMyAPIKeys)
		user.POST("/api-keys", r.apiKeyHandler.CreateMyAPIKey)
		user.GET("/usage", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "usage endpoint - to be implemented"})
		})
	}

	// MCP endpoint (Model Context Protocol for Claude Code CLI)
	mcp := r.engine.Group("/mcp")
	{
		mcp.POST("", r.mcpHandler.HandleMCP)   // JSON-RPC requests
		mcp.GET("", r.mcpHandler.HandleSSE)    // SSE streaming
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}