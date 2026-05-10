package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// ProviderAccountRepository defines operations for multi-account provider management
// Supports failover, priority ordering, and usage tracking
type ProviderAccountRepository interface {
	Create(ctx context.Context, account *entity.ProviderAccount) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.ProviderAccount, error)
	GetByProvider(ctx context.Context, provider string) ([]entity.ProviderAccount, error)
	GetActiveByProvider(ctx context.Context, provider string) ([]entity.ProviderAccount, error)
	GetDefaultByProvider(ctx context.Context, provider string) (*entity.ProviderAccount, error)
	GetNextAvailable(ctx context.Context, provider string, excludeID uuid.UUID) (*entity.ProviderAccount, error)
	Update(ctx context.Context, account *entity.ProviderAccount) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]entity.ProviderAccount, int64, error)
	ListByStatus(ctx context.Context, status string) ([]entity.ProviderAccount, error)
	SetDefault(ctx context.Context, provider string, id uuid.UUID) error
	IncrementUsage(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error
	RecordError(ctx context.Context, id uuid.UUID, errMsg string) error
	RecordSuccess(ctx context.Context, id uuid.UUID) error
	ResetMonthlyUsage(ctx context.Context) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}