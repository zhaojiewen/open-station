package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// TenantApplicationService handles tenant application workflows
type TenantApplicationService struct {
	appRepo     repository.TenantApplicationRepository
	tenantRepo  repository.TenantRepository
	userRepo    repository.UserRepository
	apiKeyRepo  repository.APIKeyRepository
}

// NewTenantApplicationService creates a new tenant application service
func NewTenantApplicationService(
	appRepo repository.TenantApplicationRepository,
	tenantRepo repository.TenantRepository,
	userRepo repository.UserRepository,
	apiKeyRepo repository.APIKeyRepository,
) *TenantApplicationService {
	return &TenantApplicationService{
		appRepo:    appRepo,
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
		apiKeyRepo: apiKeyRepo,
	}
}

// Submit submits a new tenant application
func (s *TenantApplicationService) Submit(ctx context.Context, companyName, companySlug, contactEmail, contactPhone, contactName, requestedPlan string) (*entity.TenantApplication, error) {
	// Check if slug already exists
	existingSlug, err := s.appRepo.GetBySlug(ctx, companySlug)
	if err == nil && existingSlug != nil {
		return nil, apperrors.ErrTenantSlugExists
	}

	// Check if email already exists
	existingEmail, err := s.appRepo.GetByEmail(ctx, contactEmail)
	if err == nil && existingEmail != nil && existingEmail.Status != "rejected" {
		return nil, apperrors.NewAppError("APP_007", "application already exists for this email", nil)
	}

	app := &entity.TenantApplication{
		CompanyName:    companyName,
		CompanySlug:    companySlug,
		ContactEmail:   contactEmail,
		ContactPhone:   contactPhone,
		ContactName:    contactName,
		RequestedPlan:  requestedPlan,
		Status:         "pending",
	}

	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// GetByID gets a tenant application by ID
func (s *TenantApplicationService) GetByID(ctx context.Context, id uuid.UUID) (*entity.TenantApplication, error) {
	return s.appRepo.GetByID(ctx, id)
}

// List lists tenant applications
func (s *TenantApplicationService) List(ctx context.Context, status string, page, pageSize int) ([]entity.TenantApplication, int64, error) {
	return s.appRepo.List(ctx, status, page, pageSize)
}

// Approve approves a tenant application and creates the tenant
func (s *TenantApplicationService) Approve(ctx context.Context, appID uuid.UUID, reviewerID uuid.UUID, notes string) (*entity.Tenant, error) {
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return nil, apperrors.ErrApplicationNotFound
	}

	// Check status
	if app.Status != "pending" && app.Status != "reviewing" {
		return nil, apperrors.ErrApplicationAlreadyProcessed
	}

	// Update application status to approved
	if err := s.appRepo.Approve(ctx, appID, reviewerID, notes); err != nil {
		return nil, err
	}

	// Create tenant
	tenant := &entity.Tenant{
		Name:             app.CompanyName,
		Slug:             app.CompanySlug,
		Status:           "active",
		Plan:             app.RequestedPlan,
		BillingEmail:     app.ContactEmail,
		Balance:          decimal.NewFromInt(1000), // Initial balance
		Currency:         "USD",
		ApplicationID:    &appID,
		ApprovedBy:       &reviewerID,
		ApprovedAt:       &time.Time{},
		MaxUsers:         100,
		MaxAPIKeysPerUser: 10,
	}
	now := time.Now()
	tenant.ApprovedAt = &now

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	// Mark application as tenant created
	if err := s.appRepo.MarkTenantCreated(ctx, appID, tenant.ID); err != nil {
		return nil, err
	}

	return tenant, nil
}

// Reject rejects a tenant application
func (s *TenantApplicationService) Reject(ctx context.Context, appID uuid.UUID, reviewerID uuid.UUID, reason string) error {
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return apperrors.ErrApplicationNotFound
	}

	// Check status
	if app.Status != "pending" && app.Status != "reviewing" {
		return apperrors.ErrApplicationAlreadyProcessed
	}

	return s.appRepo.Reject(ctx, appID, reviewerID, reason)
}

// StartReview marks an application as being reviewed
func (s *TenantApplicationService) StartReview(ctx context.Context, appID uuid.UUID) error {
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return apperrors.ErrApplicationNotFound
	}

	if app.Status != "pending" {
		return apperrors.ErrApplicationInvalidStatus
	}

	return s.appRepo.SetStatus(ctx, appID, "reviewing")
}

// ApproveSimple approves a tenant application (MCP wrapper without reviewerID)
func (s *TenantApplicationService) ApproveSimple(ctx context.Context, appID uuid.UUID, notes string) (*entity.Tenant, error) {
	// Use uuid.Nil as reviewerID for MCP calls
	return s.Approve(ctx, appID, uuid.Nil, notes)
}

// RejectSimple rejects a tenant application (MCP wrapper without reviewerID)
func (s *TenantApplicationService) RejectSimple(ctx context.Context, appID uuid.UUID, reason string) error {
	// Use uuid.Nil as reviewerID for MCP calls
	return s.Reject(ctx, appID, uuid.Nil, reason)
}

// SuspendTenant suspends a tenant
func (s *TenantApplicationService) SuspendTenant(ctx context.Context, tenantID uuid.UUID) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return apperrors.ErrTenantNotFound
	}

	tenant.Status = "suspended"
	return s.tenantRepo.Update(ctx, tenant)
}

// ActivateTenant activates a suspended tenant
func (s *TenantApplicationService) ActivateTenant(ctx context.Context, tenantID uuid.UUID) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return apperrors.ErrTenantNotFound
	}

	tenant.Status = "active"
	return s.tenantRepo.Update(ctx, tenant)
}