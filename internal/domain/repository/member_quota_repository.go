package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// MemberQuotaRepository defines operations for member quota management
type MemberQuotaRepository interface {
	Create(ctx context.Context, quota *entity.MemberQuota) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.MemberQuota, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.MemberQuota, error)
	GetByTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) (*entity.MemberQuota, error)
	Update(ctx context.Context, quota *entity.MemberQuota) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]entity.MemberQuota, int64, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.MemberQuota, int64, error)

	// 配额操作
	IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error
	ResetTokensUsed(ctx context.Context, id uuid.UUID) error
	GetTokenUsage(ctx context.Context, id uuid.UUID) (used int64, limit int64, err error)

	// 费用操作
	IncrementCostUsed(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error
	ResetCostUsed(ctx context.Context, id uuid.UUID) error
	GetCostUsage(ctx context.Context, id uuid.UUID) (used decimal.Decimal, limit decimal.Decimal, err error)

	// 统计
	GetTotalTokensUsedByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	GetTotalCostUsedByTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// 状态管理
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	SetExceeded(ctx context.Context, id uuid.UUID, reason string) error

	// API Key计数
	IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error
	DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error
	GetActiveAPIKeysCount(ctx context.Context, id uuid.UUID) (int, error)
}