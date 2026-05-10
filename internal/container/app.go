package container

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/internal/infrastructure/persistence/postgres"
	"github.com/zhaojiewen/open-station/internal/infrastructure/persistence/postgres/repositories"
	redisconn "github.com/zhaojiewen/open-station/internal/infrastructure/persistence/redis"
	ratelimit "github.com/zhaojiewen/open-station/internal/infrastructure/persistence/redis"
	"github.com/zhaojiewen/open-station/internal/infrastructure/proxy"
	"github.com/zhaojiewen/open-station/pkg/config"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RepositoriesContainer holds all repository instances
type RepositoriesContainer struct {
	Tenant         repository.TenantRepository
	User           repository.UserRepository
	APIKey         repository.APIKeyRepository
	Model          repository.ModelRepository
	Usage          repository.UsageRepository
	Bill           repository.BillRepository
	Recharge       repository.RechargeRepository
	AuditLog       repository.AuditLogRepository
	ProviderAccount repository.ProviderAccountRepository
}

// ServicesContainer holds all service instances
type ServicesContainer struct {
	Auth             *auth.AuthService
	Billing          *service.BillingService
	Proxy            *proxy.ProxyService
	MCP              *service.MCPService
	ProviderAccount  *service.ProviderAccountService
	Init             *service.InitService
	AsyncBilling     *service.AsyncBillingQueue
}

// InfrastructureContainer holds infrastructure components
type InfrastructureContainer struct {
	DB              *gorm.DB
	Redis           *redis.Client
	RateLimit       *ratelimit.RateLimitService
	Safe            *ratelimit.SafeService
}

// AppContainer is the main dependency injection container
type AppContainer struct {
	Infrastructure *InfrastructureContainer
	Repositories   *RepositoriesContainer
	Services       *ServicesContainer

	mu       sync.Mutex
	stopped  bool
	stopChan chan struct{}
}

// ContainerConfig holds configuration for the container
type ContainerConfig struct {
	Config           *config.Config
	SkipInit         bool
	AsyncBillingWorkers int
	AsyncBillingQueueSize int
}

// NewAppContainer creates and initializes the entire application container
func NewAppContainer(cfg *ContainerConfig) (*AppContainer, error) {
	container := &AppContainer{
		stopChan: make(chan struct{}),
	}

	// Initialize infrastructure
	if err := container.initInfrastructure(cfg.Config); err != nil {
		return nil, fmt.Errorf("failed to initialize infrastructure: %w", err)
	}

	// Initialize repositories
	container.initRepositories()

	// Initialize services
	if err := container.initServices(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	return container, nil
}

// initInfrastructure initializes database, redis, and other infrastructure
func (c *AppContainer) initInfrastructure(cfg *config.Config) error {
	c.Infrastructure = &InfrastructureContainer{}

	// Connect to database
	db, err := postgres.Connect(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	c.Infrastructure.DB = db

	// Connect to Redis
	redisClient, err := redisconn.Connect(&cfg.Redis)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	c.Infrastructure.Redis = redisClient

	// Initialize rate limit service
	c.Infrastructure.RateLimit = ratelimit.NewRateLimitService(redisClient, cfg.RateLimit.RedisKeyPrefix)

	// Initialize safe service
	c.Infrastructure.Safe = ratelimit.NewSafeService(
		redisClient,
		cfg.Safe.RedisKeyPrefix,
		cfg.Safe.IPWhitelist,
		cfg.Safe.IPBlacklist,
	)

	return nil
}

// initRepositories initializes all repositories
func (c *AppContainer) initRepositories() {
	c.Repositories = &RepositoriesContainer{
		Tenant:          repositories.NewTenantRepository(c.Infrastructure.DB),
		User:            repositories.NewUserRepository(c.Infrastructure.DB),
		APIKey:          repositories.NewAPIKeyRepository(c.Infrastructure.DB),
		Model:           repositories.NewModelRepository(c.Infrastructure.DB),
		Usage:           repositories.NewUsageRepository(c.Infrastructure.DB),
		Bill:            repositories.NewBillRepository(c.Infrastructure.DB),
		Recharge:        repositories.NewRechargeRepository(c.Infrastructure.DB),
		AuditLog:        repositories.NewAuditLogRepository(c.Infrastructure.DB),
		ProviderAccount: repositories.NewProviderAccountRepository(c.Infrastructure.DB),
	}
}

// initServices initializes all services
func (c *AppContainer) initServices(cfg *ContainerConfig) error {
	c.Services = &ServicesContainer{}

	// Auth service
	c.Services.Auth = auth.NewAuthService(
		c.Repositories.APIKey,
		c.Repositories.User,
		c.Repositories.Tenant,
		c.Infrastructure.Redis,
	)

	// Proxy service
	c.Services.Proxy = proxy.NewProxyService(&cfg.Config.Providers)

	// Billing service
	c.Services.Billing = service.NewBillingService(
		c.Repositories.Tenant,
		c.Repositories.Usage,
		c.Repositories.Bill,
		c.Repositories.Recharge,
		c.Repositories.Model,
	)

	// Provider account service
	c.Services.ProviderAccount = service.NewProviderAccountService(
		c.Repositories.ProviderAccount,
		cfg.Config.Server.EncryptionKey,
	)

	// MCP service
	c.Services.MCP = service.NewMCPService(
		c.Services.Auth,
		c.Services.Billing,
		c.Services.ProviderAccount,
	)

	// Init service
	c.Services.Init = service.NewInitService(
		c.Repositories.Tenant,
		c.Repositories.User,
		c.Repositories.APIKey,
		&cfg.Config.Admin,
	)

	// Async billing queue
	queueSize := cfg.AsyncBillingQueueSize
	if queueSize <= 0 {
		queueSize = 10000
	}
	workers := cfg.AsyncBillingWorkers
	if workers <= 0 {
		workers = 4
	}
	c.Services.AsyncBilling = service.NewAsyncBillingQueue(
		c.Services.Billing,
		c.Repositories.APIKey,
		queueSize,
		100,
		5*time.Second,
	)

	return nil
}

// Start starts all services that require initialization
func (c *AppContainer) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return fmt.Errorf("container already stopped")
	}

	// Start async billing queue workers
	c.Services.AsyncBilling.Start(4)

	logger.Info("application container started")
	return nil
}

// Stop gracefully stops all services
func (c *AppContainer) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return nil
	}
	c.stopped = true

	// Stop async billing queue
	c.Services.AsyncBilling.Stop()

	// Stop provider account service (cancel recovery timers)
	c.Services.ProviderAccount.Stop()

	// Stop rate limit services
	c.Infrastructure.RateLimit.Stop()
	c.Infrastructure.Safe.Stop()

	// Close database connection
	if err := postgres.Close(); err != nil {
		logger.Error("database close error", zap.Error(err))
	}

	// Close Redis connection
	if err := redisconn.Close(); err != nil {
		logger.Error("redis close error", zap.Error(err))
	}

	logger.Info("application container stopped")
	return nil
}

// RunMigrations runs database migrations
func (c *AppContainer) RunMigrations() error {
	return c.Infrastructure.DB.AutoMigrate(
		&entity.Tenant{},
		&entity.User{},
		&entity.APIKey{},
		&entity.Model{},
		&entity.UsageRecord{},
		&entity.Bill{},
		&entity.RechargeRecord{},
		&entity.AuditLog{},
		&entity.ProviderAccount{},
	)
}

// InitializeDefaultAdmin creates default admin user and API key
func (c *AppContainer) InitializeDefaultAdmin(ctx context.Context) (*service.InitResult, error) {
	return c.Services.Init.InitializeDefaultAdmin(ctx)
}

// GetTenantRepository returns tenant repository for direct access (deprecated)
func (c *AppContainer) GetTenantRepository() repository.TenantRepository {
	return c.Repositories.Tenant
}

// GetUserRepository returns user repository for direct access (deprecated)
func (c *AppContainer) GetUserRepository() repository.UserRepository {
	return c.Repositories.User
}

// GetAPIKeyRepository returns API key repository for direct access (deprecated)
func (c *AppContainer) GetAPIKeyRepository() repository.APIKeyRepository {
	return c.Repositories.APIKey
}

// GetModelRepository returns model repository for direct access (deprecated)
func (c *AppContainer) GetModelRepository() repository.ModelRepository {
	return c.Repositories.Model
}

// GetProviderAccountRepository returns provider account repository for direct access (deprecated)
func (c *AppContainer) GetProviderAccountRepository() repository.ProviderAccountRepository {
	return c.Repositories.ProviderAccount
}

// BillingServiceFacade provides a simplified interface for billing operations
type BillingServiceFacade struct {
	billingService *service.BillingService
	asyncBilling   *service.AsyncBillingQueue
}

func NewBillingServiceFacade(billing *service.BillingService, async *service.AsyncBillingQueue) *BillingServiceFacade {
	return &BillingServiceFacade{
		billingService: billing,
		asyncBilling:   async,
	}
}

func (f *BillingServiceFacade) RecordUsageAsync(
	tenantID, userID, apiKeyID uuid.UUID,
	requestID, provider, modelID string,
	promptTokens, completionTokens int64,
	latencyMs, statusCode int,
) {
	f.asyncBilling.QueueBillingAsync(
		tenantID, userID, apiKeyID,
		requestID, provider, modelID,
		promptTokens, completionTokens,
		latencyMs, statusCode,
	)
}

func (f *BillingServiceFacade) RecordUsageSync(
	ctx context.Context,
	tenantID, userID, apiKeyID uuid.UUID,
	requestID, provider, modelID string,
	promptTokens, completionTokens int64,
	latencyMs, statusCode int,
) (*entity.UsageRecord, error) {
	return f.billingService.RecordUsage(ctx,
		tenantID, userID, apiKeyID,
		requestID, provider, modelID,
		promptTokens, completionTokens,
		latencyMs, statusCode,
	)
}

func (f *BillingServiceFacade) CheckBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return f.billingService.CheckBalance(ctx, tenantID)
}

func (f *BillingServiceFacade) CalculateCost(ctx context.Context, provider, modelID string, promptTokens, completionTokens int64) (decimal.Decimal, error) {
	return f.billingService.CalculateCost(ctx, provider, modelID, promptTokens, completionTokens)
}