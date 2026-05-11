package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
)

// Mock Payment Order Repository
type MockPaymentOrderRepository struct {
	orders      map[uuid.UUID]*entity.PaymentOrder
	byNumber    map[string]*entity.PaymentOrder
	byPaymentID map[string]*entity.PaymentOrder
}

func NewMockPaymentOrderRepo() *MockPaymentOrderRepository {
	return &MockPaymentOrderRepository{
		orders:      make(map[uuid.UUID]*entity.PaymentOrder),
		byNumber:    make(map[string]*entity.PaymentOrder),
		byPaymentID: make(map[string]*entity.PaymentOrder),
	}
}

func (m *MockPaymentOrderRepository) Create(ctx context.Context, order *entity.PaymentOrder) error {
	order.ID = uuid.New()
	m.orders[order.ID] = order
	m.byNumber[order.OrderNumber] = order
	return nil
}

func (m *MockPaymentOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.PaymentOrder, error) {
	if o, ok := m.orders[id]; ok {
		return o, nil
	}
	return nil, errors.New("order not found")
}

func (m *MockPaymentOrderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*entity.PaymentOrder, error) {
	if o, ok := m.byNumber[orderNumber]; ok {
		return o, nil
	}
	return nil, errors.New("order not found")
}

func (m *MockPaymentOrderRepository) GetByPaymentID(ctx context.Context, paymentID string) (*entity.PaymentOrder, error) {
	if o, ok := m.byPaymentID[paymentID]; ok {
		return o, nil
	}
	return nil, errors.New("order not found")
}

func (m *MockPaymentOrderRepository) Update(ctx context.Context, order *entity.PaymentOrder) error {
	m.orders[order.ID] = order
	m.byNumber[order.OrderNumber] = order
	if order.PaymentID != "" {
		m.byPaymentID[order.PaymentID] = order
	}
	return nil
}

func (m *MockPaymentOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.orders, id)
	return nil
}

func (m *MockPaymentOrderRepository) List(ctx context.Context, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
	var result []entity.PaymentOrder
	for _, o := range m.orders {
		result = append(result, *o)
	}
	return result, int64(len(result)), nil
}

func (m *MockPaymentOrderRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
	var result []entity.PaymentOrder
	for _, o := range m.orders {
		if o.UserID == userID {
			result = append(result, *o)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockPaymentOrderRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
	var result []entity.PaymentOrder
	for _, o := range m.orders {
		if o.TenantID != nil && *o.TenantID == tenantID {
			result = append(result, *o)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockPaymentOrderRepository) ListByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
	return m.ListByUser(ctx, userID, page, pageSize)
}

func (m *MockPaymentOrderRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
	return m.ListByTenant(ctx, tenantID, page, pageSize)
}

func (m *MockPaymentOrderRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
	var result []entity.PaymentOrder
	for _, o := range m.orders {
		if o.Status == status {
			result = append(result, *o)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockPaymentOrderRepository) ListPendingByUser(ctx context.Context, userID uuid.UUID) ([]entity.PaymentOrder, error) {
	var result []entity.PaymentOrder
	for _, o := range m.orders {
		if o.UserID == userID && o.Status == "pending" {
			result = append(result, *o)
		}
	}
	return result, nil
}

func (m *MockPaymentOrderRepository) ListPendingByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.PaymentOrder, error) {
	var result []entity.PaymentOrder
	for _, o := range m.orders {
		if o.TenantID != nil && *o.TenantID == tenantID && o.Status == "pending" {
			result = append(result, *o)
		}
	}
	return result, nil
}

func (m *MockPaymentOrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if o, ok := m.orders[id]; ok {
		o.Status = status
		return nil
	}
	return errors.New("order not found")
}

func (m *MockPaymentOrderRepository) MarkPaid(ctx context.Context, id uuid.UUID, paymentID string, callbackData string) error {
	if o, ok := m.orders[id]; ok {
		o.Status = "paid"
		o.PaymentID = paymentID
		o.CallbackData = callbackData
		now := time.Now()
		o.PaidAt = &now
		return nil
	}
	return errors.New("order not found")
}

func (m *MockPaymentOrderRepository) MarkFailed(ctx context.Context, id uuid.UUID) error {
	if o, ok := m.orders[id]; ok {
		o.Status = "failed"
		return nil
	}
	return errors.New("order not found")
}

func (m *MockPaymentOrderRepository) MarkCancelled(ctx context.Context, id uuid.UUID) error {
	if o, ok := m.orders[id]; ok {
		o.Status = "cancelled"
		return nil
	}
	return errors.New("order not found")
}

func (m *MockPaymentOrderRepository) MarkExpired(ctx context.Context) (int, error) {
	count := 0
	now := time.Now()
	for _, o := range m.orders {
		if o.Status == "pending" && o.ExpireAt != nil && now.After(*o.ExpireAt) {
			o.Status = "expired"
			count++
		}
	}
	return count, nil
}

func (m *MockPaymentOrderRepository) GetTotalAmountByUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	total := decimal.Zero
	for _, o := range m.orders {
		if o.UserID == userID {
			total = total.Add(o.Amount)
		}
	}
	return total, nil
}

func (m *MockPaymentOrderRepository) GetTotalAmountByTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	total := decimal.Zero
	for _, o := range m.orders {
		if o.TenantID != nil && *o.TenantID == tenantID {
			total = total.Add(o.Amount)
		}
	}
	return total, nil
}

func (m *MockPaymentOrderRepository) GenerateOrderNumber() string {
	return "PAY-MOCK-" + uuid.New().String()[:8]
}

// Mock UserQuota Repository for payment tests
type MockUserQuotaRepository struct {
	quotas map[uuid.UUID]*entity.UserQuota
}

func NewMockUserQuotaRepo() *MockUserQuotaRepository {
	return &MockUserQuotaRepository{
		quotas: make(map[uuid.UUID]*entity.UserQuota),
	}
}

func (m *MockUserQuotaRepository) Create(ctx context.Context, quota *entity.UserQuota) error {
	quota.ID = uuid.New()
	m.quotas[quota.ID] = quota
	return nil
}

func (m *MockUserQuotaRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.UserQuota, error) {
	if q, ok := m.quotas[id]; ok {
		return q, nil
	}
	return nil, errors.New("quota not found")
}

func (m *MockUserQuotaRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.UserQuota, error) {
	for _, q := range m.quotas {
		if q.UserID == userID {
			return q, nil
		}
	}
	return nil, errors.New("quota not found")
}

func (m *MockUserQuotaRepository) Update(ctx context.Context, quota *entity.UserQuota) error {
	m.quotas[quota.ID] = quota
	return nil
}

func (m *MockUserQuotaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.quotas, id)
	return nil
}

func (m *MockUserQuotaRepository) List(ctx context.Context, page, pageSize int) ([]entity.UserQuota, int64, error) {
	var result []entity.UserQuota
	for _, q := range m.quotas {
		result = append(result, *q)
	}
	return result, int64(len(result)), nil
}

func (m *MockUserQuotaRepository) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockUserQuotaRepository) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockUserQuotaRepository) GetTokenUsage(ctx context.Context, id uuid.UUID) (int64, int64, error) {
	return 0, 0, nil
}

func (m *MockUserQuotaRepository) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	if q, ok := m.quotas[id]; ok {
		return q.Balance, nil
	}
	return decimal.Zero, nil
}

func (m *MockUserQuotaRepository) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	if q, ok := m.quotas[id]; ok {
		q.Balance = q.Balance.Sub(amount)
		return nil
	}
	return errors.New("quota not found")
}

func (m *MockUserQuotaRepository) AddBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	if q, ok := m.quotas[id]; ok {
		q.Balance = q.Balance.Add(amount)
		return nil
	}
	return errors.New("quota not found")
}

func (m *MockUserQuotaRepository) IncrementMonthlyCost(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error {
	return nil
}

func (m *MockUserQuotaRepository) ResetMonthlyCost(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockUserQuotaRepository) GetMonthlyCost(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *MockUserQuotaRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if q, ok := m.quotas[id]; ok {
		q.Status = status
		return nil
	}
	return errors.New("quota not found")
}

func (m *MockUserQuotaRepository) GetStatus(ctx context.Context, id uuid.UUID) (string, error) {
	if q, ok := m.quotas[id]; ok {
		return q.Status, nil
	}
	return "", nil
}

// Mock Tenant Repository for payment tests
type MockTenantPaymentRepo struct {
	tenants map[uuid.UUID]*entity.Tenant
}

func NewMockTenantPaymentRepo() *MockTenantPaymentRepo {
	return &MockTenantPaymentRepo{
		tenants: make(map[uuid.UUID]*entity.Tenant),
	}
}

func (m *MockTenantPaymentRepo) Create(ctx context.Context, tenant *entity.Tenant) error {
	tenant.ID = uuid.New()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockTenantPaymentRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *MockTenantPaymentRepo) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	return nil, errors.New("not found")
}

func (m *MockTenantPaymentRepo) Update(ctx context.Context, tenant *entity.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockTenantPaymentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.tenants, id)
	return nil
}

func (m *MockTenantPaymentRepo) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	var result []entity.Tenant
	for _, t := range m.tenants {
		result = append(result, *t)
	}
	return result, int64(len(result)), nil
}

func (m *MockTenantPaymentRepo) ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}

func (m *MockTenantPaymentRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockTenantPaymentRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *MockTenantPaymentRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockTenantPaymentRepo) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockTenantPaymentRepo) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockTenantPaymentRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}

func (m *MockTenantPaymentRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockTenantPaymentRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Interface verification
var _ repository.PaymentOrderRepository = (*MockPaymentOrderRepository)(nil)
var _ repository.UserQuotaRepository = (*MockUserQuotaRepository)(nil)
var _ repository.TenantRepository = (*MockTenantPaymentRepo)(nil)