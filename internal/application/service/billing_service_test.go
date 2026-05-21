package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// Mock implementations for billing service tests

type MockTenantRepository struct {
	tenants map[uuid.UUID]*entity.Tenant
	balance map[uuid.UUID]decimal.Decimal
}

func NewMockTenantRepo() *MockTenantRepository {
	return &MockTenantRepository{
		tenants: make(map[uuid.UUID]*entity.Tenant),
		balance: make(map[uuid.UUID]decimal.Decimal),
	}
}

func (m *MockTenantRepository) Create(ctx context.Context, tenant *entity.Tenant) error {
	tenant.ID = uuid.New()
	m.tenants[tenant.ID] = tenant
	m.balance[tenant.ID] = decimal.Zero
	return nil
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *MockTenantRepository) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	for _, t := range m.tenants {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (m *MockTenantRepository) Update(ctx context.Context, tenant *entity.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.tenants, id)
	delete(m.balance, id)
	return nil
}

func (m *MockTenantRepository) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	var result []entity.Tenant
	for _, t := range m.tenants {
		result = append(result, *t)
	}
	return result, int64(len(result)), nil
}

func (m *MockTenantRepository) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	current, ok := m.balance[id]
	if !ok {
		return errors.New("tenant not found")
	}
	m.balance[id] = current.Add(amount)
	return nil
}

func (m *MockTenantRepository) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	if bal, ok := m.balance[id]; ok {
		return bal, nil
	}
	return decimal.Zero, errors.New("tenant not found")
}

func (m *MockTenantRepository) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	current, ok := m.balance[id]
	if !ok {
		return errors.New("tenant not found")
	}
	if current.LessThan(amount) {
		return apperrors.ErrInsufficientBalance
	}
	m.balance[id] = current.Sub(amount)
	return nil
}

func (m *MockTenantRepository) ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error) {
	var result []entity.Tenant
	for _, t := range m.tenants {
		if t.CreditStatus == creditStatus {
			result = append(result, *t)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockTenantRepository) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	if bal, ok := m.balance[id]; ok {
		m.balance[id] = bal.Add(amount)
	}
	return nil
}

func (m *MockTenantRepository) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockTenantRepository) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, int64, error) {
	if bal, ok := m.balance[id]; ok {
		return bal, 0, nil
	}
	return decimal.Zero, 0, nil
}

func (m *MockTenantRepository) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockTenantRepository) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

type MockUsageRepository struct {
	records []entity.UsageRecord
}

func NewMockUsageRepo() *MockUsageRepository {
	return &MockUsageRepository{
		records: []entity.UsageRecord{},
	}
}

func (m *MockUsageRepository) Create(ctx context.Context, record *entity.UsageRecord) error {
	record.ID = uuid.New()
	m.records = append(m.records, *record)
	return nil
}

func (m *MockUsageRepository) CreateBatch(ctx context.Context, records []*entity.UsageRecord) error {
	for _, record := range records {
		record.ID = uuid.New()
		m.records = append(m.records, *record)
	}
	return nil
}

func (m *MockUsageRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.UsageRecord, error) {
	for _, r := range m.records {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, errors.New("record not found")
}

func (m *MockUsageRepository) GetByRequestID(ctx context.Context, requestID string) (*entity.UsageRecord, error) {
	for _, r := range m.records {
		if r.RequestID == requestID {
			return &r, nil
		}
	}
	return nil, errors.New("record not found")
}

func (m *MockUsageRepository) List(ctx context.Context, tenantID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error) {
	var result []entity.UsageRecord
	for _, r := range m.records {
		if r.TenantID == tenantID && !r.CreatedAt.Before(start) && !r.CreatedAt.After(end) {
			result = append(result, r)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockUsageRepository) ListByUser(ctx context.Context, userID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error) {
	var result []entity.UsageRecord
	for _, r := range m.records {
		if r.UserID == userID && !r.CreatedAt.Before(start) && !r.CreatedAt.After(end) {
			result = append(result, r)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockUsageRepository) GetTotalCost(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (decimal.Decimal, int64, error) {
	var total decimal.Decimal
	var count int64
	for _, r := range m.records {
		if r.TenantID == tenantID && !r.CreatedAt.Before(start) && !r.CreatedAt.After(end) {
			total = total.Add(r.Cost)
			count += int64(r.TotalTokens)
		}
	}
	return total, count, nil
}

type MockBillRepository struct {
	bills []entity.Bill
}

func NewMockBillRepo() *MockBillRepository {
	return &MockBillRepository{
		bills: []entity.Bill{},
	}
}

func (m *MockBillRepository) Create(ctx context.Context, bill *entity.Bill) error {
	bill.ID = uuid.New()
	m.bills = append(m.bills, *bill)
	return nil
}

func (m *MockBillRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Bill, error) {
	for _, b := range m.bills {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, errors.New("bill not found")
}

func (m *MockBillRepository) GetByBillNumber(ctx context.Context, billNumber string) (*entity.Bill, error) {
	for _, b := range m.bills {
		if b.BillNumber == billNumber {
			return &b, nil
		}
	}
	return nil, errors.New("bill not found")
}

func (m *MockBillRepository) Update(ctx context.Context, bill *entity.Bill) error {
	for i, b := range m.bills {
		if b.ID == bill.ID {
			m.bills[i] = *bill
			return nil
		}
	}
	return errors.New("bill not found")
}

func (m *MockBillRepository) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.Bill, int64, error) {
	var result []entity.Bill
	for _, b := range m.bills {
		if b.TenantID == tenantID {
			result = append(result, b)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockBillRepository) GetByPeriod(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*entity.Bill, error) {
	for _, b := range m.bills {
		if b.TenantID == tenantID && b.PeriodStart == start && b.PeriodEnd == end {
			return &b, nil
		}
	}
	return nil, errors.New("bill not found")
}

func (m *MockBillRepository) MarkPaid(ctx context.Context, id uuid.UUID) error {
	for i, b := range m.bills {
		if b.ID == id {
			m.bills[i].Status = "paid"
			now := time.Now()
			m.bills[i].PaidAt = &now
			return nil
		}
	}
	return errors.New("bill not found")
}

func (m *MockBillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	for i, b := range m.bills {
		if b.ID == id {
			m.bills = append(m.bills[:i], m.bills[i+1:]...)
			return nil
		}
	}
	return errors.New("bill not found")
}

func (m *MockBillRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.Bill, int64, error) {
	var result []entity.Bill
	for _, b := range m.bills {
		if b.Status == status {
			result = append(result, b)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockBillRepository) ListByType(ctx context.Context, billType string, page, pageSize int) ([]entity.Bill, int64, error) {
	var result []entity.Bill
	for _, b := range m.bills {
		if b.Type == billType {
			result = append(result, b)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockBillRepository) MarkPartialPaid(ctx context.Context, id uuid.UUID, remainingAmount decimal.Decimal) error {
	for i, b := range m.bills {
		if b.ID == id {
			m.bills[i].Status = "partial_paid"
			m.bills[i].TotalCost = remainingAmount
			return nil
		}
	}
	return errors.New("bill not found")
}

type MockRechargeRepository struct {
	records []entity.RechargeRecord
}

func NewMockRechargeRepo() *MockRechargeRepository {
	return &MockRechargeRepository{
		records: []entity.RechargeRecord{},
	}
}

func (m *MockRechargeRepository) Create(ctx context.Context, record *entity.RechargeRecord) error {
	record.ID = uuid.New()
	m.records = append(m.records, *record)
	return nil
}

func (m *MockRechargeRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.RechargeRecord, error) {
	for _, r := range m.records {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, errors.New("record not found")
}

func (m *MockRechargeRepository) Update(ctx context.Context, record *entity.RechargeRecord) error {
	for i, r := range m.records {
		if r.ID == record.ID {
			m.records[i] = *record
			return nil
		}
	}
	return errors.New("record not found")
}

func (m *MockRechargeRepository) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.RechargeRecord, int64, error) {
	var result []entity.RechargeRecord
	for _, r := range m.records {
		if r.TenantID == tenantID {
			result = append(result, r)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockRechargeRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	for i, r := range m.records {
		if r.ID == id {
			m.records[i].Status = "completed"
			now := time.Now()
			m.records[i].CompletedAt = &now
			return nil
		}
	}
	return errors.New("record not found")
}

type MockModelRepository struct {
	models []entity.Model
}

func NewMockModelRepo() *MockModelRepository {
	return &MockModelRepository{
		models: []entity.Model{},
	}
}

func (m *MockModelRepository) Create(ctx context.Context, model *entity.Model) error {
	model.ID = uuid.New()
	m.models = append(m.models, *model)
	return nil
}

func (m *MockModelRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Model, error) {
	for _, m := range m.models {
		if m.ID == id {
			return &m, nil
		}
	}
	return nil, errors.New("model not found")
}

func (m *MockModelRepository) GetByProviderModel(ctx context.Context, provider, modelID string) (*entity.Model, error) {
	for _, m := range m.models {
		if m.Provider == provider && m.ModelID == modelID {
			return &m, nil
		}
	}
	return nil, errors.New("model not found")
}

func (m *MockModelRepository) Update(ctx context.Context, model *entity.Model) error {
	for i, md := range m.models {
		if md.ID == model.ID {
			m.models[i] = *model
			return nil
		}
	}
	return errors.New("model not found")
}

func (m *MockModelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	for i, md := range m.models {
		if md.ID == id {
			m.models = append(m.models[:i], m.models[i+1:]...)
			return nil
		}
	}
	return errors.New("model not found")
}

func (m *MockModelRepository) List(ctx context.Context, provider string) ([]entity.Model, error) {
	var result []entity.Model
	for _, md := range m.models {
		if md.Provider == provider {
			result = append(result, md)
		}
	}
	return result, nil
}

func (m *MockModelRepository) ListActive(ctx context.Context) ([]entity.Model, error) {
	var result []entity.Model
	for _, md := range m.models {
		if md.Status == "active" {
			result = append(result, md)
		}
	}
	return result, nil
}

func (m *MockModelRepository) GetPricing(ctx context.Context, provider, modelID string) (*entity.Model, error) {
	for _, md := range m.models {
		if md.Provider == provider && md.ModelID == modelID {
			return &md, nil
		}
	}
	return nil, errors.New("model not found")
}

type MockUserRepo struct {
	users   map[uuid.UUID]*entity.User
	balance map[uuid.UUID]decimal.Decimal
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		users:   make(map[uuid.UUID]*entity.User),
		balance: make(map[uuid.UUID]decimal.Decimal),
	}
}

func (m *MockUserRepo) Create(ctx context.Context, user *entity.User) error {
	user.ID = uuid.New()
	m.users[user.ID] = user
	m.balance[user.ID] = user.Balance
	return nil
}
func (m *MockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}
func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return nil, nil
}
func (m *MockUserRepo) GetByVerificationToken(ctx context.Context, token string) (*entity.User, error) {
	return nil, nil
}
func (m *MockUserRepo) Update(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	return nil
}
func (m *MockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.users, id)
	delete(m.balance, id)
	return nil
}
func (m *MockUserRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error) {
	return nil, 0, nil
}
func (m *MockUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockUserRepo) IncrementMonthlyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockUserRepo) IncrementDailyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockUserRepo) ResetMonthlyBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockUserRepo) ResetDailyBudgetUsed(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *MockUserRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, int64, error) {
	return decimal.Zero, decimal.Zero, 0, nil
}
func (m *MockUserRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockUserRepo) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error  { return nil }
func (m *MockUserRepo) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error  { return nil }
func (m *MockUserRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	if bal, ok := m.balance[id]; ok {
		return bal, nil
	}
	return decimal.Zero, errors.New("user not found")
}
func (m *MockUserRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	current, ok := m.balance[id]
	if !ok {
		return errors.New("user not found")
	}
	if current.LessThan(amount) {
		return apperrors.ErrInsufficientBalance
	}
	m.balance[id] = current.Sub(amount)
	return nil
}
func (m *MockUserRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	current, ok := m.balance[id]
	if !ok {
		return errors.New("user not found")
	}
	m.balance[id] = current.Add(amount)
	return nil
}

// Tests

func TestNewBillingService(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	if service == nil {
		t.Error("NewBillingService should not return nil")
	}
}

func TestBillingService_CalculateCost(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	// Add a model with pricing
	model := &entity.Model{
		Provider:         "openai",
		ModelID:          "gpt-4",
		PromptPrice:      decimal.NewFromFloat(0.03),
		CompletionPrice:  decimal.NewFromFloat(0.06),
		Currency:         "USD",
	}
	modelRepo.Create(context.Background(), model)

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	tests := []struct {
		name            string
		provider        string
		modelID         string
		promptTokens    int64
		completionTokens int64
		wantErr         bool
	}{
		{
			name:            "valid calculation",
			provider:        "openai",
			modelID:         "gpt-4",
			promptTokens:    1000,
			completionTokens: 500,
			wantErr:         false,
		},
		{
			name:            "zero tokens",
			provider:        "openai",
			modelID:         "gpt-4",
			promptTokens:    0,
			completionTokens: 0,
			wantErr:         false,
		},
		{
			name:            "model not found",
			provider:        "unknown",
			modelID:         "unknown-model",
			promptTokens:    100,
			completionTokens: 100,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := service.CalculateCost(context.Background(), tt.provider, tt.modelID, tt.promptTokens, tt.completionTokens, 0, 0)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// For valid calculation, verify cost
				if tt.name == "valid calculation" {
					// Cost = (prompt * 0.03/1000) + (completion * 0.06/1000)
					expectedPromptCost := decimal.NewFromFloat(0.03).Mul(decimal.NewFromFloat(float64(tt.promptTokens))).Div(decimal.NewFromInt(1000))
					expectedCompletionCost := decimal.NewFromFloat(0.06).Mul(decimal.NewFromFloat(float64(tt.completionTokens))).Div(decimal.NewFromInt(1000))
					expectedCost := expectedPromptCost.Add(expectedCompletionCost)

					if !cost.Equals(expectedCost) {
						t.Errorf("cost = %v, want %v", cost, expectedCost)
					}
				}

				if tt.name == "zero tokens" {
					if !cost.IsZero() {
						t.Errorf("cost should be zero for zero tokens, got %v", cost)
					}
				}
			}
		})
	}
}

func TestBillingService_CheckBalance(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	userRepo := NewMockUserRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{
		Name:   "Test Tenant",
		Slug:   "test-tenant",
		Status: "active",
	}
	tenantRepo.Create(context.Background(), tenant)

	testUser := &entity.User{
		Email:   "test@test.com",
		Balance: decimal.NewFromInt(100),
	}
	userRepo.Create(context.Background(), testUser)

	service := NewBillingService(tenantRepo, userRepo, usageRepo, billRepo, rechargeRepo, modelRepo)

	balance, err := service.CheckBalance(context.Background(), testUser.ID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !balance.Equals(decimal.NewFromInt(100)) {
		t.Errorf("balance = %v, want 100", balance)
	}
}

func TestBillingService_Recharge(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	// Create a tenant
	tenant := &entity.Tenant{
		Name:   "Test Tenant",
		Slug:   "test-tenant",
	}
	tenantRepo.Create(context.Background(), tenant)

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	tests := []struct {
		name          string
		amount        decimal.Decimal
		paymentMethod string
		wantErr       bool
	}{
		{
			name:          "valid recharge",
			amount:        decimal.NewFromInt(100),
			paymentMethod: "credit_card",
			wantErr:       false,
		},
		{
			name:          "zero amount",
			amount:        decimal.Zero,
			paymentMethod: "credit_card",
			wantErr:       true,
		},
		{
			name:          "negative amount",
			amount:        decimal.NewFromInt(-50),
			paymentMethod: "credit_card",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := service.Recharge(context.Background(), tenant.ID, tt.amount, tt.paymentMethod, "payment-id", "test notes")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if !errors.Is(err, apperrors.ErrInvalidAmount) {
					t.Errorf("expected ErrInvalidAmount, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if record == nil {
					t.Error("record should not be nil")
				}
				if record.Status != "completed" {
					t.Errorf("record status = %v, want completed", record.Status)
				}
				if !record.Amount.Equals(tt.amount) {
					t.Errorf("record amount = %v, want %v", record.Amount, tt.amount)
				}

				// Check balance was updated
				balance, _ := tenantRepo.GetBalance(context.Background(), tenant.ID)
				if !balance.Equals(tt.amount) {
					t.Errorf("balance after recharge = %v, want %v", balance, tt.amount)
				}
			}
		})
	}
}

func TestBillingService_GetUsage(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	// Add some usage records
	now := time.Now()
	for i := 0; i < 5; i++ {
		record := &entity.UsageRecord{
			TenantID:         tenant.ID,
			UserID:           uuid.New(),
			RequestID:        "req-" + uuid.New().String(),
			Provider:         "openai",
			ModelID:          "gpt-4",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			Cost:             decimal.NewFromFloat(0.01),
			CreatedAt:        now,
		}
		usageRepo.Create(context.Background(), record)
	}

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	records, total, err := service.GetUsage(context.Background(), tenant.ID, start, end, 1, 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(records) != 5 {
		t.Errorf("records count = %d, want 5", len(records))
	}

	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
}

func TestBillingService_GenerateBill(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	// Add usage records
	now := time.Now()
	record := &entity.UsageRecord{
		TenantID:         tenant.ID,
		UserID:           uuid.New(),
		RequestID:        "req-" + uuid.New().String(),
		Provider:         "openai",
		ModelID:          "gpt-4",
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
		Cost:             decimal.NewFromFloat(0.05),
		CreatedAt:        now,
	}
	usageRepo.Create(context.Background(), record)

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	bill, err := service.GenerateBill(context.Background(), tenant.ID, start, end)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if bill == nil {
		t.Error("bill should not be nil")
	}

	if bill.TenantID != tenant.ID {
		t.Errorf("bill tenant ID = %v, want %v", bill.TenantID, tenant.ID)
	}

	if bill.Status != "pending" {
		t.Errorf("bill status = %v, want pending", bill.Status)
	}

	// Test generating bill for empty period
	emptyStart := now.Add(-24 * time.Hour)
	emptyEnd := now.Add(-23 * time.Hour)
	emptyBill, err := service.GenerateBill(context.Background(), tenant.ID, emptyStart, emptyEnd)
	if err == nil {
		t.Error("expected error for empty period")
	}
	if emptyBill != nil {
		t.Error("bill should be nil for empty period")
	}
}

func TestBillingService_GetBills(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	billRepo := NewMockBillRepo()
	usageRepo := NewMockUsageRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	// Create some bills
	for i := 0; i < 3; i++ {
		bill := &entity.Bill{
			TenantID:    tenant.ID,
			BillNumber:  "BILL-" + uuid.New().String(),
			PeriodStart: time.Now().Add(-24 * time.Duration(i) * time.Hour),
			PeriodEnd:   time.Now().Add(-23 * time.Duration(i) * time.Hour),
			TotalTokens: 1000,
			TotalCost:   decimal.NewFromFloat(10.0),
			Status:      "pending",
		}
		billRepo.Create(context.Background(), bill)
	}

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	bills, total, err := service.GetBills(context.Background(), tenant.ID, 1, 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(bills) != 3 {
		t.Errorf("bills count = %d, want 3", len(bills))
	}

	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestBillingService_GetRechargeRecords(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	rechargeRepo := NewMockRechargeRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	// Create some recharge records
	for i := 0; i < 3; i++ {
		record := &entity.RechargeRecord{
			TenantID:      tenant.ID,
			Amount:        decimal.NewFromInt(100),
			PaymentMethod: "credit_card",
			Status:        "completed",
		}
		rechargeRepo.Create(context.Background(), record)
	}

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	records, total, err := service.GetRechargeRecords(context.Background(), tenant.ID, 1, 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("records count = %d, want 3", len(records))
	}

	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestBillingService_RecordUsage_InsufficientBalance(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	userRepo := NewMockUserRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	testUser := &entity.User{Email: "test@test.com", Balance: decimal.Zero}
	userRepo.Create(context.Background(), testUser)

	// Add model pricing
	model := &entity.Model{
		Provider:        "openai",
		ModelID:         "gpt-4",
		PromptPrice:     decimal.NewFromFloat(0.03),
		CompletionPrice: decimal.NewFromFloat(0.06),
	}
	modelRepo.Create(context.Background(), model)

	service := NewBillingService(tenantRepo, userRepo, usageRepo, billRepo, rechargeRepo, modelRepo)

	apiKeyID := uuid.New()

	_, err := service.RecordUsage(
		context.Background(),
		tenant.ID,
		testUser.ID,
		apiKeyID,
		"req-123",
		"openai",
		"gpt-4",
		1000, 500, 0, 0,
		100, 200,
	)

	if err == nil {
		t.Error("expected error for insufficient balance")
	}

	if !errors.Is(err, apperrors.ErrInsufficientBalance) {
		t.Errorf("expected apperrors.ErrInsufficientBalance, got %v", err)
	}
}

func TestErrorVariables(t *testing.T) {
	if apperrors.ErrInsufficientBalance == nil {
		t.Error("apperrors.ErrInsufficientBalance should not be nil")
	}
	if apperrors.ErrInvalidAmount == nil {
		t.Error("apperrors.ErrInvalidAmount should not be nil")
	}

	if apperrors.ErrInsufficientBalance.Error() != "BILL_001: insufficient balance" {
		t.Errorf("apperrors.ErrInsufficientBalance message incorrect")
	}
	if apperrors.ErrInvalidAmount.Error() != "BILL_002: invalid amount" {
		t.Errorf("apperrors.ErrInvalidAmount message incorrect")
	}
}

func TestBillingService_RecordUsage_Success(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	userRepo := NewMockUserRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	testUser := &entity.User{Email: "test@test.com", Balance: decimal.NewFromFloat(100)}
	userRepo.Create(context.Background(), testUser)

	model := &entity.Model{
		Provider:        "openai",
		ModelID:         "gpt-4",
		PromptPrice:     decimal.NewFromFloat(0.03),
		CompletionPrice: decimal.NewFromFloat(0.06),
	}
	modelRepo.Create(context.Background(), model)

	service := NewBillingService(tenantRepo, userRepo, usageRepo, billRepo, rechargeRepo, modelRepo)

	apiKeyID := uuid.New()
	record, err := service.RecordUsage(
		context.Background(),
		tenant.ID,
		testUser.ID,
		apiKeyID,
		"req-success",
		"openai",
		"gpt-4",
		1000, 500, 0, 0,
		100, 200,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record == nil {
		t.Fatal("record should not be nil")
	}
	if record.RequestID != "req-success" {
		t.Errorf("request ID = %v, want req-success", record.RequestID)
	}
	if record.TotalTokens != 1500 {
		t.Errorf("total tokens = %d, want 1500", record.TotalTokens)
	}
	if record.TenantID != tenant.ID {
		t.Errorf("tenant ID = %v, want %v", record.TenantID, tenant.ID)
	}

	// User balance should have been deducted
	balance, _ := userRepo.GetBalance(context.Background(), testUser.ID)
	expectedCost := decimal.NewFromFloat(0.03).Mul(decimal.NewFromInt(1000)).Div(decimal.NewFromInt(1000)).
		Add(decimal.NewFromFloat(0.06).Mul(decimal.NewFromInt(500)).Div(decimal.NewFromInt(1000)))
	expectedBalance := decimal.NewFromFloat(100).Sub(expectedCost)
	if !balance.Equals(expectedBalance) {
		t.Errorf("balance = %v, want %v", balance, expectedBalance)
	}
}

func TestBillingService_RecordUsage_Rollback(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	model := &entity.Model{
		Provider:        "openai",
		ModelID:         "gpt-4",
		PromptPrice:     decimal.NewFromFloat(0.03),
		CompletionPrice: decimal.NewFromFloat(0.06),
	}
	modelRepo.Create(context.Background(), model)

	// Create a failing usage repo
	usageRepo := &MockUsageRepositoryFailing{}

	userRepo := NewMockUserRepo()
	initialBalance := decimal.NewFromFloat(100)
	testUser := &entity.User{Email: "test@test.com", Balance: initialBalance}
	userRepo.Create(context.Background(), testUser)

	service := NewBillingService(tenantRepo, userRepo, usageRepo, billRepo, rechargeRepo, modelRepo)

	apiKeyID := uuid.New()
	_, err := service.RecordUsage(
		context.Background(),
		tenant.ID,
		testUser.ID,
		apiKeyID,
		"req-rollback",
		"openai",
		"gpt-4",
		1000, 500, 0, 0,
		100, 200,
	)

	if err == nil {
		t.Fatal("expected error from failing usage repo")
	}

	// User balance should have been rolled back
	balance, _ := userRepo.GetBalance(context.Background(), testUser.ID)
	if !balance.Equals(initialBalance) {
		t.Errorf("balance should have been rolled back to %v, got %v", initialBalance, balance)
	}
}

type MockUsageRepositoryFailing struct{}

func (m *MockUsageRepositoryFailing) Create(ctx context.Context, record *entity.UsageRecord) error {
	return errors.New("database connection failed")
}
func (m *MockUsageRepositoryFailing) CreateBatch(ctx context.Context, records []*entity.UsageRecord) error {
	return errors.New("database connection failed")
}
func (m *MockUsageRepositoryFailing) GetByID(ctx context.Context, id uuid.UUID) (*entity.UsageRecord, error) {
	return nil, errors.New("not found")
}
func (m *MockUsageRepositoryFailing) GetByRequestID(ctx context.Context, requestID string) (*entity.UsageRecord, error) {
	return nil, errors.New("not found")
}
func (m *MockUsageRepositoryFailing) List(ctx context.Context, tenantID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error) {
	return nil, 0, nil
}
func (m *MockUsageRepositoryFailing) ListByUser(ctx context.Context, userID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error) {
	return nil, 0, nil
}
func (m *MockUsageRepositoryFailing) GetTotalCost(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}

func TestBillingService_RecordUsage_ModelNotFound(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)
	tenantRepo.UpdateBalance(context.Background(), tenant.ID, decimal.NewFromFloat(100))

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	_, err := service.RecordUsage(
		context.Background(),
		tenant.ID,
		uuid.New(),
		uuid.New(),
		"req-unknown-model",
		"unknown",
		"unknown-model",
		100, 50, 0, 0,
		100, 200,
	)

	if err == nil {
		t.Fatal("expected error for unknown model")
	}
}

func TestBillingService_GetTotalCost(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	now := time.Now()
	for i := 0; i < 3; i++ {
		record := &entity.UsageRecord{
			TenantID:  tenant.ID,
			UserID:    uuid.New(),
			RequestID: "req-" + uuid.New().String(),
			Provider:  "openai",
			ModelID:   "gpt-4",
			Cost:      decimal.NewFromFloat(float64(10 + i)),
			CreatedAt: now,
		}
		usageRepo.Create(context.Background(), record)
	}

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	totalCost, totalTokens, err := service.GetTotalCost(context.Background(), tenant.ID, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTotal := decimal.NewFromFloat(10).Add(decimal.NewFromFloat(11)).Add(decimal.NewFromFloat(12))
	if !totalCost.Equals(expectedTotal) {
		t.Errorf("total cost = %v, want %v", totalCost, expectedTotal)
	}
	if totalTokens != 0 {
		t.Errorf("total tokens = %d, want 0", totalTokens)
	}
}

func TestBillingService_GenerateBill_Dedup(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	now := time.Now()
	record := &entity.UsageRecord{
		TenantID:  tenant.ID,
		UserID:    uuid.New(),
		RequestID: "req-" + uuid.New().String(),
		Provider:  "openai",
		ModelID:   "gpt-4",
		Cost:      decimal.NewFromFloat(10),
		CreatedAt: now,
	}
	usageRepo.Create(context.Background(), record)

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	// First generate
	bill1, err := service.GenerateBill(context.Background(), tenant.ID, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second generate with same period should return existing bill (dedup)
	bill2, err := service.GenerateBill(context.Background(), tenant.ID, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bill1.ID != bill2.ID {
		t.Errorf("dedup failed: bill1.ID=%v, bill2.ID=%v", bill1.ID, bill2.ID)
	}
}

func TestBillingService_CalculateCost_EdgeCases(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	rechargeRepo := NewMockRechargeRepo()
	modelRepo := NewMockModelRepo()

	model := &entity.Model{
		Provider:        "openai",
		ModelID:         "gpt-4",
		PromptPrice:     decimal.NewFromFloat(0.03),
		CompletionPrice: decimal.NewFromFloat(0.06),
	}
	modelRepo.Create(context.Background(), model)

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)

	// Test with only prompt tokens
	cost, err := service.CalculateCost(context.Background(), "openai", "gpt-4", 2000, 0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedPrompt := decimal.NewFromFloat(0.03).Mul(decimal.NewFromInt(2000)).Div(decimal.NewFromInt(1000))
	if !cost.Equals(expectedPrompt) {
		t.Errorf("cost = %v, want %v", cost, expectedPrompt)
	}

	// Test with only completion tokens
	cost, err = service.CalculateCost(context.Background(), "openai", "gpt-4", 0, 1000, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedCompletion := decimal.NewFromFloat(0.06).Mul(decimal.NewFromInt(1000)).Div(decimal.NewFromInt(1000))
	if !cost.Equals(expectedCompletion) {
		t.Errorf("cost = %v, want %v", cost, expectedCompletion)
	}

	// Test with large token counts
	cost, err = service.CalculateCost(context.Background(), "openai", "gpt-4", 1000000, 500000, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cost.IsZero() {
		t.Error("cost should not be zero for large token counts")
	}
}

func TestBillingService_Recharge_UpdateBalanceFailure(t *testing.T) {
	tenantRepo := NewMockTenantRepo()
	usageRepo := NewMockUsageRepo()
	billRepo := NewMockBillRepo()
	modelRepo := NewMockModelRepo()

	tenant := &entity.Tenant{Name: "Test", Slug: "test"}
	tenantRepo.Create(context.Background(), tenant)

	// Use a recharge repo that succeeds creation but fails MarkCompleted
	rechargeRepo := &MockRechargeRepositoryFailing{
		MockRechargeRepository: *NewMockRechargeRepo(),
	}

	service := NewBillingService(tenantRepo, NewMockUserRepo(), usageRepo, billRepo, rechargeRepo, modelRepo)
	_, err := service.Recharge(context.Background(), tenant.ID, decimal.NewFromInt(50), "card", "pay-1", "notes")

	if err == nil {
		t.Fatal("expected error from failing recharge repo")
	}
}

type MockRechargeRepositoryFailing struct {
	MockRechargeRepository
}

func (m *MockRechargeRepositoryFailing) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	return errors.New("mark completed failed")
}