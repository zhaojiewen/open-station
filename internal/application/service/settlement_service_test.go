package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// Settlement-specific mock tenant repo with credit field support
type MockSettlementTenantRepo struct {
	tenants map[uuid.UUID]*entity.Tenant
}

func NewMockSettlementTenantRepo() *MockSettlementTenantRepo {
	return &MockSettlementTenantRepo{tenants: make(map[uuid.UUID]*entity.Tenant)}
}

func (m *MockSettlementTenantRepo) addTenant(t *entity.Tenant) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	m.tenants[t.ID] = t
}

func (m *MockSettlementTenantRepo) Create(ctx context.Context, tenant *entity.Tenant) error {
	tenant.ID = uuid.New()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockSettlementTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *MockSettlementTenantRepo) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	return nil, errors.New("not found")
}

func (m *MockSettlementTenantRepo) Update(ctx context.Context, tenant *entity.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockSettlementTenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.tenants, id)
	return nil
}

func (m *MockSettlementTenantRepo) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	var result []entity.Tenant
	for _, t := range m.tenants {
		result = append(result, *t)
	}
	return result, int64(len(result)), nil
}

func (m *MockSettlementTenantRepo) ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error) {
	var result []entity.Tenant
	for _, t := range m.tenants {
		if t.CreditStatus == creditStatus {
			result = append(result, *t)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockSettlementTenantRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockSettlementTenantRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *MockSettlementTenantRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockSettlementTenantRepo) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockSettlementTenantRepo) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockSettlementTenantRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}

func (m *MockSettlementTenantRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockSettlementTenantRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Settlement-specific mock bill repo
type MockSettlementBillRepo struct {
	bills map[uuid.UUID]*entity.Bill
}

func NewMockSettlementBillRepo() *MockSettlementBillRepo {
	return &MockSettlementBillRepo{bills: make(map[uuid.UUID]*entity.Bill)}
}

func (m *MockSettlementBillRepo) Create(ctx context.Context, bill *entity.Bill) error {
	bill.ID = uuid.New()
	m.bills[bill.ID] = bill
	return nil
}

func (m *MockSettlementBillRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Bill, error) {
	if b, ok := m.bills[id]; ok {
		return b, nil
	}
	return nil, errors.New("bill not found")
}

func (m *MockSettlementBillRepo) GetByBillNumber(ctx context.Context, billNumber string) (*entity.Bill, error) {
	return nil, errors.New("not found")
}

func (m *MockSettlementBillRepo) Update(ctx context.Context, bill *entity.Bill) error {
	m.bills[bill.ID] = bill
	return nil
}

func (m *MockSettlementBillRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.bills, id)
	return nil
}

func (m *MockSettlementBillRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.Bill, int64, error) {
	var result []entity.Bill
	for _, b := range m.bills {
		if b.TenantID == tenantID {
			result = append(result, *b)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockSettlementBillRepo) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.Bill, int64, error) {
	var result []entity.Bill
	for _, b := range m.bills {
		if b.Status == status {
			result = append(result, *b)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockSettlementBillRepo) ListByType(ctx context.Context, billType string, page, pageSize int) ([]entity.Bill, int64, error) {
	var result []entity.Bill
	for _, b := range m.bills {
		if b.Type == billType {
			result = append(result, *b)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockSettlementBillRepo) GetByPeriod(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*entity.Bill, error) {
	return nil, errors.New("not found")
}

func (m *MockSettlementBillRepo) MarkPaid(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockSettlementBillRepo) MarkPartialPaid(ctx context.Context, id uuid.UUID, remainingAmount decimal.Decimal) error {
	return nil
}

// --- Settlement Service Tests ---

func TestNewSettlementService(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	service := NewSettlementService(tenantRepo, billRepo, nil)

	if service == nil {
		t.Error("NewSettlementService should not return nil")
	}
}

func TestCheckSettlementTrigger_NoCredit(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	tenant := &entity.Tenant{
		Name:         "Test",
		Slug:         "test",
		CreditStatus: "none",
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("should not trigger for tenant without credit")
	}
}

func TestCheckSettlementTrigger_NilCreditLimit(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	tenant := &entity.Tenant{
		Name:         "Test",
		Slug:         "test",
		CreditStatus: "approved",
		CreditLimit:  nil,
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("should not trigger when credit limit is nil")
	}
}

func TestCheckSettlementTrigger_Monthly(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	today := time.Now().Day()
	settlementDay := today

	tenant := &entity.Tenant{
		Name:            "Test",
		Slug:            "test",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "monthly",
		SettlementDay:   &settlementDay,
		CreditUsed:      decimal.NewFromInt(100),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ShouldTrigger {
		t.Error("should trigger monthly settlement on settlement day")
	}
}

func TestCheckSettlementTrigger_Monthly_WrongDay(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	today := time.Now().Day()
	// Set settlement day to a different day
	settlementDay := today + 1
	if settlementDay > 28 {
		settlementDay = 1
	}

	tenant := &entity.Tenant{
		Name:            "Test",
		Slug:            "test",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "monthly",
		SettlementDay:   &settlementDay,
		CreditUsed:      decimal.NewFromInt(100),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("should not trigger on wrong day")
	}
}

func TestCheckSettlementTrigger_Monthly_NoSettlementDay(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:            "Test",
		Slug:            "test",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "monthly",
		SettlementDay:   nil,
		CreditUsed:      decimal.NewFromInt(100),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("should not trigger when settlement day not set")
	}
}

func TestCheckSettlementTrigger_Weekly(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	today := int(time.Now().Weekday())

	tenant := &entity.Tenant{
		Name:            "Test",
		Slug:            "test",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "weekly",
		SettlementDay:   &today,
		CreditUsed:      decimal.NewFromInt(50),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ShouldTrigger {
		t.Error("should trigger weekly settlement on correct weekday")
	}
}

func TestCheckSettlementTrigger_Weekly_NoSettlementDay(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:            "Test",
		Slug:            "test",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "weekly",
		SettlementDay:   nil,
		CreditUsed:      decimal.NewFromInt(50),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("should not trigger when weekly settlement day not set")
	}
}

func TestCheckSettlementTrigger_Threshold(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	threshold := decimal.NewFromInt(500)

	tenant := &entity.Tenant{
		Name:             "Test",
		Slug:             "test",
		CreditStatus:     "approved",
		CreditLimit:      &creditLimit,
		SettlementCycle:  "threshold",
		ThresholdAmount:  &threshold,
		CreditUsed:       decimal.NewFromInt(600),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ShouldTrigger {
		t.Error("should trigger when credit used >= threshold")
	}
}

func TestCheckSettlementTrigger_Threshold_Below(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	threshold := decimal.NewFromInt(500)

	tenant := &entity.Tenant{
		Name:             "Test",
		Slug:             "test",
		CreditStatus:     "approved",
		CreditLimit:      &creditLimit,
		SettlementCycle:  "threshold",
		ThresholdAmount:  &threshold,
		CreditUsed:       decimal.NewFromInt(400),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("should not trigger below threshold")
	}
}

func TestCheckSettlementTrigger_Threshold_NoThresholdAmount(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:             "Test",
		Slug:             "test",
		CreditStatus:     "approved",
		CreditLimit:      &creditLimit,
		SettlementCycle:  "threshold",
		ThresholdAmount:  nil,
		CreditUsed:       decimal.NewFromInt(600),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("should not trigger when threshold amount not set")
	}
}

func TestCheckSettlementTrigger_Custom(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:            "Test",
		Slug:            "test",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "custom",
		CreditUsed:      decimal.NewFromInt(100),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("custom settlement should not auto-trigger (manual only)")
	}
}

func TestCheckSettlementTrigger_DefaultCycle(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:            "Test",
		Slug:            "test",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "unknown",
		CreditUsed:      decimal.NewFromInt(100),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	result, err := service.CheckSettlementTrigger(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldTrigger {
		t.Error("should not trigger for unknown cycle")
	}
}

func TestTriggerSettlement(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:            "Test",
		Slug:            "test",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "monthly",
		CreditUsed:      decimal.NewFromInt(300),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	bill, err := service.TriggerSettlement(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bill == nil {
		t.Fatal("bill should not be nil")
	}
	if bill.Type != "credit_settlement" {
		t.Errorf("bill type = %v, want credit_settlement", bill.Type)
	}
	if !bill.TotalCost.Equals(decimal.NewFromInt(300)) {
		t.Errorf("total cost = %v, want 300", bill.TotalCost)
	}
	if bill.Status != "pending" {
		t.Errorf("status = %v, want pending", bill.Status)
	}
	if bill.DueDate == nil {
		t.Error("due date should be set")
	}

	// Verify credit was reset on tenant
	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if !updated.CreditUsed.IsZero() {
		t.Errorf("credit used should be reset to 0, got %v", updated.CreditUsed)
	}
}

func TestTriggerSettlement_NotApproved(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	tenant := &entity.Tenant{
		Name:         "Test",
		Slug:         "test",
		CreditStatus: "none",
		CreditUsed:   decimal.NewFromInt(100),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	_, err := service.TriggerSettlement(context.Background(), tenant.ID)
	if err == nil {
		t.Fatal("expected error for non-approved tenant")
	}
}

func TestTriggerSettlement_NoCreditUsed(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:         "Test",
		Slug:         "test",
		CreditStatus: "approved",
		CreditLimit:  &creditLimit,
		CreditUsed:   decimal.Zero,
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	_, err := service.TriggerSettlement(context.Background(), tenant.ID)
	if err == nil {
		t.Fatal("expected error for zero credit used")
	}
}

func TestCalculateDueDate_Monthly(t *testing.T) {
	dueDate := calculateDueDate("monthly")
	if dueDate == nil {
		t.Fatal("due date should not be nil")
	}
	expected := time.Now().AddDate(0, 0, 15)
	diff := dueDate.Sub(expected)
	if diff < -time.Hour || diff > time.Hour {
		t.Errorf("monthly due date should be ~15 days, got %v", dueDate)
	}
}

func TestCalculateDueDate_Weekly(t *testing.T) {
	dueDate := calculateDueDate("weekly")
	if dueDate == nil {
		t.Fatal("due date should not be nil")
	}
	expected := time.Now().AddDate(0, 0, 7)
	diff := dueDate.Sub(expected)
	if diff < -time.Hour || diff > time.Hour {
		t.Errorf("weekly due date should be ~7 days, got %v", dueDate)
	}
}

func TestCalculateDueDate_Threshold(t *testing.T) {
	dueDate := calculateDueDate("threshold")
	if dueDate == nil {
		t.Fatal("due date should not be nil")
	}
	expected := time.Now().AddDate(0, 0, 10)
	diff := dueDate.Sub(expected)
	if diff < -time.Hour || diff > time.Hour {
		t.Errorf("threshold due date should be ~10 days, got %v", dueDate)
	}
}

func TestCalculateDueDate_Default(t *testing.T) {
	dueDate := calculateDueDate("unknown")
	if dueDate == nil {
		t.Fatal("due date should not be nil")
	}
	expected := time.Now().AddDate(0, 0, 15)
	diff := dueDate.Sub(expected)
	if diff < -time.Hour || diff > time.Hour {
		t.Errorf("default due date should be ~15 days, got %v", dueDate)
	}
}

func TestProcessSettlementBill_FullPayment(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	bill := &entity.Bill{
		TenantID:  uuid.New(),
		Type:      "credit_settlement",
		TotalCost: decimal.NewFromInt(500),
		Status:    "pending",
	}
	billRepo.Create(context.Background(), bill)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	err := service.ProcessSettlementBill(context.Background(), bill.ID, decimal.NewFromInt(500))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := billRepo.GetByID(context.Background(), bill.ID)
	if updated.Status != "paid" {
		t.Errorf("status = %v, want paid", updated.Status)
	}
	if updated.PaidAt == nil {
		t.Error("paid_at should be set")
	}
}

func TestProcessSettlementBill_PartialPayment(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	bill := &entity.Bill{
		TenantID:  uuid.New(),
		Type:      "credit_settlement",
		TotalCost: decimal.NewFromInt(500),
		Status:    "pending",
	}
	billRepo.Create(context.Background(), bill)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	err := service.ProcessSettlementBill(context.Background(), bill.ID, decimal.NewFromInt(200))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := billRepo.GetByID(context.Background(), bill.ID)
	if updated.Status != "partial_paid" {
		t.Errorf("status = %v, want partial_paid", updated.Status)
	}
	// Remaining = 500 - 200 = 300
	if !updated.TotalCost.Equals(decimal.NewFromInt(300)) {
		t.Errorf("remaining cost = %v, want 300", updated.TotalCost)
	}
}

func TestProcessSettlementBill_Overpayment(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	bill := &entity.Bill{
		TenantID:  uuid.New(),
		Type:      "credit_settlement",
		TotalCost: decimal.NewFromInt(500),
		Status:    "pending",
	}
	billRepo.Create(context.Background(), bill)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	err := service.ProcessSettlementBill(context.Background(), bill.ID, decimal.NewFromInt(600))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := billRepo.GetByID(context.Background(), bill.ID)
	if updated.Status != "paid" {
		t.Errorf("status = %v, want paid", updated.Status)
	}
}

func TestProcessSettlementBill_NotSettlementBill(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	bill := &entity.Bill{
		TenantID:  uuid.New(),
		Type:      "usage",
		TotalCost: decimal.NewFromInt(500),
		Status:    "pending",
	}
	billRepo.Create(context.Background(), bill)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	err := service.ProcessSettlementBill(context.Background(), bill.ID, decimal.NewFromInt(500))
	if err == nil {
		t.Fatal("expected error for non-settlement bill")
	}
}

func TestProcessSettlementBill_NotFound(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	service := NewSettlementService(tenantRepo, billRepo, nil)

	err := service.ProcessSettlementBill(context.Background(), uuid.New(), decimal.NewFromInt(500))
	if err == nil {
		t.Fatal("expected error for non-existent bill")
	}
}

func TestCheckOverdueBills(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	tenant := &entity.Tenant{
		Name:   "Test",
		Slug:   "test",
		Status: "active",
	}
	tenantRepo.addTenant(tenant)

	// Create an overdue bill
	pastDue := time.Now().Add(-24 * time.Hour)
	bill := &entity.Bill{
		TenantID:  tenant.ID,
		Type:      "credit_settlement",
		TotalCost: decimal.NewFromInt(300),
		Status:    "pending",
		DueDate:   &pastDue,
	}
	billRepo.Create(context.Background(), bill)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	results, err := service.CheckOverdueBills(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("should have overdue bills")
	}
	if results[0].BillID != bill.ID {
		t.Errorf("bill ID = %v, want %v", results[0].BillID, bill.ID)
	}

	// Tenant should be suspended
	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if updated.Status != "suspended" {
		t.Errorf("tenant status = %v, want suspended", updated.Status)
	}
}

func TestCheckOverdueBills_NoOverdue(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	// Create a non-overdue bill (due in future)
	futureDue := time.Now().Add(24 * time.Hour)
	bill := &entity.Bill{
		TenantID:  uuid.New(),
		Type:      "credit_settlement",
		TotalCost: decimal.NewFromInt(300),
		Status:    "pending",
		DueDate:   &futureDue,
	}
	billRepo.Create(context.Background(), bill)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	results, err := service.CheckOverdueBills(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("should have no overdue bills, got %d", len(results))
	}
}

func TestCheckOverdueBills_SkipsNonSettlement(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	pastDue := time.Now().Add(-24 * time.Hour)
	bill := &entity.Bill{
		TenantID:  uuid.New(),
		Type:      "usage",
		TotalCost: decimal.NewFromInt(300),
		Status:    "pending",
		DueDate:   &pastDue,
	}
	billRepo.Create(context.Background(), bill)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	results, err := service.CheckOverdueBills(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("should skip non-settlement bills, got %d", len(results))
	}
}

func TestCheckOverdueBills_NoDueDate(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	tenant := &entity.Tenant{
		Name:   "Test",
		Slug:   "test",
		Status: "active",
	}
	tenantRepo.addTenant(tenant)

	bill := &entity.Bill{
		TenantID:  tenant.ID,
		Type:      "credit_settlement",
		TotalCost: decimal.NewFromInt(300),
		Status:    "pending",
		DueDate:   nil,
	}
	billRepo.Create(context.Background(), bill)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	results, err := service.CheckOverdueBills(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("should skip bills without due date, got %d", len(results))
	}
}

func TestRunScheduledSettlement(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	today := time.Now().Day()
	settlementDay := today

	tenant := &entity.Tenant{
		Name:            "Test Monthly",
		Slug:            "test-monthly",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "monthly",
		SettlementDay:   &settlementDay,
		CreditUsed:      decimal.NewFromInt(100),
	}
	tenantRepo.addTenant(tenant)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	results, err := service.RunScheduledSettlement(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("should have settlement results")
	}
	if !results[0].Triggered {
		t.Error("settlement should be triggered")
	}
	if results[0].TenantID != tenant.ID {
		t.Errorf("tenant ID = %v, want %v", results[0].TenantID, tenant.ID)
	}
}

func TestRunScheduledSettlement_WithErrors(t *testing.T) {
	tenantRepo := NewMockSettlementTenantRepo()
	billRepo := NewMockSettlementBillRepo()

	creditLimit := decimal.NewFromInt(1000)
	today := time.Now().Day()
	settlementDay := today

	// Tenant that triggers but has no credit used -> error on TriggerSettlement
	tenantNoUsage := &entity.Tenant{
		Name:            "Test No Usage",
		Slug:            "test-no-usage",
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		SettlementCycle: "monthly",
		SettlementDay:   &settlementDay,
		CreditUsed:      decimal.Zero,
	}
	tenantRepo.addTenant(tenantNoUsage)

	service := NewSettlementService(tenantRepo, billRepo, nil)

	results, err := service.RunScheduledSettlement(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// It should not trigger because CreditUsed is zero
	if len(results) > 0 && results[0].Triggered {
		t.Error("should not trigger with zero credit used")
	}
}