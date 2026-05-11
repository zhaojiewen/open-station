package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// UserApplicationService handles user application and invitation workflows
type UserApplicationService struct {
	appRepo    repository.UserApplicationRepository
	userRepo   repository.UserRepository
	tenantRepo repository.TenantRepository
}

// NewUserApplicationService creates a new user application service
func NewUserApplicationService(
	appRepo repository.UserApplicationRepository,
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
) *UserApplicationService {
	return &UserApplicationService{
		appRepo:    appRepo,
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
	}
}

// SubmitRequest submits a user request to join a tenant
func (s *UserApplicationService) SubmitRequest(ctx context.Context, tenantID uuid.UUID, email, name, requestedRole string) (*entity.UserApplication, error) {
	// Check tenant exists and is active
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, apperrors.ErrTenantNotFound
	}
	if tenant.Status != "active" {
		return nil, apperrors.ErrTenantSuspended
	}

	// Check if user already exists with this email
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, apperrors.ErrUserEmailExists
	}

	// Check if application already exists
	existingApp, err := s.appRepo.GetByEmail(ctx, tenantID, email)
	if err == nil && existingApp != nil && existingApp.Status == "pending" {
		return nil, apperrors.NewAppError("APP_007", "application already pending", nil)
	}

	app := &entity.UserApplication{
		TenantID:        tenantID,
		Email:           email,
		Name:            name,
		RequestedRole:   requestedRole,
		ApplicationType: "request",
		Status:          "pending",
	}

	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// SendInvitation sends an invitation to a user
func (s *UserApplicationService) SendInvitation(ctx context.Context, tenantID uuid.UUID, email, name, requestedRole string, invitedBy uuid.UUID, expiresIn int) (*entity.UserApplication, error) {
	// Check tenant exists and is active
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, apperrors.ErrTenantNotFound
	}
	if tenant.Status != "active" {
		return nil, apperrors.ErrTenantSuspended
	}

	// Check if user already exists with this email
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, apperrors.ErrUserEmailExists
	}

	// Generate invitation token
	token, err := generateInviteToken()
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	// Calculate expiration time (default 7 days)
	if expiresIn <= 0 {
		expiresIn = 7 * 24 * 60 * 60 // 7 days in seconds
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	app := &entity.UserApplication{
		TenantID:        tenantID,
		Email:           email,
		Name:            name,
		RequestedRole:   requestedRole,
		ApplicationType: "invitation",
		InvitedBy:       &invitedBy,
		InviteToken:     token,
		ExpiresAt:       &expiresAt,
		Status:          "pending",
	}

	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// CreateDirect directly creates a user without approval
func (s *UserApplicationService) CreateDirect(ctx context.Context, tenantID uuid.UUID, email, name, role, password string, createdBy uuid.UUID) (*entity.User, error) {
	// Check tenant exists and is active
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, apperrors.ErrTenantNotFound
	}
	if tenant.Status != "active" {
		return nil, apperrors.ErrTenantSuspended
	}

	// Check max users
	users, _, err := s.userRepo.List(ctx, tenantID, 1, 1000)
	if err == nil && len(users) >= tenant.MaxUsers {
		return nil, apperrors.ErrTenantMaxUsersReached
	}

	// Check if user already exists with this email
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, apperrors.ErrUserEmailExists
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	// Create application record
	app := &entity.UserApplication{
		TenantID:        tenantID,
		Email:           email,
		Name:            name,
		RequestedRole:   role,
		ApplicationType: "direct_create",
		CreatedBy:       &createdBy,
		Status:          "user_created",
	}
	s.appRepo.Create(ctx, app)

	// Create user
	now := time.Now()
	user := &entity.User{
		TenantID:      tenantID,
		Email:         email,
		PasswordHash:  string(passwordHash),
		Name:          name,
		Role:          role,
		Status:        "active",
		ApplicationID: &app.ID,
		ApprovedBy:    &createdBy,
		ApprovedAt:    &now,
		MaxAPIKeys:    &tenant.MaxAPIKeysPerUser,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Mark application as user created
	s.appRepo.MarkUserCreated(ctx, app.ID, user.ID)

	return user, nil
}

// AcceptInvitation accepts an invitation with token
func (s *UserApplicationService) AcceptInvitation(ctx context.Context, token, name, password string) (*entity.User, error) {
	// Find application by token
	app, err := s.appRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, apperrors.ErrInviteInvalidToken
	}

	// Check if invitation is pending
	if app.Status != "pending" {
		return nil, apperrors.ErrInviteAlreadyAccepted
	}

	// Check if invitation expired
	if app.ExpiresAt != nil && app.ExpiresAt.Before(time.Now()) {
		s.appRepo.MarkExpired(ctx, app.ID)
		return nil, apperrors.ErrInviteExpired
	}

	// Check tenant
	tenant, err := s.tenantRepo.GetByID(ctx, app.TenantID)
	if err != nil {
		return nil, apperrors.ErrTenantNotFound
	}
	if tenant.Status != "active" {
		return nil, apperrors.ErrTenantSuspended
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	// Accept invitation
	s.appRepo.Accept(ctx, app.ID)

	// Create user
	now := time.Now()
	user := &entity.User{
		TenantID:      app.TenantID,
		Email:         app.Email,
		PasswordHash:  string(passwordHash),
		Name:          name,
		Role:          app.RequestedRole,
		Status:        "active",
		ApplicationID: &app.ID,
		ApprovedBy:    app.InvitedBy,
		ApprovedAt:    &now,
		MaxAPIKeys:    &tenant.MaxAPIKeysPerUser,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Mark application as user created
	s.appRepo.MarkUserCreated(ctx, app.ID, user.ID)

	return user, nil
}

// ApproveRequest approves a user request
func (s *UserApplicationService) ApproveRequest(ctx context.Context, appID uuid.UUID, reviewerID uuid.UUID, password string) (*entity.User, error) {
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return nil, apperrors.ErrApplicationNotFound
	}

	// Check status
	if app.Status != "pending" {
		return nil, apperrors.ErrApplicationAlreadyProcessed
	}

	// Check tenant
	tenant, err := s.tenantRepo.GetByID(ctx, app.TenantID)
	if err != nil {
		return nil, apperrors.ErrTenantNotFound
	}
	if tenant.Status != "active" {
		return nil, apperrors.ErrTenantSuspended
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	// Approve
	s.appRepo.Approve(ctx, appID, reviewerID)

	// Create user
	now := time.Now()
	user := &entity.User{
		TenantID:      app.TenantID,
		Email:         app.Email,
		PasswordHash:  string(passwordHash),
		Name:          app.Name,
		Role:          app.RequestedRole,
		Status:        "active",
		ApplicationID: &app.ID,
		ApprovedBy:    &reviewerID,
		ApprovedAt:    &now,
		MaxAPIKeys:    &tenant.MaxAPIKeysPerUser,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Mark application as user created
	s.appRepo.MarkUserCreated(ctx, app.ID, user.ID)

	return user, nil
}

// RejectRequest rejects a user request
func (s *UserApplicationService) RejectRequest(ctx context.Context, appID uuid.UUID, reviewerID uuid.UUID) error {
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return apperrors.ErrApplicationNotFound
	}

	// Check status
	if app.Status != "pending" {
		return apperrors.ErrApplicationAlreadyProcessed
	}

	return s.appRepo.Reject(ctx, appID, reviewerID)
}

// GetByID gets an application by ID
func (s *UserApplicationService) GetByID(ctx context.Context, id uuid.UUID) (*entity.UserApplication, error) {
	return s.appRepo.GetByID(ctx, id)
}

// GetByToken gets an application by invitation token
func (s *UserApplicationService) GetByToken(ctx context.Context, token string) (*entity.UserApplication, error) {
	return s.appRepo.GetByToken(ctx, token)
}

// List lists applications for a tenant
func (s *UserApplicationService) List(ctx context.Context, tenantID uuid.UUID, status string, page, pageSize int) ([]entity.UserApplication, int64, error) {
	return s.appRepo.List(ctx, tenantID, status, page, pageSize)
}

// CancelInvitation cancels an invitation
func (s *UserApplicationService) CancelInvitation(ctx context.Context, id uuid.UUID) error {
	app, err := s.appRepo.GetByID(ctx, id)
	if err != nil {
		return apperrors.ErrInviteNotFound
	}

	if app.ApplicationType != "invitation" || app.Status != "pending" {
		return apperrors.ErrApplicationInvalidStatus
	}

	return s.appRepo.Delete(ctx, id)
}

// Helper functions

func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// MCP wrapper methods (simplified signatures for MCP tools)

// SendInvitationSimple sends an invitation without tenantID (uses default tenant from context)
func (s *UserApplicationService) SendInvitationSimple(ctx context.Context, email, name, requestedRole string, expiresIn int64) (*entity.UserApplication, error) {
	// For MCP tools, we need to find an active tenant or use a default
	// In practice, this should be called with tenant context from session
	// Here we use a placeholder - the actual implementation should get tenantID from context
	tenants, _, err := s.tenantRepo.List(ctx, 1, 1)
	if err != nil || len(tenants) == 0 {
		return nil, apperrors.ErrTenantNotFound
	}
	tenantID := tenants[0].ID

	// Convert expiresIn from int64 to int (seconds)
	expiresInInt := int(expiresIn)
	if expiresInInt <= 0 {
		expiresInInt = 7 * 24 * 60 * 60 // 7 days
	}

	return s.SendInvitation(ctx, tenantID, email, name, requestedRole, uuid.Nil, expiresInInt)
}

// ListSimple lists applications without tenantID (lists all for MCP)
func (s *UserApplicationService) ListSimple(ctx context.Context, status string, page, pageSize int) ([]entity.UserApplication, int64, error) {
	// List all applications across tenants for MCP admin use
	// This requires adding a ListAll method to the repository
	return s.appRepo.ListAll(ctx, status, page, pageSize)
}

// ApproveRequestSimple approves a request without reviewerID
func (s *UserApplicationService) ApproveRequestSimple(ctx context.Context, appID uuid.UUID, password string) (*entity.User, error) {
	return s.ApproveRequest(ctx, appID, uuid.Nil, password)
}

// RejectRequestSimple rejects a request without reviewerID
func (s *UserApplicationService) RejectRequestSimple(ctx context.Context, appID uuid.UUID) error {
	return s.RejectRequest(ctx, appID, uuid.Nil)
}

// CreateDirectSimple creates a user directly without tenantID and createdBy
func (s *UserApplicationService) CreateDirectSimple(ctx context.Context, email, name, password, role string) (*entity.User, error) {
	// Find an active tenant for MCP use
	tenants, _, err := s.tenantRepo.List(ctx, 1, 1)
	if err != nil || len(tenants) == 0 {
		return nil, apperrors.ErrTenantNotFound
	}
	tenantID := tenants[0].ID

	return s.CreateDirect(ctx, tenantID, email, name, role, password, uuid.Nil)
}