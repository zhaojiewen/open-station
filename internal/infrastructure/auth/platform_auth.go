package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/internal/domain/role"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// PlatformAuthService provides authentication for platform admins
type PlatformAuthService struct {
	adminRepo    repository.PlatformAdminRepository
	auditLogRepo repository.AuditLogRepository
	cache      map[uuid.UUID]*cachedPlatformAdmin // Simple in-memory cache
	cacheMutex sync.RWMutex
	sessionTokens map[string]uuid.UUID
	sessionMutex  sync.RWMutex
}

type cachedPlatformAdmin struct {
	admin      *entity.PlatformAdmin
	permissions []string
	expiry     time.Time
}

// NewPlatformAuthService creates a new platform auth service
func NewPlatformAuthService(adminRepo repository.PlatformAdminRepository, auditLogRepo repository.AuditLogRepository) *PlatformAuthService {
	return &PlatformAuthService{
		adminRepo:     adminRepo,
		auditLogRepo:  auditLogRepo,
		cache:         make(map[uuid.UUID]*cachedPlatformAdmin),
		sessionTokens: make(map[string]uuid.UUID),
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

	// Store session token mapping
	s.sessionMutex.Lock()
	s.sessionTokens[token] = admin.ID
	s.sessionMutex.Unlock()

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

// ValidateToken validates a session token and returns the platform admin.
func (s *PlatformAuthService) ValidateToken(ctx context.Context, token string) (*entity.PlatformAdmin, error) {
	s.sessionMutex.RLock()
	adminID, ok := s.sessionTokens[token]
	s.sessionMutex.RUnlock()
	if !ok {
		return nil, apperrors.ErrUnauthorized
	}
	return s.ValidateSession(ctx, adminID)
}

// Logout invalidates a session token.
func (s *PlatformAuthService) Logout(token string) {
	s.sessionMutex.Lock()
	delete(s.sessionTokens, token)
	s.sessionMutex.Unlock()
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
	return s.HasRole(ctx, adminID, role.PlatformRoleSuperAdmin)
}

// CreateAdmin creates a new platform admin
func (s *PlatformAuthService) CreateAdmin(ctx context.Context, actorID uuid.UUID, email, password, name, adminRole string, permissions []string) (*entity.PlatformAdmin, error) {
	if !role.IsValidPlatformRole(adminRole) {
		return nil, apperrors.ErrInvalidPlatformRole
	}

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

	// Serialize permissions (use defaults from PlatformRolePermissions if none provided)
	if len(permissions) == 0 {
		if defaults, ok := role.PlatformRolePermissions[adminRole]; ok {
			permissions = defaults
		}
	}
	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	admin := &entity.PlatformAdmin{
		Email:        email,
		PasswordHash: string(passwordHash),
		Name:         name,
		Role:         adminRole,
		Permissions:  string(permsJSON),
		Status:       "active",
	}

	if err := s.adminRepo.Create(ctx, admin); err != nil {
		return nil, err
	}

	s.auditLog(ctx, entity.AuditLog{
		UserID:       actorID,
		Action:       "platform_admin_created",
		ResourceType: "platform_admin",
		ResourceID:   admin.ID,
		NewValues:    toJSON(map[string]interface{}{"email": email, "name": name, "role": adminRole}),
	})

	return admin, nil
}

// UpdateAdmin updates a platform admin
func (s *PlatformAuthService) UpdateAdmin(ctx context.Context, actorID uuid.UUID, adminID uuid.UUID, updates map[string]interface{}) error {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return apperrors.ErrPlatformAdminNotFound
	}

	oldValues := map[string]interface{}{
		"name":        admin.Name,
		"role":        admin.Role,
		"status":      admin.Status,
		"permissions": admin.Permissions,
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		admin.Name = name
	}
	if newRole, ok := updates["role"].(string); ok {
		if !role.IsValidPlatformRole(newRole) {
			return apperrors.ErrInvalidPlatformRole
		}
		// Prevent self-demotion of last super admin
		if role.IsSuperAdmin(admin.Role) && !role.IsSuperAdmin(newRole) {
			count, err := s.superAdminCount(ctx)
			if err == nil && count <= 1 {
				return apperrors.ErrCannotDemoteLastSuperAdmin
			}
		}
		admin.Role = newRole
	}
	if status, ok := updates["status"].(string); ok {
		// Prevent self-status-change to inactive/suspended
		if actorID == adminID && (status == "inactive" || status == "suspended") {
			return apperrors.ErrCannotDeleteSelf
		}
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

	if err := s.adminRepo.Update(ctx, admin); err != nil {
		return err
	}

	s.auditLog(ctx, entity.AuditLog{
		UserID:       actorID,
		Action:       "platform_admin_updated",
		ResourceType: "platform_admin",
		ResourceID:   adminID,
		OldValues:    toJSON(oldValues),
		NewValues:    toJSON(updates),
	})

	return nil
}

// DeleteAdmin deletes a platform admin
func (s *PlatformAuthService) DeleteAdmin(ctx context.Context, actorID uuid.UUID, adminID uuid.UUID) error {
	if actorID == adminID {
		return apperrors.ErrCannotDeleteSelf
	}

	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return apperrors.ErrPlatformAdminNotFound
	}

	// Prevent deletion of last super admin
	if role.IsSuperAdmin(admin.Role) {
		count, err := s.superAdminCount(ctx)
		if err == nil && count <= 1 {
			return apperrors.ErrCannotDeleteLastSuperAdmin
		}
	}

	// Invalidate cache
	s.invalidateCache(adminID)

	if err := s.adminRepo.Delete(ctx, adminID); err != nil {
		return err
	}

	s.auditLog(ctx, entity.AuditLog{
		UserID:       actorID,
		Action:       "platform_admin_deleted",
		ResourceType: "platform_admin",
		ResourceID:   adminID,
		OldValues:    toJSON(map[string]interface{}{"email": admin.Email, "name": admin.Name, "role": admin.Role}),
	})

	return nil
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

func (s *PlatformAuthService) superAdminCount(ctx context.Context) (int, error) {
	admins, _, err := s.adminRepo.List(ctx, 1, 1000)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, a := range admins {
		if role.IsSuperAdmin(a.Role) && a.Status == "active" {
			count++
		}
	}
	return count, nil
}

func (s *PlatformAuthService) auditLog(ctx context.Context, entry entity.AuditLog) {
	go func() {
		if s.auditLogRepo == nil {
			return
		}
		if err := s.auditLogRepo.Create(context.Background(), &entry); err != nil {
			log.Printf("platform_admin: failed to write audit log: %v", err)
		}
	}()
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}