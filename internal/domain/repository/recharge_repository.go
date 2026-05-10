package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// RechargeRepository defines operations for recharge/payment tracking
type RechargeRepository interface {
	Create(ctx context.Context, record *entity.RechargeRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.RechargeRecord, error)
	Update(ctx context.Context, record *entity.RechargeRecord) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.RechargeRecord, int64, error)
	MarkCompleted(ctx context.Context, id uuid.UUID) error
}