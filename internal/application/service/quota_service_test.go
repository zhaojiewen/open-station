package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// Mock MemberQuota Repository
type MockMemberQuotaRepo struct {
	quotas map[uuid.UUID]*entity.MemberQuota
}

func NewMockMemberQuotaRepo() *MockMemberQuotaRepo {
	return &MockMemberQuotaRepo{quotas: make(map[uuid.UUID]*entity.MemberQuota)}
}

func (m *MockMemberQuotaRepo) addQuota(q *entity.MemberQuota) {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	m.quotas[q.ID] = q
}

func (m *MockMemberQuotaRepo) Create(ctx context.Context, quota *entity.MemberQuota) error {
	quota.ID = uuid.New()
	m.quotas[quota.ID] = quota
	return nil
}

func (m *MockMemberQuotaRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.MemberQuota, error) {
	if q, ok := m.quotas[id]; ok {
		return q, nil
	}
	return nil, errors.New("member quota not found")
}

func (m *MockMemberQuotaRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.MemberQuota, error) {
	for _, q := range m.quotas {
		if q.UserID == userID {
			return q, nil
		}
	}
	return nil, errors.New("member quota not found")
}

func (m *MockMemberQuotaRepo) GetByTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) (*entity.MemberQuota, error) {
	return nil, errors.New("not found")
}

func (m *MockMemberQuotaRepo) Update(ctx context.Context, quota *entity.MemberQuota) error {
	m.quotas[quota.ID] = quota
	return nil
}

func (m *MockMemberQuotaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.quotas, id)
	return nil
}

func (m *MockMemberQuotaRepo) List(ctx context.Context, page, pageSize int) ([]entity.MemberQuota, int64, error) {
	var result []entity.MemberQuota
	for _, q := range m.quotas {
		result = append(result, *q)
	}
	return result, int64(len(result)), nil
}

func (m *MockMemberQuotaRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.MemberQuota, int64, error) {
	var result []entity.MemberQuota
	for _, q := range m.quotas {
		if q.TenantID == tenantID {
			result = append(result, *q)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockMemberQuotaRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	if q, ok := m.quotas[id]; ok {
		q.TokensUsed += tokens
		return nil
	}
	return errors.New("not found")
}

func (m *MockMemberQuotaRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
	if q, ok := m.quotas[id]; ok {
		q.TokensUsed = 0
		return nil
	}
	return errors.New("not found")
}

func (m *MockMemberQuotaRepo) GetTokenUsage(ctx context.Context, id uuid.UUID) (int64, int64, error) {
	if q, ok := m.quotas[id]; ok {
		limit := int64(0)
		if q.TokenQuotaLimit != nil {
			limit = *q.TokenQuotaLimit
		}
		return q.TokensUsed, limit, nil
	}
	return 0, 0, errors.New("not found")
}

func (m *MockMemberQuotaRepo) IncrementCostUsed(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error {
	if q, ok := m.quotas[id]; ok {
		q.CostUsed = q.CostUsed.Add(cost)
		return nil
	}
	return errors.New("not found")
}

func (m *MockMemberQuotaRepo) ResetCostUsed(ctx context.Context, id uuid.UUID) error {
	if q, ok := m.quotas[id]; ok {
		q.CostUsed = decimal.Zero
		return nil
	}
	return errors.New("not found")
}

func (m *MockMemberQuotaRepo) GetCostUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, error) {
	if q, ok := m.quotas[id]; ok {
		limit := decimal.Zero
		if q.CostLimit != nil {
			limit = *q.CostLimit
		}
		return q.CostUsed, limit, nil
	}
	return decimal.Zero, decimal.Zero, errors.New("not found")
}

func (m *MockMemberQuotaRepo) GetTotalTokensUsedByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var total int64
	for _, q := range m.quotas {
		if q.TenantID == tenantID {
			total += q.TokensUsed
		}
	}
	return total, nil
}

func (m *MockMemberQuotaRepo) GetTotalCostUsedByTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	total := decimal.Zero
	for _, q := range m.quotas {
		if q.TenantID == tenantID {
			total = total.Add(q.CostUsed)
		}
	}
	return total, nil
}

func (m *MockMemberQuotaRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if q, ok := m.quotas[id]; ok {
		q.Status = status
		return nil
	}
	return errors.New("not found")
}

func (m *MockMemberQuotaRepo) SetExceeded(ctx context.Context, id uuid.UUID, reason string) error {
	return nil
}

func (m *MockMemberQuotaRepo) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockMemberQuotaRepo) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockMemberQuotaRepo) GetActiveAPIKeysCount(ctx context.Context, id uuid.UUID) (int, error) {
	return 0, nil
}

// Quota-specific mock UserQuota repository
type MockQuotaUserRepo struct {
	quotas map[uuid.UUID]*entity.UserQuota
}

func NewMockQuotaUserRepo() *MockQuotaUserRepo {
	return &MockQuotaUserRepo{quotas: make(map[uuid.UUID]*entity.UserQuota)}
}

func (m *MockQuotaUserRepo) Create(ctx context.Context, quota *entity.UserQuota) error {
	quota.ID = uuid.New()
	m.quotas[quota.ID] = quota
	return nil
}

func (m *MockQuotaUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.UserQuota, error) {
	if q, ok := m.quotas[id]; ok {
		return q, nil
	}
	return nil, errors.New("quota not found")
}

func (m *MockQuotaUserRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.UserQuota, error) {
	for _, q := range m.quotas {
		if q.UserID == userID {
			return q, nil
		}
	}
	return nil, errors.New("quota not found")
}

func (m *MockQuotaUserRepo) Update(ctx context.Context, quota *entity.UserQuota) error {
	m.quotas[quota.ID] = quota
	return nil
}

func (m *MockQuotaUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.quotas, id)
	return nil
}

func (m *MockQuotaUserRepo) List(ctx context.Context, page, pageSize int) ([]entity.UserQuota, int64, error) {
	var result []entity.UserQuota
	for _, q := range m.quotas {
		result = append(result, *q)
	}
	return result, int64(len(result)), nil
}

func (m *MockQuotaUserRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockQuotaUserRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockQuotaUserRepo) GetTokenUsage(ctx context.Context, id uuid.UUID) (int64, int64, error) {
	return 0, 0, nil
}

func (m *MockQuotaUserRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	if q, ok := m.quotas[id]; ok {
		return q.Balance, nil
	}
	return decimal.Zero, nil
}

func (m *MockQuotaUserRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockQuotaUserRepo) AddBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	if q, ok := m.quotas[id]; ok {
		q.Balance = q.Balance.Add(amount)
		return nil
	}
	return errors.New("not found")
}

func (m *MockQuotaUserRepo) IncrementMonthlyCost(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error {
	return nil
}

func (m *MockQuotaUserRepo) ResetMonthlyCost(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockQuotaUserRepo) GetMonthlyCost(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *MockQuotaUserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}

func (m *MockQuotaUserRepo) GetStatus(ctx context.Context, id uuid.UUID) (string, error) {
	return "active", nil
}
func (m *MockQuotaUserRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

// Quota-specific mock tenant repo with credit/token fields
type MockQuotaTenantRepo struct {
	tenants map[uuid.UUID]*entity.Tenant
}

func NewMockQuotaTenantRepo() *MockQuotaTenantRepo {
	return &MockQuotaTenantRepo{tenants: make(map[uuid.UUID]*entity.Tenant)}
}

func (m *MockQuotaTenantRepo) Create(ctx context.Context, tenant *entity.Tenant) error {
	tenant.ID = uuid.New()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockQuotaTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *MockQuotaTenantRepo) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	return nil, errors.New("not found")
}

func (m *MockQuotaTenantRepo) Update(ctx context.Context, tenant *entity.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockQuotaTenantRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockQuotaTenantRepo) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}
func (m *MockQuotaTenantRepo) ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}
func (m *MockQuotaTenantRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockQuotaTenantRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (m *MockQuotaTenantRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockQuotaTenantRepo) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockQuotaTenantRepo) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockQuotaTenantRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}
func (m *MockQuotaTenantRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockQuotaTenantRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error { return nil }

// --- Quota Service Tests ---

func TestNewQuotaService(t *testing.T) {
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())
	if service == nil {
		t.Error("NewQuotaService should not return nil")
	}
}

func TestCheckUsageAllowance_InvalidQuotaType(t *testing.T) {
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	apiKey := &entity.APIKey{QuotaType: "unknown", QuotaID: uuid.New()}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 100, decimal.NewFromInt(10))
	if err == nil {
		t.Fatal("expected error for invalid quota type")
	}
}

func TestCheckUsageAllowance_Individual_TokenQuota(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	tokenQuota := int64(10000)
	quota := &entity.UserQuota{
		UserID:     uuid.New(),
		TokenQuota: tokenQuota,
		TokensUsed: 5000,
		Balance:    decimal.Zero,
		Status:     "active",
	}
	userRepo.Create(context.Background(), quota)

	apiKey := &entity.APIKey{QuotaType: "individual", QuotaID: quota.ID}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 3000, decimal.NewFromInt(5))
	if err != nil {
		t.Fatalf("should pass with token quota sufficient: %v", err)
	}
}

func TestCheckUsageAllowance_Individual_Balance(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:     uuid.New(),
		TokenQuota: 1000,
		TokensUsed: 1000, // quota exhausted
		Balance:    decimal.NewFromInt(100),
		Status:     "active",
	}
	userRepo.Create(context.Background(), quota)

	apiKey := &entity.APIKey{QuotaType: "individual", QuotaID: quota.ID}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 100, decimal.NewFromInt(50))
	if err != nil {
		t.Fatalf("should pass with balance sufficient: %v", err)
	}
}

func TestCheckUsageAllowance_Individual_Insufficient(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:     uuid.New(),
		TokenQuota: 0,
		TokensUsed: 0,
		Balance:    decimal.NewFromInt(10),
		Status:     "active",
	}
	userRepo.Create(context.Background(), quota)

	apiKey := &entity.APIKey{QuotaType: "individual", QuotaID: quota.ID}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 100, decimal.NewFromInt(50))
	if err == nil {
		t.Fatal("expected insufficient balance error")
	}
}

func TestCheckUsageAllowance_Individual_Suspended(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:  uuid.New(),
		Balance: decimal.NewFromInt(100),
		Status:  "suspended",
	}
	userRepo.Create(context.Background(), quota)

	apiKey := &entity.APIKey{QuotaType: "individual", QuotaID: quota.ID}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 100, decimal.NewFromInt(50))
	if err == nil {
		t.Fatal("expected suspended user error")
	}
}

func TestCheckUsageAllowance_Individual_QuotaNotFound(t *testing.T) {
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	apiKey := &entity.APIKey{QuotaType: "individual", QuotaID: uuid.New()}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 100, decimal.NewFromInt(50))
	if err == nil {
		t.Fatal("expected error for non-existent quota")
	}
}

func TestCheckUsageAllowance_Member_TokenQuota(t *testing.T) {
	memberRepo := NewMockMemberQuotaRepo()
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), memberRepo, tenantRepo)

	tenLimit := int64(100000)
	tenant := &entity.Tenant{
		Name:       "Test Org",
		Slug:       "test-org",
		Status:     "active",
		TokenLimit: &tenLimit,
		Balance:    decimal.NewFromInt(1000),
	}
	tenantRepo.Create(context.Background(), tenant)

	memLimit := int64(5000)
	memberQuota := &entity.MemberQuota{
		UserID:          uuid.New(),
		TenantID:        tenant.ID,
		TokenQuotaLimit: &memLimit,
		TokensUsed:      0,
		CostLimit:       nil,
		CostUsed:        decimal.Zero,
		Status:          "active",
	}
	memberRepo.Create(context.Background(), memberQuota)

	apiKey := &entity.APIKey{QuotaType: "member", QuotaID: memberQuota.ID, TenantID: tenant.ID}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 3000, decimal.NewFromInt(5))
	if err != nil {
		t.Fatalf("should pass member token quota check: %v", err)
	}
}

func TestCheckUsageAllowance_Member_TokenExceeded(t *testing.T) {
	memberRepo := NewMockMemberQuotaRepo()
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), memberRepo, tenantRepo)

	memLimit := int64(1000)
	memberQuota := &entity.MemberQuota{
		UserID:          uuid.New(),
		TenantID:        uuid.New(),
		TokenQuotaLimit: &memLimit,
		TokensUsed:      800,
		Status:          "active",
	}
	memberRepo.Create(context.Background(), memberQuota)

	apiKey := &entity.APIKey{QuotaType: "member", QuotaID: memberQuota.ID, TenantID: uuid.New()}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 300, decimal.NewFromInt(5))
	if err == nil {
		t.Fatal("expected member token quota exceeded error")
	}
}

func TestCheckUsageAllowance_Member_CostExceeded(t *testing.T) {
	memberRepo := NewMockMemberQuotaRepo()
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), memberRepo, tenantRepo)

	costLimit := decimal.NewFromInt(100)
	memberQuota := &entity.MemberQuota{
		UserID:    uuid.New(),
		TenantID:  uuid.New(),
		CostLimit: &costLimit,
		CostUsed:  decimal.NewFromInt(80),
		Status:    "active",
	}
	memberRepo.Create(context.Background(), memberQuota)

	apiKey := &entity.APIKey{QuotaType: "member", QuotaID: memberQuota.ID, TenantID: uuid.New()}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 100, decimal.NewFromInt(30))
	if err == nil {
		t.Fatal("expected member cost limit exceeded error")
	}
}

func TestCheckUsageAllowance_Member_Suspended(t *testing.T) {
	memberRepo := NewMockMemberQuotaRepo()
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), memberRepo, tenantRepo)

	memberQuota := &entity.MemberQuota{
		UserID:  uuid.New(),
		TenantID: uuid.New(),
		Status:  "suspended",
	}
	memberRepo.Create(context.Background(), memberQuota)

	apiKey := &entity.APIKey{QuotaType: "member", QuotaID: memberQuota.ID, TenantID: uuid.New()}
	err := service.CheckUsageAllowance(context.Background(), apiKey, 100, decimal.NewFromInt(5))
	if err == nil {
		t.Fatal("expected member suspended error")
	}
}

func TestCheckOrganizationQuota_TokenQuota(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	tenLimit := int64(10000)
	tenant := &entity.Tenant{
		Name:            "Test Org",
		Slug:            "test-org",
		Status:          "active",
		TokenLimit:      &tenLimit,
		TokensUsedMonth: 2000,
		Balance:         decimal.Zero,
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.checkOrganizationQuota(context.Background(), tenant.ID, 5000, decimal.NewFromInt(10))
	if err != nil {
		t.Fatalf("should pass with token quota: %v", err)
	}
}

func TestCheckOrganizationQuota_Balance(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	tenant := &entity.Tenant{
		Name:   "Test Org",
		Slug:   "test-org",
		Status: "active",
		Balance: decimal.NewFromInt(500),
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.checkOrganizationQuota(context.Background(), tenant.ID, 100, decimal.NewFromInt(200))
	if err != nil {
		t.Fatalf("should pass with balance: %v", err)
	}
}

func TestCheckOrganizationQuota_Credit(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:         "Test Org",
		Slug:         "test-org",
		Status:       "active",
		Balance:      decimal.Zero,
		CreditStatus: "approved",
		CreditLimit:  &creditLimit,
		CreditUsed:   decimal.NewFromInt(200),
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.checkOrganizationQuota(context.Background(), tenant.ID, 100, decimal.NewFromInt(500))
	if err != nil {
		t.Fatalf("should pass with credit: %v", err)
	}
}

func TestCheckOrganizationQuota_CreditExceeded(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:         "Test Org",
		Slug:         "test-org",
		Status:       "active",
		Balance:      decimal.Zero,
		CreditStatus: "approved",
		CreditLimit:  &creditLimit,
		CreditUsed:   decimal.NewFromInt(800),
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.checkOrganizationQuota(context.Background(), tenant.ID, 100, decimal.NewFromInt(300))
	if err == nil {
		t.Fatal("expected credit limit exceeded error")
	}
}

func TestCheckOrganizationQuota_NoPaymentSource(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	tenant := &entity.Tenant{
		Name:   "Test Org",
		Slug:   "test-org",
		Status: "active",
		Balance: decimal.Zero,
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.checkOrganizationQuota(context.Background(), tenant.ID, 100, decimal.NewFromInt(50))
	if err == nil {
		t.Fatal("expected no payment source error")
	}
}

func TestCheckOrganizationQuota_TenantSuspended(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	tenant := &entity.Tenant{
		Name:   "Test Org",
		Slug:   "test-org",
		Status: "suspended",
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.checkOrganizationQuota(context.Background(), tenant.ID, 100, decimal.NewFromInt(50))
	if err == nil {
		t.Fatal("expected tenant suspended error")
	}
}

func TestCheckOrganizationQuota_TenantNotFound(t *testing.T) {
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())
	err := service.checkOrganizationQuota(context.Background(), uuid.New(), 100, decimal.NewFromInt(50))
	if err == nil {
		t.Fatal("expected error for non-existent tenant")
	}
}

func TestDeductUsage_InvalidQuotaType(t *testing.T) {
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())
	apiKey := &entity.APIKey{QuotaType: "invalid", QuotaID: uuid.New()}
	err := service.DeductUsage(context.Background(), apiKey, 100, decimal.NewFromInt(5))
	if err == nil {
		t.Fatal("expected error for invalid quota type")
	}
}

// Individual deduction tests
func TestDeductIndividualUsage_FullTokenQuota(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:       uuid.New(),
		TokenQuota:   10000,
		TokensUsed:   2000,
		Balance:      decimal.NewFromInt(100),
		MonthlyCost:  decimal.Zero,
		Status:       "active",
	}
	userRepo.Create(context.Background(), quota)

	err := service.deductIndividualUsage(context.Background(), quota.ID, 3000, decimal.NewFromInt(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := userRepo.GetByID(context.Background(), quota.ID)
	if updated.TokensUsed != 5000 {
		t.Errorf("tokens used = %d, want 5000", updated.TokensUsed)
	}
	if !updated.MonthlyCost.Equals(decimal.NewFromInt(5)) {
		t.Errorf("monthly cost = %v, want 5", updated.MonthlyCost)
	}
}

func TestDeductIndividualUsage_PartialTokenQuota(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:       uuid.New(),
		TokenQuota:   5000,
		TokensUsed:   4000,
		Balance:      decimal.NewFromInt(100),
		MonthlyCost:  decimal.Zero,
		Status:       "active",
	}
	userRepo.Create(context.Background(), quota)

	err := service.deductIndividualUsage(context.Background(), quota.ID, 3000, decimal.NewFromInt(50))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := userRepo.GetByID(context.Background(), quota.ID)
	if updated.TokensUsed != 5000 {
		t.Errorf("tokens used should be at quota limit 5000, got %d", updated.TokensUsed)
	}
}

func TestDeductIndividualUsage_BalanceOnly(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:       uuid.New(),
		TokenQuota:   0,
		TokensUsed:   0,
		Balance:      decimal.NewFromInt(100),
		MonthlyCost:  decimal.Zero,
		Status:       "active",
	}
	userRepo.Create(context.Background(), quota)

	err := service.deductIndividualUsage(context.Background(), quota.ID, 500, decimal.NewFromInt(30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := userRepo.GetByID(context.Background(), quota.ID)
	if !updated.Balance.Equals(decimal.NewFromInt(70)) {
		t.Errorf("balance = %v, want 70", updated.Balance)
	}
}

func TestDeductIndividualUsage_Suspends(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:       uuid.New(),
		TokenQuota:   0,
		TokensUsed:   0,
		Balance:      decimal.NewFromInt(50),
		MonthlyCost:  decimal.Zero,
		Status:       "active",
	}
	userRepo.Create(context.Background(), quota)

	err := service.deductIndividualUsage(context.Background(), quota.ID, 500, decimal.NewFromInt(50))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := userRepo.GetByID(context.Background(), quota.ID)
	if updated.Status != "suspended" {
		t.Errorf("status = %v, want suspended", updated.Status)
	}
	if !updated.Balance.IsZero() {
		t.Errorf("balance = %v, want 0", updated.Balance)
	}
}

func TestDeductMemberUsage(t *testing.T) {
	memberRepo := NewMockMemberQuotaRepo()
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), memberRepo, tenantRepo)

	tenLimit := int64(10000)
	tenant := &entity.Tenant{
		Name:            "Test Org",
		Slug:            "test-org",
		Status:          "active",
		TokenLimit:      &tenLimit,
		TokensUsedMonth: 2000,
		Balance:         decimal.NewFromInt(500),
	}
	tenantRepo.Create(context.Background(), tenant)

	memberQuota := &entity.MemberQuota{
		UserID:   uuid.New(),
		TenantID: tenant.ID,
		Status:   "active",
		CostUsed: decimal.Zero,
	}
	memberRepo.Create(context.Background(), memberQuota)

	err := service.deductMemberUsage(context.Background(), memberQuota.ID, tenant.ID, 3000, decimal.NewFromInt(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := memberRepo.GetByID(context.Background(), memberQuota.ID)
	if updated.TokensUsed != 3000 {
		t.Errorf("member tokens used = %d, want 3000", updated.TokensUsed)
	}
	if !updated.CostUsed.Equals(decimal.NewFromInt(10)) {
		t.Errorf("member cost used = %v, want 10", updated.CostUsed)
	}
}

func TestDeductOrganizationUsage_WithinTokenQuota(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	tenLimit := int64(10000)
	tenant := &entity.Tenant{
		Name:            "Test Org",
		Slug:            "test-org",
		Status:          "active",
		TokenLimit:      &tenLimit,
		TokensUsedMonth: 2000,
		Balance:         decimal.NewFromInt(500),
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.deductOrganizationUsage(context.Background(), tenant.ID, 3000, decimal.NewFromInt(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if updated.TokensUsedMonth != 5000 {
		t.Errorf("tokens used = %d, want 5000", updated.TokensUsedMonth)
	}
	// Balance should not be deducted (within token quota)
	if !updated.Balance.Equals(decimal.NewFromInt(500)) {
		t.Errorf("balance should remain 500, got %v", updated.Balance)
	}
}

func TestDeductOrganizationUsage_Balance(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	tenant := &entity.Tenant{
		Name:            "Test Org",
		Slug:            "test-org",
		Status:          "active",
		TokenLimit:      nil,
		TokensUsedMonth: 0,
		Balance:         decimal.NewFromInt(500),
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.deductOrganizationUsage(context.Background(), tenant.ID, 2000, decimal.NewFromInt(200))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if !updated.Balance.Equals(decimal.NewFromInt(300)) {
		t.Errorf("balance = %v, want 300", updated.Balance)
	}
}

func TestDeductOrganizationUsage_Credit(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:         "Test Org",
		Slug:         "test-org",
		Status:       "active",
		Balance:      decimal.Zero,
		CreditStatus: "approved",
		CreditLimit:  &creditLimit,
		CreditUsed:   decimal.Zero,
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.deductOrganizationUsage(context.Background(), tenant.ID, 100, decimal.NewFromInt(200))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if !updated.CreditUsed.Equals(decimal.NewFromInt(200)) {
		t.Errorf("credit used = %v, want 200", updated.CreditUsed)
	}
}

func TestDeductOrganizationUsage_PartialBalanceCredit(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	creditLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name:         "Test Org",
		Slug:         "test-org",
		Status:       "active",
		Balance:      decimal.NewFromInt(100),
		CreditStatus: "approved",
		CreditLimit:  &creditLimit,
		CreditUsed:   decimal.Zero,
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.deductOrganizationUsage(context.Background(), tenant.ID, 100, decimal.NewFromInt(300))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if !updated.Balance.IsZero() {
		t.Errorf("balance should be 0, got %v", updated.Balance)
	}
	if !updated.CreditUsed.Equals(decimal.NewFromInt(300)) {
		t.Errorf("credit used should be 300 (full cost goes to credit due to balance zeroing before calc), got %v", updated.CreditUsed)
	}
}

func TestDeductOrganizationUsage_NoPaymentSource(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	tenant := &entity.Tenant{
		Name:   "Test Org",
		Slug:   "test-org",
		Status: "active",
		Balance: decimal.Zero,
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.deductOrganizationUsage(context.Background(), tenant.ID, 100, decimal.NewFromInt(50))
	if err == nil {
		t.Fatal("expected no payment source error")
	}

	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if updated.Status != "suspended" {
		t.Errorf("tenant should be suspended, got %v", updated.Status)
	}
}

func TestDeductOrganizationUsage_BalanceInsufficientNoCredit(t *testing.T) {
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), tenantRepo)

	tenant := &entity.Tenant{
		Name:   "Test Org",
		Slug:   "test-org",
		Status: "active",
		Balance: decimal.NewFromInt(50),
	}
	tenantRepo.Create(context.Background(), tenant)

	err := service.deductOrganizationUsage(context.Background(), tenant.ID, 100, decimal.NewFromInt(200))
	if err == nil {
		t.Fatal("expected no payment source error when balance insufficient and no credit")
	}

	updated, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if updated.Status != "suspended" {
		t.Errorf("tenant should be suspended, got %v", updated.Status)
	}
}

func TestGetQuotaInfo_Individual(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:       uuid.New(),
		TokenQuota:   10000,
		TokensUsed:   3000,
		Balance:      decimal.NewFromInt(200),
		Status:       "active",
	}
	userRepo.Create(context.Background(), quota)

	apiKey := &entity.APIKey{QuotaType: "individual", QuotaID: quota.ID}
	info, err := service.GetQuotaInfo(context.Background(), apiKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.QuotaType != "individual" {
		t.Errorf("quota type = %v, want individual", info.QuotaType)
	}
	if info.TokenRemaining != 7000 {
		t.Errorf("token remaining = %d, want 7000", info.TokenRemaining)
	}
	if !info.Balance.Equals(decimal.NewFromInt(200)) {
		t.Errorf("balance = %v, want 200", info.Balance)
	}
}

func TestGetQuotaInfo_Individual_NegativeRemaining(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:       uuid.New(),
		TokenQuota:   1000,
		TokensUsed:   2000,
		Balance:      decimal.Zero,
		Status:       "active",
	}
	userRepo.Create(context.Background(), quota)

	apiKey := &entity.APIKey{QuotaType: "individual", QuotaID: quota.ID}
	info, err := service.GetQuotaInfo(context.Background(), apiKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TokenRemaining != 0 {
		t.Errorf("token remaining should floor to 0, got %d", info.TokenRemaining)
	}
}

func TestGetQuotaInfo_Member(t *testing.T) {
	memberRepo := NewMockMemberQuotaRepo()
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), memberRepo, tenantRepo)

	tenLimit := int64(50000)
	creditLimit := decimal.NewFromInt(5000)
	tenant := &entity.Tenant{
		Name:            "Test Org",
		Slug:            "test-org",
		Status:          "active",
		TokenLimit:      &tenLimit,
		TokensUsedMonth: 15000,
		Balance:         decimal.NewFromInt(1000),
		CreditStatus:    "approved",
		CreditLimit:     &creditLimit,
		CreditUsed:      decimal.NewFromInt(2000),
	}
	tenantRepo.Create(context.Background(), tenant)

	memLimit := int64(10000)
	costLimit := decimal.NewFromInt(500)
	memberQuota := &entity.MemberQuota{
		UserID:          uuid.New(),
		TenantID:        tenant.ID,
		TokenQuotaLimit: &memLimit,
		TokensUsed:      3000,
		CostLimit:       &costLimit,
		CostUsed:        decimal.NewFromInt(150),
		Status:          "active",
	}
	memberRepo.Create(context.Background(), memberQuota)

	apiKey := &entity.APIKey{QuotaType: "member", QuotaID: memberQuota.ID, TenantID: tenant.ID}
	info, err := service.GetQuotaInfo(context.Background(), apiKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.QuotaType != "member" {
		t.Errorf("quota type = %v, want member", info.QuotaType)
	}
	// Member token remaining = member limit - member used = 10000 - 3000 = 7000
	if info.TokenRemaining != 7000 {
		t.Errorf("token remaining = %d, want 7000", info.TokenRemaining)
	}
	if info.CreditLimit == nil || !info.CreditLimit.Equals(creditLimit) {
		t.Errorf("credit limit mismatch")
	}
	if info.CostLimit == nil || !info.CostLimit.Equals(costLimit) {
		t.Errorf("cost limit mismatch")
	}
}

func TestGetQuotaInfo_Member_NoMemberLimit(t *testing.T) {
	memberRepo := NewMockMemberQuotaRepo()
	tenantRepo := NewMockQuotaTenantRepo()
	service := NewQuotaService(NewMockQuotaUserRepo(), memberRepo, tenantRepo)

	tenLimit := int64(50000)
	tenant := &entity.Tenant{
		Name:            "Test Org",
		Slug:            "test-org",
		Status:          "active",
		TokenLimit:      &tenLimit,
		TokensUsedMonth: 10000,
		Balance:         decimal.NewFromInt(1000),
	}
	tenantRepo.Create(context.Background(), tenant)

	memberQuota := &entity.MemberQuota{
		UserID:   uuid.New(),
		TenantID: tenant.ID,
		Status:   "active",
	}
	memberRepo.Create(context.Background(), memberQuota)

	apiKey := &entity.APIKey{QuotaType: "member", QuotaID: memberQuota.ID, TenantID: tenant.ID}
	info, err := service.GetQuotaInfo(context.Background(), apiKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No member limit, uses tenant token remaining: 50000 - 10000 = 40000
	if info.TokenRemaining != 40000 {
		t.Errorf("token remaining = %d, want 40000", info.TokenRemaining)
	}
}

func TestGetQuotaInfo_InvalidType(t *testing.T) {
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())
	apiKey := &entity.APIKey{QuotaType: "invalid", QuotaID: uuid.New()}
	_, err := service.GetQuotaInfo(context.Background(), apiKey)
	if err == nil {
		t.Fatal("expected error for invalid quota type")
	}
}

func TestGetQuotaInfo_Individual_QuotaNotFound(t *testing.T) {
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())
	apiKey := &entity.APIKey{QuotaType: "individual", QuotaID: uuid.New()}
	_, err := service.GetQuotaInfo(context.Background(), apiKey)
	if err == nil {
		t.Fatal("expected error for non-existent quota")
	}
}

func TestDeductIndividualUsage_QuotaExhaustedNoBalance(t *testing.T) {
	userRepo := NewMockQuotaUserRepo()
	service := NewQuotaService(userRepo, NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())

	quota := &entity.UserQuota{
		UserID:       uuid.New(),
		TokenQuota:   5000,
		TokensUsed:   5000, // quota exhausted
		Balance:      decimal.NewFromInt(20),
		MonthlyCost:  decimal.Zero,
		Status:       "active",
	}
	userRepo.Create(context.Background(), quota)

	err := service.deductIndividualUsage(context.Background(), quota.ID, 1000, decimal.NewFromInt(20))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := userRepo.GetByID(context.Background(), quota.ID)
	if updated.Status != "suspended" {
		t.Errorf("status = %v, want suspended (balance exhausted)", updated.Status)
	}
}

func TestDeductOrganizationUsage_TenantNotFound(t *testing.T) {
	service := NewQuotaService(NewMockQuotaUserRepo(), NewMockMemberQuotaRepo(), NewMockQuotaTenantRepo())
	err := service.deductOrganizationUsage(context.Background(), uuid.New(), 100, decimal.NewFromInt(10))
	if err == nil {
		t.Fatal("expected error for non-existent tenant")
	}
}