package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

func TestNewPaymentService(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	if service == nil {
		t.Error("NewPaymentService should not return nil")
	}
}

func TestCreatePaymentOrder(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	userID := uuid.New()
	req := &PaymentOrderRequest{
		PaymentMode:     "individual",
		UserID:          userID,
		OrderType:       "recharge",
		Amount:          decimal.NewFromInt(100),
		Currency:        "USD",
		PaymentProvider: "alipay",
		PaymentMethod:   "qr_code",
	}

	order, err := service.CreatePaymentOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order == nil {
		t.Fatal("order should not be nil")
	}
	if order.Status != "pending" {
		t.Errorf("status = %v, want pending", order.Status)
	}
	if order.OrderType != "recharge" {
		t.Errorf("order type = %v, want recharge", order.OrderType)
	}
	if order.PaymentProvider != "alipay" {
		t.Errorf("payment provider = %v, want alipay", order.PaymentProvider)
	}
	if order.ExpireAt == nil {
		t.Error("expire_at should be set")
	}
	// Default 30 min expiry (not 24h bank)
	if order.ExpireAt != nil {
		diff := order.ExpireAt.Sub(time.Now()).Minutes()
		if diff < 28 || diff > 31 {
			t.Errorf("expiry should be ~30 min, got %.0f min", diff)
		}
	}
}

func TestCreatePaymentOrder_BankTransfer(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	req := &PaymentOrderRequest{
		PaymentMode:     "organization",
		UserID:          uuid.New(),
		OrderType:       "recharge",
		Amount:          decimal.NewFromInt(500),
		Currency:        "USD",
		PaymentProvider: "bank",
		PaymentMethod:   "bank_transfer",
	}

	order, err := service.CreatePaymentOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.ExpireAt != nil {
		diff := order.ExpireAt.Sub(time.Now()).Hours()
		if diff < 23 || diff > 25 {
			t.Errorf("bank expiry should be ~24h, got %.0f h", diff)
		}
	}
}

func TestGetPaymentOrder(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)

	retrieved, err := service.GetPaymentOrder(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.ID != created.ID {
		t.Errorf("retrieved ID = %v, want %v", retrieved.ID, created.ID)
	}
}

func TestGetPaymentOrderByNumber(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)

	retrieved, err := service.GetPaymentOrderByNumber(context.Background(), created.OrderNumber)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.OrderNumber != created.OrderNumber {
		t.Errorf("retrieved number = %v, want %v", retrieved.OrderNumber, created.OrderNumber)
	}
}

func TestCancelPaymentOrder(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)

	err := service.CancelPaymentOrder(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, _ := service.GetPaymentOrder(context.Background(), created.ID)
	if retrieved.Status != "cancelled" {
		t.Errorf("status = %v, want cancelled", retrieved.Status)
	}
}

func TestCancelPaymentOrder_AlreadyProcessed(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)

	// First cancellation
	_ = service.CancelPaymentOrder(context.Background(), created.ID)

	// Second cancellation should fail
	err := service.CancelPaymentOrder(context.Background(), created.ID)
	if err == nil {
		t.Fatal("expected error for already processed order")
	}
}

func TestCancelPaymentOrder_NotFound(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	err := service.CancelPaymentOrder(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent order")
	}
}

func TestProcessPaymentCallback_Success(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	userQuotaID := uuid.New()
	quota := &entity.UserQuota{
		UserID:  uuid.New(),
		Balance: decimal.NewFromInt(0),
		Status:  "active",
	}
	quotaRepo.Create(context.Background(), quota)
	userQuotaID = quota.ID

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      quota.UserID,
		UserQuotaID: &userQuotaID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)

	callbackReq := &PaymentCallbackRequest{
		OrderNumber:  created.OrderNumber,
		PaymentID:    "ext-pay-123",
		PaidAmount:   decimal.NewFromInt(100),
		CallbackData: `{"status":"success"}`,
	}

	order, err := service.ProcessPaymentCallback(context.Background(), callbackReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Status != "paid" {
		t.Errorf("status = %v, want paid", order.Status)
	}
	if order.PaymentID != "ext-pay-123" {
		t.Errorf("payment_id = %v, want ext-pay-123", order.PaymentID)
	}
	if order.PaidAt == nil {
		t.Error("paid_at should be set")
	}
}

func TestProcessPaymentCallback_OrderNotFound(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	callbackReq := &PaymentCallbackRequest{
		OrderNumber: "NONEXISTENT",
		PaymentID:   "ext-pay-456",
		PaidAmount:  decimal.NewFromInt(100),
	}

	_, err := service.ProcessPaymentCallback(context.Background(), callbackReq)
	if err == nil {
		t.Fatal("expected error for non-existent order")
	}
}

func TestProcessPaymentCallback_AlreadyProcessed(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)

	// First callback succeeds
	callbackReq := &PaymentCallbackRequest{
		OrderNumber: created.OrderNumber,
		PaymentID:   "ext-pay-123",
		PaidAmount:  decimal.NewFromInt(100),
	}
	_, _ = service.ProcessPaymentCallback(context.Background(), callbackReq)

	// Second callback fails
	callbackReq.PaymentID = "ext-pay-456"
	_, err := service.ProcessPaymentCallback(context.Background(), callbackReq)
	if err == nil {
		t.Fatal("expected error for already processed order")
	}
}

func TestProcessPaymentCallback_AmountTooLow(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)

	// Pay only 50 (which is < 99% of 100)
	callbackReq := &PaymentCallbackRequest{
		OrderNumber: created.OrderNumber,
		PaymentID:   "ext-pay-456",
		PaidAmount:  decimal.NewFromInt(50),
	}

	_, err := service.ProcessPaymentCallback(context.Background(), callbackReq)
	if err == nil {
		t.Fatal("expected error for insufficient payment amount")
	}
}

func TestProcessPaymentCallback_ByPaymentID(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	quota := &entity.UserQuota{
		UserID:  uuid.New(),
		Balance: decimal.Zero,
		Status:  "active",
	}
	quotaRepo.Create(context.Background(), quota)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      quota.UserID,
		UserQuotaID: &quota.ID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)
	// Manually associate payment_id
	created.PaymentID = "ext-pay-lookup"
	paymentRepo.Update(context.Background(), created)

	callbackReq := &PaymentCallbackRequest{
		PaymentID:  "ext-pay-lookup",
		PaidAmount: decimal.NewFromInt(100),
	}

	order, err := service.ProcessPaymentCallback(context.Background(), callbackReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Status != "paid" {
		t.Errorf("status = %v, want paid", order.Status)
	}
}

func TestProcessPaymentCallback_ByOrderNumber(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	quota := &entity.UserQuota{
		UserID:  uuid.New(),
		Balance: decimal.Zero,
		Status:  "active",
	}
	quotaRepo.Create(context.Background(), quota)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      quota.UserID,
		UserQuotaID: &quota.ID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Currency:    "USD",
	}
	created, _ := service.CreatePaymentOrder(context.Background(), req)

	// Callback with no payment ID but valid order number
	callbackReq := &PaymentCallbackRequest{
		OrderNumber: created.OrderNumber,
		PaidAmount:  decimal.NewFromInt(100),
	}

	order, err := service.ProcessPaymentCallback(context.Background(), callbackReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Status != "paid" {
		t.Errorf("status = %v, want paid", order.Status)
	}
}

func TestProcessIndividualPayment_Recharge(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	quota := &entity.UserQuota{
		UserID:  uuid.New(),
		Balance: decimal.NewFromInt(50),
		Status:  "active",
	}
	quotaRepo.Create(context.Background(), quota)

	order := &entity.PaymentOrder{
		PaymentMode: "individual",
		UserID:      quota.UserID,
		UserQuotaID: &quota.ID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Status:      "paid",
	}

	err := service.processIndividualPayment(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := quotaRepo.GetByID(context.Background(), quota.ID)
	if !updated.Balance.Equals(decimal.NewFromInt(150)) {
		t.Errorf("balance = %v, want 150", updated.Balance)
	}
}

func TestProcessIndividualPayment_ActivateQuota(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	quota := &entity.UserQuota{
		UserID:  uuid.New(),
		Balance: decimal.Zero,
		Status:  "pending_payment",
	}
	quotaRepo.Create(context.Background(), quota)

	order := &entity.PaymentOrder{
		PaymentMode: "individual",
		UserID:      quota.UserID,
		UserQuotaID: &quota.ID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Status:      "paid",
	}

	err := service.processIndividualPayment(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := quotaRepo.GetByID(context.Background(), quota.ID)
	if updated.Status != "active" {
		t.Errorf("status = %v, want active", updated.Status)
	}
	if updated.LastPaymentAt == nil {
		t.Error("LastPaymentAt should be set")
	}
}

func TestProcessIndividualPayment_NoQuotaID(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	order := &entity.PaymentOrder{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		UserQuotaID: nil,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
	}

	err := service.processIndividualPayment(context.Background(), order)
	if err == nil {
		t.Fatal("expected error for missing quota ID")
	}
}

func TestProcessIndividualPayment_QuotaNotFound(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	nonExistentID := uuid.New()
	order := &entity.PaymentOrder{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		UserQuotaID: &nonExistentID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
	}

	err := service.processIndividualPayment(context.Background(), order)
	if err == nil {
		t.Fatal("expected error for non-existent quota")
	}
}

func TestProcessIndividualPayment_Subscription(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	quota := &entity.UserQuota{
		UserID:  uuid.New(),
		Balance: decimal.Zero,
		Status:  "active",
	}
	quotaRepo.Create(context.Background(), quota)

	order := &entity.PaymentOrder{
		PaymentMode: "individual",
		UserID:      quota.UserID,
		UserQuotaID: &quota.ID,
		OrderType:   "subscription",
		Amount:      decimal.NewFromInt(100),
	}

	// Subscription payment for individual is handled (TODO branch), no error
	err := service.processIndividualPayment(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessIndividualPayment_UnsupportedType(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	quota := &entity.UserQuota{
		UserID:  uuid.New(),
		Balance: decimal.Zero,
		Status:  "active",
	}
	quotaRepo.Create(context.Background(), quota)

	order := &entity.PaymentOrder{
		PaymentMode: "individual",
		UserID:      quota.UserID,
		UserQuotaID: &quota.ID,
		OrderType:   "unknown_type",
		Amount:      decimal.NewFromInt(100),
	}

	err := service.processIndividualPayment(context.Background(), order)
	if err == nil {
		t.Fatal("expected error for unsupported order type")
	}
}

func TestProcessOrganizationPayment_Recharge(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	tenant := &entity.Tenant{
		Name:    "Test Org",
		Slug:    "test-org",
		Balance: decimal.NewFromInt(200),
		Type:    "organization",
	}
	tenantRepo.Create(context.Background(), tenant)

	order := &entity.PaymentOrder{
		PaymentMode: "organization",
		TenantID:    &tenant.ID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(300),
		Status:      "paid",
	}

	err := service.processOrganizationPayment(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if !updated.Balance.Equals(decimal.NewFromInt(500)) {
		t.Errorf("balance = %v, want 500", updated.Balance)
	}
}

func TestProcessOrganizationPayment_NoTenantID(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	order := &entity.PaymentOrder{
		PaymentMode: "organization",
		TenantID:    nil,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
	}

	err := service.processOrganizationPayment(context.Background(), order)
	if err == nil {
		t.Fatal("expected error for missing tenant ID")
	}
}

func TestProcessOrganizationPayment_TenantNotFound(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	nonExistentID := uuid.New()
	order := &entity.PaymentOrder{
		PaymentMode: "organization",
		TenantID:    &nonExistentID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
	}

	err := service.processOrganizationPayment(context.Background(), order)
	if err == nil {
		t.Fatal("expected error for non-existent tenant")
	}
}

func TestProcessOrganizationPayment_CreditSettlement(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	tenant := &entity.Tenant{
		Name:    "Test Org",
		Slug:    "test-org",
		Balance: decimal.NewFromInt(200),
		Type:    "organization",
	}
	tenantRepo.Create(context.Background(), tenant)

	order := &entity.PaymentOrder{
		PaymentMode: "organization",
		TenantID:    &tenant.ID,
		OrderType:   "credit_settlement",
		Amount:      decimal.NewFromInt(100),
		Status:      "paid",
	}

	err := service.processOrganizationPayment(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessOrganizationPayment_Subscription(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	tenant := &entity.Tenant{
		Name:    "Test Org",
		Slug:    "test-org",
		Balance: decimal.NewFromInt(200),
		Type:    "organization",
	}
	tenantRepo.Create(context.Background(), tenant)

	order := &entity.PaymentOrder{
		PaymentMode: "organization",
		TenantID:    &tenant.ID,
		OrderType:   "subscription",
		Amount:      decimal.NewFromInt(100),
		Status:      "paid",
	}

	err := service.processOrganizationPayment(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessOrganizationPayment_UnsupportedType(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	tenant := &entity.Tenant{
		Name:    "Test Org",
		Slug:    "test-org",
		Balance: decimal.NewFromInt(200),
		Type:    "organization",
	}
	tenantRepo.Create(context.Background(), tenant)

	order := &entity.PaymentOrder{
		PaymentMode: "organization",
		TenantID:    &tenant.ID,
		OrderType:   "unknown_type",
		Amount:      decimal.NewFromInt(100),
		Status:      "paid",
	}

	err := service.processOrganizationPayment(context.Background(), order)
	if err == nil {
		t.Fatal("expected error for unsupported order type")
	}
}

func TestProcessPayment_InvalidMode(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	order := &entity.PaymentOrder{
		PaymentMode: "invalid_mode",
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
	}

	err := service.processPayment(context.Background(), order)
	if err == nil {
		t.Fatal("expected error for invalid payment mode")
	}
}

func TestListPaymentOrders_Individual(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	userID := uuid.New()
	for i := 0; i < 3; i++ {
		req := &PaymentOrderRequest{
			PaymentMode: "individual",
			UserID:      userID,
			OrderType:   "recharge",
			Amount:      decimal.NewFromInt(int64(100 + i*10)),
		}
		service.CreatePaymentOrder(context.Background(), req)
	}

	listReq := &PaymentOrderListRequest{
		UserID:      userID,
		PaymentMode: "individual",
		Page:        1,
		PageSize:    10,
	}
	orders, total, err := service.ListPaymentOrders(context.Background(), listReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 3 {
		t.Errorf("orders count = %d, want 3", len(orders))
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestListPaymentOrders_Organization(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	tenantID := uuid.New()
	for i := 0; i < 2; i++ {
		req := &PaymentOrderRequest{
			PaymentMode: "organization",
			UserID:      uuid.New(),
			TenantID:    &tenantID,
			OrderType:   "recharge",
			Amount:      decimal.NewFromInt(100),
		}
		service.CreatePaymentOrder(context.Background(), req)
	}

	listReq := &PaymentOrderListRequest{
		TenantID:    tenantID,
		PaymentMode: "organization",
		Page:        1,
		PageSize:    10,
	}
	orders, _, err := service.ListPaymentOrders(context.Background(), listReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("orders count = %d, want 2", len(orders))
	}
}

func TestListPaymentOrders_ByStatus(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
	}
	service.CreatePaymentOrder(context.Background(), req)

	listReq := &PaymentOrderListRequest{
		Status:   "pending",
		Page:     1,
		PageSize: 10,
	}
	orders, _, err := service.ListPaymentOrders(context.Background(), listReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("orders count = %d, want 1", len(orders))
	}
}

func TestGetPendingOrders_User(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	userID := uuid.New()
	req := &PaymentOrderRequest{
		PaymentMode: "individual",
		UserID:      userID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
	}
	service.CreatePaymentOrder(context.Background(), req)

	orders, err := service.GetPendingOrders(context.Background(), userID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("orders count = %d, want 1", len(orders))
	}
}

func TestGetPendingOrders_Tenant(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	tenantID := uuid.New()
	req := &PaymentOrderRequest{
		PaymentMode: "organization",
		UserID:      uuid.New(),
		TenantID:    &tenantID,
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
	}
	service.CreatePaymentOrder(context.Background(), req)

	orders, err := service.GetPendingOrders(context.Background(), uuid.Nil, &tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("orders count = %d, want 1", len(orders))
	}
}

func TestCheckExpiredOrders(t *testing.T) {
	paymentRepo := NewMockPaymentOrderRepo()
	quotaRepo := NewMockUserQuotaRepo()
	tenantRepo := NewMockTenantPaymentRepo()

	service := NewPaymentService(paymentRepo, quotaRepo, tenantRepo, nil)

	// Create order with past expiry
	pastTime := time.Now().Add(-1 * time.Hour)
	order := &entity.PaymentOrder{
		PaymentMode: "individual",
		UserID:      uuid.New(),
		OrderNumber: "PAY-EXPIRED",
		OrderType:   "recharge",
		Amount:      decimal.NewFromInt(100),
		Status:      "pending",
		ExpireAt:    &pastTime,
	}
	paymentRepo.Create(context.Background(), order)

	count, err := service.CheckExpiredOrders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expired count = %d, want 1", count)
	}
}

func TestGenerateOrderNumber(t *testing.T) {
	num := generateOrderNumber()
	if num == "" {
		t.Error("order number should not be empty")
	}
	if len(num) < 10 {
		t.Errorf("order number too short: %s", num)
	}
	// Should start with PAY-
	if num[:4] != "PAY-" {
		t.Errorf("order number should start with PAY-, got %s", num)
	}

	// Should generate unique numbers
	num2 := generateOrderNumber()
	if num == num2 {
		t.Error("should generate unique order numbers")
	}
}