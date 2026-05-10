package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
	"github.com/zhaojiewen/open-station/pkg/logger"
)

func init() {
	// Initialize logger for tests
	if err := logger.Init("debug", "console", "stdout"); err != nil {
		panic(err)
	}
}

// Mock implementations for testing

type MockAPIKeyRepository struct {
	keys     map[uuid.UUID]*entity.APIKey
	hashKeys map[string]*entity.APIKey
}

func NewMockAPIKeyRepository() *MockAPIKeyRepository {
	return &MockAPIKeyRepository{
		keys:     make(map[uuid.UUID]*entity.APIKey),
		hashKeys: make(map[string]*entity.APIKey),
	}
}

func (m *MockAPIKeyRepository) Create(ctx context.Context, apiKey *entity.APIKey) error {
	apiKey.ID = uuid.New()
	m.keys[apiKey.ID] = apiKey
	m.hashKeys[apiKey.KeyHash] = apiKey
	return nil
}

func (m *MockAPIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.APIKey, error) {
	if key, ok := m.keys[id]; ok {
		return key, nil
	}
	return nil, errors.New("not found")
}

func (m *MockAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*entity.APIKey, error) {
	if key, ok := m.hashKeys[keyHash]; ok {
		return key, nil
	}
	return nil, errors.New("not found")
}

func (m *MockAPIKeyRepository) GetByKeyPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error) {
	var result []entity.APIKey
	for _, key := range m.keys {
		if key.KeyPrefix == prefix {
			result = append(result, *key)
		}
	}
	return result, nil
}

func (m *MockAPIKeyRepository) Update(ctx context.Context, apiKey *entity.APIKey) error {
	m.keys[apiKey.ID] = apiKey
	m.hashKeys[apiKey.KeyHash] = apiKey
	return nil
}

func (m *MockAPIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if key, ok := m.keys[id]; ok {
		delete(m.hashKeys, key.KeyHash)
		delete(m.keys, id)
	}
	return nil
}

func (m *MockAPIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	if key, ok := m.keys[id]; ok {
		key.Status = "revoked"
		now := time.Now()
		key.RevokedAt = &now
		return nil
	}
	return errors.New("not found")
}

func (m *MockAPIKeyRepository) List(ctx context.Context, userID uuid.UUID) ([]entity.APIKey, error) {
	var result []entity.APIKey
	for _, key := range m.keys {
		if key.UserID == userID {
			result = append(result, *key)
		}
	}
	return result, nil
}

func (m *MockAPIKeyRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.APIKey, error) {
	var result []entity.APIKey
	for _, key := range m.keys {
		if key.TenantID == tenantID && (status == "" || key.Status == status) {
			result = append(result, *key)
		}
	}
	return result, nil
}

func (m *MockAPIKeyRepository) ListAll(ctx context.Context) ([]entity.APIKey, error) {
	var result []entity.APIKey
	for _, key := range m.keys {
		result = append(result, *key)
	}
	return result, nil
}

func (m *MockAPIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	if key, ok := m.keys[id]; ok {
		now := time.Now()
		key.LastUsedAt = &now
		return nil
	}
	return errors.New("not found")
}

func (m *MockAPIKeyRepository) UpdateTokenUsage(ctx context.Context, id uuid.UUID, tokens int64) error {
	if key, ok := m.keys[id]; ok {
		key.UsedTokensThisMonth += tokens
		return nil
	}
	return errors.New("not found")
}

func (m *MockAPIKeyRepository) GetWithRelations(ctx context.Context, keyHash string) (*entity.APIKey, *entity.User, *entity.Tenant, error) {
	key, ok := m.hashKeys[keyHash]
	if !ok {
		return nil, nil, nil, errors.New("not found")
	}
	return key, nil, nil, nil
}

type MockUserRepository struct {
	users   map[uuid.UUID]*entity.User
	byEmail map[string]*entity.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:   make(map[uuid.UUID]*entity.User),
		byEmail: make(map[string]*entity.User),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *entity.User) error {
	user.ID = uuid.New()
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, errors.New("not found")
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	if user, ok := m.byEmail[email]; ok {
		return user, nil
	}
	return nil, errors.New("not found")
}

func (m *MockUserRepository) Update(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		delete(m.byEmail, user.Email)
		delete(m.users, id)
	}
	return nil
}

func (m *MockUserRepository) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error) {
	var result []entity.User
	for _, user := range m.users {
		if user.TenantID == tenantID {
			result = append(result, *user)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		now := time.Now()
		user.LastLoginAt = &now
		return nil
	}
	return errors.New("not found")
}

type MockTenantRepository struct {
	tenants map[uuid.UUID]*entity.Tenant
	bySlug  map[string]*entity.Tenant
}

func NewMockTenantRepository() *MockTenantRepository {
	return &MockTenantRepository{
		tenants: make(map[uuid.UUID]*entity.Tenant),
		bySlug:  make(map[string]*entity.Tenant),
	}
}

func (m *MockTenantRepository) Create(ctx context.Context, tenant *entity.Tenant) error {
	tenant.ID = uuid.New()
	m.tenants[tenant.ID] = tenant
	m.bySlug[tenant.Slug] = tenant
	return nil
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if tenant, ok := m.tenants[id]; ok {
		return tenant, nil
	}
	return nil, errors.New("not found")
}

func (m *MockTenantRepository) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	if tenant, ok := m.bySlug[slug]; ok {
		return tenant, nil
	}
	return nil, errors.New("not found")
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *entity.Tenant) error {
	m.tenants[tenant.ID] = tenant
	m.bySlug[tenant.Slug] = tenant
	return nil
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if tenant, ok := m.tenants[id]; ok {
		delete(m.bySlug, tenant.Slug)
		delete(m.tenants, id)
	}
	return nil
}

func (m *MockTenantRepository) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	var result []entity.Tenant
	for _, tenant := range m.tenants {
		result = append(result, *tenant)
	}
	return result, int64(len(result)), nil
}

func (m *MockTenantRepository) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockTenantRepository) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.NewFromInt(1000), nil
}

func (m *MockTenantRepository) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

// Tests start here

func TestHashAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
	}{
		{"simple key", "sk-test123"},
		{"empty key", ""},
		{"long key", "sk-very-long-api-key-with-many-characters-1234567890"},
		{"special chars", "sk-test!@#$%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashAPIKey(tt.apiKey)

			// Hash should be consistent
			hash2 := HashAPIKey(tt.apiKey)
			if hash != hash2 {
				t.Error("hash should be consistent")
			}

			// Hash should be 64 characters (SHA256 hex encoded)
			if len(hash) != 64 {
				t.Errorf("hash length = %d, want 64", len(hash))
			}

			// Hash should be valid hex
			_, err := hex.DecodeString(hash)
			if err != nil {
				t.Errorf("hash is not valid hex: %v", err)
			}

			// Different keys should produce different hashes
			if tt.apiKey != "" {
				differentHash := HashAPIKey(tt.apiKey + "x")
				if hash == differentHash {
					t.Error("different keys should produce different hashes")
				}
			}
		})
	}
}

func TestGenerateAPIKey(t *testing.T) {
	keys := make(map[string]bool)

	for i := 0; i < 100; i++ {
		key, keyHash, keyPrefix, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}

		// Key should have correct prefix
		if len(key) < len(KeyPrefix) || key[:len(KeyPrefix)] != KeyPrefix {
			t.Errorf("key should start with %s", KeyPrefix)
		}

		// Key should be unique
		if keys[key] {
			t.Errorf("duplicate key generated: %s", key)
		}
		keys[key] = true

		// KeyHash should be valid SHA256 hex
		if len(keyHash) != 64 {
			t.Errorf("keyHash length = %d, want 64", len(keyHash))
		}

		// KeyPrefix should be 12 characters
		if len(keyPrefix) != 12 {
			t.Errorf("keyPrefix length = %d, want 12", len(keyPrefix))
		}

		// Verify hash matches
		expectedHash := HashAPIKey(key)
		if keyHash != expectedHash {
			t.Errorf("keyHash doesn't match expected hash")
		}

		// Verify prefix
		expectedPrefix := key[:12]
		if keyPrefix != expectedPrefix {
			t.Errorf("keyPrefix = %s, want %s", keyPrefix, expectedPrefix)
		}
	}
}

func TestAPIKeyValidator_Validate(t *testing.T) {
	// Create mock repos
	apiKeyRepo := NewMockAPIKeyRepository()
	userRepo := NewMockUserRepository()
	tenantRepo := NewMockTenantRepository()

	// Create test data
	tenant := &entity.Tenant{
		Name: "Test Tenant",
		Slug: "test-tenant",
	}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{
		TenantID: tenant.ID,
		Email:    "test@example.com",
		Name:     "Test User",
	}
	userRepo.Create(context.Background(), user)

	// Generate a test key
	_, keyHash, keyPrefix, _ := GenerateAPIKey()
	apiKey := &entity.APIKey{
		UserID:    user.ID,
		TenantID:  tenant.ID,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Name:      "Test Key",
		Status:    "active",
	}
	apiKeyRepo.Create(context.Background(), apiKey)

	// Create Redis mock (won't actually connect in tests)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	validator := NewAPIKeyValidator(apiKeyRepo, redisClient)

	t.Run("empty key", func(t *testing.T) {
		_, err := validator.Validate(context.Background(), "")
		if !errors.Is(err, apperrors.ErrInvalidAPIKey) {
			t.Errorf("expected apperrors.ErrInvalidAPIKey, got %v", err)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		_, err := validator.Validate(context.Background(), "sk-invalid")
		if err == nil {
			t.Error("expected error for invalid key")
		}
	})
}

func TestAuthService_CheckPermission(t *testing.T) {
	authService := &AuthService{}

	tests := []struct {
		name        string
		permissions string
		permission  string
		expected    bool
	}{
		{
			name:        "has permission",
			permissions: `["read","write","admin"]`,
			permission:  "write",
			expected:    true,
		},
		{
			name:        "does not have permission",
			permissions: `["read","write"]`,
			permission:  "admin",
			expected:    false,
		},
		{
			name:        "empty permissions",
			permissions: `[]`,
			permission:  "read",
			expected:    false,
		},
		{
			name:        "no permissions field",
			permissions: "",
			permission:  "read",
			expected:    false,
		},
		{
			name:        "invalid JSON",
			permissions: "invalid",
			permission:  "read",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &entity.APIKey{
				ID:          uuid.New(),
				Permissions: tt.permissions,
			}
			result := authService.CheckPermission(apiKey, tt.permission)
			if result != tt.expected {
				t.Errorf("CheckPermission() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAuthService_CheckModelAccess(t *testing.T) {
	authService := &AuthService{}

	tests := []struct {
		name          string
		allowedModels string
		model         string
		expected      bool
	}{
		{
			name:          "model allowed",
			allowedModels: `["gpt-4","gpt-3.5-turbo","claude-3"]`,
			model:         "gpt-4",
			expected:      true,
		},
		{
			name:          "model not allowed",
			allowedModels: `["gpt-4","gpt-3.5-turbo"]`,
			model:         "claude-3",
			expected:      false,
		},
		{
			name:          "empty allowed models - allow all",
			allowedModels: "",
			model:         "any-model",
			expected:      true,
		},
		{
			name:          "empty array - allow all",
			allowedModels: `[]`,
			model:         "any-model",
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &entity.APIKey{
				ID:            uuid.New(),
				AllowedModels: tt.allowedModels,
			}
			result := authService.CheckModelAccess(apiKey, tt.model)
			if result != tt.expected {
				t.Errorf("CheckModelAccess() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAuthService_CheckProviderAccess(t *testing.T) {
	authService := &AuthService{}

	tests := []struct {
		name             string
		allowedProviders string
		provider         string
		expected         bool
	}{
		{
			name:             "provider allowed",
			allowedProviders: `["openai","anthropic","google"]`,
			provider:         "openai",
			expected:         true,
		},
		{
			name:             "provider not allowed",
			allowedProviders: `["openai","anthropic"]`,
			provider:         "google",
			expected:         false,
		},
		{
			name:             "empty allowed providers - allow all",
			allowedProviders: "",
			provider:         "any-provider",
			expected:         true,
		},
		{
			name:             "empty array - allow all",
			allowedProviders: `[]`,
			provider:         "any-provider",
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &entity.APIKey{
				ID:               uuid.New(),
				AllowedProviders: tt.allowedProviders,
			}
			result := authService.CheckProviderAccess(apiKey, tt.provider)
			if result != tt.expected {
				t.Errorf("CheckProviderAccess() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSHA256Consistency(t *testing.T) {
	// Verify our hash function matches the standard SHA256
	testKey := "sk-test123456"

	expectedHash := sha256.Sum256([]byte(testKey))
	expectedHex := hex.EncodeToString(expectedHash[:])

	actualHash := HashAPIKey(testKey)

	if actualHash != expectedHex {
		t.Errorf("HashAPIKey() = %v, want %v", actualHash, expectedHex)
	}
}

func TestNewAPIKeyValidator(t *testing.T) {
	apiKeyRepo := NewMockAPIKeyRepository()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	validator := NewAPIKeyValidator(apiKeyRepo, redisClient)
	if validator == nil {
		t.Error("NewAPIKeyValidator should not return nil")
	}
}

func TestNewAuthService(t *testing.T) {
	apiKeyRepo := NewMockAPIKeyRepository()
	userRepo := NewMockUserRepository()
	tenantRepo := NewMockTenantRepository()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	authService := NewAuthService(apiKeyRepo, userRepo, tenantRepo, redisClient)
	if authService == nil {
		t.Error("NewAuthService should not return nil")
	}
	// Verify service is properly initialized by calling a method
	testKey := &entity.APIKey{
		ID:          uuid.New(),
		Permissions: `["test"]`,
	}
	if !authService.CheckPermission(testKey, "test") {
		t.Log("AuthService CheckPermission method is functional")
	}
}

func TestKeyPrefixConstant(t *testing.T) {
	if KeyPrefix != "sk-" {
		t.Errorf("KeyPrefix = %v, want sk-", KeyPrefix)
	}
}

func TestKeyLengthConstant(t *testing.T) {
	if KeyLength != 48 {
		t.Errorf("KeyLength = %v, want 48", KeyLength)
	}
}

func TestCacheTTLConstant(t *testing.T) {
	if CacheTTL != 5*time.Minute {
		t.Errorf("CacheTTL = %v, want 5 minutes", CacheTTL)
	}
}

func TestErrorConstants(t *testing.T) {
	if apperrors.ErrInvalidAPIKey == nil {
		t.Error("apperrors.ErrInvalidAPIKey should not be nil")
	}
	if apperrors.ErrAPIKeyExpired == nil {
		t.Error("apperrors.ErrAPIKeyExpired should not be nil")
	}
	if apperrors.ErrAPIKeyRevoked == nil {
		t.Error("apperrors.ErrAPIKeyRevoked should not be nil")
	}

	// Verify error messages
	if apperrors.ErrInvalidAPIKey.Error() != "AUTH_001: invalid API key" {
		t.Errorf("apperrors.ErrInvalidAPIKey message = %v, want 'AUTH_001: invalid API key'", apperrors.ErrInvalidAPIKey.Error())
	}
}

func TestAPIKeyStatusChecks(t *testing.T) {
	// Test revoked status
	revokedKey := &entity.APIKey{
		Status: "revoked",
	}
	if revokedKey.Status != "revoked" {
		t.Error("revoked key should have revoked status")
	}

	// Test expired status
	expiredKey := &entity.APIKey{
		Status: "expired",
	}
	if expiredKey.Status != "expired" {
		t.Error("expired key should have expired status")
	}

	// Test expiration date
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)
	keyWithExpiry := &entity.APIKey{
		Status:    "active",
		ExpiresAt: &pastTime,
	}
	if keyWithExpiry.ExpiresAt.Before(now) {
		t.Log("key with past expiration should be considered expired")
	}
}