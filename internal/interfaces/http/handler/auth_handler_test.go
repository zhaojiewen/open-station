package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Create mock services for handler tests
type mockUserAuthService struct {
	loginResponse         *auth.LoginResponse
	registerResponse      *auth.RegisterResponse
	tenantRegResponse     *auth.RegisterTenantResponse
	userTenants           []entity.UserTenant
	loginError            error
	registerError         error
	tenantRegError        error
	switchTenantError     error
	changePasswordError   error
	logoutError           error
	logoutAllError        error
	refreshError          error
	refreshToken          string
	newAccessToken        string
	verifyEmailFunc       func(ctx context.Context, token string) (*entity.User, error)
	resendVerificationFunc func(ctx context.Context, email string) error
	GetUserTenantsFunc    func(ctx context.Context, id uuid.UUID) ([]entity.UserTenant, error)
}

func (m *mockUserAuthService) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	if m.loginError != nil {
		return nil, m.loginError
	}
	return m.loginResponse, nil
}

func (m *mockUserAuthService) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error) {
	if m.registerError != nil {
		return nil, m.registerError
	}
	return m.registerResponse, nil
}

func (m *mockUserAuthService) RegisterTenant(ctx context.Context, req *auth.RegisterTenantRequest) (*auth.RegisterTenantResponse, error) {
	if m.tenantRegError != nil {
		return nil, m.tenantRegError
	}
	return m.tenantRegResponse, nil
}

func (m *mockUserAuthService) ValidateToken(ctx context.Context, token string) (*entity.User, *entity.UserTenant, *auth.JWTClaims, error) {
	return nil, nil, nil, nil
}

func (m *mockUserAuthService) SwitchTenant(ctx context.Context, userID, tenantID uuid.UUID, currentToken string) (string, error) {
	if m.switchTenantError != nil {
		return "", m.switchTenantError
	}
	return m.newAccessToken, nil
}

func (m *mockUserAuthService) Logout(ctx context.Context, token string) error {
	return m.logoutError
}

func (m *mockUserAuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if m.logoutAllError != nil {
		return m.logoutAllError
	}
	return nil
}

func (m *mockUserAuthService) RefreshToken(ctx context.Context, refreshToken string, deviceID string) (string, error) {
	if m.refreshError != nil {
		return "", m.refreshError
	}
	return m.newAccessToken, nil
}

func (m *mockUserAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	return m.changePasswordError
}

func (m *mockUserAuthService) GetUserTenants(ctx context.Context, userID uuid.UUID) ([]entity.UserTenant, error) {
	if m.GetUserTenantsFunc != nil {
		return m.GetUserTenantsFunc(ctx, userID)
	}
	return m.userTenants, nil
}

type mockJWTService struct {
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func (m *mockJWTService) GenerateToken(userID, tenantID uuid.UUID, email, role, deviceID string) (string, string, uuid.UUID, error) {
	return "access-token", "refresh-token", uuid.New(), nil
}

func (m *mockJWTService) ValidateToken(token string) (*auth.JWTClaims, error) {
	return nil, nil
}

func (m *mockJWTService) InvalidateToken(token string) error {
	return nil
}

func (m *mockJWTService) InvalidateTokenByID(tokenID uuid.UUID, expiry time.Duration) error {
	return nil
}

func (m *mockJWTService) RefreshToken(refreshToken string, tenantID uuid.UUID, role string) (string, error) {
	return "new-access-token", nil
}

func (m *mockJWTService) IsTokenIDBlacklisted(tokenID uuid.UUID) bool {
	return false
}

func (m *mockJWTService) GetAccessTokenExpiry() time.Duration {
	return m.accessExpiry
}

func (m *mockJWTService) GetRefreshTokenExpiry() time.Duration {
	return m.refreshExpiry
}

func (m *mockJWTService) hashToken(token string) string {
	return "hashed-token"
}

func setupAuthHandler() (*AuthHandler, *mockUserAuthService, *mockJWTService) {
	mockAuthService := &mockUserAuthService{
		loginResponse: &auth.LoginResponse{
			User: &entity.User{
				ID:    uuid.New(),
				Email: "test@example.com",
				Name:  "Test User",
				Role:  "member",
			},
			UserTenants: []entity.UserTenant{
				{
					ID:        uuid.New(),
					UserID:    uuid.New(),
					TenantID:  uuid.New(),
					Role:      "member",
					Status:    "active",
					IsDefault: true,
				},
			},
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresAt:    time.Now().Add(15 * time.Minute),
		},
		registerResponse: &auth.RegisterResponse{
			User: &entity.User{
				ID:    uuid.New(),
				Email: "new@example.com",
				Name:  "New User",
			},
			UserTenant: &entity.UserTenant{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				Role:      "member",
				IsDefault: true,
			},
			AccessToken:  "register-access-token",
			RefreshToken: "register-refresh-token",
		},
		tenantRegResponse: &auth.RegisterTenantResponse{
			Tenant: &entity.Tenant{
				ID:     uuid.New(),
				Name:   "Test Tenant",
				Slug:   "test-tenant",
				Status: "active",
				Plan:   "free",
				Balance: decimal.Zero,
			},
			User: &entity.User{
				ID:    uuid.New(),
				Email: "admin@testtenant.com",
				Name:  "Admin",
				Role:  "admin",
			},
			UserTenant: &entity.UserTenant{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				Role:      "admin",
			},
			AccessToken:  "tenant-access-token",
			RefreshToken: "tenant-refresh-token",
		},
		newAccessToken: "switched-access-token",
	}

	mockJWT := &mockJWTService{
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}

	handler := NewAuthHandler(mockAuthService, mockJWT)
	return handler, mockAuthService, mockJWT
}

func TestAuthHandler_Login(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	// Test successful login
	t.Run("successful login", func(t *testing.T) {
		body := LoginRequest{
			Email:    "test@example.com",
			Password: "TestUserPass123!",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/login", handler.Login)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["access_token"] == nil {
			t.Error("response should contain access_token")
		}
		if resp["user"] == nil {
			t.Error("response should contain user")
		}
	})

	// Test invalid request
	t.Run("invalid request", func(t *testing.T) {
		body := map[string]string{
			"email": "", // missing email
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/login", handler.Login)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test too many attempts
	t.Run("too many attempts", func(t *testing.T) {
		mockService.loginError = apperrors.ErrTooManyAttempts
		defer func() { mockService.loginError = nil }()

		body := LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpass123", // 11 chars to pass min=8 validation
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/login", handler.Login)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status 429, got %d", w.Code)
		}
	})

	// Test invalid credentials
	t.Run("invalid credentials", func(t *testing.T) {
		mockService.loginError = apperrors.ErrInvalidCredentials
		defer func() { mockService.loginError = nil }()

		body := LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpass123", // 11 chars to pass min=8 validation
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/login", handler.Login)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestAuthHandler_Register(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	// Test successful registration
	t.Run("successful register", func(t *testing.T) {
		body := RegisterRequest{
			Email:    "new@example.com",
			Password: "NewSecurePass123!",
			Name:     "New User",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/register", handler.Register)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["access_token"] == nil {
			t.Error("response should contain access_token")
		}
	})

	// Test invalid request
	t.Run("invalid request", func(t *testing.T) {
		body := map[string]string{
			"email": "invalid-email",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/register", handler.Register)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test email exists
	t.Run("email exists", func(t *testing.T) {
		mockService.registerError = apperrors.ErrEmailExists
		defer func() { mockService.registerError = nil }()

		body := RegisterRequest{
			Email:    "existing@example.com",
			Password: "SecurePass123!",
			Name:     "Existing",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/register", handler.Register)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestAuthHandler_RegisterTenant(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	// Test successful tenant registration
	t.Run("successful tenant register", func(t *testing.T) {
		body := RegisterTenantRequest{
			TenantName: "Test Company",
			TenantSlug: "test-company",
			Email:      "admin@testcompany.com",
			Password:   "ManagerPass123!",
			Name:       "Admin User",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/tenant/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/tenant/register", handler.RegisterTenant)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["tenant"] == nil {
			t.Error("response should contain tenant")
		}
		if resp["user"] == nil {
			t.Error("response should contain user")
		}
	})

	// Test invalid request
	t.Run("invalid request", func(t *testing.T) {
		body := map[string]string{
			"tenant_name": "", // missing required
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/tenant/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/tenant/register", handler.RegisterTenant)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test slug exists
	t.Run("slug exists", func(t *testing.T) {
		mockService.tenantRegError = apperrors.ErrTenantSlugExists
		defer func() { mockService.tenantRegError = nil }()

		body := RegisterTenantRequest{
			TenantName: "Duplicate",
			TenantSlug: "existing-slug",
			Email:      "admin@dup.com",
			Password:   "SecurePass123!",
			Name:       "Admin",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/tenant/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/tenant/register", handler.RegisterTenant)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	handler, _, _ := setupAuthHandler()

	// Test successful logout
	t.Run("successful logout", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/logout", handler.Logout)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test no token
	t.Run("no token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/auth/logout", nil)
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/logout", handler.Logout)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	// Test successful refresh
	t.Run("successful refresh", func(t *testing.T) {
		body := RefreshTokenRequest{
			RefreshToken: "test-refresh-token",
			DeviceID:     "device123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/refresh", handler.RefreshToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["access_token"] == nil {
			t.Error("response should contain access_token")
		}
	})

	// Test invalid request
	t.Run("invalid request", func(t *testing.T) {
		body := map[string]string{
			"refresh_token": "", // missing
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/refresh", handler.RefreshToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test invalid refresh token
	t.Run("invalid refresh token", func(t *testing.T) {
		mockService.refreshError = apperrors.ErrRefreshTokenInvalid
		defer func() { mockService.refreshError = nil }()

		body := RefreshTokenRequest{
			RefreshToken: "invalid-token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/refresh", handler.RefreshToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestAuthHandler_GetProfile(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	userID := uuid.New()
	tenantID := uuid.New()

	// Test successful get profile
	t.Run("successful get profile", func(t *testing.T) {
		mockService.userTenants = []entity.UserTenant{
			{
				ID:        uuid.New(),
				UserID:    userID,
				TenantID:  tenantID,
				Role:      "member",
				Status:    "active",
				IsDefault: true,
			},
		}

		req := httptest.NewRequest("GET", "/auth/profile", nil)
		w := httptest.NewRecorder()

		router := gin.New()
		router.GET("/auth/profile", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Set("user", &entity.User{
				ID:     userID,
				Email:  "test@example.com",
				Name:   "Test User",
				Role:   "member",
				Status: "active",
			})
			c.Set("tenant_id", tenantID)
			c.Next()
		}, handler.GetProfile)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test unauthorized
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/profile", nil)
		w := httptest.NewRecorder()

		router := gin.New()
		router.GET("/auth/profile", handler.GetProfile)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestAuthHandler_GetTenants(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	userID := uuid.New()

	// Test successful get tenants
	t.Run("successful get tenants", func(t *testing.T) {
		mockService.userTenants = []entity.UserTenant{
			{
				ID:        uuid.New(),
				UserID:    userID,
				TenantID:  uuid.New(),
				Role:      "admin",
				Status:    "active",
				IsDefault: true,
			},
			{
				ID:        uuid.New(),
				UserID:    userID,
				TenantID:  uuid.New(),
				Role:      "member",
				Status:    "active",
				IsDefault: false,
			},
		}

		req := httptest.NewRequest("GET", "/auth/tenants", nil)
		w := httptest.NewRecorder()

		router := gin.New()
		router.GET("/auth/tenants", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		}, handler.GetTenants)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test unauthorized
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/tenants", nil)
		w := httptest.NewRecorder()

		router := gin.New()
		router.GET("/auth/tenants", handler.GetTenants)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestAuthHandler_SwitchTenant(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	userID := uuid.New()
	tenantID := uuid.New()

	// Test successful switch
	t.Run("successful switch", func(t *testing.T) {
		body := SwitchTenantRequest{
			TenantID: tenantID.String(),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/switch-tenant", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/switch-tenant", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		}, handler.SwitchTenant)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test invalid tenant ID
	t.Run("invalid tenant ID", func(t *testing.T) {
		body := SwitchTenantRequest{
			TenantID: "invalid-uuid",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/switch-tenant", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/switch-tenant", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		}, handler.SwitchTenant)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test unauthorized
	t.Run("unauthorized", func(t *testing.T) {
		body := SwitchTenantRequest{
			TenantID: tenantID.String(),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/switch-tenant", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/switch-tenant", handler.SwitchTenant)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	// Test not in tenant
	t.Run("not in tenant", func(t *testing.T) {
		mockService.switchTenantError = apperrors.ErrUserNotInTenant
		defer func() { mockService.switchTenantError = nil }()

		body := SwitchTenantRequest{
			TenantID: tenantID.String(),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/switch-tenant", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/switch-tenant", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		}, handler.SwitchTenant)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	handler, _, _ := setupAuthHandler()

	userID := uuid.New()

	// Test successful change
	t.Run("successful change", func(t *testing.T) {
		body := ChangePasswordRequest{
			CurrentPassword: "OldSecurePass123!",
			NewPassword:     "NewSecurePass123!",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/auth/password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.PUT("/auth/password", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		}, handler.ChangePassword)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test invalid request
	t.Run("invalid request", func(t *testing.T) {
		body := map[string]string{
			"current_password": "",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/auth/password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.PUT("/auth/password", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		}, handler.ChangePassword)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test unauthorized
	t.Run("unauthorized", func(t *testing.T) {
		body := ChangePasswordRequest{
			CurrentPassword: "OldSecurePass123!",
			NewPassword:     "NewSecurePass123!",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/auth/password", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.PUT("/auth/password", handler.ChangePassword)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestAuthHandler_LogoutAll(t *testing.T) {
	handler, _, _ := setupAuthHandler()

	userID := uuid.New()

	// Test successful logout all
	t.Run("successful logout all", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/auth/logout-all", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/logout-all", func(c *gin.Context) {
			c.Set("user_id", userID)
			c.Next()
		}, handler.LogoutAll)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test unauthorized
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/auth/logout-all", nil)
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/logout-all", handler.LogoutAll)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestAuthHandler_extractToken(t *testing.T) {
	handler, _, _ := setupAuthHandler()

	// Test Bearer token
	t.Run("Bearer token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		router := gin.New()
		router.GET("/", func(c *gin.Context) {
			token := handler.extractToken(c)
			if token != "test-token" {
				t.Errorf("expected test-token, got %s", token)
			}
		})
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test direct token
	t.Run("direct token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "test-token")

		router := gin.New()
		router.GET("/", func(c *gin.Context) {
			token := handler.extractToken(c)
			if token != "test-token" {
				t.Errorf("expected test-token, got %s", token)
			}
		})
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test no token
	t.Run("no token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		router := gin.New()
		router.GET("/", func(c *gin.Context) {
			token := handler.extractToken(c)
			if token != "" {
				t.Errorf("expected empty string, got %s", token)
			}
		})
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestGetUserIDFromContext(t *testing.T) {
	router := gin.New()

	// Test with valid user ID
	t.Run("valid user ID", func(t *testing.T) {
		userID := uuid.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set("user_id", userID)
			result := GetUserIDFromContext(c)
			if result != userID {
				t.Errorf("expected %s, got %s", userID, result)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test without user ID
	t.Run("no user ID", func(t *testing.T) {
		router.GET("/no-user", func(c *gin.Context) {
			result := GetUserIDFromContext(c)
			if result != uuid.Nil {
				t.Errorf("expected Nil UUID, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/no-user", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestGetTenantIDFromContext(t *testing.T) {
	router := gin.New()

	// Test with valid tenant ID
	t.Run("valid tenant ID", func(t *testing.T) {
		tenantID := uuid.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set("tenant_id", tenantID)
			result := GetTenantIDFromContext(c)
			if result != tenantID {
				t.Errorf("expected %s, got %s", tenantID, result)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test without tenant ID
	t.Run("no tenant ID", func(t *testing.T) {
		router.GET("/no-tenant", func(c *gin.Context) {
			result := GetTenantIDFromContext(c)
			if result != uuid.Nil {
				t.Errorf("expected Nil UUID, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/no-tenant", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestAuthHandler_LoginWithEmailNotVerified(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	mockService.loginError = apperrors.ErrEmailNotVerified
	defer func() { mockService.loginError = nil }()

	body := LoginRequest{
		Email:    "unverified@example.com",
		Password: "TestPass123!", // 11 chars
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["require_verify"] != true {
		t.Error("response should indicate require_verify")
	}
	if resp["code"] != "VERIFY_004" {
		t.Errorf("expected code VERIFY_004, got %s", resp["code"])
	}
}

func TestAuthHandler_LoginWithInternalError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	mockService.loginError = errors.New("internal server error")
	defer func() { mockService.loginError = nil }()

	body := LoginRequest{
		Email:    "internal@example.com",
		Password: "TestPass123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_LoginWithDeviceIDHeader(t *testing.T) {
	handler, _, _ := setupAuthHandler()

	body := LoginRequest{
		Email:    "device@example.com",
		Password: "TestPass123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-ID", "custom-device-id")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	// Should succeed with custom device ID
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuthHandler_LoginWithAnomaly(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	mockService.loginResponse.IsAnomaly = true
	mockService.loginResponse.AnomalyType = "new_device"
	defer func() {
		mockService.loginResponse.IsAnomaly = false
		mockService.loginResponse.AnomalyType = ""
	}()

	body := LoginRequest{
		Email:    "anomaly@example.com",
		Password: "TestPass123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/login", handler.Login)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["is_anomaly"] != true {
		t.Error("response should indicate anomaly")
	}
	if resp["anomaly_type"] != "new_device" {
		t.Errorf("expected anomaly_type new_device, got %s", resp["anomaly_type"])
	}
}

func TestAuthHandler_RegisterWithEmailVerificationRequired(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	// Set up response without tokens (email verification required)
	mockService.registerResponse.AccessToken = ""
	mockService.registerResponse.RefreshToken = ""
	defer func() {
		mockService.registerResponse.AccessToken = "register-access-token"
		mockService.registerResponse.RefreshToken = "register-refresh-token"
	}()

	body := RegisterRequest{
		Email:    "verify@example.com",
		Password: "SecurePass123!",
		Name:     "Verify User",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["require_verify"] != true {
		t.Error("response should indicate require_verify")
	}
	if resp["message"] == nil {
		t.Error("response should have message")
	}
}

func TestAuthHandler_RegisterWithInternalError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	mockService.registerError = errors.New("internal error")
	defer func() { mockService.registerError = nil }()

	body := RegisterRequest{
		Email:    "internal@example.com",
		Password: "SecurePass123!",
		Name:     "Internal",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/register", handler.Register)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_RegisterTenantWithExistingUser(t *testing.T) {
	handler, _, _ := setupAuthHandler()

	// Success case - should return tenant and user
	body := RegisterTenantRequest{
		TenantName: "Existing User Tenant",
		TenantSlug: "existing-user-tenant",
		Email:      "existing@example.com",
		Password:   "SecurePass123!",
		Name:       "Existing",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/tenant/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/tenant/register", handler.RegisterTenant)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["tenant"] == nil {
		t.Error("response should contain tenant")
	}
	if resp["user"] == nil {
		t.Error("response should contain user")
	}
}

func TestAuthHandler_RegisterTenantWithInternalError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	mockService.tenantRegError = errors.New("internal error")
	defer func() { mockService.tenantRegError = nil }()

	body := RegisterTenantRequest{
		TenantName: "Error Tenant",
		TenantSlug: "error-tenant",
		Email:      "error@example.com",
		Password:   "SecurePass123!",
		Name:       "Error",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/tenant/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/tenant/register", handler.RegisterTenant)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_LogoutWithError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	mockService.logoutError = errors.New("logout failed")
	defer func() { mockService.logoutError = nil }()

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/logout", handler.Logout)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_RefreshTokenWithInternalError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	mockService.refreshError = errors.New("refresh failed")
	defer func() { mockService.refreshError = nil }()

	body := RefreshTokenRequest{
		RefreshToken: "test-refresh-token",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/refresh", handler.RefreshToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_GetProfileWithoutUser(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()
	userID := uuid.New()

	mockService.userTenants = []entity.UserTenant{}

	req := httptest.NewRequest("GET", "/auth/profile", nil)
	w := httptest.NewRecorder()

	router := gin.New()
	router.GET("/auth/profile", func(c *gin.Context) {
		c.Set("user_id", userID)
		// Don't set user - simulate missing user
		c.Next()
	}, handler.GetProfile)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_GetProfileWithInternalError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()
	userID := uuid.New()
	tenantID := uuid.New()

	// Set up mock to return error
	mockService.userTenants = nil
	mockService.GetUserTenantsFunc = func(ctx context.Context, id uuid.UUID) ([]entity.UserTenant, error) {
		return nil, errors.New("database error")
	}

	req := httptest.NewRequest("GET", "/auth/profile", nil)
	w := httptest.NewRecorder()

	router := gin.New()
	router.GET("/auth/profile", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("user", &entity.User{
			ID:     userID,
			Email:  "test@example.com",
			Name:   "Test User",
			Role:   "member",
			Status: "active",
		})
		c.Set("tenant_id", tenantID)
		c.Next()
	}, handler.GetProfile)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_GetTenantsWithError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()
	userID := uuid.New()

	mockService.GetUserTenantsFunc = func(ctx context.Context, id uuid.UUID) ([]entity.UserTenant, error) {
		return nil, errors.New("database error")
	}

	req := httptest.NewRequest("GET", "/auth/tenants", nil)
	w := httptest.NewRecorder()

	router := gin.New()
	router.GET("/auth/tenants", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetTenants)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_SwitchTenantWithInternalError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()
	userID := uuid.New()
	tenantID := uuid.New()

	mockService.switchTenantError = errors.New("internal error")
	defer func() { mockService.switchTenantError = nil }()

	body := SwitchTenantRequest{
		TenantID: tenantID.String(),
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/auth/switch-tenant", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/switch-tenant", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.SwitchTenant)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePasswordWithInternalError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()
	userID := uuid.New()

	mockService.changePasswordError = errors.New("internal error")
	defer func() { mockService.changePasswordError = nil }()

	body := ChangePasswordRequest{
		CurrentPassword: "OldPass123!",
		NewPassword:     "NewPass123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/auth/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.PUT("/auth/password", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ChangePassword)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePasswordWithAuthError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()
	userID := uuid.New()

	mockService.changePasswordError = apperrors.ErrInvalidCredentials
	defer func() { mockService.changePasswordError = nil }()

	body := ChangePasswordRequest{
		CurrentPassword: "WrongPass123!",
		NewPassword:     "NewPass123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/auth/password", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.PUT("/auth/password", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.ChangePassword)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_VerifyEmail(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	// Test successful verification
	t.Run("successful verification", func(t *testing.T) {
		mockService.verifyEmailFunc = func(ctx context.Context, token string) (*entity.User, error) {
			now := time.Now()
			return &entity.User{
				ID:              uuid.New(),
				Email:           "verified@example.com",
				EmailVerified:   true,
				EmailVerifiedAt: &now,
			}, nil
		}

		body := VerifyEmailRequest{
			Token: "valid_token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/verify-email", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/verify-email", handler.VerifyEmail)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["message"] != "email verified successfully" {
			t.Errorf("unexpected message: %s", resp["message"])
		}
	})

	// Test invalid token
	t.Run("invalid token", func(t *testing.T) {
		mockService.verifyEmailFunc = func(ctx context.Context, token string) (*entity.User, error) {
			return nil, apperrors.ErrInvalidVerificationToken
		}

		body := VerifyEmailRequest{
			Token: "invalid_token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/verify-email", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/verify-email", handler.VerifyEmail)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test expired token
	t.Run("expired token", func(t *testing.T) {
		mockService.verifyEmailFunc = func(ctx context.Context, token string) (*entity.User, error) {
			return nil, apperrors.ErrVerificationTokenExpired
		}

		body := VerifyEmailRequest{
			Token: "expired_token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/verify-email", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/verify-email", handler.VerifyEmail)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test already verified
	t.Run("already verified", func(t *testing.T) {
		mockService.verifyEmailFunc = func(ctx context.Context, token string) (*entity.User, error) {
			return nil, apperrors.ErrEmailAlreadyVerified
		}

		body := VerifyEmailRequest{
			Token: "already_verified_token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/verify-email", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/verify-email", handler.VerifyEmail)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 for already verified, got %d", w.Code)
		}
	})

	// Test internal error
	t.Run("internal error", func(t *testing.T) {
		mockService.verifyEmailFunc = func(ctx context.Context, token string) (*entity.User, error) {
			return nil, errors.New("internal error")
		}

		body := VerifyEmailRequest{
			Token: "error_token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/verify-email", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/verify-email", handler.VerifyEmail)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	// Test missing token
	t.Run("missing token", func(t *testing.T) {
		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/verify-email", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/verify-email", handler.VerifyEmail)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestAuthHandler_ResendVerification(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()

	// Test successful resend
	t.Run("successful resend", func(t *testing.T) {
		mockService.resendVerificationFunc = func(ctx context.Context, email string) error {
			return nil
		}

		body := ResendVerificationRequest{
			Email: "resend@example.com",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/resend-verification", handler.ResendVerification)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test user not found (should still return success to prevent enumeration)
	t.Run("user not found", func(t *testing.T) {
		mockService.resendVerificationFunc = func(ctx context.Context, email string) error {
			return apperrors.ErrUserNotFound
		}

		body := ResendVerificationRequest{
			Email: "notfound@example.com",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/resend-verification", handler.ResendVerification)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 (prevent enumeration), got %d", w.Code)
		}
	})

	// Test already verified
	t.Run("already verified", func(t *testing.T) {
		mockService.resendVerificationFunc = func(ctx context.Context, email string) error {
			return apperrors.ErrEmailAlreadyVerified
		}

		body := ResendVerificationRequest{
			Email: "verified@example.com",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/resend-verification", handler.ResendVerification)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test internal error
	t.Run("internal error", func(t *testing.T) {
		mockService.resendVerificationFunc = func(ctx context.Context, email string) error {
			return errors.New("internal error")
		}

		body := ResendVerificationRequest{
			Email: "error@example.com",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/resend-verification", handler.ResendVerification)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	// Test invalid email format
	t.Run("invalid email", func(t *testing.T) {
		body := map[string]string{
			"email": "invalid-email",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/resend-verification", handler.ResendVerification)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test missing email
	t.Run("missing email", func(t *testing.T) {
		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/resend-verification", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.New()
		router.POST("/auth/resend-verification", handler.ResendVerification)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestAuthHandler_LogoutAllWithError(t *testing.T) {
	handler, mockService, _ := setupAuthHandler()
	userID := uuid.New()

	mockService.logoutAllError = errors.New("logout all failed")
	defer func() { mockService.logoutAllError = nil }()

	req := httptest.NewRequest("POST", "/auth/logout-all", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/auth/logout-all", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.LogoutAll)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// Add missing functions to mockUserAuthService
func (m *mockUserAuthService) VerifyEmail(ctx context.Context, token string) (*entity.User, error) {
	if m.verifyEmailFunc != nil {
		return m.verifyEmailFunc(ctx, token)
	}
	return nil, nil
}

func (m *mockUserAuthService) ResendVerification(ctx context.Context, email string) error {
	if m.resendVerificationFunc != nil {
		return m.resendVerificationFunc(ctx, email)
	}
	return nil
}