package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
)

// PaymentService handles payment order management
type PaymentService struct {
	paymentOrderRepo repository.PaymentOrderRepository
	userQuotaRepo    repository.UserQuotaRepository
	tenantRepo       repository.TenantRepository
	notificationSvc  *NotificationService
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	paymentOrderRepo repository.PaymentOrderRepository,
	userQuotaRepo repository.UserQuotaRepository,
	tenantRepo repository.TenantRepository,
	notificationSvc *NotificationService,
) *PaymentService {
	return &PaymentService{
		paymentOrderRepo: paymentOrderRepo,
		userQuotaRepo:    userQuotaRepo,
		tenantRepo:       tenantRepo,
		notificationSvc:  notificationSvc,
	}
}

// CreatePaymentOrder creates a new payment order
func (s *PaymentService) CreatePaymentOrder(ctx context.Context, req *PaymentOrderRequest) (*entity.PaymentOrder, error) {
	// Generate order number
	orderNumber := generateOrderNumber()

	order := &entity.PaymentOrder{
		PaymentMode:     req.PaymentMode,
		UserID:          req.UserID,
		UserQuotaID:     req.UserQuotaID,
		TenantID:        req.TenantID,
		OrderNumber:     orderNumber,
		OrderType:       req.OrderType,
		Amount:          req.Amount,
		Currency:        req.Currency,
		PaymentProvider: req.PaymentProvider,
		PaymentMethod:   req.PaymentMethod,
		Status:          "pending",
	}

	// Set expiration time (30 minutes for most payment methods)
	expireAt := time.Now().Add(30 * time.Minute)
	if req.PaymentProvider == "bank" {
		// Bank transfer: 24 hours
		expireAt = time.Now().Add(24 * time.Hour)
	}
	order.ExpireAt = &expireAt

	if err := s.paymentOrderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create payment order: %w", err)
	}

	return order, nil
}

// PaymentOrderRequest represents the payment order creation request
type PaymentOrderRequest struct {
	PaymentMode     string          // individual, organization
	UserID          uuid.UUID
	UserQuotaID     *uuid.UUID      // for individual mode
	TenantID        *uuid.UUID      // for organization mode
	OrderType       string          // recharge, subscription, credit_settlement
	Amount          decimal.Decimal
	Currency        string
	PaymentProvider string          // alipay, wechat, stripe, paypal, bank
	PaymentMethod   string          // qr_code, web, app, bank_transfer
}

// generateOrderNumber generates a unique order number
func generateOrderNumber() string {
	return fmt.Sprintf("PAY-%d-%s", time.Now().UnixNano()/1000000, uuid.New().String()[:8])
}

// GetPaymentOrder retrieves a payment order by ID
func (s *PaymentService) GetPaymentOrder(ctx context.Context, orderID uuid.UUID) (*entity.PaymentOrder, error) {
	return s.paymentOrderRepo.GetByID(ctx, orderID)
}

// GetPaymentOrderByNumber retrieves a payment order by order number
func (s *PaymentService) GetPaymentOrderByNumber(ctx context.Context, orderNumber string) (*entity.PaymentOrder, error) {
	return s.paymentOrderRepo.GetByOrderNumber(ctx, orderNumber)
}

// CancelPaymentOrder cancels a pending payment order
func (s *PaymentService) CancelPaymentOrder(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.paymentOrderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != "pending" {
		return fmt.Errorf("order already processed")
	}

	order.Status = "cancelled"
	if err := s.paymentOrderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

// ProcessPaymentCallback processes payment callback from payment providers
func (s *PaymentService) ProcessPaymentCallback(ctx context.Context, req *PaymentCallbackRequest) (*entity.PaymentOrder, error) {
	// Get order by payment ID or order number
	order, err := s.paymentOrderRepo.GetByPaymentID(ctx, req.PaymentID)
	if err != nil {
		// Try by order number if payment ID not found
		order, err = s.paymentOrderRepo.GetByOrderNumber(ctx, req.OrderNumber)
		if err != nil {
			return nil, fmt.Errorf("order not found: %w", err)
		}
	}

	// Validate order status
	if order.Status != "pending" {
		return nil, fmt.Errorf("order already processed")
	}

	// Validate amount (allow small tolerance for currency conversion)
	if req.PaidAmount.LessThan(order.Amount.Mul(decimal.NewFromFloat(0.99))) {
		return nil, fmt.Errorf("paid amount less than order amount")
	}

	// Update order status
	now := time.Now()
	order.Status = "paid"
	order.PaidAt = &now
	order.PaymentID = req.PaymentID
	order.CallbackData = req.CallbackData
	order.CallbackAt = &now

	if err := s.paymentOrderRepo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	// Process payment based on mode and type
	if err := s.processPayment(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}

	// Send notification
	if s.notificationSvc != nil {
		// TODO: Send payment success notification
	}

	return order, nil
}

// PaymentCallbackRequest represents payment callback data
type PaymentCallbackRequest struct {
	OrderNumber   string
	PaymentID     string
	PaidAmount    decimal.Decimal
	CallbackData  string
	PaymentStatus string // success, failed
}

// processPayment handles the actual payment processing after callback
func (s *PaymentService) processPayment(ctx context.Context, order *entity.PaymentOrder) error {
	switch order.PaymentMode {
	case "individual":
		return s.processIndividualPayment(ctx, order)
	case "organization":
		return s.processOrganizationPayment(ctx, order)
	default:
		return fmt.Errorf("invalid payment mode")
	}
}

// processIndividualPayment handles payment for individual users
func (s *PaymentService) processIndividualPayment(ctx context.Context, order *entity.PaymentOrder) error {
	if order.UserQuotaID == nil {
		return fmt.Errorf("user quota ID required for individual payment")
	}

	quota, err := s.userQuotaRepo.GetByID(ctx, *order.UserQuotaID)
	if err != nil {
		return fmt.Errorf("failed to get user quota: %w", err)
	}

	switch order.OrderType {
	case "recharge":
		// Add balance to user quota
		quota.Balance = quota.Balance.Add(order.Amount)
		if quota.Status == "pending_payment" {
			quota.Status = "active"
		}
		quota.LastPaymentAt = &time.Time{}
		*quota.LastPaymentAt = time.Now()
		if err := s.userQuotaRepo.Update(ctx, quota); err != nil {
			return fmt.Errorf("failed to update user quota: %w", err)
		}
	case "subscription":
		// TODO: Handle subscription payment
	default:
		return fmt.Errorf("unsupported order type for individual")
	}

	return nil
}

// processOrganizationPayment handles payment for organization tenants
func (s *PaymentService) processOrganizationPayment(ctx context.Context, order *entity.PaymentOrder) error {
	if order.TenantID == nil {
		return fmt.Errorf("tenant ID required for organization payment")
	}

	tenant, err := s.tenantRepo.GetByID(ctx, *order.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	switch order.OrderType {
	case "recharge":
		// Add balance to tenant
		tenant.Balance = tenant.Balance.Add(order.Amount)
		if err := s.tenantRepo.Update(ctx, tenant); err != nil {
			return fmt.Errorf("failed to update tenant: %w", err)
		}
	case "credit_settlement":
		// Credit settlement bill payment
		// The credit is already reset when the bill was created
		// Optionally convert to prepaid balance
		// tenant.Balance = tenant.Balance.Add(order.Amount)
		// s.tenantRepo.Update(ctx, tenant)
	case "subscription":
		// TODO: Handle subscription payment
	default:
		return fmt.Errorf("unsupported order type for organization")
	}

	return nil
}

// ListPaymentOrders lists payment orders with filters
func (s *PaymentService) ListPaymentOrders(ctx context.Context, req *PaymentOrderListRequest) ([]entity.PaymentOrder, int64, error) {
	if req.UserID != uuid.Nil && req.PaymentMode == "individual" {
		return s.paymentOrderRepo.ListByUser(ctx, req.UserID, req.Page, req.PageSize)
	}
	if req.TenantID != uuid.Nil && req.PaymentMode == "organization" {
		return s.paymentOrderRepo.ListByTenant(ctx, req.TenantID, req.Page, req.PageSize)
	}
	if req.Status != "" {
		return s.paymentOrderRepo.ListByStatus(ctx, req.Status, req.Page, req.PageSize)
	}
	return s.paymentOrderRepo.List(ctx, req.Page, req.PageSize)
}

// PaymentOrderListRequest represents list request
type PaymentOrderListRequest struct {
	UserID     uuid.UUID
	TenantID   uuid.UUID
	PaymentMode string
	Status     string
	Page       int
PageSize   int
}

// GetPendingOrders returns pending orders for a user or tenant
func (s *PaymentService) GetPendingOrders(ctx context.Context, userID uuid.UUID, tenantID *uuid.UUID) ([]entity.PaymentOrder, error) {
	if tenantID != nil {
		return s.paymentOrderRepo.ListPendingByTenant(ctx, *tenantID)
	}
	return s.paymentOrderRepo.ListPendingByUser(ctx, userID)
}

// CheckExpiredOrders marks expired orders as expired
func (s *PaymentService) CheckExpiredOrders(ctx context.Context) (int, error) {
	return s.paymentOrderRepo.MarkExpired(ctx)
}