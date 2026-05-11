package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// UserQuotaRepository defines operations for user quota management
type UserQuotaRepository interface {
	Create(ctx context.Context, quota *entity.UserQuota) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.UserQuota, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.UserQuota, error)
	Update(ctx context.Context, quota *entity.UserQuota) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]entity.UserQuota, int64, error)

	// 配额操作
	IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error
	ResetTokensUsed(ctx context.Context, id uuid.UUID) error
	GetTokenUsage(ctx context.Context, id uuid.UUID) (used int64, quota int64, err error)

	// 余额操作
	GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error)
	DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	AddBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error

	// 月度统计
	IncrementMonthlyCost(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error
	ResetMonthlyCost(ctx context.Context, id uuid.UUID) error
	GetMonthlyCost(ctx context.Context, id uuid.UUID) (decimal.Decimal, error)

	// 状态管理
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	GetStatus(ctx context.Context, id uuid.UUID) (string, error)
}