package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// CreditApplicationRepository defines operations for credit application management
type CreditApplicationRepository interface {
	Create(ctx context.Context, application *entity.CreditApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.CreditApplication, error)
	GetLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.CreditApplication, error)
	Update(ctx context.Context, application *entity.CreditApplication) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]entity.CreditApplication, int64, error)
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.CreditApplication, int64, error)

	// 审核操作
	Approve(ctx context.Context, id uuid.UUID, approvedLimit decimal.Decimal, reviewedBy uuid.UUID, reviewNotes string) error
	Reject(ctx context.Context, id uuid.UUID, reviewedBy uuid.UUID, reviewNotes string) error

	// 状态查询
	GetPendingCount(ctx context.Context) (int64, error)
}