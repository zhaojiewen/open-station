package repository

import (
		"context"

		"github.com/google/uuid"
		"github.com/shopspring/decimal"
		"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// TenantRepository defines operations for tenant management
type TenantRepository interface {
		Create(ctx context.Context, tenant *entity.Tenant) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error)
		Update(ctx context.Context, tenant *entity.Tenant) error
		Delete(ctx context.Context, id uuid.UUID) error
		List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error)
		ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error)
		UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
		GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error)
		DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error

		// 费用限制相关 (新增)
		IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
		ResetBudgetUsed(ctx context.Context, id uuid.UUID) error
		GetBudgetUsage(ctx context.Context, id uuid.UUID) (monthlyUsed decimal.Decimal, tokensUsed int64, err error)
		IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error
		ResetTokensUsed(ctx context.Context, id uuid.UUID) error
}