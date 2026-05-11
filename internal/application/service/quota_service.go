package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// QuotaService provides unified quota check and deduction logic
type QuotaService struct {
	userQuotaRepo   repository.UserQuotaRepository
	memberQuotaRepo repository.MemberQuotaRepository
	tenantRepo      repository.TenantRepository
}

// NewQuotaService creates a new quota service
func NewQuotaService(
	userQuotaRepo repository.UserQuotaRepository,
	memberQuotaRepo repository.MemberQuotaRepository,
	tenantRepo repository.TenantRepository,
) *QuotaService {
	return &QuotaService{
		userQuotaRepo:   userQuotaRepo,
		memberQuotaRepo: memberQuotaRepo,
		tenantRepo:      tenantRepo,
	}
}

// CheckUsageAllowance checks if usage is allowed based on quota
// Priority: Token quota -> Balance -> Credit (org only)
func (s *QuotaService) CheckUsageAllowance(ctx context.Context, apiKey *entity.APIKey, tokens int64, cost decimal.Decimal) error {
	switch apiKey.QuotaType {
	case "individual":
		return s.checkIndividualQuota(ctx, apiKey.QuotaID, tokens, cost)
	case "member":
		return s.checkMemberQuota(ctx, apiKey.QuotaID, apiKey.TenantID, tokens, cost)
	default:
		return apperrors.ErrInvalidQuotaType
	}
}

// checkIndividualQuota checks quota for individual mode (public tenant user)
// Priority: Token quota -> Balance -> Stop (no credit for individuals)
func (s *QuotaService) checkIndividualQuota(ctx context.Context, quotaID uuid.UUID, tokens int64, cost decimal.Decimal) error {
	quota, err := s.userQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return fmt.Errorf("failed to get user quota: %w", err)
	}

	// 1. Status check
	if quota.Status != "active" {
		return apperrors.ErrUserSuspended
	}

	// 2. Token quota check (priority 1)
	if quota.TokenQuota > 0 && quota.TokensUsed+tokens <= quota.TokenQuota {
		// Token quota sufficient, pass
		return nil
	}

	// 3. Balance check (priority 2)
	if quota.Balance.GreaterThan(decimal.Zero) && quota.Balance.GreaterThanOrEqual(cost) {
		// Balance sufficient, pass
		return nil
	}

	// 4. Individual has no credit, insufficient balance stops service
	return apperrors.ErrInsufficientBalance
}

// checkMemberQuota checks quota for member mode (organization tenant)
// Two-layer check: Member layer + Tenant layer
func (s *QuotaService) checkMemberQuota(ctx context.Context, memberQuotaID, tenantID uuid.UUID, tokens int64, cost decimal.Decimal) error {
	// First layer: Member quota limit check
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, memberQuotaID)
	if err != nil {
		return fmt.Errorf("failed to get member quota: %w", err)
	}

	// 1. Member status check
	if memberQuota.Status != "active" {
		return apperrors.ErrMemberSuspended
	}

	// 2. Member token quota limit check (admin assigned)
	if memberQuota.TokenQuotaLimit != nil {
		if memberQuota.TokensUsed+tokens > *memberQuota.TokenQuotaLimit {
			return apperrors.ErrMemberTokenQuotaExceeded
		}
	}

	// 3. Member cost limit check
	if memberQuota.CostLimit != nil {
		if memberQuota.CostUsed.Add(cost).GreaterThan(*memberQuota.CostLimit) {
			return apperrors.ErrMemberCostLimitExceeded
		}
	}

	// Second layer: Tenant quota check (unified priority)
	return s.checkOrganizationQuota(ctx, tenantID, tokens, cost)
}

// checkOrganizationQuota checks quota for organization mode
// Priority: Token quota -> Balance -> Credit (approved only)
func (s *QuotaService) checkOrganizationQuota(ctx context.Context, tenantID uuid.UUID, tokens int64, cost decimal.Decimal) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// 1. Tenant status check
	if tenant.Status == "suspended" {
		return apperrors.ErrTenantSuspended
	}

	// 2. Token quota check (priority 1)
	if tenant.TokenLimit != nil && tenant.TokensUsedMonth+tokens <= *tenant.TokenLimit {
		// Token quota sufficient, pass
		return nil
	}

	// 3. Balance check (priority 2)
	if tenant.Balance.GreaterThan(decimal.Zero) && tenant.Balance.GreaterThanOrEqual(cost) {
		// Balance sufficient, pass
		return nil
	}

	// 4. Credit limit check (priority 3, requires approval)
	if tenant.CreditStatus == "approved" && tenant.CreditLimit != nil {
		if tenant.CreditUsed.Add(cost).LessThanOrEqual(*tenant.CreditLimit) {
			// Credit sufficient, pass
			return nil
		}
		return apperrors.ErrCreditLimitExceeded
	}

	// 5. No available payment source, stop service
	return apperrors.ErrNoPaymentSource
}

// DeductUsage deducts usage after request completed
// Priority: Token quota -> Balance -> Credit
func (s *QuotaService) DeductUsage(ctx context.Context, apiKey *entity.APIKey, tokens int64, cost decimal.Decimal) error {
	switch apiKey.QuotaType {
	case "individual":
		return s.deductIndividualUsage(ctx, apiKey.QuotaID, tokens, cost)
	case "member":
		return s.deductMemberUsage(ctx, apiKey.QuotaID, apiKey.TenantID, tokens, cost)
	default:
		return apperrors.ErrInvalidQuotaType
	}
}

// deductIndividualUsage deducts usage for individual mode
// Priority: Token quota -> Balance -> Stop (no credit)
func (s *QuotaService) deductIndividualUsage(ctx context.Context, quotaID uuid.UUID, tokens int64, cost decimal.Decimal) error {
	quota, err := s.userQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return fmt.Errorf("failed to get user quota: %w", err)
	}

	// 1. Deduct token quota first (priority 1)
	if quota.TokenQuota > 0 && quota.TokensUsed < quota.TokenQuota {
		remainingTokens := quota.TokenQuota - quota.TokensUsed
		if tokens <= remainingTokens {
			// Fully within quota
			quota.TokensUsed += tokens
			quota.MonthlyCost = quota.MonthlyCost.Add(cost)
			if err := s.userQuotaRepo.Update(ctx, quota); err != nil {
				return fmt.Errorf("failed to update user quota: %w", err)
			}
			return nil
		}
		// Partial within quota, remaining deduct from balance
		quota.TokensUsed = quota.TokenQuota // quota exhausted
		// Calculate partial cost for tokens outside quota
		// Note: In quota-first model, tokens outside quota cost from balance
		quota.Balance = quota.Balance.Sub(cost)
		quota.MonthlyCost = quota.MonthlyCost.Add(cost)
		if err := s.userQuotaRepo.Update(ctx, quota); err != nil {
			return fmt.Errorf("failed to update user quota: %w", err)
		}
		// Check if balance exhausted
		if quota.Balance.LessThanOrEqual(decimal.Zero) {
			quota.Status = "suspended"
			if err := s.userQuotaRepo.Update(ctx, quota); err != nil {
				return fmt.Errorf("failed to suspend user quota: %w", err)
			}
		}
		return nil
	}

	// 2. Quota exhausted or no quota, deduct from balance
	quota.Balance = quota.Balance.Sub(cost)
	quota.MonthlyCost = quota.MonthlyCost.Add(cost)
	if err := s.userQuotaRepo.Update(ctx, quota); err != nil {
		return fmt.Errorf("failed to update user quota: %w", err)
	}

	// Check if balance exhausted
	if quota.Balance.LessThanOrEqual(decimal.Zero) {
		quota.Status = "suspended"
		if err := s.userQuotaRepo.Update(ctx, quota); err != nil {
			return fmt.Errorf("failed to suspend user quota: %w", err)
		}
	}

	return nil
}

// deductMemberUsage deducts usage for member mode (two-layer)
func (s *QuotaService) deductMemberUsage(ctx context.Context, memberQuotaID, tenantID uuid.UUID, tokens int64, cost decimal.Decimal) error {
	// 1. Deduct member quota statistics
	if err := s.memberQuotaRepo.IncrementTokensUsed(ctx, memberQuotaID, tokens); err != nil {
		return fmt.Errorf("failed to increment member tokens: %w", err)
	}
	if err := s.memberQuotaRepo.IncrementCostUsed(ctx, memberQuotaID, cost); err != nil {
		return fmt.Errorf("failed to increment member cost: %w", err)
	}

	// 2. Deduct tenant quota (unified priority)
	return s.deductOrganizationUsage(ctx, tenantID, tokens, cost)
}

// deductOrganizationUsage deducts usage for organization mode
// Priority: Token quota -> Balance -> Credit
func (s *QuotaService) deductOrganizationUsage(ctx context.Context, tenantID uuid.UUID, tokens int64, cost decimal.Decimal) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// Deduct token statistics
	tenant.TokensUsedMonth += tokens
	tenant.BudgetUsedMonth = tenant.BudgetUsedMonth.Add(cost)

	// 1. Check if within token quota (priority 1)
	if tenant.TokenLimit != nil && tenant.TokensUsedMonth <= *tenant.TokenLimit {
		// Within quota, no cost deduction
		if err := s.tenantRepo.Update(ctx, tenant); err != nil {
			return fmt.Errorf("failed to update tenant: %w", err)
		}
		return nil
	}

	// 2. Quota exhausted or no quota, deduct from balance (priority 2)
	if tenant.Balance.GreaterThan(decimal.Zero) {
		if tenant.Balance.GreaterThanOrEqual(cost) {
			tenant.Balance = tenant.Balance.Sub(cost)
			if err := s.tenantRepo.Update(ctx, tenant); err != nil {
				return fmt.Errorf("failed to update tenant: %w", err)
			}
			return nil
		}
		// Balance insufficient, partial from balance, remaining from credit
		tenant.Balance = decimal.Zero
		remainingCost := cost.Sub(tenant.Balance)

		// 3. Deduct from credit (requires approval, priority 3)
		if tenant.CreditStatus == "approved" && tenant.CreditLimit != nil {
			tenant.CreditUsed = tenant.CreditUsed.Add(remainingCost)
			if err := s.tenantRepo.Update(ctx, tenant); err != nil {
				return fmt.Errorf("failed to update tenant credit: %w", err)
			}
			return nil
		}

		// No credit, suspend tenant
		tenant.Status = "suspended"
		if err := s.tenantRepo.Update(ctx, tenant); err != nil {
			return fmt.Errorf("failed to suspend tenant: %w", err)
		}
		return apperrors.ErrNoPaymentSource
	}

	// 5. Balance is zero, directly deduct from credit
	if tenant.CreditStatus == "approved" && tenant.CreditLimit != nil {
		tenant.CreditUsed = tenant.CreditUsed.Add(cost)
		if err := s.tenantRepo.Update(ctx, tenant); err != nil {
			return fmt.Errorf("failed to update tenant credit: %w", err)
		}
		return nil
	}

	// No available payment source, suspend tenant
	tenant.Status = "suspended"
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to suspend tenant: %w", err)
	}
	return apperrors.ErrNoPaymentSource
}

// GetQuotaInfo returns quota information for an API key
func (s *QuotaService) GetQuotaInfo(ctx context.Context, apiKey *entity.APIKey) (*QuotaInfo, error) {
	switch apiKey.QuotaType {
	case "individual":
		return s.getIndividualQuotaInfo(ctx, apiKey.QuotaID)
	case "member":
		return s.getMemberQuotaInfo(ctx, apiKey.QuotaID, apiKey.TenantID)
	default:
		return nil, apperrors.ErrInvalidQuotaType
	}
}

// QuotaInfo contains quota status information
type QuotaInfo struct {
	QuotaType       string          // individual or member
	Status          string          // active, suspended, etc.
	TokenQuota      int64           // total token quota
	TokensUsed      int64           // tokens used
	TokenRemaining  int64           // tokens remaining
	Balance         decimal.Decimal // current balance
	CreditLimit     *decimal.Decimal // credit limit (org only)
	CreditUsed      decimal.Decimal // credit used (org only)
	CreditStatus    string          // credit approval status (org only)
	CostLimit       *decimal.Decimal // member cost limit
	CostUsed        decimal.Decimal // member cost used
}

func (s *QuotaService) getIndividualQuotaInfo(ctx context.Context, quotaID uuid.UUID) (*QuotaInfo, error) {
	quota, err := s.userQuotaRepo.GetByID(ctx, quotaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user quota: %w", err)
	}

	remaining := quota.TokenQuota - quota.TokensUsed
	if remaining < 0 {
		remaining = 0
	}

	return &QuotaInfo{
		QuotaType:      "individual",
		Status:         quota.Status,
		TokenQuota:     quota.TokenQuota,
		TokensUsed:     quota.TokensUsed,
		TokenRemaining: remaining,
		Balance:        quota.Balance,
		// Individual has no credit
	}, nil
}

func (s *QuotaService) getMemberQuotaInfo(ctx context.Context, memberQuotaID, tenantID uuid.UUID) (*QuotaInfo, error) {
	memberQuota, err := s.memberQuotaRepo.GetByID(ctx, memberQuotaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member quota: %w", err)
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	var memberTokenRemaining int64
	if memberQuota.TokenQuotaLimit != nil {
		memberTokenRemaining = *memberQuota.TokenQuotaLimit - memberQuota.TokensUsed
		if memberTokenRemaining < 0 {
			memberTokenRemaining = 0
		}
	} else {
		// No member limit, use tenant limit
		if tenant.TokenLimit != nil {
			memberTokenRemaining = *tenant.TokenLimit - tenant.TokensUsedMonth
			if memberTokenRemaining < 0 {
				memberTokenRemaining = 0
			}
		}
	}

	tenantTokenQuota := int64(0)
	if tenant.TokenLimit != nil {
		tenantTokenQuota = *tenant.TokenLimit
	}

	return &QuotaInfo{
		QuotaType:      "member",
		Status:         memberQuota.Status,
		TokenQuota:     tenantTokenQuota,
		TokensUsed:     tenant.TokensUsedMonth,
		TokenRemaining: memberTokenRemaining,
		Balance:        tenant.Balance,
		CreditLimit:    tenant.CreditLimit,
		CreditUsed:     tenant.CreditUsed,
		CreditStatus:   tenant.CreditStatus,
		CostLimit:      memberQuota.CostLimit,
		CostUsed:       memberQuota.CostUsed,
	}, nil
}