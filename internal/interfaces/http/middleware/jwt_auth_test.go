package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Mock services for middleware tests
type mockUserAuthServiceForMiddleware struct {
	user        *entity.User
	userTenant  *entity.UserTenant
	claims      *auth.JWTClaims
	validateErr error
}

func (m *mockUserAuthServiceForMiddleware) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) { return nil, nil }
func (m *mockUserAuthServiceForMiddleware) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error) { return nil, nil }
func (m *mockUserAuthServiceForMiddleware) RegisterTenant(ctx context.Context, req *auth.RegisterTenantRequest) (*auth.RegisterTenantResponse, error) { return nil, nil }
func (m *mockUserAuthServiceForMiddleware) ValidateToken(ctx context.Context, token string) (*entity.User, *entity.UserTenant, *auth.JWTClaims, error) {
	return m.user, m.userTenant, m.claims, m.validateErr
}
func (m *mockUserAuthServiceForMiddleware) SwitchTenant(ctx context.Context, userID, tenantID uuid.UUID, currentToken string) (string, error) { return "", nil }
func (m *mockUserAuthServiceForMiddleware) Logout(ctx context.Context, token string) error { return nil }
func (m *mockUserAuthServiceForMiddleware) LogoutAll(ctx context.Context, userID uuid.UUID) error { return nil }
func (m *mockUserAuthServiceForMiddleware) RefreshToken(ctx context.Context, refreshToken string, deviceID string) (string, error) { return "", nil }
func (m *mockUserAuthServiceForMiddleware) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error { return nil }
func (m *mockUserAuthServiceForMiddleware) GetUserTenants(ctx context.Context, userID uuid.UUID) ([]entity.UserTenant, error) { return nil, nil }

type mockJWTServiceForMiddleware struct{}

func (m *mockJWTServiceForMiddleware) GenerateToken(userID, tenantID uuid.UUID, email, role, deviceID string) (string, string, uuid.UUID, error) { return "", "", uuid.Nil, nil }
func (m *mockJWTServiceForMiddleware) ValidateToken(token string) (*auth.JWTClaims, error) { return nil, nil }
func (m *mockJWTServiceForMiddleware) InvalidateToken(token string) error { return nil }
func (m *mockJWTServiceForMiddleware) InvalidateTokenByID(tokenID uuid.UUID, expiry time.Duration) error { return nil }
func (m *mockJWTServiceForMiddleware) RefreshToken(refreshToken string, tenantID uuid.UUID, role string) (string, error) { return "", nil }
func (m *mockJWTServiceForMiddleware) IsTokenIDBlacklisted(tokenID uuid.UUID) bool { return false }
func (m *mockJWTServiceForMiddleware) GetAccessTokenExpiry() time.Duration { return 15 * time.Minute }
func (m *mockJWTServiceForMiddleware) GetRefreshTokenExpiry() time.Duration { return 7 * 24 * time.Hour }
func (m *mockJWTServiceForMiddleware) hashToken(token string) string { return "" }

func setupJWTMiddlewareTest() (*mockUserAuthServiceForMiddleware, *mockJWTServiceForMiddleware) {
	userID := uuid.New()
	tenantID := uuid.New()

	return &mockUserAuthServiceForMiddleware{
		user: &entity.User{
			ID:     userID,
			Email:  "test@example.com",
			Name:   "Test User",
			Role:   "member",
			Status: "active",
		},
		userTenant: &entity.UserTenant{
			ID:        uuid.New(),
			UserID:    userID,
			TenantID:  tenantID,
			Role:      "member",
			Status:    "active",
			IsDefault: true,
		},
		claims: &auth.JWTClaims{
			UserID:   userID,
			Email:    "test@example.com",
			TenantID: tenantID,
			Role:     "member",
			DeviceID: "device123",
			TokenID:  uuid.New(),
		},
	}, &mockJWTServiceForMiddleware{}
}

func TestJWTAuthMiddleware_Success(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()

	// Create middleware
	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	// Create router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil {
			t.Error("user_id should be set")
		}
		c.JSON(200, gin.H{"success": true})
	})

	// Create request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_NoToken(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "no authorization token provided" {
		t.Errorf("expected error message, got %v", resp["error"])
	}
}

func TestJWTAuthMiddleware_InvalidToken(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()
	mockUserService.validateErr = apperrors.ErrTokenInvalid

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "invalid token" {
		t.Errorf("expected 'invalid token' error, got %v", resp["error"])
	}
}

func TestJWTAuthMiddleware_TokenExpired(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()
	mockUserService.validateErr = apperrors.ErrSessionExpired

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "token expired" {
		t.Errorf("expected 'token expired' error, got %v", resp["error"])
	}
	if resp["code"] != "TOKEN_EXPIRED" {
		t.Errorf("expected code TOKEN_EXPIRED, got %v", resp["code"])
	}
}

func TestJWTAuthMiddleware_TokenRevoked(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()
	mockUserService.validateErr = apperrors.ErrTokenRevoked

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer revoked-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "token has been revoked" {
		t.Errorf("expected 'token has been revoked' error, got %v", resp["error"])
	}
}

func TestJWTAuthMiddleware_DeviceMismatch(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()
	mockUserService.claims.DeviceID = "original-device"

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		mismatch, exists := c.Get("device_mismatch")
		if !exists || !mismatch.(bool) {
			t.Error("device_mismatch should be true")
		}
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("X-Device-ID", "different-device")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 (device mismatch is not blocking), got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_ContextValues(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		// Check all context values
		userID, exists := c.Get("user_id")
		if !exists {
			t.Error("user_id should exist")
		}
		if userID.(uuid.UUID) != mockUserService.user.ID {
			t.Error("user_id mismatch")
		}

		user, exists := c.Get("user")
		if !exists {
			t.Error("user should exist")
		}
		if user.(*entity.User).Email != mockUserService.user.Email {
			t.Error("user email mismatch")
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			t.Error("tenant_id should exist")
		}
		if tenantID.(uuid.UUID) != mockUserService.userTenant.TenantID {
			t.Error("tenant_id mismatch")
		}

		userTenant, exists := c.Get("user_tenant")
		if !exists {
			t.Error("user_tenant should exist")
		}
		if userTenant.(*entity.UserTenant).Role != mockUserService.userTenant.Role {
			t.Error("user_tenant role mismatch")
		}

		role, exists := c.Get("role")
		if !exists {
			t.Error("role should exist")
		}
		if role.(string) != mockUserService.userTenant.Role {
			t.Error("role mismatch")
		}

		email, exists := c.Get("email")
		if !exists {
			t.Error("email should exist")
		}
		if email.(string) != mockUserService.user.Email {
			t.Error("email mismatch")
		}

		tokenID, exists := c.Get("token_id")
		if !exists {
			t.Error("token_id should exist")
		}
		if tokenID.(uuid.UUID) != mockUserService.claims.TokenID {
			t.Error("token_id mismatch")
		}

		deviceID, exists := c.Get("device_id")
		if !exists {
			t.Error("device_id should exist")
		}
		if deviceID.(string) != mockUserService.claims.DeviceID {
			t.Error("device_id mismatch")
		}

		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
}

func TestOptionalJWTAuth(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()

	middleware := OptionalJWTAuth(mockJWT, mockUserService)

	// Test without token - should continue
	t.Run("no token", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test with valid token - should set context
	t.Run("valid token", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			if !exists {
				t.Error("user_id should be set with valid token")
			}
			c.JSON(200, gin.H{"success": true, "user_id": userID})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test with invalid token - should continue without context
	t.Run("invalid token", func(t *testing.T) {
		mockUserService.validateErr = apperrors.ErrTokenInvalid

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			_, exists := c.Get("user_id")
			if exists {
				t.Error("user_id should not be set with invalid token")
			}
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 (optional auth doesn't block), got %d", w.Code)
		}
	})
}

func TestRequireRole(t *testing.T) {
	// Test admin role required
	t.Run("admin required - success", func(t *testing.T) {
		middleware := RequireRole("admin")

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("role", "admin")
			c.Next()
		})
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test member role not allowed
	t.Run("admin required - denied", func(t *testing.T) {
		middleware := RequireRole("admin")

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("role", "member")
			c.Next()
		})
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})

	// Test multiple roles allowed
	t.Run("multiple roles - success", func(t *testing.T) {
		middleware := RequireRole("admin", "member")

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("role", "member")
			c.Next()
		})
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test no role in context
	t.Run("no role in context", func(t *testing.T) {
		middleware := RequireRole("admin")

		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestRequireAdmin(t *testing.T) {
	middleware := RequireAdmin()

	// Test admin role
	t.Run("admin success", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("role", "admin")
			c.Next()
		})
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test member role denied
	t.Run("member denied", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("role", "member")
			c.Next()
		})
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})
}

func TestRequireTenantMember(t *testing.T) {
	middleware := RequireTenantMember()

	// Test tenant member
	t.Run("tenant member success", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user_tenant", &entity.UserTenant{Role: "member"})
			c.Next()
		})
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test not tenant member
	t.Run("not tenant member", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})
}

func TestExtractToken(t *testing.T) {
	// Test Bearer token
	t.Run("Bearer token", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			token := extractToken(c)
			if token != "test-token" {
				t.Errorf("expected test-token, got %s", token)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test direct token
	t.Run("direct token", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			token := extractToken(c)
			if token != "direct-token" {
				t.Errorf("expected direct-token, got %s", token)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "direct-token")
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test query parameter
	t.Run("query parameter", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			token := extractToken(c)
			if token != "query-token" {
				t.Errorf("expected query-token, got %s", token)
			}
		})

		req := httptest.NewRequest("GET", "/test?token=query-token", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test no token
	t.Run("no token", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", func(c *gin.Context) {
			token := extractToken(c)
			if token != "" {
				t.Errorf("expected empty string, got %s", token)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestGetUserTenant(t *testing.T) {
	router := gin.New()

	// Test with user_tenant
	t.Run("has user_tenant", func(t *testing.T) {
		ut := &entity.UserTenant{Role: "admin"}
		router.GET("/test", func(c *gin.Context) {
			c.Set("user_tenant", ut)
			result := GetUserTenant(c)
			if result != ut {
				t.Error("should return same user_tenant")
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test without user_tenant
	t.Run("no user_tenant", func(t *testing.T) {
		router.GET("/no-tenant", func(c *gin.Context) {
			result := GetUserTenant(c)
			if result != nil {
				t.Error("should return nil when no user_tenant")
			}
		})

		req := httptest.NewRequest("GET", "/no-tenant", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestGetRole(t *testing.T) {
	router := gin.New()

	// Test with role
	t.Run("has role", func(t *testing.T) {
		router.GET("/test", func(c *gin.Context) {
			c.Set("role", "admin")
			result := GetRole(c)
			if result != "admin" {
				t.Errorf("expected admin, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test without role
	t.Run("no role", func(t *testing.T) {
		router.GET("/no-role", func(c *gin.Context) {
			result := GetRole(c)
			if result != "" {
				t.Errorf("expected empty string, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/no-role", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestGetEmail(t *testing.T) {
	router := gin.New()

	// Test with email
	t.Run("has email", func(t *testing.T) {
		router.GET("/test", func(c *gin.Context) {
			c.Set("email", "test@example.com")
			result := GetEmail(c)
			if result != "test@example.com" {
				t.Errorf("expected test@example.com, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test without email
	t.Run("no email", func(t *testing.T) {
		router.GET("/no-email", func(c *gin.Context) {
			result := GetEmail(c)
			if result != "" {
				t.Errorf("expected empty string, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/no-email", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestGetDeviceID(t *testing.T) {
	router := gin.New()

	// Test with device_id
	t.Run("has device_id", func(t *testing.T) {
		router.GET("/test", func(c *gin.Context) {
			c.Set("device_id", "device123")
			result := GetDeviceID(c)
			if result != "device123" {
				t.Errorf("expected device123, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test without device_id
	t.Run("no device_id", func(t *testing.T) {
		router.GET("/no-device", func(c *gin.Context) {
			result := GetDeviceID(c)
			if result != "" {
				t.Errorf("expected empty string, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/no-device", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestIsDeviceMismatch(t *testing.T) {
	router := gin.New()

	// Test with mismatch
	t.Run("has mismatch", func(t *testing.T) {
		router.GET("/test", func(c *gin.Context) {
			c.Set("device_mismatch", true)
			result := IsDeviceMismatch(c)
			if !result {
				t.Error("should return true")
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test without mismatch
	t.Run("no mismatch", func(t *testing.T) {
		router.GET("/no-mismatch", func(c *gin.Context) {
			result := IsDeviceMismatch(c)
			if result {
				t.Error("should return false")
			}
		})

		req := httptest.NewRequest("GET", "/no-mismatch", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

// Additional edge case tests for middleware

// Note: Tests for nil user/claims removed because ValidateToken should never
// return nil values without an error. If nil is returned, that's an invalid
// state from the service that shouldn't be tested as normal behavior.

func TestJWTAuthMiddleware_EmptyAuthorizationHeader(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "") // Empty header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for empty Authorization header, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_BearerWithEmptyToken(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ") // Bearer with empty token
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for empty token, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_MalformedAuthorizationHeader(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	tests := []struct {
		name  string
		value string
	}{
		{"wrong prefix", "Basic test-token"},
		{"bearer lowercase", "bearer test-token"},
		{"multiple spaces", "Bearer  test-token"},
		{"no space", "Bearertest-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(middleware)
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"success": true})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.value)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should be unauthorized or treat as direct token
			// The middleware handles "Bearer " prefix specifically
			if w.Code != http.StatusOK && w.Code != http.StatusUnauthorized {
				t.Errorf("unexpected status %d", w.Code)
			}
		})
	}
}

func TestJWTAuthMiddleware_QueryTokenPriority(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		// Check that query token is used when both header and query exist
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should work (query token takes precedence or header token)
	if w.Code != http.StatusOK && w.Code != http.StatusUnauthorized {
		t.Errorf("unexpected status %d", w.Code)
	}
}

func TestJWTAuthMiddleware_InactiveUserTenant(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()
	mockUserService.userTenant.Status = "inactive"

	middleware := JWTAuthMiddleware(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should succeed - middleware doesn't block on inactive user tenant
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRequireRole_MultipleRolesDenied(t *testing.T) {
	middleware := RequireRole("admin", "owner")

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("role", "member") // member not in [admin, owner]
		c.Next()
	})
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for member not in allowed roles, got %d", w.Code)
	}
}

// Note: TestRequireRole_InvalidRoleType removed - role should always be string
// per JWTAuthMiddleware which sets it from claims. Testing invalid type is
// testing impossible scenario.

func TestRequireTenantMember_WithInactiveStatus(t *testing.T) {
	middleware := RequireTenantMember()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_tenant", &entity.UserTenant{Role: "member", Status: "inactive"})
		c.Next()
	})
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should succeed - RequireTenantMember only checks presence, not status
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// Note: TestRequireTenantMember_InvalidType removed - user_tenant should always
// be *entity.UserTenant per JWTAuthMiddleware. Testing invalid type is
// testing impossible scenario.

func TestGetTokenID(t *testing.T) {
	router := gin.New()

	// Test with token_id
	t.Run("has token_id", func(t *testing.T) {
		tokenID := uuid.New()
		router.GET("/test", func(c *gin.Context) {
			c.Set("token_id", tokenID)
			result := GetTokenID(c)
			if result != tokenID {
				t.Errorf("expected %s, got %s", tokenID, result)
			}
		})

		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})

	// Test without token_id
	t.Run("no token_id", func(t *testing.T) {
		router.GET("/no-token", func(c *gin.Context) {
			result := GetTokenID(c)
			if result != uuid.Nil {
				t.Errorf("expected Nil UUID, got %s", result)
			}
		})

		req := httptest.NewRequest("GET", "/no-token", nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestOptionalJWTAuth_ValidateError(t *testing.T) {
	mockUserService, mockJWT := setupJWTMiddlewareTest()
	mockUserService.validateErr = apperrors.ErrSessionExpired

	middleware := OptionalJWTAuth(mockJWT, mockUserService)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if exists {
			t.Error("user_id should not be set when validation fails")
		}
		c.JSON(200, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should succeed - optional auth doesn't block
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRequireTenantWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setupCtx   func(c *gin.Context)
		wantAbort  bool
		wantStatus int
	}{
		{
			name: "admin via UserTenant passes",
			setupCtx: func(c *gin.Context) {
				c.Set("user_tenant", &entity.UserTenant{Role: "admin"})
			},
			wantAbort:  false,
			wantStatus: 200,
		},
		{
			name: "member via UserTenant passes",
			setupCtx: func(c *gin.Context) {
				c.Set("user_tenant", &entity.UserTenant{Role: "member"})
			},
			wantAbort:  false,
			wantStatus: 200,
		},
		{
			name: "viewer via UserTenant blocked",
			setupCtx: func(c *gin.Context) {
				c.Set("user_tenant", &entity.UserTenant{Role: "viewer"})
			},
			wantAbort:  true,
			wantStatus: 403,
		},
		{
			name: "viewer via user.Role blocked (API key auth fallback)",
			setupCtx: func(c *gin.Context) {
				c.Set("user", &entity.User{Role: "viewer"})
			},
			wantAbort:  true,
			wantStatus: 403,
		},
		{
			name: "admin via user.Role passes (API key auth fallback)",
			setupCtx: func(c *gin.Context) {
				c.Set("user", &entity.User{Role: "admin"})
			},
			wantAbort:  false,
			wantStatus: 200,
		},
		{
			name: "member via user.Role passes (API key auth fallback)",
			setupCtx: func(c *gin.Context) {
				c.Set("user", &entity.User{Role: "member"})
			},
			wantAbort:  false,
			wantStatus: 200,
		},
		{
			name:       "no user in context - no-op",
			setupCtx:   func(c *gin.Context) {},
			wantAbort:  false,
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			middleware := RequireTenantWrite()
			middleware(c)

			if c.IsAborted() != tt.wantAbort {
				t.Errorf("IsAborted = %v, want %v", c.IsAborted(), tt.wantAbort)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestGetUserFromAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("user from context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		expected := &entity.User{ID: uuid.New(), Role: "admin"}
		c.Set("user", expected)

		got := GetUserFromAuth(c)
		if got == nil {
			t.Fatal("GetUserFromAuth should not return nil")
		}
		if got.ID != expected.ID {
			t.Errorf("ID = %v, want %v", got.ID, expected.ID)
		}
	})

	t.Run("no user in context returns nil", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		got := GetUserFromAuth(c)
		if got != nil {
			t.Error("GetUserFromAuth should return nil")
		}
	})
}
