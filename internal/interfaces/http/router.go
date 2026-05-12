package http

import (
	"github.com/gin-gonic/gin"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/internal/infrastructure/payment"
	"github.com/zhaojiewen/open-station/internal/infrastructure/proxy"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/handler"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
	"github.com/zhaojiewen/open-station/internal/version"
)

type Router struct {
	engine             *gin.Engine
	proxyHandler       *handler.ProxyHandler
	dynamicProxyHandler *handler.DynamicProxyHandler
	providerAccountHandler *handler.ProviderAccountHandler
	anthropicHandler   *handler.AnthropicHandler
	mcpHandler         *handler.MCPHandler
	billingHandler     *handler.BillingHandler
	apiKeyHandler      *handler.APIKeyHandler
	userHandler        *handler.UserHandler
	pluginHandler      *handler.PluginHandler
	platformHandler    *handler.PlatformHandler
	tenantAppHandler   *handler.TenantApplicationHandler
	userAppHandler     *handler.UserApplicationHandler
	budgetAlertHandler *handler.BudgetAlertHandler
	// Payment system handlers
	creditAppHandler    *handler.CreditApplicationHandler
	memberQuotaHandler  *handler.MemberQuotaHandler
	paymentHandler      *handler.PaymentHandler
	settlementHandler   *handler.SettlementHandler
	// Auth handlers
	authHandler         *handler.AuthHandler
	authMiddleware      gin.HandlerFunc
	adminMiddleware     gin.HandlerFunc
	platformMiddleware  gin.HandlerFunc
	jwtAuthMiddleware   gin.HandlerFunc
	rateLimitMiddleware gin.HandlerFunc
	loginRateLimitMiddleware gin.HandlerFunc
	safeMiddleware      gin.HandlerFunc
	loggingMiddleware   gin.HandlerFunc
	recoveryMiddleware  gin.HandlerFunc
	securityHeadersMiddleware gin.HandlerFunc
}

func NewRouter(
	proxyService *proxy.ProxyService,
	dynamicProxyHandler *handler.DynamicProxyHandler,
	providerAccountHandler *handler.ProviderAccountHandler,
	billingService *service.BillingService,
	asyncBilling *service.AsyncBillingQueue,
	authService *auth.AuthService,
	mcpService *service.MCPService,
	pluginService *service.PluginService,
	platformAuth *auth.PlatformAuthService,
	tenantAppService *service.TenantApplicationService,
	userAppService *service.UserApplicationService,
	budgetAlertService *service.BudgetAlertService,
	// Payment system services
	creditAppService *service.CreditApplicationService,
	memberQuotaService *service.MemberQuotaService,
	paymentService *service.PaymentService,
	settlementService *service.SettlementService,
	gatewayService *payment.PaymentGatewayService,
	// Auth services for JWT authentication
	jwtService *auth.JWTService,
	userAuthService *auth.UserAuthService,
	tenantRepo repository.TenantRepository,
	userRepo repository.UserRepository,
	authMiddleware gin.HandlerFunc,
	adminMiddleware gin.HandlerFunc,
	platformMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	safeMiddleware gin.HandlerFunc,
loginRateLimitMiddleware gin.HandlerFunc,
) *Router {
	r := &Router{
		engine: gin.New(),
		proxyHandler: handler.NewProxyHandler(proxyService, billingService, asyncBilling, authService),
			dynamicProxyHandler: dynamicProxyHandler,
			providerAccountHandler: providerAccountHandler,
		anthropicHandler: handler.NewAnthropicHandler(proxyService, authService, billingService),
		mcpHandler: handler.NewMCPHandler(mcpService),
		billingHandler: handler.NewBillingHandler(billingService),
		apiKeyHandler: handler.NewAPIKeyHandler(authService),
		userHandler: handler.NewUserHandler(),
		authMiddleware: authMiddleware,
		adminMiddleware: adminMiddleware,
		platformMiddleware: platformMiddleware,
		rateLimitMiddleware: rateLimitMiddleware,
		safeMiddleware: safeMiddleware,
			loginRateLimitMiddleware: loginRateLimitMiddleware,
		loggingMiddleware: middleware.LoggingMiddleware(),
		recoveryMiddleware: middleware.RecoveryMiddleware(),
		securityHeadersMiddleware: middleware.SecurityHeaders(),
	}

	// Initialize JWT auth middleware if services provided
	if jwtService != nil && userAuthService != nil {
		r.jwtAuthMiddleware = middleware.JWTAuthMiddleware(jwtService, userAuthService)
		r.authHandler = handler.NewAuthHandler(userAuthService, jwtService)
	}

	// Plugin handler is optional - only set if plugin service is provided
	if pluginService != nil {
		r.pluginHandler = handler.NewPluginHandler(pluginService)
	}

	// Platform handlers - only set if platform auth is provided
	if platformAuth != nil && tenantAppService != nil {
		r.platformHandler = handler.NewPlatformHandler(platformAuth, tenantAppService, tenantRepo)
		r.tenantAppHandler = handler.NewTenantApplicationHandler(tenantAppService)
	}

	// User application handler - only set if user app service is provided
	if userAppService != nil {
		r.userAppHandler = handler.NewUserApplicationHandler(userAppService, authService, userRepo, tenantRepo)
	}

	// Budget alert handler - only set if budget alert service is provided
	if budgetAlertService != nil {
		r.budgetAlertHandler = handler.NewBudgetAlertHandler(budgetAlertService)
	}

	// Payment system handlers
	if creditAppService != nil {
		r.creditAppHandler = handler.NewCreditApplicationHandler(creditAppService)
	}
	if memberQuotaService != nil {
		r.memberQuotaHandler = handler.NewMemberQuotaHandler(memberQuotaService)
	}
	if paymentService != nil {
		r.paymentHandler = handler.NewPaymentHandler(paymentService, gatewayService)
	}
	if settlementService != nil {
		r.settlementHandler = handler.NewSettlementHandler(settlementService)
	}

	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	r.engine.Use(r.safeMiddleware)
	r.engine.Use(r.recoveryMiddleware)
	r.engine.Use(r.loggingMiddleware)
	r.engine.Use(r.securityHeadersMiddleware)

	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.engine.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	r.engine.GET("/version", func(c *gin.Context) {
		c.JSON(200, version.GetVersionInfo())
	})

	// Auth routes (public - no authentication required)
	if r.authHandler != nil {
		authPublic := r.engine.Group("/auth")
			if r.loginRateLimitMiddleware != nil {
				authPublic.Use(r.loginRateLimitMiddleware)
			}
		{
			authPublic.POST("/login", r.authHandler.Login)
			authPublic.POST("/register", r.authHandler.Register)
			authPublic.POST("/tenant/register", r.authHandler.RegisterTenant)
			authPublic.POST("/refresh", r.authHandler.RefreshToken)
			authPublic.POST("/verify-email", r.authHandler.VerifyEmail)
			authPublic.POST("/resend-verification", r.authHandler.ResendVerification)
		}

		// Auth routes (require JWT authentication)
		authProtected := r.engine.Group("/auth")
		authProtected.Use(r.jwtAuthMiddleware)
		{
			authProtected.POST("/logout", r.authHandler.Logout)
			authProtected.POST("/logout-all", r.authHandler.LogoutAll)
			authProtected.GET("/profile", r.authHandler.GetProfile)
			authProtected.GET("/tenants", r.authHandler.GetTenants)
			authProtected.POST("/switch-tenant", r.authHandler.SwitchTenant)
			authProtected.PUT("/password", r.authHandler.ChangePassword)
		}
	}

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
			proxyGroup.POST("/chat/completions", r.dynamicProxyHandler.ChatCompletions)
			proxyGroup.POST("/embeddings", r.proxyHandler.Embeddings)
		}

		directGroup := v1.Group("/:provider")
		directGroup.Use(r.authMiddleware)
		directGroup.Use(r.rateLimitMiddleware)
		{
			directGroup.POST("/chat/completions", r.dynamicProxyHandler.ChatCompletions)
			directGroup.POST("/messages", r.dynamicProxyHandler.ChatCompletions)
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
		// admin.POST("/users") - implemented below via userAppHandler.AdminCreateUser
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

			// User application management (tenant admin)
			admin.GET("/applications", r.userAppHandler.AdminListApplications)
			admin.POST("/applications/:id/approve", r.userAppHandler.AdminApproveRequest)
			admin.POST("/applications/:id/reject", r.userAppHandler.AdminRejectRequest)

			// User invitation management
			admin.POST("/invitations", r.userAppHandler.AdminSendInvitation)
			admin.GET("/invitations", r.userAppHandler.AdminListApplications) // Same as applications
			admin.DELETE("/invitations/:id", r.userAppHandler.AdminCancelInvitation)

			// Direct user creation
			admin.POST("/users", r.userAppHandler.AdminCreateUser)

			// Budget alert management
			admin.GET("/budget/alerts", r.budgetAlertHandler.List)
			admin.POST("/budget/alerts", r.budgetAlertHandler.Create)
			admin.GET("/budget/alerts/:id", r.budgetAlertHandler.Get)
			admin.PUT("/budget/alerts/:id", r.budgetAlertHandler.Update)
			admin.POST("/budget/alerts/:id/enable", r.budgetAlertHandler.Enable)
			admin.POST("/budget/alerts/:id/disable", r.budgetAlertHandler.Disable)
			admin.DELETE("/budget/alerts/:id", r.budgetAlertHandler.Delete)

			// Provider account management (multi-key support with failover)
			if r.providerAccountHandler != nil {
				admin.GET("/providers/:provider/status", r.providerAccountHandler.GetProviderStatus)
				admin.GET("/providers/status", r.providerAccountHandler.GetAllProvidersStatus)
				admin.POST("/providers/:provider/switch", r.providerAccountHandler.SwitchAccount)
				admin.GET("/providers/accounts/:account_id", r.providerAccountHandler.GetAccountDetail)
				admin.GET("/providers/:provider/history", r.providerAccountHandler.GetSwitchHistory)
				admin.POST("/providers/accounts/:account_id/recover", r.providerAccountHandler.ManualRecoverAccount)
				admin.GET("/providers/metrics", r.providerAccountHandler.GetRealTimeMetrics)
				admin.POST("/providers/cache/refresh", r.providerAccountHandler.ForceRefreshCache)
				admin.GET("/providers/cache/stats", r.providerAccountHandler.GetCacheStats)
			}
	}

	user := r.engine.Group("/user")
	user.Use(r.authMiddleware)
	{
		user.GET("/profile", r.userHandler.GetProfile)
		user.GET("/api-keys", r.apiKeyHandler.ListMyAPIKeys)
		user.GET("/usage", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "usage endpoint - to be implemented"})
		})
	}

	// API key write operations require non-viewer role
	apiKeyWrite := r.engine.Group("/user")
	apiKeyWrite.Use(r.authMiddleware)
	apiKeyWrite.Use(middleware.RequireTenantWrite())
	{
		apiKeyWrite.POST("/api-keys", r.apiKeyHandler.CreateMyAPIKey)
	}

	// MCP endpoint (Model Context Protocol for Claude Code CLI)
	mcp := r.engine.Group("/mcp")
	{
		mcp.POST("", r.mcpHandler.HandleMCP)   // JSON-RPC requests
		mcp.GET("", r.mcpHandler.HandleSSE)    // SSE streaming
	}

	// Platform admin routes (requires platform admin authentication)
	platform := r.engine.Group("/platform")
	{
		// Login endpoint (no auth required)
		platform.POST("/login", r.platformHandler.Login)

		// Admin management (requires platform admin auth)
		adminGroup := platform.Group("")
		adminGroup.Use(r.platformMiddleware)
		{
			// Read operations — any platform admin can access
			adminGroup.GET("/admins", r.platformHandler.ListAdmins)
			adminGroup.GET("/admins/:id", r.platformHandler.GetAdmin)
			adminGroup.GET("/applications", r.platformHandler.ListApplications)
			adminGroup.GET("/applications/:id", r.platformHandler.GetApplication)
			adminGroup.GET("/tenants", r.platformHandler.ListTenants)

			// Super admin only — admin CRUD
			superGroup := adminGroup.Group("")
			superGroup.Use(middleware.SuperAdminMiddleware())
			{
				superGroup.POST("/admins", r.platformHandler.CreateAdmin)
				superGroup.PUT("/admins/:id", r.platformHandler.UpdateAdmin)
				superGroup.DELETE("/admins/:id", r.platformHandler.DeleteAdmin)
			}

			// Billing write — billing_admin and super_admin can approve/manage
			billingGroup := adminGroup.Group("")
			billingGroup.Use(middleware.PlatformPermissionMiddleware("billing:write"))
			{
				billingGroup.POST("/applications/:id/approve", r.platformHandler.ApproveApplication)
				billingGroup.POST("/applications/:id/reject", r.platformHandler.RejectApplication)
				billingGroup.PUT("/tenants/:id/suspend", r.platformHandler.SuspendTenant)
				billingGroup.PUT("/tenants/:id/activate", r.platformHandler.ActivateTenant)
			}
		}
	}

	// Public tenant application endpoint (for new tenants to apply)
	apply := r.engine.Group("/apply")
	{
		apply.POST("/tenant", r.tenantAppHandler.Submit)
		apply.GET("/tenant/:id/status", r.tenantAppHandler.GetStatus)
		apply.POST("/user", r.userAppHandler.SubmitRequest)
	}

	// Public invitation endpoint (for users to accept invitations)
	invite := r.engine.Group("/invite")
	{
		invite.POST("/accept", r.userAppHandler.AcceptInvitation)
		invite.GET("/verify/:token", r.userAppHandler.VerifyInvitation)
	}

	// Payment system routes
	// Tenant credit application routes (requires tenant admin auth)
	tenantCredit := r.engine.Group("/tenant")
	tenantCredit.Use(r.authMiddleware)
	tenantCredit.Use(r.adminMiddleware)
	{
		tenantCredit.POST("/credit-application", r.creditAppHandler.ApplyForCredit)
		tenantCredit.GET("/credit-application", r.creditAppHandler.GetApplication)
		tenantCredit.PUT("/credit-application", r.creditAppHandler.UpdateApplication)
		tenantCredit.DELETE("/credit-application", r.creditAppHandler.CancelApplication)
	}

	// Member quota management routes (requires tenant admin auth)
	memberQuotaAdmin := r.engine.Group("/admin/member-quotas")
	memberQuotaAdmin.Use(r.authMiddleware)
	memberQuotaAdmin.Use(r.adminMiddleware)
	{
		memberQuotaAdmin.GET("", r.memberQuotaHandler.ListMemberQuotas)
		memberQuotaAdmin.POST("", r.memberQuotaHandler.CreateMemberQuota)
		memberQuotaAdmin.GET("/:id", r.memberQuotaHandler.GetMemberQuota)
		memberQuotaAdmin.PUT("/:id", r.memberQuotaHandler.UpdateMemberQuota)
		memberQuotaAdmin.DELETE("/:id", r.memberQuotaHandler.DeleteMemberQuota)
		memberQuotaAdmin.PUT("/:id/token-limit", r.memberQuotaHandler.SetTokenLimit)
		memberQuotaAdmin.PUT("/:id/cost-limit", r.memberQuotaHandler.SetCostLimit)
		memberQuotaAdmin.GET("/:id/usage", r.memberQuotaHandler.GetMemberUsage)
		memberQuotaAdmin.POST("/:id/reset", r.memberQuotaHandler.ResetMemberQuota)
	}

	// User member quota routes (requires user auth)
	userQuota := r.engine.Group("/user")
	userQuota.Use(r.authMiddleware)
	{
		userQuota.GET("/member-quota", r.memberQuotaHandler.GetMyMemberQuota)
		userQuota.GET("/member-usage", r.memberQuotaHandler.GetMyMemberUsage)
	}

	// Platform admin credit review routes
	platformCredit := r.engine.Group("/platform")
	platformCredit.Use(r.platformMiddleware)
	{
		// Read operations
		platformCredit.GET("/credit-applications", r.creditAppHandler.ListApplications)
		platformCredit.GET("/credit-applications/pending-count", r.creditAppHandler.GetPendingCount)
		platformCredit.GET("/credit-applications/:id", r.creditAppHandler.GetApplicationDetail)
		platformCredit.GET("/member-quotas", r.memberQuotaHandler.ListAllMemberQuotas)

		// Write operations — require billing:write
		creditWrite := platformCredit.Group("")
		creditWrite.Use(middleware.PlatformPermissionMiddleware("billing:write"))
		{
			creditWrite.POST("/credit-applications/:id/review", r.creditAppHandler.ReviewApplication)
			creditWrite.PUT("/tenants/:id/credit", r.creditAppHandler.AdjustCreditLimit)
		}
	}

	// Payment routes - User level (individual mode)
	if r.paymentHandler != nil {
		userPayments := r.engine.Group("/user/payments")
		userPayments.Use(r.authMiddleware)
		{
			userPayments.POST("", r.paymentHandler.CreateOrder)
			userPayments.GET("", r.paymentHandler.ListOrders)
			userPayments.GET("/pending", r.paymentHandler.GetPendingOrders)
			userPayments.GET("/:id", r.paymentHandler.GetOrder)
			userPayments.POST("/:id/cancel", r.paymentHandler.CancelOrder)
		}

		// Payment routes - Tenant Admin level (organization mode)
		adminPayments := r.engine.Group("/admin/payments")
		adminPayments.Use(r.authMiddleware)
		adminPayments.Use(r.adminMiddleware)
		{
			adminPayments.POST("", r.paymentHandler.CreateOrder)
			adminPayments.GET("", r.paymentHandler.ListOrders)
			adminPayments.GET("/:id", r.paymentHandler.GetOrder)
		}

		// Public payment endpoints (no auth)
		r.engine.POST("/payments/callback/:provider", r.paymentHandler.ProcessCallback)
		r.engine.GET("/payments/:order_number", r.paymentHandler.GetOrderByNumber)
	}

	// Settlement routes
	if r.settlementHandler != nil {
		// Tenant Admin settlement routes
		tenantSettlement := r.engine.Group("/tenant/settlement")
		tenantSettlement.Use(r.authMiddleware)
		tenantSettlement.Use(r.adminMiddleware)
		{
			tenantSettlement.GET("/check", r.settlementHandler.CheckTrigger)
			tenantSettlement.POST("/trigger", r.settlementHandler.TriggerSettlement)
		}

		// Settlement payment route
		adminSettlement := r.engine.Group("/admin/settlement")
		adminSettlement.Use(r.authMiddleware)
		adminSettlement.Use(r.adminMiddleware)
		{
			adminSettlement.POST("/:bill_id/pay", r.settlementHandler.ProcessBillPayment)
		}

		// Platform settlement management
		platformSettlement := r.engine.Group("/platform/settlement")
		platformSettlement.Use(r.platformMiddleware)
		{
			platformSettlement.GET("/overdue", r.settlementHandler.CheckOverdue)
			platformSettlement.POST("/run", middleware.PlatformPermissionMiddleware("billing:write"), r.settlementHandler.RunScheduledSettlement)
		}
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}