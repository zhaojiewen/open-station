package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// PlatformAuthService provides authentication for platform admins
type PlatformAuthService struct {
	adminRepo  repository.PlatformAdminRepository
	cache      map[uuid.UUID]*cachedPlatformAdmin // Simple in-memory cache
	cacheMutex sync.RWMutex
}

type cachedPlatformAdmin struct {
	admin      *entity.PlatformAdmin
	permissions []string
	expiry     time.Time
}

// NewPlatformAuthService creates a new platform auth service
func NewPlatformAuthService(adminRepo repository.PlatformAdminRepository) *PlatformAuthService {
	return &PlatformAuthService{
		adminRepo: adminRepo,
		cache:     make(map[uuid.UUID]*cachedPlatformAdmin),
	}
}

// Login authenticates a platform admin and returns a session token
func (s *PlatformAuthService) Login(ctx context.Context, email, password string) (*entity.PlatformAdmin, string, error) {
	admin, err := s.adminRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", apperrors.ErrPlatformAdminNotFound
	}

	if admin.Status != "active" {
		return nil, "", apperrors.ErrPlatformAdminInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, "", apperrors.ErrUnauthorized
	}

	// Generate session token
	token, err := generateSessionToken()
	if err != nil {
		return nil, "", apperrors.ErrInternal
	}

	// Update last login
	s.adminRepo.UpdateLastLogin(ctx, admin.ID)

	// Cache admin info
	s.cacheAdmin(admin)

	return admin, token, nil
}

// ValidateSession validates a platform admin session token
func (s *PlatformAuthService) ValidateSession(ctx context.Context, adminID uuid.UUID) (*entity.PlatformAdmin, error) {
	// Check cache first
	cached := s.getCachedAdmin(adminID)
	if cached != nil && time.Now().Before(cached.expiry) {
		return cached.admin, nil
	}

	// Fetch from database
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, apperrors.ErrPlatformAdminNotFound
	}

	if admin.Status != "active" {
		return nil, apperrors.ErrPlatformAdminInactive
	}

	// Cache admin info
	s.cacheAdmin(admin)

	return admin, nil
}

// CheckPermission checks if a platform admin has a specific permission
func (s *PlatformAuthService) CheckPermission(ctx context.Context, adminID uuid.UUID, permission string) (bool, error) {
	// Check cache first
	cached := s.getCachedAdmin(adminID)
	if cached != nil && time.Now().Before(cached.expiry) {
		for _, p := range cached.permissions {
			if p == permission || p == "*" {
				return true, nil
			}
		}
		return false, nil
	}

	// Check from repository
	return s.adminRepo.CheckPermission(ctx, adminID, permission)
}

// HasRole checks if admin has a specific role
func (s *PlatformAuthService) HasRole(ctx context.Context, adminID uuid.UUID, role string) (bool, error) {
	admin, err := s.ValidateSession(ctx, adminID)
	if err != nil {
		return false, err
	}

	return admin.Role == role, nil
}

// IsSuperAdmin checks if admin is super admin
func (s *PlatformAuthService) IsSuperAdmin(ctx context.Context, adminID uuid.UUID) (bool, error) {
	return s.HasRole(ctx, adminID, "super_admin")
}

// CreateAdmin creates a new platform admin
func (s *PlatformAuthService) CreateAdmin(ctx context.Context, email, password, name, role string, permissions []string) (*entity.PlatformAdmin, error) {
	// Check if email already exists
	existing, err := s.adminRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, apperrors.ErrPlatformAdminExists
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	// Serialize permissions
	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	admin := &entity.PlatformAdmin{
		Email:        email,
		PasswordHash: string(passwordHash),
		Name:         name,
		Role:         role,
		Permissions:  string(permsJSON),
		Status:       "active",
	}

	if err := s.adminRepo.Create(ctx, admin); err != nil {
		return nil, err
	}

	return admin, nil
}

// UpdateAdmin updates a platform admin
func (s *PlatformAuthService) UpdateAdmin(ctx context.Context, adminID uuid.UUID, updates map[string]interface{}) error {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return apperrors.ErrPlatformAdminNotFound
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		admin.Name = name
	}
	if role, ok := updates["role"].(string); ok {
		admin.Role = role
	}
	if status, ok := updates["status"].(string); ok {
		admin.Status = status
	}
	if permissions, ok := updates["permissions"].([]string); ok {
		permsJSON, _ := json.Marshal(permissions)
		admin.Permissions = string(permsJSON)
	}
	if password, ok := updates["password"].(string); ok {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return apperrors.ErrInternal
		}
		admin.PasswordHash = string(passwordHash)
	}

	// Invalidate cache
	s.invalidateCache(adminID)

	return s.adminRepo.Update(ctx, admin)
}

// DeleteAdmin deletes a platform admin
func (s *PlatformAuthService) DeleteAdmin(ctx context.Context, adminID uuid.UUID) error {
	// Invalidate cache
	s.invalidateCache(adminID)

	return s.adminRepo.Delete(ctx, adminID)
}

// ListAdmins lists all platform admins
func (s *PlatformAuthService) ListAdmins(ctx context.Context, page, pageSize int) ([]entity.PlatformAdmin, int64, error) {
	return s.adminRepo.List(ctx, page, pageSize)
}

// Helper functions

func (s *PlatformAuthService) cacheAdmin(admin *entity.PlatformAdmin) {
	var permissions []string
	if admin.Permissions != "" {
		json.Unmarshal([]byte(admin.Permissions), &permissions)
	}

	s.cacheMutex.Lock()
	s.cache[admin.ID] = &cachedPlatformAdmin{
		admin:      admin,
		permissions: permissions,
		expiry:     time.Now().Add(5 * time.Minute),
	}
	s.cacheMutex.Unlock()
}

func (s *PlatformAuthService) getCachedAdmin(id uuid.UUID) *cachedPlatformAdmin {
	s.cacheMutex.RLock()
	cached := s.cache[id]
	s.cacheMutex.RUnlock()
	return cached
}

func (s *PlatformAuthService) invalidateCache(id uuid.UUID) {
	s.cacheMutex.Lock()
	delete(s.cache, id)
	s.cacheMutex.Unlock()
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}