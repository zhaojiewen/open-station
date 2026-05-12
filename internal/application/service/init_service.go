package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/role"
	"github.com/zhaojiewen/open-station/pkg/config"
	"golang.org/x/crypto/bcrypt"
)

type InitService struct {
	tenantRepo    TenantRepository
	userRepo      UserRepository
	apiKeyRepo    APIKeyRepository
	cfg           *config.AdminConfig
}

type TenantRepository interface {
	Create(ctx context.Context, tenant *entity.Tenant) error
	GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error)
}

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
}

type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *entity.APIKey) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.APIKey, error)
}

type InitResult struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	APIKeyID    uuid.UUID
	APIKeyRaw   string
	IsNewSetup  bool
	WarningMsg  string
}

func NewInitService(
	tenantRepo TenantRepository,
	userRepo UserRepository,
	apiKeyRepo APIKeyRepository,
	cfg *config.AdminConfig,
) *InitService {
	return &InitService{
		tenantRepo:    tenantRepo,
		userRepo:      userRepo,
		apiKeyRepo:    apiKeyRepo,
		cfg:           cfg,
	}
}

func (s *InitService) InitializeDefaultAdmin(ctx context.Context) (*InitResult, error) {
	result := &InitResult{}

	// Check if tenant already exists
	tenant, err := s.tenantRepo.GetBySlug(ctx, s.cfg.DefaultTenantSlug)
	if err == nil {
		// Tenant exists, check for existing API keys
		result.TenantID = tenant.ID
		result.IsNewSetup = false

		keys, err := s.apiKeyRepo.ListByTenant(ctx, tenant.ID, "active")
		if err == nil && len(keys) > 0 {
			// Already has active keys, no need to create
			result.WarningMsg = "Default tenant already configured with active API keys"
			return result, nil
		}
	}

	// Create tenant if not exists
	if tenant == nil {
		tenantID := uuid.New()
		tenant = &entity.Tenant{
			ID:        tenantID,
			Slug:      s.cfg.DefaultTenantSlug,
			Name:      "Admin Tenant",
			Status:    "active",
			Balance:   decimal.NewFromInt(1000), // Initial balance
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.tenantRepo.Create(ctx, tenant); err != nil {
			return nil, fmt.Errorf("failed to create default tenant: %w", err)
		}
		result.TenantID = tenant.ID
	}

	// Create admin user
	userID := uuid.New()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.DefaultAdminPass), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &entity.User{
		ID:           userID,
		TenantID:     tenant.ID,
		Email:        s.cfg.SuperAdminEmail,
		Name:         s.cfg.DefaultAdminUser,
		Role:         role.TenantRoleAdmin,
		Status:       "active",
		PasswordHash: string(passwordHash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		// User might already exist, try to get it
		existingUser, getErr := s.userRepo.GetByEmail(ctx, s.cfg.SuperAdminEmail)
		if getErr != nil {
			return nil, fmt.Errorf("failed to create/get default admin user: %w", err)
		}
		user = existingUser
		userID = user.ID
	}
	result.UserID = userID

	// Generate API key
	apiKeyID := uuid.New()
	rawKey := generateAPIKey()
	keyHash := sha256.Sum256([]byte(rawKey))
	keyHashStr := hex.EncodeToString(keyHash[:])
	keyPrefix := rawKey[:12]

	permissionsJSON, _ := json.Marshal([]string{role.PermAdmin, role.PermManage, role.PermChat})
	emptyJSON, _ := json.Marshal([]string{})

	apiKey := &entity.APIKey{
		ID:             apiKeyID,
		TenantID:       tenant.ID,
		UserID:         userID,
		Name:           s.cfg.InitialAPIKeyName,
		KeyHash:        keyHashStr,
		KeyPrefix:      keyPrefix,
		Status:         "active",
		Permissions:    string(permissionsJSON),
		AllowedModels:  string(emptyJSON),
		AllowedProviders: string(emptyJSON),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create initial API key: %w", err)
	}

	result.APIKeyID = apiKeyID
	result.APIKeyRaw = rawKey
	result.IsNewSetup = true

	result.WarningMsg = fmt.Sprintf(`
================================================================================
                        SECURITY WARNING
================================================================================
Default admin credentials have been created!

Username: %s
Password: %s
API Key:  %s

PLEASE CHANGE THESE IMMEDIATELY:
1. Update config.yaml with new credentials
2. Create a new API key via admin panel
3. Revoke this initial API key after creating new one

Command to revoke this key:
  curl -X POST http://localhost:8080/admin/api-keys/%s/revoke \
    -H "Authorization: Bearer %s"
================================================================================
`, s.cfg.DefaultAdminUser, s.cfg.DefaultAdminPass, rawKey, apiKeyID, rawKey)

	return result, nil
}

func generateAPIKey() string {
	b := make([]byte, 48)
	rand.Read(b)
	return "sk-" + hex.EncodeToString(b)
}