package repository

import (
		"context"

		"github.com/google/uuid"
		"github.com/shopspring/decimal"
		"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// UserRepository defines operations for user management
type UserRepository interface {
		Create(ctx context.Context, user *entity.User) error
		GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
		GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByVerificationToken(ctx context.Context, token string) (*entity.User, error)
		Update(ctx context.Context, user *entity.User) error
		Delete(ctx context.Context, id uuid.UUID) error
		List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error)
		UpdateLastLogin(ctx context.Context, id uuid.UUID) error

		// 费用限制相关 (新增)
		IncrementMonthlyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
		IncrementDailyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
		ResetMonthlyBudgetUsed(ctx context.Context, id uuid.UUID) error
		ResetDailyBudgetUsed(ctx context.Context, id uuid.UUID) error
		GetBudgetUsage(ctx context.Context, id uuid.UUID) (monthlyUsed decimal.Decimal, dailyUsed decimal.Decimal, tokensUsed int64, err error)
		IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error
		IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error
		DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error
}