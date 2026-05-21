package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// APIKeyRepository defines operations for API key management
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *entity.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.APIKey, error)
	GetByHash(ctx context.Context, keyHash string) (*entity.APIKey, error)
	GetWithRelations(ctx context.Context, keyHash string) (*entity.APIKey, *entity.User, *entity.Tenant, error)
	GetByKeyPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error)
	Update(ctx context.Context, apiKey *entity.APIKey) error
	Delete(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID) ([]entity.APIKey, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.APIKey, error)
	ListAll(ctx context.Context) ([]entity.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	UpdateTokenUsage(ctx context.Context, id uuid.UUID, tokens int64) error

	// 费用限制相关 (新增)
	IncrementMonthlyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	IncrementDailyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	ResetMonthlyCostUsed(ctx context.Context, id uuid.UUID) error
	ResetDailyCostUsed(ctx context.Context, id uuid.UUID) error
	GetCostUsage(ctx context.Context, id uuid.UUID) (monthlyUsed decimal.Decimal, dailyUsed decimal.Decimal, tokensMonth int64, tokensToday int64, err error)
	IncrementDailyTokens(ctx context.Context, id uuid.UUID, tokens int64) error
	ResetDailyTokens(ctx context.Context, id uuid.UUID) error

	// Per-provider usage tracking
	UpdateProviderUsage(ctx context.Context, id uuid.UUID, provider string, tokens int64, cost decimal.Decimal) error
}
