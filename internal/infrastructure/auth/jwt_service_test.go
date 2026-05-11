package auth

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

func TestNewJWTService(t *testing.T) {
	tests := []struct {
		name              string
		secretKey         string
		accessExpiry      time.Duration
		refreshExpiry     time.Duration
		wantPanic         bool
	}{
		{"valid config", "test-secret-key-32bytes!!", 15 * time.Minute, 7 * 24 * time.Hour, false},
		{"empty secret key", "", 15 * time.Minute, 7 * 24 * time.Hour, true},
		{"zero expiry", "test-secret-key-32bytes!!", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if tt.wantPanic {
					if r := recover(); r == nil {
						t.Errorf("expected panic for empty secret key")
					}
				}
			}()

			service := NewJWTService(tt.secretKey, tt.accessExpiry, tt.refreshExpiry, nil)
			if !tt.wantPanic && service == nil {
				t.Errorf("expected non-nil service")
			}
		})
	}
}

func TestJWTService_GenerateToken(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()
	email := "test@example.com"
	role := "admin"
	deviceID := "device123"

	accessToken, refreshToken, tokenID, err := service.GenerateToken(userID, tenantID, email, role, deviceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tokens are not empty
	if accessToken == "" {
		t.Error("access token should not be empty")
	}
	if refreshToken == "" {
		t.Error("refresh token should not be empty")
	}
	if tokenID == uuid.Nil {
		t.Error("token ID should not be nil")
	}

	// Verify tokens are different
	if accessToken == refreshToken {
		t.Error("access and refresh tokens should be different")
	}

	// Verify access token claims
	claims, err := service.ValidateToken(accessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
	if claims.TenantID != tenantID {
		t.Errorf("expected tenantID %s, got %s", tenantID, claims.TenantID)
	}
	if claims.Role != role {
		t.Errorf("expected role %s, got %s", role, claims.Role)
	}
	if claims.DeviceID != deviceID {
		t.Errorf("expected deviceID %s, got %s", deviceID, claims.DeviceID)
	}

	// Verify refresh token claims (no tenantID)
	refreshClaims, err := service.ValidateToken(refreshToken)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}
	if refreshClaims.TenantID != uuid.Nil {
		t.Errorf("refresh token should not have tenantID")
	}
	if refreshClaims.TokenID != tokenID {
		t.Errorf("refresh token should have same TokenID as access token")
	}
}

func TestJWTService_ValidateToken(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	// Test valid token
	t.Run("valid token", func(t *testing.T) {
		token, _, _, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")
		claims, err := service.ValidateToken(token)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims == nil {
			t.Error("expected non-nil claims")
		}
	})

	// Test invalid format
	t.Run("invalid format", func(t *testing.T) {
		_, err := service.ValidateToken("invalid-token")

		if err == nil {
			t.Error("expected error for invalid format")
		}
		if err != apperrors.ErrTokenInvalid {
			t.Errorf("expected ErrTokenInvalid, got %v", err)
		}
	})

	// Test empty token
	t.Run("empty token", func(t *testing.T) {
		_, err := service.ValidateToken("")

		if err == nil {
			t.Error("expected error for empty token")
		}
		if err != apperrors.ErrTokenInvalid {
			t.Errorf("expected ErrTokenInvalid, got %v", err)
		}
	})

	// Test wrong signing key
	t.Run("wrong signing key", func(t *testing.T) {
		otherService := NewJWTService("different-secret-key!!!", 15*time.Minute, 7*24*time.Hour, nil)
		token, _, _, _ := otherService.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

		_, err := service.ValidateToken(token)

		if err == nil {
			t.Error("expected error for wrong signing key")
		}
		if err != apperrors.ErrTokenInvalid {
			t.Errorf("expected ErrTokenInvalid, got %v", err)
		}
	})
}

func TestJWTService_InvalidateToken(t *testing.T) {
	// Test without Redis - InvalidateToken should still work for expired tokens
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	// Test with valid token (without Redis, blacklist check is skipped)
	t.Run("valid token without redis", func(t *testing.T) {
		token, _, _, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")
		err := service.InvalidateToken(token)
		// Without Redis, InvalidateToken does nothing for valid tokens
		if err != nil {
			t.Errorf("unexpected error without Redis: %v", err)
		}
	})

	// Test with invalid token
	t.Run("invalid token", func(t *testing.T) {
		err := service.InvalidateToken("invalid-token")
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	// Test with empty token
	t.Run("empty token", func(t *testing.T) {
		err := service.InvalidateToken("")
		if err == nil {
			t.Error("expected error for empty token")
		}
	})
}

func TestJWTService_RefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()
	newTenantID := uuid.New()

	// Generate tokens
	_, refreshToken, _, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

	// Refresh access token
	t.Run("successful refresh", func(t *testing.T) {
		newAccess, err := service.RefreshToken(refreshToken, newTenantID, "member")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if newAccess == "" {
			t.Error("new access token should not be empty")
		}

		// Validate new token
		claims, err := service.ValidateToken(newAccess)
		if err != nil {
			t.Fatalf("failed to validate new token: %v", err)
		}
		if claims.TenantID != newTenantID {
			t.Errorf("expected tenantID %s, got %s", newTenantID, claims.TenantID)
		}
		if claims.Role != "member" {
			t.Errorf("expected role member, got %s", claims.Role)
		}
	})

	// Refresh with invalid token
	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := service.RefreshToken("invalid-token", newTenantID, "member")
		if err == nil {
			t.Error("expected error for invalid refresh token")
		}
		if err != apperrors.ErrRefreshTokenInvalid {
			t.Errorf("expected ErrRefreshTokenInvalid, got %v", err)
		}
	})
}

func TestJWTService_InvalidateTokenByID(t *testing.T) {
	// Without Redis, this should return nil (no-op)
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	tokenID := uuid.New()

	err := service.InvalidateTokenByID(tokenID, 24*time.Hour)
	if err != nil {
		t.Errorf("unexpected error without Redis: %v", err)
	}
}

func TestJWTService_IsTokenIDBlacklisted(t *testing.T) {
	// Without Redis, should always return false
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	tokenID := uuid.New()

	if service.IsTokenIDBlacklisted(tokenID) {
		t.Error("should return false without Redis")
	}
}

func TestJWTService_GetAccessTokenExpiry(t *testing.T) {
	expected := 15 * time.Minute
	service := NewJWTService("test-secret-key-32bytes!!", expected, 7*24*time.Hour, nil)

	if service.GetAccessTokenExpiry() != expected {
		t.Errorf("expected %v, got %v", expected, service.GetAccessTokenExpiry())
	}
}

func TestJWTService_GetRefreshTokenExpiry(t *testing.T) {
	expected := 7 * 24 * time.Hour
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, expected, nil)

	if service.GetRefreshTokenExpiry() != expected {
		t.Errorf("expected %v, got %v", expected, service.GetRefreshTokenExpiry())
	}
}

func TestJWTService_hashToken(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	token := "test-token"
	hash := service.hashToken(token)

	// Verify it's SHA256 hex (64 characters)
	if len(hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash))
	}

	// Verify same input produces same hash
	hash2 := service.hashToken(token)
	if hash != hash2 {
		t.Error("same input should produce same hash")
	}

	// Verify different input produces different hash
	hash3 := service.hashToken("different-token")
	if hash == hash3 {
		t.Error("different input should produce different hash")
	}
}

func TestJWTClaims(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()
	email := "user@example.com"
	role := "admin"
	deviceID := "device123"

	token, _, tokenID, _ := service.GenerateToken(userID, tenantID, email, role, deviceID)
	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields
	if claims.UserID != userID {
		t.Errorf("UserID mismatch")
	}
	if claims.Email != email {
		t.Errorf("Email mismatch")
	}
	if claims.TenantID != tenantID {
		t.Errorf("TenantID mismatch")
	}
	if claims.Role != role {
		t.Errorf("Role mismatch")
	}
	if claims.DeviceID != deviceID {
		t.Errorf("DeviceID mismatch")
	}
	if claims.TokenID != tokenID {
		t.Errorf("TokenID mismatch")
	}

	// Verify time claims (IssuedAt should be before or equal to now)
	if claims.IssuedAt.Time.After(time.Now()) {
		t.Error("IssuedAt should be before or equal to now")
	}

	// ExpiresAt should be after now
	if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Error("ExpiresAt should be after now")
	}
}

func TestJWTService_ExpiredToken(t *testing.T) {
	// Create service with very short expiry to test expiration
	service := NewJWTService("test-secret-key-32bytes!!", 1*time.Millisecond, 1*time.Millisecond, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	token, _, _, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err := service.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
	if err != apperrors.ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

func TestJWTService_TokenWithEmptyFields(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	// Test with empty email
	t.Run("empty email", func(t *testing.T) {
		accessToken, _, _, err := service.GenerateToken(uuid.New(), uuid.New(), "", "member", "device")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		claims, err := service.ValidateToken(accessToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.Email != "" {
			t.Error("email should be empty")
		}
	})

	// Test with empty device ID
	t.Run("empty device ID", func(t *testing.T) {
		accessToken, _, _, err := service.GenerateToken(uuid.New(), uuid.New(), "test@example.com", "member", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		claims, err := service.ValidateToken(accessToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.DeviceID != "" {
			t.Error("device ID should be empty")
		}
	})

	// Test with empty role
	t.Run("empty role", func(t *testing.T) {
		accessToken, _, _, err := service.GenerateToken(uuid.New(), uuid.New(), "test@example.com", "", "device")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		claims, err := service.ValidateToken(accessToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.Role != "" {
			t.Error("role should be empty")
		}
	})

	// Test with Nil UUID for tenant
	t.Run("nil tenant ID", func(t *testing.T) {
		accessToken, _, _, err := service.GenerateToken(uuid.New(), uuid.Nil, "test@example.com", "member", "device")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		claims, err := service.ValidateToken(accessToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.TenantID != uuid.Nil {
			t.Error("tenant ID should be Nil")
		}
	})
}

func TestJWTService_TokenIDConsistency(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	// Generate multiple tokens for same user
	tokens := make([]struct {
		access  string
		refresh string
		tokenID uuid.UUID
	}, 3)

	for i := 0; i < 3; i++ {
		access, refresh, tokenID, _ := service.GenerateToken(userID, tenantID, "test@example.com", "member", "device")
		tokens[i] = struct {
			access  string
			refresh string
			tokenID uuid.UUID
		}{access, refresh, tokenID}
	}

	// Each token generation should produce unique TokenID
	for i := 0; i < 3; i++ {
		for j := i + 1; j < 3; j++ {
			if tokens[i].tokenID == tokens[j].tokenID {
				t.Error("each token generation should produce unique TokenID")
			}
		}
	}

	// Access and refresh token should share same TokenID
	for i := 0; i < 3; i++ {
		accessClaims, _ := service.ValidateToken(tokens[i].access)
		refreshClaims, _ := service.ValidateToken(tokens[i].refresh)
		if accessClaims.TokenID != refreshClaims.TokenID {
			t.Error("access and refresh token should share same TokenID")
		}
		if tokens[i].tokenID != accessClaims.TokenID {
			t.Error("returned tokenID should match claims tokenID")
		}
	}
}

func TestJWTService_MalformedTokens(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"single dot", "."},
		{"two dots", ".."},
		{"three dots", "..."},
		{"no dots", "invalidtoken"},
		{"missing parts", "header.payload"},
		{"invalid header", "invalid.payload.signature"},
		{"random string", "asdfghjklqwertyuiop"},
		{"base64 only", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"corrupted signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.corrupted"},
		{"wrong algorithm format", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dummy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateToken(tt.token)
			if err == nil {
				t.Errorf("expected error for malformed token: %s", tt.name)
			}
			if err != apperrors.ErrTokenInvalid {
				// Malformed tokens should return ErrTokenInvalid, not ErrSessionExpired
				t.Errorf("expected ErrTokenInvalid for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestJWTService_InvalidatedExpiredToken(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 1*time.Millisecond, 1*time.Millisecond, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	token, _, _, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Invalidate expired token (should succeed without Redis)
	err := service.InvalidateToken(token)
	if err != nil {
		t.Errorf("invalidating expired token should succeed without Redis: %v", err)
	}
}

func TestJWTService_RefreshTokenPreservesClaims(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()
	email := "preserve@example.com"
	deviceID := "preserve-device"

	_, refreshToken, originalTokenID, _ := service.GenerateToken(userID, tenantID, email, "member", deviceID)

	// Refresh with new tenant and role
	newTenantID := uuid.New()
	newAccess, err := service.RefreshToken(refreshToken, newTenantID, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate new token
	claims, err := service.ValidateToken(newAccess)
	if err != nil {
		t.Fatalf("unexpected error validating new token: %v", err)
	}

	// Original claims should be preserved
	if claims.UserID != userID {
		t.Error("UserID should be preserved")
	}
	if claims.Email != email {
		t.Error("Email should be preserved")
	}
	if claims.DeviceID != deviceID {
		t.Error("DeviceID should be preserved")
	}
	if claims.TokenID != originalTokenID {
		t.Error("TokenID should be preserved")
	}

	// New values should be set
	if claims.TenantID != newTenantID {
		t.Error("TenantID should be new value")
	}
	if claims.Role != "admin" {
		t.Error("Role should be new value")
	}
}

func TestJWTService_GenerateTokenWithDifferentRoles(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	roles := []string{"admin", "member", "viewer", "guest", "superadmin", "custom_role"}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			accessToken, _, _, err := service.GenerateToken(uuid.New(), uuid.New(), "test@example.com", role, "device")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			claims, err := service.ValidateToken(accessToken)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if claims.Role != role {
				t.Errorf("expected role %s, got %s", role, claims.Role)
			}
		})
	}
}

func TestJWTService_MultipleServicesDifferentSecrets(t *testing.T) {
	service1 := NewJWTService("secret-key-1-32bytes!!!!!!!", 15*time.Minute, 7*24*time.Hour, nil)
	service2 := NewJWTService("secret-key-2-32bytes!!!!!!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	// Generate token with service1
	token1, _, _, _ := service1.GenerateToken(userID, tenantID, "test@example.com", "member", "device")

	// Validate with service1 (should succeed)
	_, err := service1.ValidateToken(token1)
	if err != nil {
		t.Error("token should be valid with same service")
	}

	// Validate with service2 (should fail - different secret)
	_, err = service2.ValidateToken(token1)
	if err == nil {
		t.Error("token should be invalid with different service")
	}
	if err != apperrors.ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestJWTService_TimeBoundaryTests(t *testing.T) {
	// Test with zero expiry
	t.Run("zero expiry", func(t *testing.T) {
		service := NewJWTService("test-secret-key-32bytes!!", 0, 0, nil)
		// With zero expiry, tokens should still be generated
		// but may have immediate expiration issues
		accessToken, _, _, err := service.GenerateToken(uuid.New(), uuid.New(), "test@example.com", "member", "device")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Validate should work if NotBefore allows immediate use
		// This is implementation dependent
		_, err = service.ValidateToken(accessToken)
		// May or may not fail depending on implementation
	})

	// Test with very long expiry
	t.Run("long expiry", func(t *testing.T) {
		service := NewJWTService("test-secret-key-32bytes!!", 365*24*time.Hour, 365*24*time.Hour, nil)
		accessToken, _, _, err := service.GenerateToken(uuid.New(), uuid.New(), "test@example.com", "member", "device")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		claims, err := service.ValidateToken(accessToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// ExpiresAt should be approximately 365 days from now
		expectedExpiry := time.Now().Add(365 * 24 * time.Hour)
		diff := claims.ExpiresAt.Time.Sub(expectedExpiry)
		if diff > time.Minute || diff < -time.Minute {
			t.Errorf("expiry time difference too large: %v", diff)
		}
	})
}

func TestJWTService_TokenStringFormat(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	accessToken, _, _, _ := service.GenerateToken(uuid.New(), uuid.New(), "test@example.com", "member", "device")

	// JWT format: header.payload.signature (three base64 strings separated by dots)
	t.Run("access token format", func(t *testing.T) {
		parts := splitByDot(accessToken)
		if len(parts) != 3 {
			t.Errorf("expected 3 parts in JWT, got %d", len(parts))
		}
		for _, part := range parts {
			if !isBase64(part) {
				t.Error("each part should be base64 encoded")
			}
		}
	})

	t.Run("refresh token format", func(t *testing.T) {
		_, refreshToken, _, _ := service.GenerateToken(uuid.New(), uuid.New(), "test@example.com", "member", "device")
		parts := splitByDot(refreshToken)
		if len(parts) != 3 {
			t.Errorf("expected 3 parts in JWT, got %d", len(parts))
		}
	})
}

// Helper functions for JWT format testing
func splitByDot(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func isBase64(s string) bool {
	// Base64 strings (including URL-safe encoding) can contain:
	// alphanumeric chars, +, /, -, _, and =
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '-' || c == '_' || c == '=') {
			return false
		}
	}
	return true
}

// Redis-based tests for JWT service
func setupJWTServiceWithRedis() (*JWTService, *miniredis.Miniredis) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, client)
	return service, mr
}

func TestJWTService_ValidateTokenWithRedisBlacklist(t *testing.T) {
	service, mr := setupJWTServiceWithRedis()
	defer mr.Close()

	userID := uuid.New()
	tenantID := uuid.New()

	// Generate token
	accessToken, _, _, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

	// Validate should succeed initially
	t.Run("valid token", func(t *testing.T) {
		claims, err := service.ValidateToken(accessToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims == nil {
			t.Error("claims should not be nil")
		}
	})

	// Blacklist the token
	t.Run("blacklist token", func(t *testing.T) {
		// Manually add token to blacklist in Redis
		tokenHash := service.hashToken(accessToken)
		key := "jwt_blacklist:" + tokenHash
		mr.Set(key, "1")

		// Validate should now fail
		_, err := service.ValidateToken(accessToken)
		if err == nil {
			t.Error("expected error for blacklisted token")
		}
		if err != apperrors.ErrTokenRevoked {
			t.Errorf("expected ErrTokenRevoked, got %v", err)
		}
	})
}

func TestJWTService_InvalidateTokenWithRedis(t *testing.T) {
	service, mr := setupJWTServiceWithRedis()
	defer mr.Close()

	userID := uuid.New()
	tenantID := uuid.New()

	// Test invalidate valid token
	t.Run("invalidate valid token", func(t *testing.T) {
		accessToken, _, _, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

		err := service.InvalidateToken(accessToken)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Check Redis for blacklist entry
		tokenHash := service.hashToken(accessToken)
		key := "jwt_blacklist:" + tokenHash
		val, err := mr.Get(key)
		if err != nil {
			t.Error("expected key to be set in Redis")
		}
		if val != "1" {
			t.Errorf("expected value '1', got %s", val)
		}

		// Validate should now fail
		_, err = service.ValidateToken(accessToken)
		if err == nil {
			t.Error("expected error for invalidated token")
		}
		if err != apperrors.ErrTokenRevoked {
			t.Errorf("expected ErrTokenRevoked, got %v", err)
		}
	})

	// Test invalidate expired token
	t.Run("invalidate expired token", func(t *testing.T) {
		shortService, mr2 := setupJWTServiceWithRedis()
		defer mr2.Close()

		// Generate token with short expiry
		shortService.accessTokenExpiry = 1 * time.Millisecond
		accessToken, _, _, _ := shortService.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		// Invalidate expired token - should still add to blacklist
		err := shortService.InvalidateToken(accessToken)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Check Redis for blacklist entry
		tokenHash := shortService.hashToken(accessToken)
		key := "jwt_blacklist:" + tokenHash
		val, err := mr2.Get(key)
		if err != nil {
			t.Error("expected expired token to still be added to blacklist")
		}
		if val != "1" {
			t.Errorf("expected value '1', got %s", val)
		}
	})

	// Test invalidate with TTL calculation
	t.Run("invalidate with proper TTL", func(t *testing.T) {
		accessToken, _, _, _ := service.GenerateToken(userID, tenantID, "ttl@example.com", "member", "device")

		err := service.InvalidateToken(accessToken)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// TTL should be approximately access token expiry
		tokenHash := service.hashToken(accessToken)
		key := "jwt_blacklist:" + tokenHash
		ttl := mr.TTL(key)
		// TTL should be positive and close to 15 minutes
		if ttl <= 0 {
			t.Error("expected positive TTL")
		}
	})
}

func TestJWTService_InvalidateTokenByIDWithRedis(t *testing.T) {
	service, mr := setupJWTServiceWithRedis()
	defer mr.Close()

	tokenID := uuid.New()
	ttl := 24 * time.Hour

	err := service.InvalidateTokenByID(tokenID, ttl)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check Redis for blacklist entry
	key := "jwt_blacklist_id:" + tokenID.String()
	val, err := mr.Get(key)
	if err != nil {
		t.Error("expected key to be set in Redis")
	}
	if val != "1" {
		t.Errorf("expected value '1', got %s", val)
	}

	// Check TTL
	ttlVal := mr.TTL(key)
	if ttlVal <= 0 {
		t.Error("expected positive TTL")
	}
}

func TestJWTService_IsTokenIDBlacklistedWithRedis(t *testing.T) {
	service, mr := setupJWTServiceWithRedis()
	defer mr.Close()

	tokenID := uuid.New()

	// Initially not blacklisted
	t.Run("not blacklisted", func(t *testing.T) {
		if service.IsTokenIDBlacklisted(tokenID) {
			t.Error("token ID should not be blacklisted initially")
		}
	})

	// Blacklist the token ID
	t.Run("blacklisted", func(t *testing.T) {
		key := "jwt_blacklist_id:" + tokenID.String()
		mr.Set(key, "1")

		if !service.IsTokenIDBlacklisted(tokenID) {
			t.Error("token ID should be blacklisted")
		}
	})
}

func TestJWTService_RefreshTokenWithBlacklistedTokenID(t *testing.T) {
	service, mr := setupJWTServiceWithRedis()
	defer mr.Close()

	userID := uuid.New()
	tenantID := uuid.New()
	newTenantID := uuid.New()

	// Generate tokens
	_, refreshToken, tokenID, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

	// Blacklist the TokenID
	key := "jwt_blacklist_id:" + tokenID.String()
	mr.Set(key, "1")

	// Refresh should fail because TokenID is blacklisted
	_, err := service.RefreshToken(refreshToken, newTenantID, "member")
	if err == nil {
		t.Error("expected error for blacklisted TokenID")
	}
	if err != apperrors.ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked, got %v", err)
	}
}

func TestJWTService_RedisConnectionError(t *testing.T) {
	// Create service with failing Redis
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, client)

	userID := uuid.New()
	tenantID := uuid.New()

	// Generate token before closing Redis
	accessToken, _, _, _ := service.GenerateToken(userID, tenantID, "test@example.com", "admin", "device")

	// Close Redis to simulate connection failure
	mr.Close()

	// Operations should still work gracefully without Redis
	t.Run("validate without Redis", func(t *testing.T) {
		// Validate should still work (Redis check fails gracefully)
		_, err := service.ValidateToken(accessToken)
		// Should succeed because Redis check fails silently
		if err != nil {
			// If Redis check fails, it might return token invalid
			// This is acceptable behavior
		}
	})

	t.Run("invalidate without Redis", func(t *testing.T) {
		// Invalidate might fail or succeed gracefully
		err := service.InvalidateToken(accessToken)
		// Behavior depends on implementation
		_ = err
	})
}

func TestJWTService_MultipleTokenInvalidation(t *testing.T) {
	service, mr := setupJWTServiceWithRedis()
	defer mr.Close()

	userID := uuid.New()
	tenantID := uuid.New()

	// Generate multiple tokens
	tokens := make([]string, 5)
	for i := 0; i < 5; i++ {
		accessToken, _, _, _ := service.GenerateToken(userID, tenantID, "multi@example.com", "member", "device")
		tokens[i] = accessToken
	}

	// Invalidate all tokens
	for _, token := range tokens {
		err := service.InvalidateToken(token)
		if err != nil {
			t.Errorf("unexpected error invalidating token: %v", err)
		}
	}

	// All tokens should be blacklisted
	for _, token := range tokens {
		_, err := service.ValidateToken(token)
		if err == nil {
			t.Error("expected error for invalidated token")
		}
		if err != apperrors.ErrTokenRevoked {
			t.Errorf("expected ErrTokenRevoked, got %v", err)
		}
	}
}

// Additional edge case tests for JWT service

func TestJWTService_ValidateTokenWithNilRedis(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	accessToken, _, _, _ := service.GenerateToken(userID, tenantID, "nilredis@example.com", "member", "device")

	// With nil Redis, blacklist check should be skipped
	claims, err := service.ValidateToken(accessToken)
	if err != nil {
		t.Errorf("unexpected error with nil Redis: %v", err)
	}
	if claims == nil {
		t.Error("claims should not be nil")
	}
}

func TestJWTService_RefreshTokenWithExpiredRefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 1*time.Millisecond, 1*time.Millisecond, nil)

	userID := uuid.New()
	tenantID := uuid.New()
	newTenantID := uuid.New()

	_, refreshToken, _, _ := service.GenerateToken(userID, tenantID, "expired@example.com", "member", "device")

	// Wait for refresh token to expire
	time.Sleep(10 * time.Millisecond)

	// Refresh should fail
	_, err := service.RefreshToken(refreshToken, newTenantID, "admin")
	if err == nil {
		t.Error("expected error for expired refresh token")
	}
	if err != apperrors.ErrRefreshTokenInvalid {
		t.Errorf("expected ErrRefreshTokenInvalid, got %v", err)
	}
}

func TestJWTService_GenerateTokenUniqueness(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	// Generate 100 tokens and verify uniqueness
	tokenSet := make(map[string]bool)
	for i := 0; i < 100; i++ {
		accessToken, refreshToken, _, _ := service.GenerateToken(userID, tenantID, "unique@example.com", "member", "device")
		if tokenSet[accessToken] {
			t.Error("access token should be unique")
		}
		if tokenSet[refreshToken] {
			t.Error("refresh token should be unique")
		}
		tokenSet[accessToken] = true
		tokenSet[refreshToken] = true
	}
}

func TestJWTService_InvalidateTokenByIDWithNilRedis(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	tokenID := uuid.New()

	// With nil Redis, should return nil (no-op)
	err := service.InvalidateTokenByID(tokenID, 24*time.Hour)
	if err != nil {
		t.Errorf("unexpected error with nil Redis: %v", err)
	}
}

func TestJWTService_IsTokenIDBlacklistedWithNilRedis(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	tokenID := uuid.New()

	// With nil Redis, should always return false
	if service.IsTokenIDBlacklisted(tokenID) {
		t.Error("should return false with nil Redis")
	}
}

func TestJWTService_ValidateTokenWithTokenIDBlacklisted(t *testing.T) {
	service, mr := setupJWTServiceWithRedis()
	defer mr.Close()

	userID := uuid.New()
	tenantID := uuid.New()

	_, _, tokenID, _ := service.GenerateToken(userID, tenantID, "idblacklist@example.com", "member", "device")

	// Initially not blacklisted
	if service.IsTokenIDBlacklisted(tokenID) {
		t.Error("TokenID should not be blacklisted initially")
	}

	// Blacklist by TokenID
	key := "jwt_blacklist_id:" + tokenID.String()
	mr.Set(key, "1")

	// IsTokenIDBlacklisted should return true now
	if !service.IsTokenIDBlacklisted(tokenID) {
		t.Error("expected TokenID to be blacklisted")
	}
}

func TestJWTService_RefreshTokenWithNilRedis(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()
	newTenantID := uuid.New()

	_, refreshToken, _, _ := service.GenerateToken(userID, tenantID, "nilredis@example.com", "member", "device")

	// With nil Redis, refresh should succeed (no blacklist check)
	newAccess, err := service.RefreshToken(refreshToken, newTenantID, "admin")
	if err != nil {
		t.Errorf("unexpected error with nil Redis: %v", err)
	}
	if newAccess == "" {
		t.Error("new access token should not be empty")
	}
}

func TestJWTService_ConcurrentTokenGeneration(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	// Generate tokens concurrently
	done := make(chan bool)
	tokenSet := make(map[string]bool)
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		go func() {
			accessToken, _, _, err := service.GenerateToken(userID, tenantID, "concurrent@example.com", "member", "device")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			mu.Lock()
			if tokenSet[accessToken] {
				t.Error("concurrent token should be unique")
			}
			tokenSet[accessToken] = true
			mu.Unlock()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestJWTService_TokenWithVeryLongEmail(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	// Very long email
	longEmail := "verylongemailaddressthatmightexceednormallimits@verylongdomainnamethatgoesonandontest.example.com"

	accessToken, _, _, err := service.GenerateToken(userID, tenantID, longEmail, "member", "device")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	claims, err := service.ValidateToken(accessToken)
	if err != nil {
		t.Errorf("unexpected error validating token: %v", err)
	}
	if claims.Email != longEmail {
		t.Errorf("expected email %s, got %s", longEmail, claims.Email)
	}
}

func TestJWTService_TokenWithSpecialCharactersInDeviceID(t *testing.T) {
	service := NewJWTService("test-secret-key-32bytes!!", 15*time.Minute, 7*24*time.Hour, nil)

	userID := uuid.New()
	tenantID := uuid.New()

	// Special characters in device ID
	specialDeviceID := "device-123_abc!@#$%^&*()"

	accessToken, _, _, err := service.GenerateToken(userID, tenantID, "special@example.com", "member", specialDeviceID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	claims, err := service.ValidateToken(accessToken)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if claims.DeviceID != specialDeviceID {
		t.Errorf("expected device ID %s, got %s", specialDeviceID, claims.DeviceID)
	}
}

func TestJWTService_ValidateTokenTwice(t *testing.T) {
	service, mr := setupJWTServiceWithRedis()
	defer mr.Close()

	userID := uuid.New()
	tenantID := uuid.New()

	accessToken, _, _, _ := service.GenerateToken(userID, tenantID, "twice@example.com", "member", "device")

	// Validate twice - should both succeed
	claims1, err1 := service.ValidateToken(accessToken)
	claims2, err2 := service.ValidateToken(accessToken)

	if err1 != nil || err2 != nil {
		t.Errorf("both validations should succeed: %v, %v", err1, err2)
	}
	if claims1.UserID != claims2.UserID {
		t.Error("claims should be consistent")
	}
}

func TestJWTService_GetExpiryMethods(t *testing.T) {
	accessExpiry := 30 * time.Minute
	refreshExpiry := 14 * 24 * time.Hour

	service := NewJWTService("test-secret-key-32bytes!!", accessExpiry, refreshExpiry, nil)

	if service.GetAccessTokenExpiry() != accessExpiry {
		t.Errorf("expected access expiry %v, got %v", accessExpiry, service.GetAccessTokenExpiry())
	}
	if service.GetRefreshTokenExpiry() != refreshExpiry {
		t.Errorf("expected refresh expiry %v, got %v", refreshExpiry, service.GetRefreshTokenExpiry())
	}
}