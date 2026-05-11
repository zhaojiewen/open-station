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

// MemberQuotaService handles member quota management for organization tenants
type MemberQuotaService struct {
	memberQuotaRepo repository.MemberQuotaRepository
	tenantRepo      repository.TenantRepository
	userRepo        repository.UserRepository
}

// NewMemberQuotaService creates a new member quota service
func NewMemberQuotaService(
	memberQuotaRepo repository.MemberQuotaRepository,
	tenantRepo repository.TenantRepository,
	userRepo repository.UserRepository,
) *MemberQuotaService {
	return &MemberQuotaService{
		memberQuotaRepo: memberQuotaRepo,
		tenantRepo:      tenantRepo,
		userRepo:        userRepo,
	}
}

// CreateMemberQuota creates a member quota for an organization member
func (s *MemberQuotaService) CreateMemberQuota(ctx context.Context, req *MemberQuotaCreateRequest) (*entity.MemberQuota, error) {
	// 1. Check tenant is organization type
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.Type != "organization" {
		return nil, apperrors.ErrInvalidQuotaType
	}

	// 2. Check user exists and belongs to tenant
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.TenantID != req.TenantID {
		return nil, fmt.Errorf("user does not belong to tenant")
	}

	// 3. Check if member quota already exists
	existing, err := s.memberQuotaRepo.GetByUserID(ctx, req.UserID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("member quota already exists for this user")
	}

	// 4. Create member quota
	memberQuota := &entity.MemberQuota{
		TenantID:        req.TenantID,
		UserID:          req.UserID,
		TokenQuotaLimit: req.TokenQuotaLimit,
		TokensUsed:      0,
		CostLimit:       req.CostLimit,
		CostLimitType:   req.CostLimitType,
		CostUsed:        decimal.Zero,
		MaxAPIKeys:      req.MaxAPIKeys,
		ActiveAPIKeys:   0,
		Status:          "active",
	}

	if err := s.memberQuotaRepo.Create(ctx, memberQuota); err != nil {
		return nil, fmt.Errorf("failed to create member quota: %w", err)
	}

	// 5. Update user with member quota reference
	user.MemberQuotaID = &memberQuota.ID
	user.UserMode = "member"
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return memberQuota, nil
}

// MemberQuotaCreateRequest represents the create request
type MemberQuotaCreateRequest struct {
	TenantID        uuid.UUID
	UserID          uuid.UUID
	TokenQuotaLimit *int64
	CostLimit       *decimal.Decimal
	CostLimitType   string // monthly, daily
	MaxAPIKeys      *int
}

// GetMemberQuota retrieves member quota by ID
func (s *MemberQuotaService) GetMemberQuota(ctx context.Context, quotaID uuid.UUID) (*entity.MemberQuota, error) {
	return s.memberQuotaRepo.GetByID(ctx, quotaID)
}

// GetMemberQuotaByUser retrieves member quota by user ID
func (s *MemberQuotaService) GetMemberQuotaByUser(ctx context.Context, userID uuid.UUID) (*entity.MemberQuota, error) {
	return s.memberQuotaRepo.GetByUserID(ctx, userID)
}

// ListMemberQuotas lists all member quotas for a tenant
func (s *MemberQuotaService) ListMemberQuotas(ctx context.Context, tenantID uuid.UUID) ([]entity.MemberQuota, error) {
	quotas, _, err := s.memberQuotaRepo.ListByTenant(ctx, tenantID, 1, 1000)
	return quotas, err
}

// UpdateMemberQuota updates member quota settings
func (s *MemberQuotaService) UpdateMemberQuota(ctx context.Context, quotaID uuid.UUID, req *MemberQuotaUpdateRequest) (*entity.MemberQuota, error) {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member quota: %w", err)
	}

	// Update fields
	if req.TokenQuotaLimit != nil {
		memberQuota.TokenQuotaLimit = req.TokenQuotaLimit
	}
	if req.CostLimit != nil {
		memberQuota.CostLimit = req.CostLimit
	}
	if req.CostLimitType != "" {
		memberQuota.CostLimitType = req.CostLimitType
	}
	if req.MaxAPIKeys != nil {
		memberQuota.MaxAPIKeys = req.MaxAPIKeys
	}
	if req.Status != "" {
		memberQuota.Status = req.Status
	}

	if err := s.memberQuotaRepo.Update(ctx, memberQuota); err != nil {
		return nil, fmt.Errorf("failed to update member quota: %w", err)
	}

	return memberQuota, nil
}

// MemberQuotaUpdateRequest represents the update request
type MemberQuotaUpdateRequest struct {
	TokenQuotaLimit *int64
	CostLimit       *decimal.Decimal
	CostLimitType   string
	MaxAPIKeys      *int
	Status          string
}

// SetTokenQuotaLimit sets member token quota limit
func (s *MemberQuotaService) SetTokenQuotaLimit(ctx context.Context, quotaID uuid.UUID, limit int64) error {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return fmt.Errorf("failed to get member quota: %w", err)
	}

	memberQuota.TokenQuotaLimit = &limit
	if err := s.memberQuotaRepo.Update(ctx, memberQuota); err != nil {
		return fmt.Errorf("failed to set token quota limit: %w", err)
	}

	return nil
}

// SetCostLimit sets member cost limit
func (s *MemberQuotaService) SetCostLimit(ctx context.Context, quotaID uuid.UUID, limit decimal.Decimal, limitType string) error {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return fmt.Errorf("failed to get member quota: %w", err)
	}

	memberQuota.CostLimit = &limit
	memberQuota.CostLimitType = limitType
	if err := s.memberQuotaRepo.Update(ctx, memberQuota); err != nil {
		return fmt.Errorf("failed to set cost limit: %w", err)
	}

	return nil
}

// ResetMemberQuota resets member quota usage
func (s *MemberQuotaService) ResetMemberQuota(ctx context.Context, quotaID uuid.UUID) error {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return fmt.Errorf("failed to get member quota: %w", err)
	}

	memberQuota.TokensUsed = 0
	memberQuota.CostUsed = decimal.Zero
	memberQuota.Status = "active"
	memberQuota.ExceededAt = nil
	memberQuota.ExceededReason = ""

	if err := s.memberQuotaRepo.Update(ctx, memberQuota); err != nil {
		return fmt.Errorf("failed to reset member quota: %w", err)
	}

	return nil
}

// DeleteMemberQuota deletes a member quota
func (s *MemberQuotaService) DeleteMemberQuota(ctx context.Context, quotaID uuid.UUID) error {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return fmt.Errorf("failed to get member quota: %w", err)
	}

	// Update user to remove member quota reference
	user, err := s.userRepo.GetByID(ctx, memberQuota.UserID)
	if err == nil {
		user.MemberQuotaID = nil
		user.UserMode = "individual"
		s.userRepo.Update(ctx, user)
	}

	return s.memberQuotaRepo.Delete(ctx, quotaID)
}

// GetMemberUsage returns member quota usage statistics
func (s *MemberQuotaService) GetMemberUsage(ctx context.Context, quotaID uuid.UUID) (*MemberUsageInfo, error) {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member quota: %w", err)
	}

	tenant, err := s.tenantRepo.GetByID(ctx, memberQuota.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	info := &MemberUsageInfo{
		MemberQuotaID:   quotaID,
		TenantID:        memberQuota.TenantID,
		UserID:          memberQuota.UserID,
		Status:          memberQuota.Status,
		TokensUsed:      memberQuota.TokensUsed,
		CostUsed:        memberQuota.CostUsed,
		TenantTokensUsed: tenant.TokensUsedMonth,
		TenantCostUsed:  tenant.BudgetUsedMonth,
	}

	if memberQuota.TokenQuotaLimit != nil {
		info.TokenQuotaLimit = *memberQuota.TokenQuotaLimit
		info.TokenRemaining = *memberQuota.TokenQuotaLimit - memberQuota.TokensUsed
		if info.TokenRemaining < 0 {
			info.TokenRemaining = 0
		}
	}

	if memberQuota.CostLimit != nil {
		info.CostLimit = *memberQuota.CostLimit
		info.CostRemaining = memberQuota.CostLimit.Sub(memberQuota.CostUsed)
		if info.CostRemaining.LessThan(decimal.Zero) {
			info.CostRemaining = decimal.Zero
		}
	}

	if tenant.TokenLimit != nil {
		info.TenantTokenLimit = *tenant.TokenLimit
	}

	return info, nil
}

// MemberUsageInfo contains member usage statistics
type MemberUsageInfo struct {
	MemberQuotaID   uuid.UUID
	TenantID        uuid.UUID
	UserID          uuid.UUID
	Status          string
	TokenQuotaLimit int64
	TokensUsed      int64
	TokenRemaining  int64
	CostLimit       decimal.Decimal
	CostUsed        decimal.Decimal
	CostRemaining   decimal.Decimal
	TenantTokenLimit int64
	TenantTokensUsed int64
	TenantCostUsed  decimal.Decimal
}

// SuspendMember suspends a member due to quota exceeded
func (s *MemberQuotaService) SuspendMember(ctx context.Context, quotaID uuid.UUID, reason string) error {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return fmt.Errorf("failed to get member quota: %w", err)
	}

	now := time.Now()
	memberQuota.Status = "suspended"
	memberQuota.ExceededAt = &now
	memberQuota.ExceededReason = reason

	if err := s.memberQuotaRepo.Update(ctx, memberQuota); err != nil {
		return fmt.Errorf("failed to suspend member: %w", err)
	}

	return nil
}

// ActivateMember activates a suspended member
func (s *MemberQuotaService) ActivateMember(ctx context.Context, quotaID uuid.UUID) error {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return fmt.Errorf("failed to get member quota: %w", err)
	}

	memberQuota.Status = "active"
	memberQuota.ExceededAt = nil
	memberQuota.ExceededReason = ""

	if err := s.memberQuotaRepo.Update(ctx, memberQuota); err != nil {
		return fmt.Errorf("failed to activate member: %w", err)
	}

	return nil
}

// ListAllMemberQuotas lists all member quotas with pagination (platform admin only)
func (s *MemberQuotaService) ListAllMemberQuotas(ctx context.Context, page, pageSize int) ([]entity.MemberQuota, int64, error) {
	return s.memberQuotaRepo.List(ctx, page, pageSize)
}