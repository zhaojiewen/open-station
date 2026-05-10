package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

const (
	KeyPrefix = "sk-"
	KeyLength = 48
	CacheTTL  = 5 * time.Minute
)

type cachedPermissions struct {
	permissions      []string
	allowedModels    []string
	allowedProviders []string
}

type APIKeyValidator struct {
	apiKeyRepo repository.APIKeyRepository
	redis      *redis.Client
}

func NewAPIKeyValidator(apiKeyRepo repository.APIKeyRepository, redisClient *redis.Client) *APIKeyValidator {
	return &APIKeyValidator{
		apiKeyRepo: apiKeyRepo,
		redis:      redisClient,
	}
}

func (v *APIKeyValidator) Validate(ctx context.Context, apiKey string) (*entity.APIKey, error) {
	if apiKey == "" {
		return nil, apperrors.ErrInvalidAPIKey
	}

	keyHash := HashAPIKey(apiKey)

	cacheKey := fmt.Sprintf("apikey:%s", keyHash)
	cached, err := v.redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		apiKeyID, err := uuid.Parse(cached)
		if err == nil {
			key, err := v.apiKeyRepo.GetByID(ctx, apiKeyID)
			if err == nil {
				logger.Debug("api key found in cache", zap.String("key_prefix", key.KeyPrefix))
				return key, nil
			}
		}
	}

	key, err := v.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		logger.Warn("api key not found", zap.String("key_hash", keyHash[:8]+"..."))
		return nil, apperrors.ErrInvalidAPIKey
	}

	if key.Status == "revoked" {
		return nil, apperrors.ErrAPIKeyRevoked
	}

	if key.Status == "expired" {
		return nil, apperrors.ErrAPIKeyExpired
	}

	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, apperrors.ErrAPIKeyExpired
	}

	v.redis.Set(ctx, cacheKey, key.ID.String(), CacheTTL)
	logger.Debug("api key validated", zap.String("key_prefix", key.KeyPrefix))

	return key, nil
}

func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

func GenerateAPIKey() (string, string, string, error) {
	bytes := make([]byte, KeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	key := KeyPrefix + hex.EncodeToString(bytes)
	keyHash := HashAPIKey(key)
	keyPrefix := key[:12]

	return key, keyHash, keyPrefix, nil
}

type AuthService struct {
	apiKeyRepo       repository.APIKeyRepository
	userRepo         repository.UserRepository
	tenantRepo       repository.TenantRepository
	validator        *APIKeyValidator
	permissionsCache sync.Map
}

func NewAuthService(
	apiKeyRepo repository.APIKeyRepository,
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
	redisClient *redis.Client,
) *AuthService {
	return &AuthService{
		apiKeyRepo:  apiKeyRepo,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		validator:   NewAPIKeyValidator(apiKeyRepo, redisClient),
	}
}

func (s *AuthService) ValidateAPIKey(ctx context.Context, apiKey string) (*entity.APIKey, *entity.User, *entity.Tenant, error) {
	keyHash := HashAPIKey(apiKey)

	cacheKey := fmt.Sprintf("apikey:%s", keyHash)
	cached, err := s.validator.redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		key, user, tenant, err := s.apiKeyRepo.GetWithRelations(ctx, keyHash)
		if err == nil && user != nil && tenant != nil {
			logger.Debug("api key with relations found via cache", zap.String("key_prefix", key.KeyPrefix))
			return key, user, tenant, nil
		}
	}

	key, user, tenant, err := s.apiKeyRepo.GetWithRelations(ctx, keyHash)
	if err != nil {
		logger.Warn("api key not found", zap.String("key_hash", keyHash[:8]+"..."))
		return nil, nil, nil, apperrors.ErrInvalidAPIKey
	}

	if key.Status == "revoked" {
		return nil, nil, nil, apperrors.ErrAPIKeyRevoked
	}

	if key.Status == "expired" {
		return nil, nil, nil, apperrors.ErrAPIKeyExpired
	}

	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, nil, nil, apperrors.ErrAPIKeyExpired
	}

	s.validator.redis.Set(ctx, cacheKey, key.ID.String(), CacheTTL)
	logger.Debug("api key validated with relations", zap.String("key_prefix", key.KeyPrefix))

	return key, user, tenant, nil
}

func (s *AuthService) CreateAPIKey(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, name string, permissions []string, allowedModels []string, allowedProviders []string, expiresAt *time.Time, monthlyTokenLimit *int64) (*entity.APIKey, string, error) {
	key, keyHash, keyPrefix, err := GenerateAPIKey()
	if err != nil {
		return nil, "", err
	}

	permissionsJSON, _ := json.Marshal(permissions)
	modelsJSON, _ := json.Marshal(allowedModels)
	providersJSON, _ := json.Marshal(allowedProviders)

	apiKey := &entity.APIKey{
		UserID:            userID,
		TenantID:          tenantID,
		KeyHash:           keyHash,
		KeyPrefix:         keyPrefix,
		Name:              name,
		Permissions:       string(permissionsJSON),
		AllowedModels:     string(modelsJSON),
		AllowedProviders:  string(providersJSON),
		Status:            "active",
		ExpiresAt:         expiresAt,
		MonthlyTokenLimit: monthlyTokenLimit,
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("failed to create api key: %w", err)
	}

	return apiKey, key, nil
}

func (s *AuthService) RevokeAPIKey(ctx context.Context, apiKeyID uuid.UUID) error {
	s.permissionsCache.Delete(apiKeyID)
	return s.apiKeyRepo.Revoke(ctx, apiKeyID)
}

func (s *AuthService) getCachedPermissions(apiKey *entity.APIKey) *cachedPermissions {
	cached, ok := s.permissionsCache.Load(apiKey.ID)
	if ok {
		return cached.(*cachedPermissions)
	}

	cp := &cachedPermissions{}
	if apiKey.Permissions != "" {
		json.Unmarshal([]byte(apiKey.Permissions), &cp.permissions)
	}
	if apiKey.AllowedModels != "" {
		json.Unmarshal([]byte(apiKey.AllowedModels), &cp.allowedModels)
	}
	if apiKey.AllowedProviders != "" {
		json.Unmarshal([]byte(apiKey.AllowedProviders), &cp.allowedProviders)
	}

	s.permissionsCache.Store(apiKey.ID, cp)
	return cp
}

func (s *AuthService) CheckPermission(apiKey *entity.APIKey, permission string) bool {
	cp := s.getCachedPermissions(apiKey)
	for _, p := range cp.permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (s *AuthService) CheckModelAccess(apiKey *entity.APIKey, model string) bool {
	cp := s.getCachedPermissions(apiKey)
	if len(cp.allowedModels) == 0 {
		return true
	}
	for _, m := range cp.allowedModels {
		if m == model {
			return true
		}
	}
	return false
}

func (s *AuthService) CheckProviderAccess(apiKey *entity.APIKey, provider string) bool {
	cp := s.getCachedPermissions(apiKey)
	if len(cp.allowedProviders) == 0 {
		return true
	}
	for _, p := range cp.allowedProviders {
		if p == provider {
			return true
		}
	}
	return false
}

// GetTenantByID retrieves a tenant by ID
func (s *AuthService) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*entity.Tenant, error) {
	return s.tenantRepo.GetByID(ctx, tenantID)
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// GetAPIKeyByID retrieves an API key by ID
func (s *AuthService) GetAPIKeyByID(ctx context.Context, apiKeyID uuid.UUID) (*entity.APIKey, error) {
	return s.apiKeyRepo.GetByID(ctx, apiKeyID)
}

// ListAPIKeys retrieves all API keys for a user
func (s *AuthService) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]entity.APIKey, error) {
	return s.apiKeyRepo.List(ctx, userID)
}

// ListAPIKeysByTenant retrieves all API keys for a tenant with optional status filter
func (s *AuthService) ListAPIKeysByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.APIKey, error) {
	return s.apiKeyRepo.ListByTenant(ctx, tenantID, status)
}

// ListAllAPIKeys retrieves all API keys (admin only)
func (s *AuthService) ListAllAPIKeys(ctx context.Context) ([]entity.APIKey, error) {
	return s.apiKeyRepo.ListAll(ctx)
}

// UpdateAPIKeyTokenUsage updates the token usage for an API key
func (s *AuthService) UpdateAPIKeyTokenUsage(ctx context.Context, apiKeyID uuid.UUID, tokens int64) error {
	s.permissionsCache.Delete(apiKeyID)
	return s.apiKeyRepo.UpdateTokenUsage(ctx, apiKeyID, tokens)
}

// UpdateAPIKeyLastUsed updates the last used timestamp for an API key
func (s *AuthService) UpdateAPIKeyLastUsed(ctx context.Context, apiKeyID uuid.UUID) error {
	return s.apiKeyRepo.UpdateLastUsed(ctx, apiKeyID)
}

// UpdateAPIKey updates an existing API key
func (s *AuthService) UpdateAPIKey(ctx context.Context, apiKey *entity.APIKey) error {
	s.permissionsCache.Delete(apiKey.ID)
	return s.apiKeyRepo.Update(ctx, apiKey)
}

// CheckTokenLimit checks if an API key has exceeded its monthly token limit
func (s *AuthService) CheckTokenLimit(apiKey *entity.APIKey) bool {
	if apiKey.MonthlyTokenLimit == nil {
		return true // No limit set
	}
	return apiKey.UsedTokensThisMonth <= *apiKey.MonthlyTokenLimit
}

// GetAPIKeysByTenant retrieves API keys for a tenant
func (s *AuthService) GetAPIKeysByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.APIKey, error) {
	return s.apiKeyRepo.ListByTenant(ctx, tenantID, status)
}

// GetUserByEmail retrieves a user by email
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// ListUsers retrieves users for a tenant with pagination
func (s *AuthService) ListUsers(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error) {
	return s.userRepo.List(ctx, tenantID, page, pageSize)
}

// CreateUser creates a new user
func (s *AuthService) CreateUser(ctx context.Context, user *entity.User) error {
	return s.userRepo.Create(ctx, user)
}

// UpdateUser updates an existing user
func (s *AuthService) UpdateUser(ctx context.Context, user *entity.User) error {
	return s.userRepo.Update(ctx, user)
}

// ListTenants retrieves all tenants with pagination
func (s *AuthService) ListTenants(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	return s.tenantRepo.List(ctx, page, pageSize)
}

// UpdateTenantBalance updates tenant balance
func (s *AuthService) UpdateTenantBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
	return s.tenantRepo.UpdateBalance(ctx, tenantID, amount)
}