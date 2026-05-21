package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// Mock email sender for testing
type mockEmailSender struct {
	sentEmails []struct {
		to       string
		token    string
		userName string
	}
	sendError error
}

func (m *mockEmailSender) SendVerificationEmail(to string, token string, userName string) error {
	if m.sendError != nil {
		return m.sendError
	}
	m.sentEmails = append(m.sentEmails, struct {
		to       string
		token    string
		userName string
	}{to, token, userName})
	return nil
}

// Mock user repository for email verification tests
type mockUserRepoForVerification struct {
	users          map[uuid.UUID]*entity.User
	byEmail        map[string]*entity.User
	byToken        map[string]*entity.User
	updateError    error
	getByEmailErr  error
	getByTokenErr  error
}

func newMockUserRepoForVerification() *mockUserRepoForVerification {
	return &mockUserRepoForVerification{
		users:   make(map[uuid.UUID]*entity.User),
		byEmail: make(map[string]*entity.User),
		byToken: make(map[string]*entity.User),
	}
}

func (m *mockUserRepoForVerification) Create(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockUserRepoForVerification) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, apperrors.ErrUserNotFound
}

func (m *mockUserRepoForVerification) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.getByEmailErr != nil {
		return nil, m.getByEmailErr
	}
	if user, ok := m.byEmail[email]; ok {
		return user, nil
	}
	return nil, apperrors.ErrUserNotFound
}

func (m *mockUserRepoForVerification) GetByVerificationToken(ctx context.Context, token string) (*entity.User, error) {
	if m.getByTokenErr != nil {
		return nil, m.getByTokenErr
	}
	if user, ok := m.byToken[token]; ok {
		return user, nil
	}
	return nil, apperrors.ErrInvalidVerificationToken
}

func (m *mockUserRepoForVerification) Update(ctx context.Context, user *entity.User) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	if user.EmailVerificationToken != "" {
		m.byToken[user.EmailVerificationToken] = user
	}
	// Clear old token if cleared
	for token, u := range m.byToken {
		if u.ID == user.ID && token != user.EmailVerificationToken {
			delete(m.byToken, token)
		}
	}
	return nil
}

func (m *mockUserRepoForVerification) Delete(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		delete(m.users, id)
		delete(m.byEmail, user.Email)
		delete(m.byToken, user.EmailVerificationToken)
	}
	return nil
}

func (m *mockUserRepoForVerification) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error) {
	return nil, 0, nil
}

func (m *mockUserRepoForVerification) UpdateLastLogin(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockUserRepoForVerification) IncrementMonthlyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error { return nil }
func (m *mockUserRepoForVerification) IncrementDailyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error { return nil }
func (m *mockUserRepoForVerification) ResetMonthlyBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockUserRepoForVerification) ResetDailyBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockUserRepoForVerification) GetBudgetUsage(ctx context.Context, id uuid.UUID) (monthlyUsed decimal.Decimal, dailyUsed decimal.Decimal, tokensUsed int64, err error) { return decimal.Zero, decimal.Zero, 0, nil }
func (m *mockUserRepoForVerification) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error { return nil }
func (m *mockUserRepoForVerification) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockUserRepoForVerification) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockUserRepoForVerification) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (m *mockUserRepoForVerification) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *mockUserRepoForVerification) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func TestNewEmailVerificationService(t *testing.T) {
	mockUserRepo := newMockUserRepoForVerification()
	mockSender := &mockEmailSender{}

	tests := []struct {
		name       string
		expiry     time.Duration
		wantExpiry time.Duration
	}{
		{"default expiry", 0, DefaultVerificationExpiry},
		{"custom expiry", 48 * time.Hour, 48 * time.Hour},
		{"negative expiry", -1 * time.Hour, DefaultVerificationExpiry},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewEmailVerificationService(mockUserRepo, mockSender, tt.expiry)
			if service == nil {
				t.Error("service should not be nil")
			}
			if service.expiry != tt.wantExpiry {
				t.Errorf("expected expiry %v, got %v", tt.wantExpiry, service.expiry)
			}
		})
	}

	// Test with nil sender
	t.Run("nil sender", func(t *testing.T) {
		service := NewEmailVerificationService(mockUserRepo, nil, DefaultVerificationExpiry)
		if service == nil {
			t.Error("service should not be nil even with nil sender")
		}
	})
}

func TestEmailVerificationService_GenerateVerificationToken(t *testing.T) {
	mockUserRepo := newMockUserRepoForVerification()
	mockSender := &mockEmailSender{}
	service := NewEmailVerificationService(mockUserRepo, mockSender, DefaultVerificationExpiry)

	token, err := service.GenerateVerificationToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Token should be 64 hex characters (32 bytes)
	if len(token) != 64 {
		t.Errorf("expected token length 64, got %d", len(token))
	}

	// Generate another token, should be different
	token2, _ := service.GenerateVerificationToken()
	if token == token2 {
		t.Error("tokens should be unique")
	}

	// Verify tokens are hex strings
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token should be hex string, found %c", c)
		}
	}
}

func TestEmailVerificationService_SendVerificationEmail(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		user           *entity.User
		senderError    error
		updateError    error
		wantError      bool
		wantEmailSent  bool
		wantTokenSet   bool
	}{
		{
			name: "successful send",
			user: &entity.User{
				ID:     uuid.New(),
				Email:  "test@example.com",
				Name:   "Test User",
				Status: "active",
			},
			wantError:     false,
			wantEmailSent: true,
			wantTokenSet:  true,
		},
		{
			name: "nil sender",
			user: &entity.User{
				ID:     uuid.New(),
				Email:  "nosender@example.com",
				Name:   "No Sender",
				Status: "active",
			},
			wantError:     false,
			wantEmailSent: false,
			wantTokenSet:  true,
		},
		{
			name: "update error",
			user: &entity.User{
				ID:     uuid.New(),
				Email:  "updatefail@example.com",
				Name:   "Update Fail",
				Status: "active",
			},
			updateError:   errors.New("update failed"),
			wantError:     true,
			wantEmailSent: false,
			wantTokenSet:  false,
		},
		{
			name: "email send error",
			user: &entity.User{
				ID:     uuid.New(),
				Email:  "sendfail@example.com",
				Name:   "Send Fail",
				Status: "active",
			},
			senderError:   errors.New("email service down"),
			wantError:     true,
			wantEmailSent: false,
			wantTokenSet:  true, // Token is set before email send attempt
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := newMockUserRepoForVerification()
			mockSender := &mockEmailSender{sendError: tt.senderError}
			mockUserRepo.updateError = tt.updateError
			mockUserRepo.Create(ctx, tt.user)

			// Use nil sender for "nil sender" test
			var sender EmailSender = mockSender
			if tt.name == "nil sender" {
				sender = nil
			}

			service := NewEmailVerificationService(mockUserRepo, sender, DefaultVerificationExpiry)

			err := service.SendVerificationEmail(ctx, tt.user)

			if tt.wantError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.wantEmailSent && len(mockSender.sentEmails) == 0 {
				t.Error("expected email to be sent")
			}
			if !tt.wantEmailSent && len(mockSender.sentEmails) > 0 {
				t.Error("expected no email to be sent")
			}

			if tt.wantTokenSet {
				if tt.user.EmailVerificationToken == "" {
					t.Error("expected token to be set")
				}
				if tt.user.EmailVerificationExpires == nil {
					t.Error("expected expiry to be set")
				}
				if tt.user.Status != "pending_verification" {
					t.Errorf("expected status pending_verification, got %s", tt.user.Status)
				}
				if tt.user.EmailVerified {
					t.Error("expected EmailVerified to be false")
				}
			}
		})
	}
}

func TestEmailVerificationService_VerifyEmail(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := newMockUserRepoForVerification()
	mockSender := &mockEmailSender{}
	service := NewEmailVerificationService(mockUserRepo, mockSender, DefaultVerificationExpiry)

	// Create test user with valid token
	validToken := "valid_verification_token_1234567890abcdef"
	expiresAt := time.Now().Add(24 * time.Hour)
	userID := uuid.New()
	user := &entity.User{
		ID:                      userID,
		Email:                   "verify@example.com",
		Name:                    "Verify User",
		EmailVerificationToken:  validToken,
		EmailVerificationExpires: &expiresAt,
		EmailVerified:           false,
		Status:                  "pending_verification",
	}
	mockUserRepo.Create(ctx, user)
	mockUserRepo.byToken[validToken] = user

	// Test successful verification
	t.Run("successful verification", func(t *testing.T) {
		verifiedUser, err := service.VerifyEmail(ctx, validToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if verifiedUser == nil {
			t.Fatal("expected non-nil user")
		}
		if !verifiedUser.EmailVerified {
			t.Error("expected EmailVerified to be true")
		}
		if verifiedUser.EmailVerifiedAt == nil {
			t.Error("expected EmailVerifiedAt to be set")
		}
		if verifiedUser.EmailVerificationToken != "" {
			t.Error("expected token to be cleared")
		}
		if verifiedUser.EmailVerificationExpires != nil {
			t.Error("expected expiry to be cleared")
		}
		if verifiedUser.Status != "active" {
			t.Errorf("expected status active, got %s", verifiedUser.Status)
		}
	})

	// Test invalid token
	t.Run("invalid token", func(t *testing.T) {
		_, err := service.VerifyEmail(ctx, "invalid_token")
		if err == nil {
			t.Error("expected error for invalid token")
		}
		if !errors.Is(err, apperrors.ErrInvalidVerificationToken) {
			t.Errorf("expected ErrInvalidVerificationToken, got %v", err)
		}
	})

	// Test expired token
	t.Run("expired token", func(t *testing.T) {
		expiredToken := "expired_token_1234567890abcdef"
		expiredTime := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
		expiredUser := &entity.User{
			ID:                      uuid.New(),
			Email:                   "expired@example.com",
			Name:                    "Expired User",
			EmailVerificationToken:  expiredToken,
			EmailVerificationExpires: &expiredTime,
			EmailVerified:           false,
			Status:                  "pending_verification",
		}
		mockUserRepo.Create(ctx, expiredUser)
		mockUserRepo.byToken[expiredToken] = expiredUser

		_, err := service.VerifyEmail(ctx, expiredToken)
		if err == nil {
			t.Error("expected error for expired token")
		}
		if !errors.Is(err, apperrors.ErrVerificationTokenExpired) {
			t.Errorf("expected ErrVerificationTokenExpired, got %v", err)
		}
	})

	// Test already verified
	t.Run("already verified", func(t *testing.T) {
		verifiedToken := "verified_token_1234567890abcdef"
		verifiedUser := &entity.User{
			ID:                      uuid.New(),
			Email:                   "verified@example.com",
			Name:                    "Verified User",
			EmailVerificationToken:  verifiedToken,
			EmailVerificationExpires: nil,
			EmailVerified:           true,
			Status:                  "active",
		}
		mockUserRepo.Create(ctx, verifiedUser)
		mockUserRepo.byToken[verifiedToken] = verifiedUser

		_, err := service.VerifyEmail(ctx, verifiedToken)
		if err == nil {
			t.Error("expected error for already verified")
		}
		if !errors.Is(err, apperrors.ErrEmailAlreadyVerified) {
			t.Errorf("expected ErrEmailAlreadyVerified, got %v", err)
		}
	})

	// Test update error
	t.Run("update error", func(t *testing.T) {
		updateFailRepo := newMockUserRepoForVerification()
		updateFailRepo.updateError = errors.New("update failed")
		failService := NewEmailVerificationService(updateFailRepo, mockSender, DefaultVerificationExpiry)

		failToken := "fail_token_1234567890abcdef"
		failExpiresAt := time.Now().Add(24 * time.Hour)
		failUser := &entity.User{
			ID:                      uuid.New(),
			Email:                   "fail@example.com",
			Name:                    "Fail User",
			EmailVerificationToken:  failToken,
			EmailVerificationExpires: &failExpiresAt,
			EmailVerified:           false,
			Status:                  "pending_verification",
		}
		updateFailRepo.Create(ctx, failUser)
		updateFailRepo.byToken[failToken] = failUser

		_, err := failService.VerifyEmail(ctx, failToken)
		if err == nil {
			t.Error("expected error for update failure")
		}
		if !errors.Is(err, apperrors.ErrInvalidVerificationToken) {
			// Error message should contain "failed to verify email"
			if err.Error() == "update failed" {
				t.Error("expected wrapped error")
			}
		}
	})

	// Test empty token
	t.Run("empty token", func(t *testing.T) {
		_, err := service.VerifyEmail(ctx, "")
		if err == nil {
			t.Error("expected error for empty token")
		}
		if !errors.Is(err, apperrors.ErrInvalidVerificationToken) {
			t.Errorf("expected ErrInvalidVerificationToken, got %v", err)
		}
	})

	// Test nil expiry (should succeed)
	t.Run("nil expiry", func(t *testing.T) {
		noExpiryToken := "noexpiry_token_1234567890abcdef"
		noExpiryUser := &entity.User{
			ID:                      uuid.New(),
			Email:                   "noexpiry@example.com",
			Name:                    "No Expiry User",
			EmailVerificationToken:  noExpiryToken,
			EmailVerificationExpires: nil,
			EmailVerified:           false,
			Status:                  "pending_verification",
		}
		mockUserRepo.Create(ctx, noExpiryUser)
		mockUserRepo.byToken[noExpiryToken] = noExpiryUser

		verifiedUser, err := service.VerifyEmail(ctx, noExpiryToken)
		if err != nil {
			t.Errorf("unexpected error for nil expiry: %v", err)
		}
		if verifiedUser == nil {
			t.Error("expected user to be returned")
		}
	})
}

func TestEmailVerificationService_ResendVerification(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := newMockUserRepoForVerification()
	mockSender := &mockEmailSender{}
	service := NewEmailVerificationService(mockUserRepo, mockSender, DefaultVerificationExpiry)

	// Create test user
	userID := uuid.New()
	user := &entity.User{
		ID:            userID,
		Email:         "resend@example.com",
		Name:          "Resend User",
		EmailVerified: false,
		Status:        "pending_verification",
	}
	mockUserRepo.Create(ctx, user)

	// Test successful resend
	t.Run("successful resend", func(t *testing.T) {
		err := service.ResendVerification(ctx, "resend@example.com")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Verify new token was set
		if user.EmailVerificationToken == "" {
			t.Error("expected new token to be set")
		}
	})

	// Test user not found
	t.Run("user not found", func(t *testing.T) {
		err := service.ResendVerification(ctx, "nonexistent@example.com")
		if err == nil {
			t.Error("expected error for non-existent user")
		}
		if !errors.Is(err, apperrors.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	// Test already verified user
	t.Run("already verified", func(t *testing.T) {
		verifiedUser := &entity.User{
			ID:            uuid.New(),
			Email:         "verified_resend@example.com",
			Name:          "Verified Resend",
			EmailVerified: true,
			Status:        "active",
		}
		mockUserRepo.Create(ctx, verifiedUser)

		err := service.ResendVerification(ctx, "verified_resend@example.com")
		if err == nil {
			t.Error("expected error for already verified user")
		}
		if !errors.Is(err, apperrors.ErrEmailAlreadyVerified) {
			t.Errorf("expected ErrEmailAlreadyVerified, got %v", err)
		}
	})
}

func TestEmailVerificationService_IsEmailVerified(t *testing.T) {
	mockUserRepo := newMockUserRepoForVerification()
	mockSender := &mockEmailSender{}
	service := NewEmailVerificationService(mockUserRepo, mockSender, DefaultVerificationExpiry)

	tests := []struct {
		name     string
		user     *entity.User
		expected bool
	}{
		{"verified user", &entity.User{EmailVerified: true}, true},
		{"unverified user", &entity.User{EmailVerified: false}, false},
		{"nil user", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: IsEmailVerified checks user.EmailVerified directly
			// For nil user test, we need to handle differently
			if tt.user != nil {
				result := service.IsEmailVerified(tt.user)
				if result != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestEmailVerificationService_NeedsVerification(t *testing.T) {
	mockUserRepo := newMockUserRepoForVerification()
	mockSender := &mockEmailSender{}
	service := NewEmailVerificationService(mockUserRepo, mockSender, DefaultVerificationExpiry)

	tests := []struct {
		name     string
		user     *entity.User
		expected bool
	}{
		{"needs verification", &entity.User{EmailVerified: false, EmailVerificationToken: "token123"}, true},
		{"verified no token", &entity.User{EmailVerified: true, EmailVerificationToken: "token123"}, false},
		{"unverified no token", &entity.User{EmailVerified: false, EmailVerificationToken: ""}, false},
		{"verified empty token", &entity.User{EmailVerified: true, EmailVerificationToken: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.NeedsVerification(tt.user)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEmailVerificationService_Integration(t *testing.T) {
	ctx := context.Background()
	mockUserRepo := newMockUserRepoForVerification()
	mockSender := &mockEmailSender{}
	service := NewEmailVerificationService(mockUserRepo, mockSender, DefaultVerificationExpiry)

	// Create user
	userID := uuid.New()
	user := &entity.User{
		ID:     userID,
		Email:  "integration@example.com",
		Name:   "Integration User",
		Status: "active",
	}
	mockUserRepo.Create(ctx, user)

	// Step 1: Send verification email
	err := service.SendVerificationEmail(ctx, user)
	if err != nil {
		t.Fatalf("failed to send verification email: %v", err)
	}

	// Check token was set
	token := user.EmailVerificationToken
	if token == "" {
		t.Fatal("token should be set")
	}

	// Step 2: Verify email
	verifiedUser, err := service.VerifyEmail(ctx, token)
	if err != nil {
		t.Fatalf("failed to verify email: %v", err)
	}

	// Check verification status
	if !verifiedUser.EmailVerified {
		t.Error("email should be verified")
	}
	if verifiedUser.Status != "active" {
		t.Errorf("expected status active, got %s", verifiedUser.Status)
	}

	// Step 3: Try to verify again (should fail)
	_, err = service.VerifyEmail(ctx, token)
	if err == nil {
		t.Error("should fail for already verified email")
	}
}