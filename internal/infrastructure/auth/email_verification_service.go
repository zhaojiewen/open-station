package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

const (
	// Email verification token expires in 24 hours
	DefaultVerificationExpiry = 24 * time.Hour
	// Verification token length in bytes
	VerificationTokenLength   = 32
)

// EmailSender interface for sending emails
type EmailSender interface {
	SendVerificationEmail(to string, token string, userName string) error
}

// EmailVerificationService handles email verification logic
type EmailVerificationService struct {
	userRepo    repository.UserRepository
	emailSender EmailSender
	expiry      time.Duration
}

// NewEmailVerificationService creates a new email verification service
func NewEmailVerificationService(
	userRepo repository.UserRepository,
	emailSender EmailSender,
	expiry time.Duration,
) *EmailVerificationService {
	if expiry <= 0 {
		expiry = DefaultVerificationExpiry
	}
	return &EmailVerificationService{
		userRepo:    userRepo,
		emailSender: emailSender,
		expiry:      expiry,
	}
}

// GenerateVerificationToken generates a random verification token
func (s *EmailVerificationService) GenerateVerificationToken() (string, error) {
	b := make([]byte, VerificationTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// SendVerificationEmail sends verification email to user
func (s *EmailVerificationService) SendVerificationEmail(ctx context.Context, user *entity.User) error {
	// Generate new token
	token, err := s.GenerateVerificationToken()
	if err != nil {
		return err
	}

	// Set token and expiry
	expiresAt := time.Now().Add(s.expiry)
	user.EmailVerificationToken = token
	user.EmailVerificationExpires = &expiresAt
	user.Status = "pending_verification"
	user.EmailVerified = false

	// Update user in database
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save verification token: %w", err)
	}

	// Send email
	if s.emailSender != nil {
		if err := s.emailSender.SendVerificationEmail(user.Email, token, user.Name); err != nil {
			// Log the error but don't fail - user can request resend
			return fmt.Errorf("failed to send verification email: %w", err)
		}
	}

	return nil
}

// VerifyEmail verifies user email with token
func (s *EmailVerificationService) VerifyEmail(ctx context.Context, token string) (*entity.User, error) {
	// Find user by verification token
	user, err := s.userRepo.GetByVerificationToken(ctx, token)
	if err != nil {
		return nil, apperrors.ErrInvalidVerificationToken
	}

	// Check if already verified
	if user.EmailVerified {
		return nil, apperrors.ErrEmailAlreadyVerified
	}

	// Check if token expired
	if user.EmailVerificationExpires != nil && time.Now().After(*user.EmailVerificationExpires) {
		return nil, apperrors.ErrVerificationTokenExpired
	}

	// Verify email
	now := time.Now()
	user.EmailVerified = true
	user.EmailVerifiedAt = &now
	user.EmailVerificationToken = "" // Clear token
	user.EmailVerificationExpires = nil
	user.Status = "active"

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}

	return user, nil
}

// ResendVerification resends verification email
func (s *EmailVerificationService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	// Check if already verified
	if user.EmailVerified {
		return apperrors.ErrEmailAlreadyVerified
	}

	// Generate new token and send
	return s.SendVerificationEmail(ctx, user)
}

// IsEmailVerified checks if user email is verified
func (s *EmailVerificationService) IsEmailVerified(user *entity.User) bool {
	return user.EmailVerified
}

// NeedsVerification checks if user needs email verification
func (s *EmailVerificationService) NeedsVerification(user *entity.User) bool {
	return !user.EmailVerified && user.EmailVerificationToken != ""
}