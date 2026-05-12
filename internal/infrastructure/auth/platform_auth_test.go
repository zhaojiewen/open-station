package auth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// Mock repository for PlatformAdminRepository
type mockPlatformAdminRepo struct {
	admins       map[uuid.UUID]*entity.PlatformAdmin
	adminsByEmail map[string]*entity.PlatformAdmin
	createError  error
	getError     error
	updateError  error
	deleteError  error
	checkPermResult bool
	checkPermError  error
}

func newMockPlatformAdminRepo() *mockPlatformAdminRepo {
	return &mockPlatformAdminRepo{
		admins:        make(map[uuid.UUID]*entity.PlatformAdmin),
		adminsByEmail: make(map[string]*entity.PlatformAdmin),
	}
}

func (m *mockPlatformAdminRepo) Create(ctx context.Context, admin *entity.PlatformAdmin) error {
	if m.createError != nil {
		return m.createError
	}
	admin.ID = uuid.New()
	m.admins[admin.ID] = admin
	m.adminsByEmail[admin.Email] = admin
	return nil
}

func (m *mockPlatformAdminRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.PlatformAdmin, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	admin, ok := m.admins[id]
	if !ok {
		return nil, errors.New("admin not found")
	}
	return admin, nil
}

func (m *mockPlatformAdminRepo) GetByEmail(ctx context.Context, email string) (*entity.PlatformAdmin, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	admin, ok := m.adminsByEmail[email]
	if !ok {
		return nil, errors.New("admin not found")
	}
	return admin, nil
}

func (m *mockPlatformAdminRepo) Update(ctx context.Context, admin *entity.PlatformAdmin) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.admins[admin.ID] = admin
	m.adminsByEmail[admin.Email] = admin
	return nil
}

func (m *mockPlatformAdminRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	admin, ok := m.admins[id]
	if ok {
		delete(m.adminsByEmail, admin.Email)
	}
	delete(m.admins, id)
	return nil
}

func (m *mockPlatformAdminRepo) List(ctx context.Context, page, pageSize int) ([]entity.PlatformAdmin, int64, error) {
	result := []entity.PlatformAdmin{}
	for _, admin := range m.admins {
		result = append(result, *admin)
	}
	return result, int64(len(result)), nil
}

func (m *mockPlatformAdminRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	admin, ok := m.admins[id]
	if ok {
		admin.LastLoginAt = &time.Time{}
		*admin.LastLoginAt = time.Now()
	}
	return nil
}

func (m *mockPlatformAdminRepo) CheckPermission(ctx context.Context, id uuid.UUID, permission string) (bool, error) {
	if m.checkPermError != nil {
		return false, m.checkPermError
	}
	return m.checkPermResult, nil
}

func (m *mockPlatformAdminRepo) GetPermissions(ctx context.Context, id uuid.UUID) ([]string, error) {
	admin, ok := m.admins[id]
	if !ok {
		return nil, errors.New("admin not found")
	}
	if admin.Permissions == "" {
		return []string{}, nil
	}
	var perms []string
	json.Unmarshal([]byte(admin.Permissions), &perms)
	return perms, nil
}

func TestNewPlatformAuthService(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)
	if service == nil {
		t.Error("service should not be nil")
	}
	if service.adminRepo == nil {
		t.Error("adminRepo should not be nil")
	}
	if service.cache == nil {
		t.Error("cache should be initialized")
	}
}

func TestPlatformAuthService_Login(t *testing.T) {
	password := "testpassword123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	tests := []struct {
		name        string
		setupAdmin  *entity.PlatformAdmin
		email       string
		password    string
		expectError error
	}{
		{
			name: "successful login",
			setupAdmin: &entity.PlatformAdmin{
				Email:        "admin@example.com",
				PasswordHash: string(passwordHash),
				Status:       "active",
			},
			email:       "admin@example.com",
			password:    password,
			expectError: nil,
		},
		{
			name:        "admin not found",
			setupAdmin:  nil,
			email:       "nonexistent@example.com",
			password:    "password",
			expectError: apperrors.ErrPlatformAdminNotFound,
		},
		{
			name: "inactive admin",
			setupAdmin: &entity.PlatformAdmin{
				Email:        "inactive@example.com",
				PasswordHash: string(passwordHash),
				Status:       "inactive",
			},
			email:       "inactive@example.com",
			password:    password,
			expectError: apperrors.ErrPlatformAdminInactive,
		},
		{
			name: "wrong password",
			setupAdmin: &entity.PlatformAdmin{
				Email:        "wrongpass@example.com",
				PasswordHash: string(passwordHash),
				Status:       "active",
			},
			email:       "wrongpass@example.com",
			password:    "wrongpassword",
			expectError: apperrors.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockPlatformAdminRepo()
			if tt.setupAdmin != nil {
				repo.Create(context.Background(), tt.setupAdmin)
			}

			service := NewPlatformAuthService(repo, nil)
			ctx := context.Background()

			admin, token, err := service.Login(ctx, tt.email, tt.password)

			if tt.expectError != nil {
				if err != tt.expectError {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if admin == nil {
					t.Error("admin should not be nil")
				}
				if token == "" {
					t.Error("token should not be empty")
				}
				// Check cache was populated
				if admin != nil {
					cached := service.getCachedAdmin(admin.ID)
					if cached == nil {
						t.Error("admin should be cached")
					}
				}
			}
		})
	}
}

func TestPlatformAuthService_LoginWithRepoError(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	repo.getError = errors.New("db error")
	service := NewPlatformAuthService(repo, nil)

	ctx := context.Background()
	_, _, err := service.Login(ctx, "admin@example.com", "password")
	if err != apperrors.ErrPlatformAdminNotFound {
		t.Errorf("expected ErrPlatformAdminNotFound, got %v", err)
	}
}

func TestPlatformAuthService_ValidateSession(t *testing.T) {
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	adminID := uuid.New()

	tests := []struct {
		name        string
		setupAdmin  *entity.PlatformAdmin
		setupCache  bool
		adminID     uuid.UUID
		expectError error
	}{
		{
			name: "valid from cache",
			setupAdmin: &entity.PlatformAdmin{
				ID:           adminID,
				Email:        "cached@example.com",
				PasswordHash: string(passwordHash),
				Status:       "active",
			},
			setupCache:  true,
			adminID:     adminID,
			expectError: nil,
		},
		{
			name: "valid from database",
			setupAdmin: &entity.PlatformAdmin{
				Email:        "database@example.com",
				PasswordHash: string(passwordHash),
				Status:       "active",
			},
			setupCache:  false,
			adminID:     uuid.Nil, // Will be set after create
			expectError: nil,
		},
		{
			name:        "admin not found",
			setupAdmin:  nil,
			adminID:     uuid.New(),
			expectError: apperrors.ErrPlatformAdminNotFound,
		},
		{
			name: "inactive admin",
			setupAdmin: &entity.PlatformAdmin{
				Email:        "inactive@example.com",
				PasswordHash: string(passwordHash),
				Status:       "inactive",
			},
			adminID:     uuid.Nil,
			expectError: apperrors.ErrPlatformAdminInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockPlatformAdminRepo()
			service := NewPlatformAuthService(repo, nil)
			ctx := context.Background()

			var testAdminID uuid.UUID
			if tt.setupAdmin != nil {
				repo.Create(ctx, tt.setupAdmin)
				testAdminID = tt.setupAdmin.ID
				if tt.setupCache {
					service.cacheAdmin(tt.setupAdmin)
				}
			} else {
				testAdminID = tt.adminID
			}

			admin, err := service.ValidateSession(ctx, testAdminID)

			if tt.expectError != nil {
				if err != tt.expectError {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if admin == nil {
					t.Error("admin should not be nil")
				}
			}
		})
	}
}

func TestPlatformAuthService_ValidateSessionWithExpiredCache(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	admin := &entity.PlatformAdmin{
		Email:        "expired@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
	}
	repo.Create(context.Background(), admin)

	// Cache with expired expiry
	service.cacheMutex.Lock()
	service.cache[admin.ID] = &cachedPlatformAdmin{
		admin:      admin,
		permissions: []string{},
		expiry:     time.Now().Add(-1 * time.Hour), // Expired
	}
	service.cacheMutex.Unlock()

	// Should fetch from database
	ctx := context.Background()
	validated, err := service.ValidateSession(ctx, admin.ID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if validated == nil {
		t.Error("admin should not be nil")
	}
}

func TestPlatformAuthService_CheckPermission(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	perms := []string{"read", "write"}
	permsJSON, _ := json.Marshal(perms)
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	admin := &entity.PlatformAdmin{
		Email:        "perms@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
		Permissions:  string(permsJSON),
	}
	repo.Create(context.Background(), admin)

	ctx := context.Background()

	// Check with cache
	t.Run("permission from cache", func(t *testing.T) {
		service.cacheAdmin(admin)
		hasPerm, err := service.CheckPermission(ctx, admin.ID, "read")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !hasPerm {
			t.Error("should have 'read' permission")
		}
	})

	// Check wildcard permission
	t.Run("wildcard permission", func(t *testing.T) {
		adminWild := &entity.PlatformAdmin{
			Email:        "wild@example.com",
			PasswordHash: string(passwordHash),
			Status:       "active",
			Permissions:  "[\"*\"]",
		}
		repo.Create(ctx, adminWild)
		service.cacheAdmin(adminWild)

		hasPerm, err := service.CheckPermission(ctx, adminWild.ID, "any_permission")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !hasPerm {
			t.Error("wildcard should grant any permission")
		}
	})

	// Check permission not in list
	t.Run("permission not granted", func(t *testing.T) {
		service.cacheAdmin(admin)
		hasPerm, _ := service.CheckPermission(ctx, admin.ID, "admin")
		if hasPerm {
			t.Error("should not have 'admin' permission")
		}
	})

	// Check with expired cache - falls back to repo
	t.Run("expired cache fallback", func(t *testing.T) {
		repo.checkPermResult = true
		service.cacheMutex.Lock()
		service.cache[admin.ID] = &cachedPlatformAdmin{
			admin:      admin,
			permissions: perms,
			expiry:     time.Now().Add(-1 * time.Hour),
		}
		service.cacheMutex.Unlock()

		hasPerm, err := service.CheckPermission(ctx, admin.ID, "read")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !hasPerm {
			t.Error("should have permission from repo")
		}
	})
}

func TestPlatformAuthService_CheckPermissionWithRepoError(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	repo.checkPermError = errors.New("repo error")
	service := NewPlatformAuthService(repo, nil)

	adminID := uuid.New()
	// Set expired cache to force repo lookup
	service.cacheMutex.Lock()
	service.cache[adminID] = &cachedPlatformAdmin{
		admin:      nil,
		permissions: []string{},
		expiry:     time.Now().Add(-1 * time.Hour),
	}
	service.cacheMutex.Unlock()

	ctx := context.Background()
	_, err := service.CheckPermission(ctx, adminID, "read")
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestPlatformAuthService_HasRole(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	admin := &entity.PlatformAdmin{
		Email:        "role@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
		Role:         "super_admin",
	}
	repo.Create(context.Background(), admin)

	ctx := context.Background()

	// Has correct role
	t.Run("has role", func(t *testing.T) {
		hasRole, err := service.HasRole(ctx, admin.ID, "super_admin")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !hasRole {
			t.Error("should have super_admin role")
		}
	})

	// Does not have different role
	t.Run("does not have role", func(t *testing.T) {
		hasRole, err := service.HasRole(ctx, admin.ID, "support")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if hasRole {
			t.Error("should not have support role")
		}
	})

	// Invalid admin - returns error
	t.Run("invalid admin", func(t *testing.T) {
		_, err := service.HasRole(ctx, uuid.New(), "super_admin")
		if err == nil {
			t.Error("expected error for nonexistent admin")
		}
	})
}

func TestPlatformAuthService_IsSuperAdmin(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	superAdmin := &entity.PlatformAdmin{
		Email:        "super@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
		Role:         "super_admin",
	}
	repo.Create(context.Background(), superAdmin)

	regularAdmin := &entity.PlatformAdmin{
		Email:        "regular@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
		Role:         "support",
	}
	repo.Create(context.Background(), regularAdmin)

	ctx := context.Background()

	// Super admin
	t.Run("is super admin", func(t *testing.T) {
		isSuper, err := service.IsSuperAdmin(ctx, superAdmin.ID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !isSuper {
			t.Error("should be super admin")
		}
	})

	// Not super admin
	t.Run("is not super admin", func(t *testing.T) {
		isSuper, err := service.IsSuperAdmin(ctx, regularAdmin.ID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if isSuper {
			t.Error("should not be super admin")
		}
	})
}

func TestPlatformAuthService_CreateAdmin(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	ctx := context.Background()

	// Successful create
	t.Run("successful create", func(t *testing.T) {
		admin, err := service.CreateAdmin(ctx, uuid.Nil, "new@example.com", "password123", "New Admin", "support", []string{"read"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if admin == nil {
			t.Error("admin should not be nil")
		}
		if admin.Email != "new@example.com" {
			t.Errorf("expected email new@example.com, got %s", admin.Email)
		}
		if admin.Name != "New Admin" {
			t.Errorf("expected name 'New Admin', got %s", admin.Name)
		}
		if admin.Role != "support" {
			t.Errorf("expected role 'support', got %s", admin.Role)
		}
		if admin.Status != "active" {
			t.Errorf("expected status 'active', got %s", admin.Status)
		}
	})

	// Duplicate email
	t.Run("duplicate email", func(t *testing.T) {
		_, err := service.CreateAdmin(ctx, uuid.Nil, "new@example.com", "password123", "Duplicate", "support", []string{})
		if err != apperrors.ErrPlatformAdminExists {
			t.Errorf("expected ErrPlatformAdminExists, got %v", err)
		}
	})

	// Create with repo error
	t.Run("repo error", func(t *testing.T) {
		repoError := newMockPlatformAdminRepo()
		repoError.createError = errors.New("create failed")
		serviceError := NewPlatformAuthService(repoError, nil)

		_, err := serviceError.CreateAdmin(ctx, uuid.Nil, "error@example.com", "password123", "Error Admin", "support", []string{})
		if err == nil {
			t.Error("expected error from repo")
		}
	})
}

func TestPlatformAuthService_UpdateAdmin(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	admin := &entity.PlatformAdmin{
		Email:        "update@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
		Name:         "Original",
		Role:         "support",
	}
	repo.Create(context.Background(), admin)

	ctx := context.Background()

	// Update name
	t.Run("update name", func(t *testing.T) {
		err := service.UpdateAdmin(ctx, uuid.Nil, admin.ID, map[string]interface{}{"name": "Updated Name"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		updated, _ := repo.GetByID(ctx, admin.ID)
		if updated.Name != "Updated Name" {
			t.Errorf("expected name 'Updated Name', got %s", updated.Name)
		}
	})

	// Update role
	t.Run("update role", func(t *testing.T) {
		err := service.UpdateAdmin(ctx, uuid.Nil, admin.ID, map[string]interface{}{"role": "billing_admin"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		updated, _ := repo.GetByID(ctx, admin.ID)
		if updated.Role != "billing_admin" {
			t.Errorf("expected role 'billing_admin', got %s", updated.Role)
		}
	})

	// Update status
	t.Run("update status", func(t *testing.T) {
		err := service.UpdateAdmin(ctx, uuid.Nil, admin.ID, map[string]interface{}{"status": "inactive"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		updated, _ := repo.GetByID(ctx, admin.ID)
		if updated.Status != "inactive" {
			t.Errorf("expected status 'inactive', got %s", updated.Status)
		}
	})

	// Update permissions
	t.Run("update permissions", func(t *testing.T) {
		err := service.UpdateAdmin(ctx, uuid.Nil, admin.ID, map[string]interface{}{"permissions": []string{"read", "write", "admin"}})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		updated, _ := repo.GetByID(ctx, admin.ID)
		var perms []string
		json.Unmarshal([]byte(updated.Permissions), &perms)
		if len(perms) != 3 {
			t.Errorf("expected 3 permissions, got %d", len(perms))
		}
	})

	// Update password
	t.Run("update password", func(t *testing.T) {
		newPassword := "newpassword123"
		err := service.UpdateAdmin(ctx, uuid.Nil, admin.ID, map[string]interface{}{"password": newPassword})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		updated, _ := repo.GetByID(ctx, admin.ID)
		// Verify new password hash works
		if bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte(newPassword)) != nil {
			t.Error("new password should work")
		}
	})

	// Admin not found
	t.Run("admin not found", func(t *testing.T) {
		err := service.UpdateAdmin(ctx, uuid.Nil, uuid.New(), map[string]interface{}{"name": "New Name"})
		if err != apperrors.ErrPlatformAdminNotFound {
			t.Errorf("expected ErrPlatformAdminNotFound, got %v", err)
		}
	})

	// Verify cache invalidated
	t.Run("cache invalidated", func(t *testing.T) {
		service.cacheAdmin(admin)
		_ = service.UpdateAdmin(ctx, uuid.Nil, admin.ID, map[string]interface{}{"name": "Cache Test"})
		cached := service.getCachedAdmin(admin.ID)
		if cached != nil {
			t.Error("cache should be invalidated after update")
		}
	})
}

func TestPlatformAuthService_DeleteAdmin(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	admin := &entity.PlatformAdmin{
		Email:        "delete@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
	}
	repo.Create(context.Background(), admin)

	ctx := context.Background()

	// Successful delete
	t.Run("successful delete", func(t *testing.T) {
		err := service.DeleteAdmin(ctx, uuid.Nil, admin.ID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_, err = repo.GetByID(ctx, admin.ID)
		if err == nil {
			t.Error("admin should be deleted")
		}
	})

	// Cache invalidated on delete
	t.Run("cache invalidated on delete", func(t *testing.T) {
		admin2 := &entity.PlatformAdmin{
			Email:        "delete2@example.com",
			PasswordHash: string(passwordHash),
			Status:       "active",
		}
		repo.Create(ctx, admin2)
		service.cacheAdmin(admin2)

		_ = service.DeleteAdmin(ctx, uuid.Nil, admin2.ID)
		cached := service.getCachedAdmin(admin2.ID)
		if cached != nil {
			t.Error("cache should be invalidated after delete")
		}
	})

	// Delete with repo error
	t.Run("repo error", func(t *testing.T) {
		repoError := newMockPlatformAdminRepo()
		repoError.deleteError = errors.New("delete failed")
		serviceError := NewPlatformAuthService(repoError, nil)

		err := serviceError.DeleteAdmin(ctx, uuid.Nil, uuid.New())
		if err == nil {
			t.Error("expected error from repo")
		}
	})
}

func TestPlatformAuthService_ListAdmins(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	for i := 0; i < 5; i++ {
		admin := &entity.PlatformAdmin{
			Email:        "list" + toString(i) + "@example.com",
			PasswordHash: string(passwordHash),
			Status:       "active",
		}
		repo.Create(context.Background(), admin)
	}

	ctx := context.Background()

	admins, total, err := service.ListAdmins(ctx, 1, 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(admins) != 5 {
		t.Errorf("expected 5 admins, got %d", len(admins))
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
}

func TestPlatformAuthService_CacheOperations(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	admin := &entity.PlatformAdmin{
		Email:        "cache@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
		Permissions:  "[\"read\",\"write\"]",
	}
	repo.Create(context.Background(), admin)

	// Test cacheAdmin
	t.Run("cache admin", func(t *testing.T) {
		service.cacheAdmin(admin)
		cached := service.getCachedAdmin(admin.ID)
		if cached == nil {
			t.Error("admin should be cached")
		}
		if cached.admin.Email != admin.Email {
			t.Errorf("expected email %s, got %s", admin.Email, cached.admin.Email)
		}
		if len(cached.permissions) != 2 {
			t.Errorf("expected 2 permissions, got %d", len(cached.permissions))
		}
	})

	// Test getCachedAdmin
	t.Run("get cached admin", func(t *testing.T) {
		cached := service.getCachedAdmin(admin.ID)
		if cached == nil {
			t.Error("cached admin should not be nil")
		}
	})

	// Test getCachedAdmin non-existent
	t.Run("get non-existent cached admin", func(t *testing.T) {
		cached := service.getCachedAdmin(uuid.New())
		if cached != nil {
			t.Error("non-existent cached admin should be nil")
		}
	})

	// Test invalidateCache
	t.Run("invalidate cache", func(t *testing.T) {
		service.cacheAdmin(admin)
		service.invalidateCache(admin.ID)
		cached := service.getCachedAdmin(admin.ID)
		if cached != nil {
			t.Error("cache should be invalidated")
		}
	})

	// Test cache with empty permissions
	t.Run("cache with empty permissions", func(t *testing.T) {
		adminNoPerms := &entity.PlatformAdmin{
			Email:        "noperms@example.com",
			PasswordHash: string(passwordHash),
			Status:       "active",
			Permissions:  "",
		}
		repo.Create(context.Background(), adminNoPerms)
		service.cacheAdmin(adminNoPerms)
		cached := service.getCachedAdmin(adminNoPerms.ID)
		if cached == nil {
			t.Error("admin should be cached")
		}
		if len(cached.permissions) != 0 {
			t.Errorf("expected 0 permissions, got %d", len(cached.permissions))
		}
	})
}

func TestGenerateSessionToken(t *testing.T) {
	// Generate multiple tokens and verify uniqueness
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := generateSessionToken()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if token == "" {
			t.Error("token should not be empty")
		}
		if len(token) < 32 {
			t.Errorf("token should be at least 32 chars, got %d", len(token))
		}
		if tokens[token] {
			t.Error("token should be unique")
		}
		tokens[token] = true
	}
}

func TestPlatformAuthService_ConcurrentCacheAccess(t *testing.T) {
	repo := newMockPlatformAdminRepo()
	service := NewPlatformAuthService(repo, nil)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	admin := &entity.PlatformAdmin{
		Email:        "concurrent@example.com",
		PasswordHash: string(passwordHash),
		Status:       "active",
	}
	repo.Create(context.Background(), admin)

	// Concurrent cache operations
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			service.cacheAdmin(admin)
			service.getCachedAdmin(admin.ID)
			service.invalidateCache(admin.ID)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}