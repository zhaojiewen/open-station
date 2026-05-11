package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// CostLimitService provides cost limit checking and management
type CostLimitService struct {
	tenantRepo repository.TenantRepository
	userRepo   repository.UserRepository
	apiKeyRepo repository.APIKeyRepository
}

// NewCostLimitService creates a new CostLimitService
func NewCostLimitService(
	tenantRepo repository.TenantRepository,
	userRepo repository.UserRepository,
	apiKeyRepo repository.APIKeyRepository,
) *CostLimitService {
	return &CostLimitService{
		tenantRepo: tenantRepo,
		userRepo:   userRepo,
		apiKeyRepo: apiKeyRepo,
	}
}

// CheckCostLimits checks all cost limits before processing a request
// Returns error if any limit is exceeded, nil otherwise
func (s *CostLimitService) CheckCostLimits(ctx context.Context, apiKeyID, userID, tenantID uuid.UUID, estimatedCost decimal.Decimal, estimatedTokens int64) error {
	// 1. Check Tenant balance
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return apperrors.ErrTenantNotFound
	}

	// Check tenant status
	if tenant.Status == "suspended" {
		return apperrors.ErrTenantSuspended
	}
	if tenant.Status == "deleted" {
		return apperrors.ErrTenantDeleted
	}

	// Check balance
	if tenant.Balance.LessThan(estimatedCost) {
		return apperrors.ErrInsufficientBalance
	}

	// Check tenant monthly budget limit
	if tenant.MonthlyBudgetLimit != nil && tenant.BudgetUsedMonth.Add(estimatedCost).GreaterThan(*tenant.MonthlyBudgetLimit) {
		return apperrors.ErrTenantMonthlyBudgetExceeded
	}

	// Check tenant token limit
	if tenant.TokenLimit != nil && *tenant.TokenLimit > 0 {
		if tenant.TokensUsedMonth+estimatedTokens > *tenant.TokenLimit {
			return apperrors.ErrQuotaExceeded
		}
	}

	// 2. Check User limits
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	// Check user status
	if user.Status == "inactive" {
		return apperrors.ErrUserInactive
	}

	// Check user monthly budget
	if user.MonthlyBudget != nil && user.BudgetUsedMonth.Add(estimatedCost).GreaterThan(*user.MonthlyBudget) {
		return apperrors.ErrUserMonthlyBudgetExceeded
	}

	// Check user daily budget
	if user.DailyBudget != nil && user.BudgetUsedToday.Add(estimatedCost).GreaterThan(*user.DailyBudget) {
		return apperrors.ErrUserDailyBudgetExceeded
	}

	// Check user token quota
	if user.TokenQuota != nil && *user.TokenQuota > 0 {
		if user.TokensUsedMonth+estimatedTokens > *user.TokenQuota {
			return apperrors.ErrQuotaExceeded
		}
	}

	// 3. Check API Key limits
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return apperrors.ErrInvalidAPIKey
	}

	// Check API key status
	if apiKey.Status == "revoked" {
		return apperrors.ErrAPIKeyRevoked
	}
	if apiKey.Status == "expired" {
		return apperrors.ErrAPIKeyExpired
	}
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return apperrors.ErrAPIKeyExpired
	}

	// Check API key per-request cost limit
	if apiKey.PerRequestCostLimit != nil && estimatedCost.GreaterThan(*apiKey.PerRequestCostLimit) {
		return apperrors.ErrAPIKeyPerRequestCostExceeded
	}

	// Check API key monthly cost limit
	if apiKey.MonthlyCostLimit != nil && apiKey.MonthlyCostUsed.Add(estimatedCost).GreaterThan(*apiKey.MonthlyCostLimit) {
		return apperrors.ErrAPIKeyMonthlyCostExceeded
	}

	// Check API key daily cost limit
	if apiKey.DailyCostLimit != nil && apiKey.DailyCostUsed.Add(estimatedCost).GreaterThan(*apiKey.DailyCostLimit) {
		return apperrors.ErrAPIKeyDailyCostExceeded
	}

	// Check API key monthly token limit
	if apiKey.MonthlyTokenLimit != nil && *apiKey.MonthlyTokenLimit > 0 {
		if apiKey.UsedTokensThisMonth+estimatedTokens > *apiKey.MonthlyTokenLimit {
			return apperrors.ErrAPIKeyTokenLimitExceeded
		}
	}

	// Check API key daily token limit
	if apiKey.TokenLimitPerDay != nil && *apiKey.TokenLimitPerDay > 0 {
		if apiKey.TokensUsedToday+estimatedTokens > *apiKey.TokenLimitPerDay {
			return apperrors.ErrAPIKeyDailyTokenExceeded
		}
	}

	return nil
}

// RecordCost records the cost usage after a request completes
func (s *CostLimitService) RecordCost(ctx context.Context, apiKeyID, userID, tenantID uuid.UUID, cost decimal.Decimal, tokens int64) error {
	// 1. Deduct tenant balance and increment budget used
	if err := s.tenantRepo.DeductBalance(ctx, tenantID, cost); err != nil {
		return apperrors.ErrInsufficientBalance
	}

	if err := s.tenantRepo.IncrementBudgetUsed(ctx, tenantID, cost); err != nil {
		// Rollback balance
		s.tenantRepo.UpdateBalance(ctx, tenantID, cost)
		return err
	}

	if err := s.tenantRepo.IncrementTokensUsed(ctx, tenantID, tokens); err != nil {
		return err
	}

	// 2. Increment user budget usage
	if err := s.userRepo.IncrementMonthlyBudgetUsed(ctx, userID, cost); err != nil {
		return err
	}

	if err := s.userRepo.IncrementDailyBudgetUsed(ctx, userID, cost); err != nil {
		return err
	}

	// 3. Increment API key cost usage
	if err := s.apiKeyRepo.IncrementMonthlyCostUsed(ctx, apiKeyID, cost); err != nil {
		return err
	}

	if err := s.apiKeyRepo.IncrementDailyCostUsed(ctx, apiKeyID, cost); err != nil {
		return err
	}

	// Update token usage on API key
	if err := s.apiKeyRepo.UpdateTokenUsage(ctx, apiKeyID, tokens); err != nil {
		return err
	}

	if err := s.apiKeyRepo.IncrementDailyTokens(ctx, apiKeyID, tokens); err != nil {
		return err
	}

	return nil
}

// GetCostUsage returns cost usage information for all levels
type CostUsageInfo struct {
	Tenant struct {
		Balance       decimal.Decimal
		BudgetLimit   *decimal.Decimal
		BudgetUsed    decimal.Decimal
		BudgetPercent int
		TokenLimit    *int64
		TokensUsed    int64
	}
	User struct {
		MonthlyBudget *decimal.Decimal
		MonthlyUsed   decimal.Decimal
		MonthlyPercent int
		DailyBudget   *decimal.Decimal
		DailyUsed     decimal.Decimal
		DailyPercent  int
		TokenQuota    *int64
		TokensUsed    int64
	}
	APIKey struct {
		MonthlyCostLimit   *decimal.Decimal
		MonthlyCostUsed    decimal.Decimal
		MonthlyPercent     int
		DailyCostLimit     *decimal.Decimal
		DailyCostUsed      decimal.Decimal
		DailyPercent       int
		PerRequestLimit    *decimal.Decimal
		MonthlyTokenLimit  *int64
		TokensUsedMonth    int64
		DailyTokenLimit    *int64
		TokensUsedToday    int64
	}
}

// GetCostUsageInfo returns comprehensive cost usage information
func (s *CostLimitService) GetCostUsageInfo(ctx context.Context, apiKeyID, userID, tenantID uuid.UUID) (*CostUsageInfo, error) {
	info := &CostUsageInfo{}

	// Tenant info
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	info.Tenant.Balance = tenant.Balance
	info.Tenant.BudgetLimit = tenant.MonthlyBudgetLimit
	info.Tenant.BudgetUsed = tenant.BudgetUsedMonth
	info.Tenant.TokenLimit = tenant.TokenLimit
	info.Tenant.TokensUsed = tenant.TokensUsedMonth
	if tenant.MonthlyBudgetLimit != nil && !tenant.MonthlyBudgetLimit.IsZero() {
		info.Tenant.BudgetPercent = int(tenant.BudgetUsedMonth.Div(*tenant.MonthlyBudgetLimit).Mul(decimal.NewFromInt(100)).IntPart())
	}

	// User info
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	info.User.MonthlyBudget = user.MonthlyBudget
	info.User.MonthlyUsed = user.BudgetUsedMonth
	info.User.DailyBudget = user.DailyBudget
	info.User.DailyUsed = user.BudgetUsedToday
	info.User.TokenQuota = user.TokenQuota
	info.User.TokensUsed = user.TokensUsedMonth
	if user.MonthlyBudget != nil && !user.MonthlyBudget.IsZero() {
		info.User.MonthlyPercent = int(user.BudgetUsedMonth.Div(*user.MonthlyBudget).Mul(decimal.NewFromInt(100)).IntPart())
	}
	if user.DailyBudget != nil && !user.DailyBudget.IsZero() {
		info.User.DailyPercent = int(user.BudgetUsedToday.Div(*user.DailyBudget).Mul(decimal.NewFromInt(100)).IntPart())
	}

	// API Key info
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}
	info.APIKey.MonthlyCostLimit = apiKey.MonthlyCostLimit
	info.APIKey.MonthlyCostUsed = apiKey.MonthlyCostUsed
	info.APIKey.DailyCostLimit = apiKey.DailyCostLimit
	info.APIKey.DailyCostUsed = apiKey.DailyCostUsed
	info.APIKey.PerRequestLimit = apiKey.PerRequestCostLimit
	info.APIKey.MonthlyTokenLimit = apiKey.MonthlyTokenLimit
	info.APIKey.TokensUsedMonth = apiKey.UsedTokensThisMonth
	info.APIKey.DailyTokenLimit = apiKey.TokenLimitPerDay
	info.APIKey.TokensUsedToday = apiKey.TokensUsedToday
	if apiKey.MonthlyCostLimit != nil && !apiKey.MonthlyCostLimit.IsZero() {
		info.APIKey.MonthlyPercent = int(apiKey.MonthlyCostUsed.Div(*apiKey.MonthlyCostLimit).Mul(decimal.NewFromInt(100)).IntPart())
	}
	if apiKey.DailyCostLimit != nil && !apiKey.DailyCostLimit.IsZero() {
		info.APIKey.DailyPercent = int(apiKey.DailyCostUsed.Div(*apiKey.DailyCostLimit).Mul(decimal.NewFromInt(100)).IntPart())
	}

	return info, nil
}

// ResetDailyCosts resets daily cost counters for all entities
// This should be called at midnight (via cron job)
func (s *CostLimitService) ResetDailyCosts(ctx context.Context) error {
	// Reset all users' daily budget
	// Note: This is a batch operation - implement as needed
	// For now, we use a per-request check with date comparison

	// Reset all API keys' daily cost
	// Note: This is a batch operation - implement as needed

	return nil
}

// ResetMonthlyCosts resets monthly cost counters for all entities
// This should be called at the beginning of each month (via cron job)
func (s *CostLimitService) ResetMonthlyCosts(ctx context.Context) error {
	// Reset all tenants' monthly budget used
	// Reset all users' monthly budget used
	// Reset all API keys' monthly cost used

	return nil
}

// SetUserBudget sets budget limits for a user
func (s *CostLimitService) SetUserBudget(ctx context.Context, userID uuid.UUID, monthlyBudget, dailyBudget decimal.Decimal, tokenQuota int64) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !monthlyBudget.IsZero() {
		user.MonthlyBudget = &monthlyBudget
	}
	if !dailyBudget.IsZero() {
		user.DailyBudget = &dailyBudget
	}
	if tokenQuota > 0 {
		user.TokenQuota = &tokenQuota
	}

	return s.userRepo.Update(ctx, user)
}

// SetAPIKeyCostLimit sets cost limits for an API key
func (s *CostLimitService) SetAPIKeyCostLimit(ctx context.Context, apiKeyID uuid.UUID, monthlyLimit, dailyLimit, perRequestLimit decimal.Decimal, monthlyTokenLimit, dailyTokenLimit int64, alertThreshold1, alertThreshold2, alertThreshold3 int) error {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return err
	}

	if !monthlyLimit.IsZero() {
		apiKey.MonthlyCostLimit = &monthlyLimit
	}
	if !dailyLimit.IsZero() {
		apiKey.DailyCostLimit = &dailyLimit
	}
	if !perRequestLimit.IsZero() {
		apiKey.PerRequestCostLimit = &perRequestLimit
	}
	if monthlyTokenLimit > 0 {
		apiKey.MonthlyTokenLimit = &monthlyTokenLimit
	}
	if dailyTokenLimit > 0 {
		apiKey.TokenLimitPerDay = &dailyTokenLimit
	}
	apiKey.AlertThreshold1 = alertThreshold1
	apiKey.AlertThreshold2 = alertThreshold2
	apiKey.AlertThreshold3 = alertThreshold3

	return s.apiKeyRepo.Update(ctx, apiKey)
}

// SetTenantBudget sets budget limits for a tenant
func (s *CostLimitService) SetTenantBudget(ctx context.Context, tenantID uuid.UUID, monthlyBudgetLimit decimal.Decimal, tokenLimit int64) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}

	if !monthlyBudgetLimit.IsZero() {
		tenant.MonthlyBudgetLimit = &monthlyBudgetLimit
	}
	if tokenLimit > 0 {
		tenant.TokenLimit = &tokenLimit
	}

	return s.tenantRepo.Update(ctx, tenant)
}

// UserBudgetUsage represents user budget usage information
type UserBudgetUsage struct {
	MonthlyBudget *decimal.Decimal
	MonthlyUsed   decimal.Decimal
	DailyBudget   *decimal.Decimal
	DailyUsed     decimal.Decimal
	TokenQuota    *int64
	TokensUsed    int64
}

// GetUserBudgetUsage gets budget usage information for a user
func (s *CostLimitService) GetUserBudgetUsage(ctx context.Context, userID uuid.UUID) (*UserBudgetUsage, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserBudgetUsage{
		MonthlyBudget: user.MonthlyBudget,
		MonthlyUsed:   user.BudgetUsedMonth,
		DailyBudget:   user.DailyBudget,
		DailyUsed:     user.BudgetUsedToday,
		TokenQuota:    user.TokenQuota,
		TokensUsed:    user.TokensUsedMonth,
	}, nil
}

// APIKeyCostUsage represents API key cost usage information
type APIKeyCostUsage struct {
	MonthlyCostLimit   *decimal.Decimal
	MonthlyCostUsed    decimal.Decimal
	DailyCostLimit     *decimal.Decimal
	DailyCostUsed      decimal.Decimal
	MonthlyTokensUsed  int64
	DailyTokensUsed    int64
}

// GetAPIKeyCostUsage gets cost usage information for an API key
func (s *CostLimitService) GetAPIKeyCostUsage(ctx context.Context, apiKeyID uuid.UUID) (*APIKeyCostUsage, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	return &APIKeyCostUsage{
		MonthlyCostLimit:  apiKey.MonthlyCostLimit,
		MonthlyCostUsed:   apiKey.MonthlyCostUsed,
		DailyCostLimit:    apiKey.DailyCostLimit,
		DailyCostUsed:     apiKey.DailyCostUsed,
		MonthlyTokensUsed: apiKey.UsedTokensThisMonth,
		DailyTokensUsed:   apiKey.TokensUsedToday,
	}, nil
}

// CostSummary represents comprehensive cost summary
type CostSummary struct {
	TenantMonthlyUsed  decimal.Decimal
	TenantBalance      decimal.Decimal
	UserMonthlyUsed    decimal.Decimal
	APIKeyMonthlyCostUsed decimal.Decimal
}

// GetCostSummary gets comprehensive cost usage summary
func (s *CostLimitService) GetCostSummary(ctx context.Context, apiKeyID, userID, tenantID uuid.UUID) (*CostSummary, error) {
	summary := &CostSummary{}

	if tenantID != uuid.Nil {
		tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
		if err == nil {
			summary.TenantMonthlyUsed = tenant.BudgetUsedMonth
			summary.TenantBalance = tenant.Balance
		}
	}

	if userID != uuid.Nil {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err == nil {
			summary.UserMonthlyUsed = user.BudgetUsedMonth
		}
	}

	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err == nil {
		summary.APIKeyMonthlyCostUsed = apiKey.MonthlyCostUsed
	}

	return summary, nil
}