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
	UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
	GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error)
	DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error
}