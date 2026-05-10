package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// BillRepository defines operations for billing management
type BillRepository interface {
	Create(ctx context.Context, bill *entity.Bill) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Bill, error)
	GetByBillNumber(ctx context.Context, billNumber string) (*entity.Bill, error)
	Update(ctx context.Context, bill *entity.Bill) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.Bill, int64, error)
	GetByPeriod(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*entity.Bill, error)
	MarkPaid(ctx context.Context, id uuid.UUID) error
}