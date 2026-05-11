package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// PaymentOrderRepository defines operations for payment order management
type PaymentOrderRepository interface {
	Create(ctx context.Context, order *entity.PaymentOrder) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.PaymentOrder, error)
	GetByOrderNumber(ctx context.Context, orderNumber string) (*entity.PaymentOrder, error)
	GetByPaymentID(ctx context.Context, paymentID string) (*entity.PaymentOrder, error)
	Update(ctx context.Context, order *entity.PaymentOrder) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]entity.PaymentOrder, int64, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error)
	ListByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error)
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.PaymentOrder, int64, error)
	ListPendingByUser(ctx context.Context, userID uuid.UUID) ([]entity.PaymentOrder, error)
	ListPendingByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.PaymentOrder, error)

	// 状态更新
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	MarkPaid(ctx context.Context, id uuid.UUID, paymentID string, callbackData string) error
	MarkFailed(ctx context.Context, id uuid.UUID) error
	MarkCancelled(ctx context.Context, id uuid.UUID) error
	MarkExpired(ctx context.Context) (int, error)

	// 金额操作
	GetTotalAmountByUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
	GetTotalAmountByTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error)

	// 订单生成
	GenerateOrderNumber() string
}