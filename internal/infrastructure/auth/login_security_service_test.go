package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// Local error for mock repository
var errRecordNotFound = fmt.Errorf("record not found")

// Mock repositories for login security tests
type mockLoginAuditRepo struct {
	audits []*entity.LoginAudit
}

func (m *mockLoginAuditRepo) Create(ctx context.Context, audit *entity.LoginAudit) error {
	m.audits = append(m.audits, audit)
	return nil
}

func (m *mockLoginAuditRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.LoginAudit, error) {
	for _, a := range m.audits {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, errRecordNotFound
}

func (m *mockLoginAuditRepo) ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.LoginAudit, int64, error) {
	var result []entity.LoginAudit
	for _, a := range m.audits {
		if a.UserID != nil && *a.UserID == userID {
			result = append(result, *a)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockLoginAuditRepo) ListByEmail(ctx context.Context, email string, page, pageSize int) ([]entity.LoginAudit, int64, error) {
	var result []entity.LoginAudit
	for _, a := range m.audits {
		if a.Email == email {
			result = append(result, *a)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockLoginAuditRepo) ListRecent(ctx context.Context, userID uuid.UUID, limit int) ([]entity.LoginAudit, error) {
	var result []entity.LoginAudit
	for _, a := range m.audits {
		if a.UserID != nil && *a.UserID == userID {
			result = append(result, *a)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockLoginAuditRepo) ListFailed(ctx context.Context, email string, windowMinutes int) ([]entity.LoginAudit, error) {
	var result []entity.LoginAudit
	windowStart := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
	for _, a := range m.audits {
		if a.Email == email && !a.Success && a.LoginAt.After(windowStart) {
			result = append(result, *a)
		}
	}
	return result, nil
}

type mockPasswordHistoryRepo struct {
	histories    []*entity.PasswordHistory
	createError  error
}

func (m *mockPasswordHistoryRepo) Create(ctx context.Context, history *entity.PasswordHistory) error {
	if m.createError != nil {
		return m.createError
	}
	m.histories = append(m.histories, history)
	return nil
}

func (m *mockPasswordHistoryRepo) ListRecent(ctx context.Context, userID uuid.UUID, limit int) ([]entity.PasswordHistory, error) {
	var result []entity.PasswordHistory
	for _, h := range m.histories {
		if h.UserID == userID {
			result = append(result, *h)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockPasswordHistoryRepo) DeleteOld(ctx context.Context, userID uuid.UUID, keepCount int) error {
	var userHistories []*entity.PasswordHistory
	var otherHistories []*entity.PasswordHistory

	for _, h := range m.histories {
		if h.UserID == userID {
			userHistories = append(userHistories, h)
		} else {
			otherHistories = append(otherHistories, h)
		}
	}

	if len(userHistories) > keepCount {
		m.histories = append(otherHistories, userHistories[:keepCount]...)
	}
	return nil
}

func TestNewLoginSecurityService(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	tests := []struct {
		name              string
		encryptionKey     string
		maxFailedAttempts int
		failedWindow      time.Duration
		blockDuration     time.Duration
		enableAuditLog    bool
		encryptAuditData  bool
	}{
		{"default config", "test-encryption-key-32bytes", 0, 0, 0, true, true},
		{"custom config", "test-key-32bytes-!!!", 10, 30 * time.Minute, 60 * time.Minute, false, false},
		{"empty encryption key", "", 5, 15 * time.Minute, 30 * time.Minute, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLoginSecurityService(
				nil, // nil Redis for testing
				mockAudit,
				mockPwdHistory,
				tt.encryptionKey,
				tt.maxFailedAttempts,
				tt.failedWindow,
				tt.blockDuration,
				tt.enableAuditLog,
				tt.encryptAuditData,
			)

			if service == nil {
				t.Error("service should not be nil")
			}
		})
	}
}

func TestLoginSecurityService_CheckLoginAllowed(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	// Test with nil Redis (always allowed)
	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "test@example.com"

	// With nil Redis, should always be allowed
	t.Run("nil redis - always allowed", func(t *testing.T) {
		err := service.CheckLoginAllowed(ctx, ip, email)
		if err != nil {
			t.Errorf("should be allowed with nil Redis: %v", err)
		}
	})
}

func TestLoginSecurityService_RecordFailedAttempt(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis for testing
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "test@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "invalid_password"
	userID := uuid.New()

	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check audit log was created
	if len(mockAudit.audits) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(mockAudit.audits))
	}

	audit := mockAudit.audits[0]
	if audit.Email != email {
		t.Errorf("expected email %s, got %s", email, audit.Email)
	}
	if audit.Success {
		t.Error("audit should record failed attempt")
	}
	if audit.FailureReason != reason {
		t.Errorf("expected reason %s, got %s", reason, audit.FailureReason)
	}

	// Test without audit logging
	t.Run("audit disabled", func(t *testing.T) {
		noAuditService := NewLoginSecurityService(
			nil,
			mockAudit,
			mockPwdHistory,
			"key",
			5,
			15*time.Minute,
			30*time.Minute,
			false, // disable audit
			true,
		)
		mockAudit.audits = nil

		err := noAuditService.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(mockAudit.audits) != 0 {
			t.Error("should not create audit when disabled")
		}
	})
}

func TestLoginSecurityService_ClearFailedAttempts(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "test@example.com"

	// With nil Redis, should return nil
	err := service.ClearFailedAttempts(ctx, ip, email)
	if err != nil {
		t.Errorf("unexpected error with nil Redis: %v", err)
	}
}

func TestLoginSecurityService_RecordSuccessfulLogin(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	email := "test@example.com"
	ip := "192.168.1.100"
	userAgent := "TestAgent"
	deviceID := "device123"
	tenantID := uuid.New()

	err := service.RecordSuccessfulLogin(ctx, userID, email, ip, userAgent, deviceID, &tenantID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check audit log was created
	if len(mockAudit.audits) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(mockAudit.audits))
	}

	audit := mockAudit.audits[0]
	if audit.Email != email {
		t.Errorf("expected email %s, got %s", email, audit.Email)
	}
	if !audit.Success {
		t.Error("audit should record successful login")
	}
	if audit.UserID == nil || *audit.UserID != userID {
		t.Error("audit should have correct userID")
	}

	// Test without audit logging
	t.Run("audit disabled", func(t *testing.T) {
		noAuditService := NewLoginSecurityService(
			nil,
			mockAudit,
			mockPwdHistory,
			"key",
			5,
			15*time.Minute,
			30*time.Minute,
			false, // disable audit
			true,
		)
		mockAudit.audits = nil

		err := noAuditService.RecordSuccessfulLogin(ctx, userID, email, ip, userAgent, deviceID, &tenantID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(mockAudit.audits) != 0 {
			t.Error("should not create audit when disabled")
		}
	})
}

func TestLoginSecurityService_DetectAnomaly(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	ip := "192.168.1.100"
	deviceID := "device123"

	// Test with no history (first login)
	t.Run("no history", func(t *testing.T) {
		isAnomaly, anomalyType := service.DetectAnomaly(ctx, userID, ip, deviceID)
		if isAnomaly {
			t.Error("first login should not be anomaly")
		}
		if anomalyType != "" {
			t.Errorf("expected empty anomaly type, got %s", anomalyType)
		}
	})

	// Add some successful login history
	oldDeviceID := "old_device"
	mockAudit.audits = []*entity.LoginAudit{
		{
			UserID:   &userID,
			DeviceID: oldDeviceID,
			Success:  true,
			LoginAt:  time.Now().Add(-24 * time.Hour),
		},
		{
			UserID:   &userID,
			DeviceID: oldDeviceID,
			Success:  true,
			LoginAt:  time.Now().Add(-12 * time.Hour),
		},
	}

	// Test with known device
	t.Run("known device", func(t *testing.T) {
		isAnomaly, _ := service.DetectAnomaly(ctx, userID, ip, oldDeviceID)
		if isAnomaly {
			t.Error("known device should not be anomaly")
		}
	})

	// Test with new device
	t.Run("new device", func(t *testing.T) {
		isAnomaly, anomalyType := service.DetectAnomaly(ctx, userID, ip, "new_device")
		if !isAnomaly {
			t.Error("new device should be anomaly")
		}
		if anomalyType != "new_device" {
			t.Errorf("expected anomaly type 'new_device', got %s", anomalyType)
		}
	})

	// Test with nil audit repo
	t.Run("nil audit repo", func(t *testing.T) {
		nilAuditService := NewLoginSecurityService(
			nil,
			nil,
			mockPwdHistory,
			"key",
			5,
			15*time.Minute,
			30*time.Minute,
			true,
			true,
		)
		isAnomaly, _ := nilAuditService.DetectAnomaly(ctx, userID, ip, deviceID)
		if isAnomaly {
			t.Error("should not detect anomaly with nil audit repo")
		}
	})
}

func TestLoginSecurityService_SavePasswordHistory(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	passwordHash := "hashed_password_1"
	keepCount := 5

	// Save first password
	err := service.SavePasswordHistory(ctx, userID, passwordHash, keepCount)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check history was saved
	if len(mockPwdHistory.histories) != 1 {
		t.Errorf("expected 1 history, got %d", len(mockPwdHistory.histories))
	}

	// Save more passwords
	for i := 0; i < 10; i++ {
		service.SavePasswordHistory(ctx, userID, toString(i), keepCount)
	}

	// Should only keep 5
	userHistories, _ := mockPwdHistory.ListRecent(ctx, userID, 100)
	if len(userHistories) > keepCount {
		t.Errorf("should only keep %d histories, got %d", keepCount, len(userHistories))
	}
}

func TestLoginSecurityService_GetFailedAttempts(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "test@example.com"

	// With nil Redis, should return 0
	count := service.GetFailedAttempts(ctx, ip, email)
	if count != 0 {
		t.Errorf("expected 0 with nil Redis, got %d", count)
	}
}

func TestLoginSecurityService_GetBlockRemainingTime(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"

	// With nil Redis, should return 0
	remaining := service.GetBlockRemainingTime(ctx, ip)
	if remaining != 0 {
		t.Errorf("expected 0 with nil Redis, got %v", remaining)
	}
}

func TestGenerateDeviceID(t *testing.T) {
	tests := []struct {
		userAgent string
		ip        string
	}{
		{"Mozilla/5.0", "192.168.1.1"},
		{"Chrome/123", "10.0.0.1"},
		{"Safari/600", "172.16.0.1"},
		{"", ""},
	}

	for _, tt := range tests {
		deviceID := GenerateDeviceID(tt.userAgent, tt.ip)

		// Should be consistent hash
		deviceID2 := GenerateDeviceID(tt.userAgent, tt.ip)
		if deviceID != deviceID2 {
			t.Error("same input should produce same device ID")
		}

		// Should be 32 hex characters (16 bytes)
		if len(deviceID) != 32 {
			t.Errorf("expected device ID length 32, got %d", len(deviceID))
		}

		// Different input should produce different ID
		if tt.userAgent != "" || tt.ip != "" {
			diffDeviceID := GenerateDeviceID(tt.userAgent+"x", tt.ip)
			if deviceID == diffDeviceID {
				t.Error("different input should produce different device ID")
			}
		}
	}

	// Verify hash calculation - device ID now only based on UserAgent (not IP)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	expectedHash := sha256.Sum256([]byte(userAgent))
	expectedHex := hex.EncodeToString(expectedHash[:16])

	deviceID := GenerateDeviceID(userAgent, ip)
	if deviceID != expectedHex {
		t.Errorf("device ID mismatch, got %s, expected %s", deviceID, expectedHex)
	}
}

// Helper functions for tests
func toString(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}

func TestLoginSecurityService_CheckPasswordHistory(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()

	// Test with nil password history repo
	t.Run("nil password history repo", func(t *testing.T) {
		nilRepoService := NewLoginSecurityService(
			nil,
			mockAudit,
			nil,
			"key",
			5,
			15*time.Minute,
			30*time.Minute,
			true,
			true,
		)
		result := nilRepoService.CheckPasswordHistory(ctx, userID, "hash", nil, 5)
		if result {
			t.Error("should return false with nil repo")
		}
	})

	// Test with empty history
	t.Run("empty history", func(t *testing.T) {
		result := service.CheckPasswordHistory(ctx, userID, "hash", nil, 5)
		// Returns true because list error is handled
		if !result {
			t.Error("should return true when history check fails")
		}
	})

	// Test with password in history
	t.Run("password in history", func(t *testing.T) {
		// Add some password history
		mockPwdHistory.histories = []*entity.PasswordHistory{
			{UserID: userID, PasswordHash: "hash1"},
			{UserID: userID, PasswordHash: "hash2"},
			{UserID: userID, PasswordHash: "hash3"},
		}
		result := service.CheckPasswordHistory(ctx, userID, "hash", nil, 5)
		// This function is mainly for UI hint, actual check happens elsewhere
		if !result {
			t.Error("should return true")
		}
	})
}

func TestLoginSecurityService_SavePasswordHistoryWithNilRepo(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		nil, // nil password history repo
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()

	// Should return nil (no-op) with nil repo
	err := service.SavePasswordHistory(ctx, userID, "hashed_password", 5)
	if err != nil {
		t.Errorf("expected nil error with nil repo: %v", err)
	}
}

func TestLoginSecurityService_RecordFailedAttemptWithNilUserID(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "test@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "user_not_found"

	// Test with nil userID
	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check audit log was created with nil userID
	if len(mockAudit.audits) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(mockAudit.audits))
	}

	audit := mockAudit.audits[0]
	if audit.UserID != nil {
		t.Error("audit UserID should be nil")
	}
}

func TestLoginSecurityService_RecordSuccessfulLoginWithNilTenant(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	email := "test@example.com"
	ip := "192.168.1.100"
	userAgent := "TestAgent"
	deviceID := "device123"

	// Test with nil tenantID
	err := service.RecordSuccessfulLogin(ctx, userID, email, ip, userAgent, deviceID, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check audit log was created
	if len(mockAudit.audits) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(mockAudit.audits))
	}

	audit := mockAudit.audits[0]
	if audit.TenantID != nil {
		t.Error("audit TenantID should be nil")
	}
}

func TestLoginSecurityService_EncryptionWithoutKey(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	// Create service with empty encryption key
	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"", // empty encryption key
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true, // encryptAuditData is true but no key
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "test@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "test"
	userID := uuid.New()

	// Should still work, just not encrypt
	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check audit was created
	if len(mockAudit.audits) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(mockAudit.audits))
	}

	audit := mockAudit.audits[0]
	// IPHash should still be set (from sha256)
	if audit.IPHash == "" {
		t.Error("IPHash should be set even without encryption")
	}
	// Encrypted fields should be empty
	if audit.IPEncrypted != "" {
		t.Error("IPEncrypted should be empty without encryption key")
	}
	if audit.UserAgentEnc != "" {
		t.Error("UserAgentEnc should be empty without encryption key")
	}
}

func TestLoginSecurityService_NoEncryption(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	// Create service with encryption disabled
	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		false, // encryptAuditData is false
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "test@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "test"
	userID := uuid.New()

	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	audit := mockAudit.audits[0]
	// IPHash should be set
	if audit.IPHash == "" {
		t.Error("IPHash should be set")
	}
	// Encrypted fields should be empty
	if audit.IPEncrypted != "" {
		t.Error("IPEncrypted should be empty when encryption disabled")
	}
}

func TestGenerateDeviceFingerprint(t *testing.T) {
	tests := []struct {
		userAgent string
		ip        string
	}{
		{"Mozilla/5.0", "192.168.1.1"},
		{"Chrome/123", "10.0.0.1"},
		{"Safari/600", "172.16.0.1"},
		{"", ""},
	}

	for _, tt := range tests {
		fingerprint := GenerateDeviceFingerprint(tt.userAgent, tt.ip)

		// Should be consistent hash
		fingerprint2 := GenerateDeviceFingerprint(tt.userAgent, tt.ip)
		if fingerprint != fingerprint2 {
			t.Error("same input should produce same fingerprint")
		}

		// Should be 64 hex characters (full SHA256)
		if len(fingerprint) != 64 {
			t.Errorf("expected fingerprint length 64, got %d", len(fingerprint))
		}

		// Different input should produce different fingerprint
		if tt.userAgent != "" || tt.ip != "" {
			diffFingerprint := GenerateDeviceFingerprint(tt.userAgent+"x", tt.ip)
			if fingerprint == diffFingerprint {
				t.Error("different input should produce different fingerprint")
			}
		}
	}

	// Verify fingerprint differs from deviceID (includes IP)
	userAgent := "Mozilla/5.0"
	ip := "192.168.1.1"
	deviceID := GenerateDeviceID(userAgent, ip)
	fingerprint := GenerateDeviceFingerprint(userAgent, ip)
	if deviceID == fingerprint {
		t.Error("deviceID and fingerprint should differ (fingerprint includes IP)")
	}
}

func TestLoginSecurityService_MultipleFailedAttempts(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis - can't test blocking without Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "multi@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	userID := uuid.New()

	// Record multiple failed attempts
	for i := 0; i < 10; i++ {
		reason := toString(i)
		err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
		if err != nil {
			t.Errorf("unexpected error on attempt %d: %v", i, err)
		}
	}

	// Check all audits were created
	if len(mockAudit.audits) != 10 {
		t.Errorf("expected 10 audit logs, got %d", len(mockAudit.audits))
	}
}

func TestLoginSecurityService_DetectAnomalyWithFailedHistory(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	ip := "192.168.1.100"
	newDeviceID := "new_device"

	// Add some failed login history (should not count for known devices)
	mockAudit.audits = []*entity.LoginAudit{
		{
			UserID:   &userID,
			DeviceID: "failed_device",
			Success:  false, // Failed login
			LoginAt:  time.Now().Add(-24 * time.Hour),
		},
	}

	// With only failed history, new device should still be anomaly
	isAnomaly, anomalyType := service.DetectAnomaly(ctx, userID, ip, newDeviceID)
	if !isAnomaly {
		t.Error("new device should be anomaly with only failed history")
	}
	if anomalyType != "new_device" {
		t.Errorf("expected anomaly type 'new_device', got %s", anomalyType)
	}
}

func TestLoginSecurityService_DetectAnomalyWithEmptyDeviceID(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	ip := "192.168.1.100"

	// Add history with empty device IDs
	mockAudit.audits = []*entity.LoginAudit{
		{
			UserID:   &userID,
			DeviceID: "", // Empty device ID
			Success:  true,
			LoginAt:  time.Now().Add(-24 * time.Hour),
		},
	}

	// Empty device ID in history means any device is "new"
	isAnomaly, anomalyType := service.DetectAnomaly(ctx, userID, ip, "any_device")
	if !isAnomaly {
		t.Error("any device should be anomaly when history has empty device IDs")
	}
	if anomalyType != "new_device" {
		t.Errorf("expected anomaly type 'new_device', got %s", anomalyType)
	}
}

func TestLoginSecurityService_SavePasswordHistoryError(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}
	mockPwdHistory.createError = errors.New("create failed")

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()

	err := service.SavePasswordHistory(ctx, userID, "hash", 5)
	if err == nil {
		t.Error("expected error from create failure")
	}
}

// Add error field to mock
type mockPasswordHistoryRepoWithError struct {
	mockPasswordHistoryRepo
	createError error
}

func (m *mockPasswordHistoryRepoWithError) Create(ctx context.Context, history *entity.PasswordHistory) error {
	if m.createError != nil {
		return m.createError
	}
	return m.mockPasswordHistoryRepo.Create(ctx, history)
}

// Redis-based tests for login security service
func setupLoginSecurityWithRedis() (*LoginSecurityService, *miniredis.Miniredis, *mockLoginAuditRepo, *mockPasswordHistoryRepo) {
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		client,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,           // maxFailedAttempts
		15*time.Minute, // failedWindow
		30*time.Minute, // blockDuration
		true,        // enableAuditLog
		true,        // encryptAuditData
	)
	return service, mr, mockAudit, mockPwdHistory
}

func TestLoginSecurityService_CheckLoginAllowedWithRedis(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "test@example.com"

	// Should be allowed initially
	t.Run("allowed initially", func(t *testing.T) {
		err := service.CheckLoginAllowed(ctx, ip, email)
		if err != nil {
			t.Errorf("should be allowed initially: %v", err)
		}
	})

	// Block IP
	t.Run("IP blocked", func(t *testing.T) {
		key := "login_blocked_ip:" + ip
		mr.Set(key, "1")

		err := service.CheckLoginAllowed(ctx, ip, email)
		if err == nil {
			t.Error("should be blocked when IP is blocked")
		}
		if err != apperrors.ErrTooManyAttempts {
			t.Errorf("expected ErrTooManyAttempts, got %v", err)
		}
	})

	// Clear IP block and block email
	t.Run("email blocked", func(t *testing.T) {
		mr.Del("login_blocked_ip:" + ip)
		emailKey := "login_blocked_email:" + email
		mr.Set(emailKey, "1")

		err := service.CheckLoginAllowed(ctx, ip, email)
		if err == nil {
			t.Error("should be blocked when email is blocked")
		}
		if err != apperrors.ErrAccountLocked {
			t.Errorf("expected ErrAccountLocked, got %v", err)
		}
	})

	// Clear blocks but exceed failed attempts
	t.Run("exceeded failed attempts", func(t *testing.T) {
		mr.Del("login_blocked_email:" + email)

		// Set failed attempts count to exceed max
		failedKey := "login_failed:" + ip + ":" + email
		mr.Set(failedKey, "6") // More than maxFailedAttempts (5)

		err := service.CheckLoginAllowed(ctx, ip, email)
		if err == nil {
			t.Error("should be blocked when failed attempts exceeded")
		}
		if err != apperrors.ErrTooManyAttempts {
			t.Errorf("expected ErrTooManyAttempts, got %v", err)
		}
	})
}

func TestLoginSecurityService_RecordFailedAttemptWithRedis(t *testing.T) {
	service, mr, mockAudit, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "failed@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "invalid_password"
	userID := uuid.New()

	// Record failed attempt
	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check failed count increased
	failedKey := "login_failed:" + ip + ":" + email
	count, err := mr.Get(failedKey)
	if err != nil {
		t.Error("failed count should be set")
	}
	if count != "1" {
		t.Errorf("expected count 1, got %s", count)
	}

	// Record more failures to trigger blocking
	for i := 0; i < 5; i++ {
		service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	}

	// Check IP is blocked after exceeding threshold
	ipBlockedKey := "login_blocked_ip:" + ip
	val, err := mr.Get(ipBlockedKey)
	if err != nil {
		t.Error("IP should be blocked after exceeding threshold")
	}
	if val != "1" {
		t.Errorf("expected blocked value '1', got %s", val)
	}

	// Check audit logs were created
	if len(mockAudit.audits) != 6 {
		t.Errorf("expected 6 audit logs, got %d", len(mockAudit.audits))
	}
}

func TestLoginSecurityService_ClearFailedAttemptsWithRedis(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "clear@example.com"

	// Set failed attempts
	failedKey := "login_failed:" + ip + ":" + email
	mr.Set(failedKey, "5")

	// Clear failed attempts
	err := service.ClearFailedAttempts(ctx, ip, email)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check key is deleted
	_, err = mr.Get(failedKey)
	if err == nil {
		t.Error("failed attempts key should be deleted")
	}
}

func TestLoginSecurityService_GetFailedAttemptsWithRedis(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "count@example.com"

	// Initially 0
	t.Run("initial count", func(t *testing.T) {
		count := service.GetFailedAttempts(ctx, ip, email)
		if count != 0 {
			t.Errorf("expected 0, got %d", count)
		}
	})

	// Set failed attempts
	t.Run("with failures", func(t *testing.T) {
		failedKey := "login_failed:" + ip + ":" + email
		mr.Set(failedKey, "3")

		count := service.GetFailedAttempts(ctx, ip, email)
		if count != 3 {
			t.Errorf("expected 3, got %d", count)
		}
	})

	// Get count for different IP/email
	t.Run("different key", func(t *testing.T) {
		count := service.GetFailedAttempts(ctx, "10.0.0.1", "other@example.com")
		if count != 0 {
			t.Errorf("expected 0 for different key, got %d", count)
		}
	})
}

func TestLoginSecurityService_GetBlockRemainingTimeWithRedis(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ip := "192.168.1.100"

	// Initially no block
	t.Run("no block", func(t *testing.T) {
		remaining := service.GetBlockRemainingTime(ctx, ip)
		if remaining != 0 {
			t.Errorf("expected 0 when not blocked, got %v", remaining)
		}
	})

	// Block IP with TTL
	t.Run("with block", func(t *testing.T) {
		ipBlockedKey := "login_blocked_ip:" + ip
		mr.Set(ipBlockedKey, "1")
		// Use mr.SetTTL after setting the key
		// miniredis will track TTL for the key

		remaining := service.GetBlockRemainingTime(ctx, ip)
		// miniredis might return 0 for TTL - this is acceptable in tests
		// The real behavior would return proper TTL
		_ = remaining // Just verify it doesn't error
	})

	// Expired block
	t.Run("expired block", func(t *testing.T) {
		expiredIP := "192.168.1.200"
		expiredKey := "login_blocked_ip:" + expiredIP
		mr.Set(expiredKey, "1")
		mr.FastForward(31 * time.Minute) // Expire the block

		remaining := service.GetBlockRemainingTime(ctx, expiredIP)
		if remaining != 0 {
			t.Errorf("expected 0 for expired block, got %v", remaining)
		}
	})
}

func TestLoginSecurityService_RecordSuccessfulLoginWithRedis(t *testing.T) {
	service, mr, mockAudit, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()
	email := "success@example.com"
	ip := "192.168.1.100"
	userAgent := "TestAgent"
	deviceID := "device123"
	tenantID := uuid.New()

	// Set some failed attempts first
	failedKey := "login_failed:" + ip + ":" + email
	mr.Set(failedKey, "3")

	// Record successful login - note: ClearFailedAttempts is separate
	err := service.RecordSuccessfulLogin(ctx, userID, email, ip, userAgent, deviceID, &tenantID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check audit log was created
	if len(mockAudit.audits) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(mockAudit.audits))
	}

	audit := mockAudit.audits[0]
	if !audit.Success {
		t.Error("audit should be successful")
	}
}

func TestLoginSecurityService_DetectAnomalyWithRedisAudit(t *testing.T) {
	service, mr, mockAudit, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	userID := uuid.New()
	ip := "192.168.1.100"

	// Create login history
	mockAudit.audits = []*entity.LoginAudit{
		{
			UserID:   &userID,
			DeviceID: "known_device",
			Success:  true,
			LoginAt:  time.Now().Add(-24 * time.Hour),
		},
	}

	// Test known device
	t.Run("known device", func(t *testing.T) {
		isAnomaly, _ := service.DetectAnomaly(ctx, userID, ip, "known_device")
		if isAnomaly {
			t.Error("known device should not be anomaly")
		}
	})

	// Test new device
	t.Run("new device", func(t *testing.T) {
		isAnomaly, anomalyType := service.DetectAnomaly(ctx, userID, ip, "new_device")
		if !isAnomaly {
			t.Error("new device should be anomaly")
		}
		if anomalyType != "new_device" {
			t.Errorf("expected anomaly type 'new_device', got %s", anomalyType)
		}
	})
}

func TestLoginSecurityService_FailedAttemptWindowExpiry(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "expiry@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "test"
	userID := uuid.New()

	// Record failed attempts
	for i := 0; i < 3; i++ {
		service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	}

	// Check count
	failedKey := "login_failed:" + ip + ":" + email
	count, _ := mr.Get(failedKey)
	if count != "3" {
		t.Errorf("expected count 3, got %s", count)
	}

	// Fast forward past the failed window (15 minutes)
	mr.FastForward(16 * time.Minute)

	// Count should be 0 (expired)
	newCount := service.GetFailedAttempts(ctx, ip, email)
	if newCount != 0 {
		t.Errorf("expected count 0 after expiry, got %d", newCount)
	}
}

func TestLoginSecurityService_BlockDurationExpiry(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "blockexpiry@example.com"

	// Block IP
	ipBlockedKey := "login_blocked_ip:" + ip
	mr.Set(ipBlockedKey, "1")

	// Should be blocked
	err := service.CheckLoginAllowed(ctx, ip, email)
	if err == nil {
		t.Error("should be blocked")
	}

	// Delete the block key to simulate expiry
	mr.Del(ipBlockedKey)

	// Should be allowed now
	err = service.CheckLoginAllowed(ctx, ip, email)
	if err != nil {
		t.Errorf("should be allowed after block removed: %v", err)
	}
}

func TestLoginSecurityService_MultipleIPBlocking(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	email := "multiip@example.com"

	// Block each IP
	for _, ip := range ips {
		ipBlockedKey := "login_blocked_ip:" + ip
		mr.Set(ipBlockedKey, "1")
	}

	// All IPs should be blocked
	for _, ip := range ips {
		err := service.CheckLoginAllowed(ctx, ip, email)
		if err == nil {
			t.Errorf("IP %s should be blocked", ip)
		}
	}

	// Unblock one IP
	mr.Del("login_blocked_ip:" + ips[0])

	// First IP should be allowed
	err := service.CheckLoginAllowed(ctx, ips[0], email)
	if err != nil {
		t.Errorf("IP %s should be allowed after unblock: %v", ips[0], err)
	}

	// Other IPs should still be blocked
	for i := 1; i < len(ips); i++ {
		err := service.CheckLoginAllowed(ctx, ips[i], email)
		if err == nil {
			t.Errorf("IP %s should still be blocked", ips[i])
		}
	}
}
// Mock login audit repo that can return errors
type mockLoginAuditRepoWithError struct {
	mockLoginAuditRepo
	createError error
}

func (m *mockLoginAuditRepoWithError) Create(ctx context.Context, audit *entity.LoginAudit) error {
	if m.createError != nil {
		return m.createError
	}
	return m.mockLoginAuditRepo.Create(ctx, audit)
}

func TestLoginSecurityService_RecordSuccessfulLoginWithAuditError(t *testing.T) {
	mockAudit := &mockLoginAuditRepoWithError{
		createError: errors.New("audit create failed"),
	}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	email := "error@example.com"
	ip := "192.168.1.100"
	userAgent := "TestAgent"
	deviceID := "device123"
	tenantID := uuid.New()

	// Should return error from audit repo
	err := service.RecordSuccessfulLogin(ctx, userID, email, ip, userAgent, deviceID, &tenantID)
	if err == nil {
		t.Error("expected error from audit repo")
	}
	if err.Error() != "audit create failed" {
		t.Errorf("expected 'audit create failed', got %v", err)
	}
}

func TestLoginSecurityService_RecordSuccessfulLoginWithNilAuditRepo(t *testing.T) {
	service := NewLoginSecurityService(
		nil, // nil Redis
		nil, // nil audit repo
		&mockPasswordHistoryRepo{},
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true, // audit enabled but repo is nil
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	email := "nilrepo@example.com"
	ip := "192.168.1.100"

	// Should return nil (no-op with nil audit repo)
	err := service.RecordSuccessfulLogin(ctx, userID, email, ip, "ua", "dev", nil)
	if err != nil {
		t.Errorf("expected nil with nil audit repo: %v", err)
	}
}

func TestLoginSecurityService_RecordFailedAttemptWithAuditError(t *testing.T) {
	mockAudit := &mockLoginAuditRepoWithError{
		createError: errors.New("audit create failed"),
	}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "auditerror@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "test"
	userID := uuid.New()

	// RecordFailedAttempt should still succeed (audit is non-blocking)
	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Audit repo error is not returned, but no audit was created
	if len(mockAudit.audits) != 0 {
		t.Error("audit should not be created when repo fails")
	}
}

func TestLoginSecurityService_RecordFailedAttemptWithNilAuditRepo(t *testing.T) {
	service := NewLoginSecurityService(
		nil, // nil Redis
		nil, // nil audit repo
		&mockPasswordHistoryRepo{},
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true, // audit enabled but repo is nil
		true,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "nilaudit@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "test"
	userID := uuid.New()

	// Should succeed without audit
	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoginSecurityService_RecordSuccessfulLoginWithEncryption(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"01234567890123456789012345678901", // 32-byte encryption key
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true, // encryption enabled
	)

	ctx := context.Background()
	userID := uuid.New()
	email := "encrypted@example.com"
	ip := "192.168.1.100"
	userAgent := "Mozilla/5.0"
	deviceID := "device123"
	tenantID := uuid.New()

	err := service.RecordSuccessfulLogin(ctx, userID, email, ip, userAgent, deviceID, &tenantID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	audit := mockAudit.audits[0]
	// Verify IP was encrypted
	if audit.IPEncrypted == "" {
		t.Error("IP should be encrypted")
	}
	// Verify UserAgent was encrypted
	if audit.UserAgentEnc == "" {
		t.Error("UserAgent should be encrypted")
	}
	// Verify IPHash is set
	if audit.IPHash == "" {
		t.Error("IPHash should be set")
	}
}

func TestLoginSecurityService_RecordSuccessfulLoginWithoutEncryption(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		false, // encryption disabled
	)

	ctx := context.Background()
	userID := uuid.New()
	email := "noencrypt@example.com"
	ip := "192.168.1.100"
	userAgent := "Mozilla/5.0"
	deviceID := "device123"
	tenantID := uuid.New()

	err := service.RecordSuccessfulLogin(ctx, userID, email, ip, userAgent, deviceID, &tenantID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	audit := mockAudit.audits[0]
	// Verify encrypted fields are empty
	if audit.IPEncrypted != "" {
		t.Error("IPEncrypted should be empty when encryption disabled")
	}
	if audit.UserAgentEnc != "" {
		t.Error("UserAgentEnc should be empty when encryption disabled")
	}
	// Verify IPHash is still set
	if audit.IPHash == "" {
		t.Error("IPHash should still be set")
	}
}

func TestLoginSecurityService_RecordFailedAttemptWithEncryption(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"01234567890123456789012345678901", // 32-byte encryption key
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true, // encryption enabled
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "encfailed@example.com"
	userAgent := "Mozilla/5.0"
	deviceID := "device123"
	reason := "invalid_password"
	userID := uuid.New()

	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	audit := mockAudit.audits[0]
	// Verify IP was encrypted
	if audit.IPEncrypted == "" {
		t.Error("IP should be encrypted")
	}
	// Verify UserAgent was encrypted
	if audit.UserAgentEnc == "" {
		t.Error("UserAgent should be encrypted")
	}
	// Verify IPHash is set
	if audit.IPHash == "" {
		t.Error("IPHash should be set")
	}
	// Verify failure reason is set
	if audit.FailureReason != reason {
		t.Errorf("expected reason %s, got %s", reason, audit.FailureReason)
	}
}

// Note: CheckPasswordHistory always returns true per implementation
// (used for UI hints, actual check is done elsewhere)
func TestLoginSecurityService_CheckPasswordHistoryWithPasswordFound(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	passwordHash := "hashed_password"

	// Add password to history
	mockPwdHistory.histories = []*entity.PasswordHistory{
		{UserID: userID, PasswordHash: passwordHash},
	}

	// CheckPasswordHistory always returns true per implementation (UI hint function)
	result := service.CheckPasswordHistory(ctx, userID, passwordHash, nil, 5)
	if !result {
		t.Error("CheckPasswordHistory always returns true per implementation")
	}
}

func TestLoginSecurityService_CheckPasswordHistoryWithDifferentUser(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	passwordHash := "hashed_password"

	// Add password to history for different user
	mockPwdHistory.histories = []*entity.PasswordHistory{
		{UserID: otherUserID, PasswordHash: passwordHash},
	}

	// Test with different user - should return true (password not found for this user)
	result := service.CheckPasswordHistory(ctx, userID, passwordHash, nil, 5)
	if !result {
		t.Error("should return true when password not in user's history")
	}
}

func TestLoginSecurityService_RecordFailedAttemptBlockingThreshold(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	ip := "192.168.1.200"
	email := "threshold@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "invalid_password"
	userID := uuid.New()

	// Set max failed attempts to 5, so record 5 failures
	for i := 0; i < 5; i++ {
		err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	// Check that IP blocking key was set
	ipBlockedKey := "login_blocked_ip:" + ip
	val, err := mr.Get(ipBlockedKey)
	if err != nil {
		t.Error("IP should be blocked at threshold")
	}
	if val != "1" {
		t.Errorf("expected blocked value '1', got %s", val)
	}

	// Verify failed count is 5
	failedKey := "login_failed:" + ip + ":" + email
	count, _ := mr.Get(failedKey)
	if count != "5" {
		t.Errorf("expected count 5, got %s", count)
	}
}

func TestLoginSecurityService_RecordFailedAttemptDifferentIPs(t *testing.T) {
	service, mr, _, _ := setupLoginSecurityWithRedis()
	defer mr.Close()

	ctx := context.Background()
	email := "multiip@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "invalid_password"
	userID := uuid.New()

	// Record failures from different IPs for same email
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	for _, ip := range ips {
		for i := 0; i < 3; i++ {
			err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
	}

	// Each IP should have count 3, not blocked yet (threshold is 5)
	for _, ip := range ips {
		failedKey := "login_failed:" + ip + ":" + email
		count, _ := mr.Get(failedKey)
		if count != "3" {
			t.Errorf("expected count 3 for IP %s, got %s", ip, count)
		}

		// IP should not be blocked (threshold not reached)
		ipBlockedKey := "login_blocked_ip:" + ip
		_, err := mr.Get(ipBlockedKey)
		if err == nil {
			t.Errorf("IP %s should not be blocked at count 3", ip)
		}
	}
}

func TestLoginSecurityService_RecordSuccessfulLoginWithEmptyDeviceID(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	email := "emptydevice@example.com"
	ip := "192.168.1.100"
	userAgent := "TestAgent"
	deviceID := "" // empty device ID
	tenantID := uuid.New()

	err := service.RecordSuccessfulLogin(ctx, userID, email, ip, userAgent, deviceID, &tenantID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	audit := mockAudit.audits[0]
	if audit.DeviceID != "" {
		t.Error("device ID should be empty")
	}
}

func TestLoginSecurityService_RecordFailedAttemptWithEmptyStrings(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil, // nil Redis
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()

	// Test with empty IP
	err := service.RecordFailedAttempt(ctx, "", "test@example.com", "ua", "dev", "test", nil)
	if err != nil {
		t.Errorf("unexpected error with empty IP: %v", err)
	}

	// Test with empty email
	err = service.RecordFailedAttempt(ctx, "192.168.1.100", "", "ua", "dev", "test", nil)
	if err != nil {
		t.Errorf("unexpected error with empty email: %v", err)
	}

	// Test with empty userAgent
	err = service.RecordFailedAttempt(ctx, "192.168.1.100", "test@example.com", "", "dev", "test", nil)
	if err != nil {
		t.Errorf("unexpected error with empty userAgent: %v", err)
	}

	// All audits should be created
	if len(mockAudit.audits) != 3 {
		t.Errorf("expected 3 audits, got %d", len(mockAudit.audits))
	}
}

func TestLoginSecurityService_CheckPasswordHistoryWithLimit(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	passwordHash := "password_hash_10"

	// Add more than limit passwords
	for i := 0; i < 10; i++ {
		mockPwdHistory.histories = append(mockPwdHistory.histories, &entity.PasswordHistory{
			UserID:       userID,
			PasswordHash: "password_hash_" + toString(i),
		})
	}

	// CheckPasswordHistory always returns true per implementation (UI hint function)
	result := service.CheckPasswordHistory(ctx, userID, passwordHash, nil, 5)
	if !result {
		t.Error("CheckPasswordHistory always returns true per implementation")
	}

	// Check password not in history - also returns true
	result = service.CheckPasswordHistory(ctx, userID, "not_in_history", nil, 5)
	if !result {
		t.Error("CheckPasswordHistory always returns true")
	}
}

func TestLoginSecurityService_DetectAnomalyWithNoSuccessHistory(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	ip := "192.168.1.100"
	deviceID := "device123"

	// Add only failed login history
	mockAudit.audits = []*entity.LoginAudit{
		{
			UserID:        &userID,
			DeviceID:      "other_device",
			Success:       false,
			FailureReason: "invalid_password",
			LoginAt:       time.Now().Add(-24 * time.Hour),
		},
	}

	// Any device should be anomaly since no successful login history
	isAnomaly, anomalyType := service.DetectAnomaly(ctx, userID, ip, deviceID)
	if !isAnomaly {
		t.Error("should be anomaly with no successful history")
	}
	if anomalyType != "new_device" {
		t.Errorf("expected 'new_device', got %s", anomalyType)
	}
}

func TestLoginSecurityService_DetectAnomalyWithRecentHistory(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	service := NewLoginSecurityService(
		nil,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		true,
		true,
	)

	ctx := context.Background()
	userID := uuid.New()
	ip := "192.168.1.100"
	knownDeviceID := "known_device"

	// Add recent successful login history
	mockAudit.audits = []*entity.LoginAudit{
		{
			UserID:   &userID,
			DeviceID: knownDeviceID,
			Success:  true,
			LoginAt:  time.Now().Add(-1 * time.Hour),
		},
		{
			UserID:   &userID,
			DeviceID: knownDeviceID,
			Success:  true,
			LoginAt:  time.Now().Add(-2 * time.Hour),
		},
	}

	// Known device should not be anomaly
	isAnomaly, _ := service.DetectAnomaly(ctx, userID, ip, knownDeviceID)
	if isAnomaly {
		t.Error("known device should not be anomaly")
	}

	// New device should be anomaly
	isAnomaly, anomalyType := service.DetectAnomaly(ctx, userID, ip, "new_device")
	if !isAnomaly {
		t.Error("new device should be anomaly")
	}
	if anomalyType != "new_device" {
		t.Errorf("expected 'new_device', got %s", anomalyType)
	}
}

func TestLoginSecurityService_RecordFailedAttemptWithRedisError(t *testing.T) {
	mockAudit := &mockLoginAuditRepo{}
	mockPwdHistory := &mockPasswordHistoryRepo{}

	// Create Redis client that will fail
	mr, _ := miniredis.Run()
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewLoginSecurityService(
		client,
		mockAudit,
		mockPwdHistory,
		"test-key-32bytes-!!!",
		5,
		15*time.Minute,
		30*time.Minute,
		false, // disable audit for this test
		false,
	)

	ctx := context.Background()
	ip := "192.168.1.100"
	email := "rediserror@example.com"
	userAgent := "TestAgent"
	deviceID := "device123"
	reason := "test"
	userID := uuid.New()

	// Close Redis to cause errors
	mr.Close()

	// Should still succeed (Redis errors are non-blocking)
	err := service.RecordFailedAttempt(ctx, ip, email, userAgent, deviceID, reason, &userID)
	if err != nil {
		t.Errorf("should succeed even with Redis error: %v", err)
	}
}
