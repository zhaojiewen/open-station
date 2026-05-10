package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// UsageRepository defines operations for usage tracking
type UsageRepository interface {
	Create(ctx context.Context, record *entity.UsageRecord) error
	CreateBatch(ctx context.Context, records []*entity.UsageRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.UsageRecord, error)
	GetByRequestID(ctx context.Context, requestID string) (*entity.UsageRecord, error)
	List(ctx context.Context, tenantID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error)
	ListByUser(ctx context.Context, userID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error)
	GetTotalCost(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (decimal.Decimal, int64, error)
}