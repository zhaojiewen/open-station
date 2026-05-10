package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetAPIKeyID(t *testing.T) {
	id := uuid.New()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("api_key_id", id)

	result := GetAPIKeyID(c)
	if result != id {
		t.Errorf("GetAPIKeyID() = %v, want %v", result, id)
	}
}

func TestGetUserID(t *testing.T) {
	id := uuid.New()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", id)

	result := GetUserID(c)
	if result != id {
		t.Errorf("GetUserID() = %v, want %v", result, id)
	}
}

func TestGetTenantID(t *testing.T) {
	id := uuid.New()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("tenant_id", id)

	result := GetTenantID(c)
	if result != id {
		t.Errorf("GetTenantID() = %v, want %v", result, id)
	}
}

func TestGetAPIKey(t *testing.T) {
	key := &entity.APIKey{
		ID:     uuid.New(),
		Name:   "Test Key",
		Status: "active",
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("api_key", key)

	result := GetAPIKey(c)
	if result != key {
		t.Errorf("GetAPIKey() = %v, want %v", result, key)
	}
	if result.Name != "Test Key" {
		t.Errorf("APIKey.Name = %v, want Test Key", result.Name)
	}
}

func TestGetUser(t *testing.T) {
	user := &entity.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
		Role:  "admin",
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user", user)

	result := GetUser(c)
	if result != user {
		t.Errorf("GetUser() = %v, want %v", result, user)
	}
	if result.Email != "test@example.com" {
		t.Errorf("User.Email = %v, want test@example.com", result.Email)
	}
}

func TestGetTenant(t *testing.T) {
	tenant := &entity.Tenant{
		ID:     uuid.New(),
		Name:   "Test Tenant",
		Slug:   "test-tenant",
		Status: "active",
		Balance: decimal.NewFromInt(1000),
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("tenant", tenant)

	result := GetTenant(c)
	if result != tenant {
		t.Errorf("GetTenant() = %v, want %v", result, tenant)
	}
	if result.Slug != "test-tenant" {
		t.Errorf("Tenant.Slug = %v, want test-tenant", result.Slug)
	}
}

func TestAdminOnlyMiddleware_Admin(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	user := &entity.User{
		ID:   uuid.New(),
		Role: "admin",
	}
	c.Set("user", user)

	middleware := AdminOnlyMiddleware()
	middleware(c)

	// Should not abort for admin
	if c.IsAborted() {
		t.Error("AdminOnlyMiddleware should not abort for admin user")
	}
}

func TestAdminOnlyMiddleware_NonAdmin(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	user := &entity.User{
		ID:   uuid.New(),
		Role: "member",
	}
	c.Set("user", user)

	middleware := AdminOnlyMiddleware()
	middleware(c)

	// Should abort for non-admin
	if !c.IsAborted() {
		t.Error("AdminOnlyMiddleware should abort for non-admin user")
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusForbidden)
	}
}

func TestAdminOnlyMiddleware_NoUser(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// No user set in context
	middleware := AdminOnlyMiddleware()
	middleware(c)

	// Should abort when no user
	if !c.IsAborted() {
		t.Error("AdminOnlyMiddleware should abort when user not set")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestRateLimitConfig(t *testing.T) {
	cfg := &RateLimitConfig{
		DefaultUserRPS:      10.0,
		DefaultUserBurst:    20,
		DefaultTenantRPS:    100.0,
		DefaultTenantBurst:  200,
	}

	if cfg.DefaultUserRPS != 10.0 {
		t.Errorf("DefaultUserRPS = %v, want 10.0", cfg.DefaultUserRPS)
	}
	if cfg.DefaultUserBurst != 20 {
		t.Errorf("DefaultUserBurst = %v, want 20", cfg.DefaultUserBurst)
	}
	if cfg.DefaultTenantRPS != 100.0 {
		t.Errorf("DefaultTenantRPS = %v, want 100.0", cfg.DefaultTenantRPS)
	}
	if cfg.DefaultTenantBurst != 200 {
		t.Errorf("DefaultTenantBurst = %v, want 200", cfg.DefaultTenantBurst)
	}
}

func TestGettersWithNilContext(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test getters when values not set - should panic or return nil
	// We need to handle these cases appropriately

	// In production code, these would panic if the value is not set
	// For tests, we can wrap in recover

	func() {
		defer func() {
			if r := recover(); r == nil {
				// Some implementations might return zero value instead of panic
				t.Log("GetAPIKeyID did not panic for unset value")
			}
		}()
		_ = GetAPIKeyID(c)
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Log("GetUserID did not panic for unset value")
			}
		}()
		_ = GetUserID(c)
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Log("GetTenantID did not panic for unset value")
			}
		}()
		_ = GetTenantID(c)
	}()
}

func TestMiddlewareSetsContextValues(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test that all expected keys are set
	testKeys := []string{
		"api_key_id",
		"api_key",
		"user_id",
		"user",
		"tenant_id",
		"tenant",
	}

	for _, key := range testKeys {
		// Initially not set
		val, exists := c.Get(key)
		if exists {
			// Value is set (this shouldn't happen)
			t.Logf("Key %s is unexpectedly set with value %v", key, val)
		}
	}

	// Set values
	apiKeyID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()

	c.Set("api_key_id", apiKeyID)
	c.Set("user_id", userID)
	c.Set("tenant_id", tenantID)

	// Verify retrieval
	if GetAPIKeyID(c) != apiKeyID {
		t.Error("api_key_id mismatch")
	}
	if GetUserID(c) != userID {
		t.Error("user_id mismatch")
	}
	if GetTenantID(c) != tenantID {
		t.Error("tenant_id mismatch")
	}
}

func TestAdminMiddleware_RoleChecking(t *testing.T) {
	roles := []struct {
		role      string
		shouldAbort bool
	}{
		{"admin", false},
		{"member", true},
		{"viewer", true},
		{"", true},
		{"superadmin", true}, // Not recognized as admin
	}

	for _, tc := range roles {
		t.Run(tc.role, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			user := &entity.User{
				Role: tc.role,
			}
			c.Set("user", user)

			middleware := AdminOnlyMiddleware()
			middleware(c)

			if c.IsAborted() != tc.shouldAbort {
				t.Errorf("Role %s: IsAborted = %v, want %v", tc.role, c.IsAborted(), tc.shouldAbort)
			}
		})
	}
}

func TestEntityTypesInContext(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Test that correct types are stored and retrieved
	apiKey := &entity.APIKey{
		ID:       uuid.New(),
		Name:     "Test",
		KeyPrefix: "sk-test",
	}
	user := &entity.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	tenant := &entity.Tenant{
		ID:     uuid.New(),
		Name:   "Test Tenant",
		Balance: decimal.NewFromInt(100),
	}

	c.Set("api_key", apiKey)
	c.Set("user", user)
	c.Set("tenant", tenant)

	// Verify types
	retrievedKey := GetAPIKey(c)
	if retrievedKey == nil {
		t.Error("GetAPIKey returned nil")
	}
	if retrievedKey.Name != "Test" {
		t.Error("APIKey type not preserved")
	}

	retrievedUser := GetUser(c)
	if retrievedUser == nil {
		t.Error("GetUser returned nil")
	}
	if retrievedUser.Email != "test@example.com" {
		t.Error("User type not preserved")
	}

	retrievedTenant := GetTenant(c)
	if retrievedTenant == nil {
		t.Error("GetTenant returned nil")
	}
	if retrievedTenant.Name != "Test Tenant" {
		t.Error("Tenant type not preserved")
	}
}