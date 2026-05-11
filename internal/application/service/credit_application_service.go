package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// CreditApplicationService handles credit application workflow
type CreditApplicationService struct {
	creditAppRepo  repository.CreditApplicationRepository
	tenantRepo     repository.TenantRepository
	notificationSvc *NotificationService
}

// NewCreditApplicationService creates a new credit application service
func NewCreditApplicationService(
	creditAppRepo repository.CreditApplicationRepository,
	tenantRepo repository.TenantRepository,
	notificationSvc *NotificationService,
) *CreditApplicationService {
	return &CreditApplicationService{
		creditAppRepo:  creditAppRepo,
		tenantRepo:     tenantRepo,
		notificationSvc: notificationSvc,
	}
}

// ApplyForCredit creates a new credit application for a tenant
func (s *CreditApplicationService) ApplyForCredit(ctx context.Context, tenantID uuid.UUID, req *CreditApplicationRequest) (*entity.CreditApplication, error) {
	// 1. Check tenant exists and is organization type
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.Type != "organization" {
		return nil, apperrors.ErrInvalidQuotaType
	}

	// 2. Check for existing pending application for this tenant
	existingApp, err := s.creditAppRepo.GetLatestByTenantID(ctx, tenantID)
	if err == nil && existingApp.Status == "pending" {
		return nil, apperrors.ErrApplicationPending
	}

	// 3. Create application
	application := &entity.CreditApplication{
		TenantID:        tenantID,
		RequestedLimit:  req.RequestedLimit,
		Reason:          req.Reason,
		Status:          "pending",
		SettlementCycle: req.SettlementCycle,
		ThresholdAmount: req.ThresholdAmount,
		SettlementDay:   req.SettlementDay,
	}

	if err := s.creditAppRepo.Create(ctx, application); err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	// 4. Update tenant credit status to pending
	tenant.CreditStatus = "pending"
	tenant.CreditAppliedAt = &time.Time{}
	*tenant.CreditAppliedAt = time.Now()
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant credit status: %w", err)
	}

	// 5. Send notification to platform admins (if configured)
	if s.notificationSvc != nil {
		// TODO: Send notification to platform admins about new application
	}

	return application, nil
}

// CreditApplicationRequest represents the credit application request
type CreditApplicationRequest struct {
	RequestedLimit  decimal.Decimal
	Reason          string
	SettlementCycle string           // monthly, weekly, threshold, custom
	ThresholdAmount *decimal.Decimal // for threshold settlement
	SettlementDay   *int             // settlement day
}

// GetApplication retrieves credit application by ID
func (s *CreditApplicationService) GetApplication(ctx context.Context, applicationID uuid.UUID) (*entity.CreditApplication, error) {
	return s.creditAppRepo.GetByID(ctx, applicationID)
}

// GetTenantApplication retrieves the latest credit application for a tenant
func (s *CreditApplicationService) GetTenantApplication(ctx context.Context, tenantID uuid.UUID) (*entity.CreditApplication, error) {
	return s.creditAppRepo.GetLatestByTenantID(ctx, tenantID)
}

// UpdateApplication updates a pending credit application
func (s *CreditApplicationService) UpdateApplication(ctx context.Context, applicationID uuid.UUID, req *CreditApplicationUpdateRequest) (*entity.CreditApplication, error) {
	application, err := s.creditAppRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// Only pending applications can be updated
	if application.Status != "pending" {
		return nil, apperrors.ErrApplicationAlreadyProcessed
	}

	// Update fields
	if req.RequestedLimit.GreaterThan(decimal.Zero) {
		application.RequestedLimit = req.RequestedLimit
	}
	if req.Reason != "" {
		application.Reason = req.Reason
	}
	if req.SettlementCycle != "" {
		application.SettlementCycle = req.SettlementCycle
	}
	if req.ThresholdAmount != nil {
		application.ThresholdAmount = req.ThresholdAmount
	}
	if req.SettlementDay != nil {
		application.SettlementDay = req.SettlementDay
	}

	if err := s.creditAppRepo.Update(ctx, application); err != nil {
		return nil, fmt.Errorf("failed to update application: %w", err)
	}

	return application, nil
}

// CreditApplicationUpdateRequest represents the update request
type CreditApplicationUpdateRequest struct {
	RequestedLimit  decimal.Decimal
	Reason          string
	SettlementCycle string
	ThresholdAmount *decimal.Decimal
	SettlementDay   *int
}

// CancelApplication cancels a pending credit application
func (s *CreditApplicationService) CancelApplication(ctx context.Context, applicationID uuid.UUID) error {
	application, err := s.creditAppRepo.GetByID(ctx, applicationID)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}

	if application.Status != "pending" {
		return apperrors.ErrApplicationAlreadyProcessed
	}

	// Delete application
	if err := s.creditAppRepo.Delete(ctx, applicationID); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	// Update tenant credit status back to none
	tenant, err := s.tenantRepo.GetByID(ctx, application.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	tenant.CreditStatus = "none"
	tenant.CreditAppliedAt = nil
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	return nil
}

// ListApplications lists all credit applications with pagination
func (s *CreditApplicationService) ListApplications(ctx context.Context, page, pageSize int) ([]entity.CreditApplication, int64, error) {
	return s.creditAppRepo.List(ctx, page, pageSize)
}

// ListApplicationsByStatus lists credit applications by status
func (s *CreditApplicationService) ListApplicationsByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.CreditApplication, int64, error) {
	return s.creditAppRepo.ListByStatus(ctx, status, page, pageSize)
}

// ==================== Platform Admin Functions ====================

// ApproveApplication approves a credit application (platform admin only)
func (s *CreditApplicationService) ApproveApplication(ctx context.Context, applicationID uuid.UUID, reviewerID uuid.UUID, req *ApprovalRequest) (*entity.CreditApplication, error) {
	application, err := s.creditAppRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	if application.Status != "pending" {
		return nil, apperrors.ErrApplicationAlreadyProcessed
	}

	// Approve application
	if err := s.creditAppRepo.Approve(ctx, applicationID, req.ApprovedLimit, reviewerID, req.ReviewNotes); err != nil {
		return nil, fmt.Errorf("failed to approve application: %w", err)
	}

	// Update tenant with approved credit limit
	tenant, err := s.tenantRepo.GetByID(ctx, application.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	now := time.Now()
	tenant.CreditLimit = &req.ApprovedLimit
	tenant.CreditStatus = "approved"
	tenant.CreditApprovedAt = &now
	tenant.CreditApprovedBy = &reviewerID
	tenant.SettlementCycle = application.SettlementCycle
	tenant.ThresholdAmount = application.ThresholdAmount
	if application.SettlementDay != nil {
		tenant.SettlementDay = application.SettlementDay
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant credit: %w", err)
	}

	// Get updated application
	application, err = s.creditAppRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated application: %w", err)
	}

	// Send approval notification to tenant
	if s.notificationSvc != nil {
		// TODO: Send approval notification
	}

	return application, nil
}

// ApprovalRequest represents the approval request
type ApprovalRequest struct {
	ApprovedLimit decimal.Decimal
	ReviewNotes   string
}

// RejectApplication rejects a credit application (platform admin only)
func (s *CreditApplicationService) RejectApplication(ctx context.Context, applicationID uuid.UUID, reviewerID uuid.UUID, reason string) (*entity.CreditApplication, error) {
	application, err := s.creditAppRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	if application.Status != "pending" {
		return nil, apperrors.ErrApplicationAlreadyProcessed
	}

	// Reject application
	if err := s.creditAppRepo.Reject(ctx, applicationID, reviewerID, reason); err != nil {
		return nil, fmt.Errorf("failed to reject application: %w", err)
	}

	// Update tenant credit status
	tenant, err := s.tenantRepo.GetByID(ctx, application.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	tenant.CreditStatus = "rejected"
	tenant.CreditRejectReason = reason
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	// Get updated application
	application, err = s.creditAppRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated application: %w", err)
	}

	// Send rejection notification to tenant
	if s.notificationSvc != nil {
		// TODO: Send rejection notification
	}

	return application, nil
}

// AdjustCreditLimit adjusts the credit limit for an approved tenant (platform admin only)
func (s *CreditApplicationService) AdjustCreditLimit(ctx context.Context, tenantID uuid.UUID, newLimit decimal.Decimal) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.CreditStatus != "approved" {
		return apperrors.ErrCreditNotApproved
	}

	tenant.CreditLimit = &newLimit
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to adjust credit limit: %w", err)
	}

	return nil
}

// GetPendingCount returns the count of pending applications
func (s *CreditApplicationService) GetPendingCount(ctx context.Context) (int64, error) {
	return s.creditAppRepo.GetPendingCount(ctx)
}