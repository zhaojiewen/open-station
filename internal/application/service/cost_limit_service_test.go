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

// Mock repos for CostLimitService
type MockCLTenantRepo struct {
	tenants map[uuid.UUID]*entity.Tenant
}

func NewMockCLTenantRepo() *MockCLTenantRepo {
	return &MockCLTenantRepo{tenants: make(map[uuid.UUID]*entity.Tenant)}
}

func (m *MockCLTenantRepo) Create(ctx context.Context, t *entity.Tenant) error {
	t.ID = uuid.New()
	m.tenants[t.ID] = t
	return nil
}
func (m *MockCLTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("not found")
}
func (m *MockCLTenantRepo) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	return nil, errors.New("not found")
}
func (m *MockCLTenantRepo) Update(ctx context.Context, t *entity.Tenant) error {
	m.tenants[t.ID] = t
	return nil
}
func (m *MockCLTenantRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLTenantRepo) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}
func (m *MockCLTenantRepo) ListByCreditStatus(ctx context.Context, cs string, p, ps int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}
func (m *MockCLTenantRepo) UpdateBalance(ctx context.Context, id uuid.UUID, a decimal.Decimal) error {
	if t, ok := m.tenants[id]; ok {
		t.Balance = t.Balance.Add(a)
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLTenantRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	if t, ok := m.tenants[id]; ok {
		return t.Balance, nil
	}
	return decimal.Zero, errors.New("not found")
}
func (m *MockCLTenantRepo) DeductBalance(ctx context.Context, id uuid.UUID, a decimal.Decimal) error {
	if t, ok := m.tenants[id]; ok {
		if t.Balance.LessThan(a) {
			return errors.New("insufficient balance")
		}
		t.Balance = t.Balance.Sub(a)
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLTenantRepo) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, a decimal.Decimal) error {
	if t, ok := m.tenants[id]; ok {
		t.BudgetUsedMonth = t.BudgetUsedMonth.Add(a)
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLTenantRepo) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLTenantRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}
func (m *MockCLTenantRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	if t, ok := m.tenants[id]; ok {
		t.TokensUsedMonth += tokens
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLTenantRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error { return nil }

type MockCLUserRepo struct {
	users map[uuid.UUID]*entity.User
}

func NewMockCLUserRepo() *MockCLUserRepo {
	return &MockCLUserRepo{users: make(map[uuid.UUID]*entity.User)}
}

func (m *MockCLUserRepo) Create(ctx context.Context, u *entity.User) error {
	u.ID = uuid.New()
	m.users[u.ID] = u
	return nil
}
func (m *MockCLUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (m *MockCLUserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return nil, errors.New("not found")
}
func (m *MockCLUserRepo) GetByVerificationToken(ctx context.Context, token string) (*entity.User, error) {
	return nil, errors.New("not found")
}
func (m *MockCLUserRepo) Update(ctx context.Context, u *entity.User) error {
	m.users[u.ID] = u
	return nil
}
func (m *MockCLUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLUserRepo) List(ctx context.Context, tid uuid.UUID, p, ps int) ([]entity.User, int64, error) {
	return nil, 0, nil
}
func (m *MockCLUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLUserRepo) IncrementMonthlyBudgetUsed(ctx context.Context, id uuid.UUID, a decimal.Decimal) error {
	if u, ok := m.users[id]; ok {
		u.BudgetUsedMonth = u.BudgetUsedMonth.Add(a)
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLUserRepo) IncrementDailyBudgetUsed(ctx context.Context, id uuid.UUID, a decimal.Decimal) error {
	if u, ok := m.users[id]; ok {
		u.BudgetUsedToday = u.BudgetUsedToday.Add(a)
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLUserRepo) ResetMonthlyBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLUserRepo) ResetDailyBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLUserRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, int64, error) {
	return decimal.Zero, decimal.Zero, 0, nil
}
func (m *MockCLUserRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockCLUserRepo) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLUserRepo) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLUserRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (m *MockCLUserRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockCLUserRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

type MockCLAPIKeyRepo struct {
	keys map[uuid.UUID]*entity.APIKey
}

func NewMockCLAPIKeyRepo() *MockCLAPIKeyRepo {
	return &MockCLAPIKeyRepo{keys: make(map[uuid.UUID]*entity.APIKey)}
}

func (m *MockCLAPIKeyRepo) Create(ctx context.Context, k *entity.APIKey) error {
	k.ID = uuid.New()
	m.keys[k.ID] = k
	return nil
}
func (m *MockCLAPIKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.APIKey, error) {
	if k, ok := m.keys[id]; ok {
		return k, nil
	}
	return nil, errors.New("not found")
}
func (m *MockCLAPIKeyRepo) GetByHash(ctx context.Context, h string) (*entity.APIKey, error) {
	return nil, errors.New("not found")
}
func (m *MockCLAPIKeyRepo) GetWithRelations(ctx context.Context, h string) (*entity.APIKey, *entity.User, *entity.Tenant, error) {
	return nil, nil, nil, errors.New("not found")
}
func (m *MockCLAPIKeyRepo) GetByKeyPrefix(ctx context.Context, p string) ([]entity.APIKey, error) {
	return nil, nil
}
func (m *MockCLAPIKeyRepo) Update(ctx context.Context, k *entity.APIKey) error {
	m.keys[k.ID] = k
	return nil
}
func (m *MockCLAPIKeyRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLAPIKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLAPIKeyRepo) List(ctx context.Context, uid uuid.UUID) ([]entity.APIKey, error) { return nil, nil }
func (m *MockCLAPIKeyRepo) ListByTenant(ctx context.Context, tid uuid.UUID, s string) ([]entity.APIKey, error) {
	return nil, nil
}
func (m *MockCLAPIKeyRepo) ListAll(ctx context.Context) ([]entity.APIKey, error) { return nil, nil }
func (m *MockCLAPIKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLAPIKeyRepo) UpdateTokenUsage(ctx context.Context, id uuid.UUID, tokens int64) error {
	if k, ok := m.keys[id]; ok {
		k.UsedTokensThisMonth += tokens
		k.TokensUsedToday += tokens
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLAPIKeyRepo) IncrementMonthlyCostUsed(ctx context.Context, id uuid.UUID, a decimal.Decimal) error {
	if k, ok := m.keys[id]; ok {
		k.MonthlyCostUsed = k.MonthlyCostUsed.Add(a)
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLAPIKeyRepo) IncrementDailyCostUsed(ctx context.Context, id uuid.UUID, a decimal.Decimal) error {
	if k, ok := m.keys[id]; ok {
		k.DailyCostUsed = k.DailyCostUsed.Add(a)
		return nil
	}
	return errors.New("not found")
}
func (m *MockCLAPIKeyRepo) ResetMonthlyCostUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLAPIKeyRepo) ResetDailyCostUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLAPIKeyRepo) GetCostUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, int64, int64, error) {
	return decimal.Zero, decimal.Zero, 0, 0, nil
}
func (m *MockCLAPIKeyRepo) IncrementDailyTokens(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockCLAPIKeyRepo) ResetDailyTokens(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockCLAPIKeyRepo) UpdateProviderUsage(ctx context.Context, id uuid.UUID, provider string, tokens int64, cost decimal.Decimal) error {
	return nil
}

// --- CostLimitService Tests ---

func TestNewCostLimitService(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	if s == nil {
		t.Error("NewCostLimitService should not return nil")
	}
}

func TestCheckCostLimits_Pass(t *testing.T) {
	tr := NewMockCLTenantRepo()
	ur := NewMockCLUserRepo()
	kr := NewMockCLAPIKeyRepo()
	s := NewCostLimitService(tr, ur, kr)

	tenant := &entity.Tenant{
		Name: "Test", Slug: "test", Status: "active", Balance: decimal.NewFromInt(1000),
	}
	tr.Create(context.Background(), tenant)
	user := &entity.User{
		Name: "User", Status: "active",
	}
	ur.Create(context.Background(), user)
	apiKey := &entity.APIKey{
		Name: "Key", Status: "active",
	}
	kr.Create(context.Background(), apiKey)

	err := s.CheckCostLimits(context.Background(), apiKey.ID, user.ID, tenant.ID, decimal.NewFromInt(100), 1000)
	if err != nil {
		t.Fatalf("should pass: %v", err)
	}
}

func TestCheckCostLimits_TenantNotFound(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uuid.New(), uuid.New(), decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckCostLimits_TenantSuspended(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "suspended", Balance: decimal.NewFromInt(1000)})
	tid := tr.tenants
	// Actually, let's get the ID properly
	ts := make([]uuid.UUID, 0, 1)
	for id := range tr.tenants {
		ts = append(ts, id)
		break
	}
	s := NewCostLimitService(tr, NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	_ = tid // not used
	err := s.CheckCostLimits(context.Background(), uuid.New(), uuid.New(), ts[0], decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected tenant suspended error")
	}
}

func TestCheckCostLimits_TenantDeleted(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "deleted", Balance: decimal.NewFromInt(1000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	s := NewCostLimitService(tr, NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uuid.New(), tid, decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected tenant deleted error")
	}
}

func TestCheckCostLimits_InsufficientBalance(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(10)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	s := NewCostLimitService(tr, NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uuid.New(), tid, decimal.NewFromInt(100), 1)
	if err == nil {
		t.Fatal("expected insufficient balance error")
	}
}

func TestCheckCostLimits_TenantBudgetExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	mb := decimal.NewFromInt(1000)
	tr.Create(context.Background(), &entity.Tenant{
		Name: "T", Slug: "t", Status: "active",
		Balance: decimal.NewFromInt(5000),
		MonthlyBudgetLimit: &mb,
		BudgetUsedMonth:    decimal.NewFromInt(800),
	})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	s := NewCostLimitService(tr, NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	err := s.CheckCostLimits(context.Background(), uuid.New(), uid, tid, decimal.NewFromInt(300), 1)
	if err == nil {
		t.Fatal("expected tenant monthly budget exceeded error")
	}
}

func TestCheckCostLimits_TenantTokenExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tl := int64(10000)
	tr.Create(context.Background(), &entity.Tenant{
		Name: "T", Slug: "t", Status: "active",
		Balance:         decimal.NewFromInt(5000),
		TokenLimit:      &tl,
		TokensUsedMonth: 8000,
	})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	s := NewCostLimitService(tr, ur, NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uid, tid, decimal.NewFromInt(100), 3000)
	if err == nil {
		t.Fatal("expected quota exceeded error")
	}
}

func TestCheckCostLimits_UserNotFound(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(1000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	s := NewCostLimitService(tr, NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uuid.New(), tid, decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected user not found error")
	}
}

func TestCheckCostLimits_UserInactive(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(1000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "inactive"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	s := NewCostLimitService(tr, ur, NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uid, tid, decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected user inactive error")
	}
}

func TestCheckCostLimits_UserMonthlyBudgetExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	mb := decimal.NewFromInt(500)
	ur.Create(context.Background(), &entity.User{
		Name: "U", Status: "active",
		MonthlyBudget:  &mb,
		BudgetUsedMonth: decimal.NewFromInt(400),
	})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	s := NewCostLimitService(tr, ur, NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uid, tid, decimal.NewFromInt(200), 1)
	if err == nil {
		t.Fatal("expected user monthly budget exceeded error")
	}
}

func TestCheckCostLimits_UserDailyBudgetExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	db := decimal.NewFromInt(100)
	ur.Create(context.Background(), &entity.User{
		Name: "U", Status: "active",
		DailyBudget:    &db,
		BudgetUsedToday: decimal.NewFromInt(80),
	})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	s := NewCostLimitService(tr, ur, NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uid, tid, decimal.NewFromInt(30), 1)
	if err == nil {
		t.Fatal("expected user daily budget exceeded error")
	}
}

func TestCheckCostLimits_UserTokenQuotaExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	tq := int64(5000)
	ur.Create(context.Background(), &entity.User{
		Name: "U", Status: "active",
		TokenQuota:      &tq,
		TokensUsedMonth: 4000,
	})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	s := NewCostLimitService(tr, ur, NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uid, tid, decimal.NewFromInt(10), 2000)
	if err == nil {
		t.Fatal("expected quota exceeded error")
	}
}

func TestCheckCostLimits_APIKeyRevoked(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	kr := NewMockCLAPIKeyRepo()
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "revoked"})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
		break
	}
	s := NewCostLimitService(tr, ur, kr)
	err := s.CheckCostLimits(context.Background(), kid, uid, tid, decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected API key revoked error")
	}
}

func TestCheckCostLimits_APIKeyExpired(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	kr := NewMockCLAPIKeyRepo()
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "expired"})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
		break
	}
	s := NewCostLimitService(tr, ur, kr)
	err := s.CheckCostLimits(context.Background(), kid, uid, tid, decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected API key expired error")
	}
}

func TestCheckCostLimits_APIKeyExpiredByDate(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	kr := NewMockCLAPIKeyRepo()
	past := time.Now().Add(-24 * time.Hour)
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active", ExpiresAt: &past})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
		break
	}
	s := NewCostLimitService(tr, ur, kr)
	err := s.CheckCostLimits(context.Background(), kid, uid, tid, decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected API key expired error (by date)")
	}
}

func TestCheckCostLimits_APIKeyPerRequestExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	kr := NewMockCLAPIKeyRepo()
	prl := decimal.NewFromInt(50)
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active", PerRequestCostLimit: &prl})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
		break
	}
	s := NewCostLimitService(tr, ur, kr)
	err := s.CheckCostLimits(context.Background(), kid, uid, tid, decimal.NewFromInt(100), 1)
	if err == nil {
		t.Fatal("expected per-request cost exceeded error")
	}
}

func TestCheckCostLimits_APIKeyMonthlyCostExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	kr := NewMockCLAPIKeyRepo()
	mcl := decimal.NewFromInt(500)
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active", MonthlyCostLimit: &mcl, MonthlyCostUsed: decimal.NewFromInt(400)})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
		break
	}
	s := NewCostLimitService(tr, ur, kr)
	err := s.CheckCostLimits(context.Background(), kid, uid, tid, decimal.NewFromInt(200), 1)
	if err == nil {
		t.Fatal("expected monthly cost exceeded error")
	}
}

func TestCheckCostLimits_APIKeyDailyCostExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	kr := NewMockCLAPIKeyRepo()
	dcl := decimal.NewFromInt(100)
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active", DailyCostLimit: &dcl, DailyCostUsed: decimal.NewFromInt(80)})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
		break
	}
	s := NewCostLimitService(tr, ur, kr)
	err := s.CheckCostLimits(context.Background(), kid, uid, tid, decimal.NewFromInt(30), 1)
	if err == nil {
		t.Fatal("expected daily cost exceeded error")
	}
}

func TestCheckCostLimits_APIKeyMonthlyTokenExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	kr := NewMockCLAPIKeyRepo()
	mtl := int64(5000)
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active", MonthlyTokenLimit: &mtl, UsedTokensThisMonth: 4000})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
		break
	}
	s := NewCostLimitService(tr, ur, kr)
	err := s.CheckCostLimits(context.Background(), kid, uid, tid, decimal.NewFromInt(10), 2000)
	if err == nil {
		t.Fatal("expected API key token limit exceeded error")
	}
}

func TestCheckCostLimits_APIKeyDailyTokenExceeded(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	kr := NewMockCLAPIKeyRepo()
	dtl := int64(1000)
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active", TokenLimitPerDay: &dtl, TokensUsedToday: 800})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
		break
	}
	s := NewCostLimitService(tr, ur, kr)
	err := s.CheckCostLimits(context.Background(), kid, uid, tid, decimal.NewFromInt(1), 500)
	if err == nil {
		t.Fatal("expected daily token exceeded error")
	}
}

func TestCheckCostLimits_APIKeyNotFound(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
		break
	}
	s := NewCostLimitService(tr, ur, NewMockCLAPIKeyRepo())
	err := s.CheckCostLimits(context.Background(), uuid.New(), uid, tid, decimal.NewFromInt(1), 1)
	if err == nil {
		t.Fatal("expected invalid API key error")
	}
}

func TestRecordCost(t *testing.T) {
	tr := NewMockCLTenantRepo()
	ur := NewMockCLUserRepo()
	kr := NewMockCLAPIKeyRepo()
	s := NewCostLimitService(tr, ur, kr)

	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(5000)})
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active"})

	var tid, uid, kid uuid.UUID
	for id := range tr.tenants {
		tid = id
	}
	for id := range ur.users {
		uid = id
	}
	for id := range kr.keys {
		kid = id
	}

	err := s.RecordCost(context.Background(), kid, uid, tid, decimal.NewFromInt(100), 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ten, _ := tr.GetByID(context.Background(), tid)
	if !ten.Balance.Equals(decimal.NewFromInt(4900)) {
		t.Errorf("tenant balance = %v, want 4900", ten.Balance)
	}
	if !ten.BudgetUsedMonth.Equals(decimal.NewFromInt(100)) {
		t.Errorf("budget used = %v, want 100", ten.BudgetUsedMonth)
	}
	if ten.TokensUsedMonth != 500 {
		t.Errorf("tokens used = %d, want 500", ten.TokensUsedMonth)
	}
}

func TestRecordCost_InsufficientBalance(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active", Balance: decimal.NewFromInt(10)})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
		break
	}
	s := NewCostLimitService(tr, NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.RecordCost(context.Background(), uuid.New(), uuid.New(), tid, decimal.NewFromInt(100), 1)
	if err == nil {
		t.Fatal("expected insufficient balance error")
	}
}

func TestGetCostUsageInfo(t *testing.T) {
	tr := NewMockCLTenantRepo()
	ur := NewMockCLUserRepo()
	kr := NewMockCLAPIKeyRepo()
	s := NewCostLimitService(tr, ur, kr)

	mb := decimal.NewFromInt(1000)
	tl := int64(10000)
	tr.Create(context.Background(), &entity.Tenant{
		Name: "T", Slug: "t", Status: "active",
		Balance:            decimal.NewFromInt(500),
		MonthlyBudgetLimit: &mb,
		BudgetUsedMonth:    decimal.NewFromInt(300),
		TokenLimit:         &tl,
		TokensUsedMonth:    2000,
	})

	umb := decimal.NewFromInt(500)
	utq := int64(5000)
	ur.Create(context.Background(), &entity.User{
		Name: "U", Status: "active",
		MonthlyBudget:  &umb,
		BudgetUsedMonth: decimal.NewFromInt(200),
		DailyBudget:     nil,
		BudgetUsedToday: decimal.Zero,
		TokenQuota:      &utq,
		TokensUsedMonth: 1000,
	})

	mcl := decimal.NewFromInt(200)
	dcl := decimal.NewFromInt(50)
	kr.Create(context.Background(), &entity.APIKey{
		Name: "K", Status: "active",
		MonthlyCostLimit: &mcl,
		MonthlyCostUsed:  decimal.NewFromInt(80),
		DailyCostLimit:   &dcl,
		DailyCostUsed:    decimal.NewFromInt(20),
	})

	var tid, uid, kid uuid.UUID
	for id := range tr.tenants {
		tid = id
	}
	for id := range ur.users {
		uid = id
	}
	for id := range kr.keys {
		kid = id
	}

	info, err := s.GetCostUsageInfo(context.Background(), kid, uid, tid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Tenant.BudgetPercent != 30 {
		t.Errorf("tenant budget percent = %d, want 30", info.Tenant.BudgetPercent)
	}
	if info.User.MonthlyPercent != 40 {
		t.Errorf("user monthly percent = %d, want 40", info.User.MonthlyPercent)
	}
	if info.APIKey.MonthlyPercent != 40 {
		t.Errorf("apikey monthly percent = %d, want 40", info.APIKey.MonthlyPercent)
	}
	if info.APIKey.DailyPercent != 40 {
		t.Errorf("apikey daily percent = %d, want 40", info.APIKey.DailyPercent)
	}
}

func TestGetCostUsageInfo_TenantNotFound(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	_, err := s.GetCostUsageInfo(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResetDailyCosts(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.ResetDailyCosts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResetMonthlyCosts(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.ResetMonthlyCosts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUserBudget(t *testing.T) {
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
	}
	s := NewCostLimitService(NewMockCLTenantRepo(), ur, NewMockCLAPIKeyRepo())

	err := s.SetUserBudget(context.Background(), uid, decimal.NewFromInt(500), decimal.NewFromInt(100), 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := ur.GetByID(context.Background(), uid)
	if updated.MonthlyBudget == nil || !updated.MonthlyBudget.Equals(decimal.NewFromInt(500)) {
		t.Error("monthly budget not set correctly")
	}
	if updated.DailyBudget == nil || !updated.DailyBudget.Equals(decimal.NewFromInt(100)) {
		t.Error("daily budget not set correctly")
	}
	if updated.TokenQuota == nil || *updated.TokenQuota != 10000 {
		t.Error("token quota not set correctly")
	}
}

func TestSetUserBudget_ZeroValues(t *testing.T) {
	ur := NewMockCLUserRepo()
	ur.Create(context.Background(), &entity.User{Name: "U", Status: "active"})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
	}
	s := NewCostLimitService(NewMockCLTenantRepo(), ur, NewMockCLAPIKeyRepo())

	err := s.SetUserBudget(context.Background(), uid, decimal.Zero, decimal.Zero, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUserBudget_NotFound(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.SetUserBudget(context.Background(), uuid.New(), decimal.NewFromInt(100), decimal.Zero, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetAPIKeyCostLimit(t *testing.T) {
	kr := NewMockCLAPIKeyRepo()
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active"})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
	}
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), kr)

	err := s.SetAPIKeyCostLimit(context.Background(), kid,
		decimal.NewFromInt(1000), decimal.NewFromInt(200), decimal.NewFromInt(50),
		50000, 10000,
		80, 90, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := kr.GetByID(context.Background(), kid)
	if updated.MonthlyCostLimit == nil || !updated.MonthlyCostLimit.Equals(decimal.NewFromInt(1000)) {
		t.Error("monthly cost limit not set")
	}
	if updated.AlertThreshold1 != 80 {
		t.Errorf("alert threshold1 = %d, want 80", updated.AlertThreshold1)
	}
}

func TestSetAPIKeyCostLimit_ZeroValues(t *testing.T) {
	kr := NewMockCLAPIKeyRepo()
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active"})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
	}
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), kr)

	err := s.SetAPIKeyCostLimit(context.Background(), kid,
		decimal.Zero, decimal.Zero, decimal.Zero,
		0, 0,
		0, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetAPIKeyCostLimit_NotFound(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.SetAPIKeyCostLimit(context.Background(), uuid.New(), decimal.Zero, decimal.Zero, decimal.Zero, 0, 0, 0, 0, 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetTenantBudget(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active"})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
	}
	s := NewCostLimitService(tr, NewMockCLUserRepo(), NewMockCLAPIKeyRepo())

	err := s.SetTenantBudget(context.Background(), tid, decimal.NewFromInt(10000), 100000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := tr.GetByID(context.Background(), tid)
	if updated.MonthlyBudgetLimit == nil || !updated.MonthlyBudgetLimit.Equals(decimal.NewFromInt(10000)) {
		t.Error("monthly budget limit not set")
	}
	if updated.TokenLimit == nil || *updated.TokenLimit != 100000 {
		t.Error("token limit not set")
	}
}

func TestSetTenantBudget_ZeroValues(t *testing.T) {
	tr := NewMockCLTenantRepo()
	tr.Create(context.Background(), &entity.Tenant{Name: "T", Slug: "t", Status: "active"})
	var tid uuid.UUID
	for id := range tr.tenants {
		tid = id
	}
	s := NewCostLimitService(tr, NewMockCLUserRepo(), NewMockCLAPIKeyRepo())

	err := s.SetTenantBudget(context.Background(), tid, decimal.Zero, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetTenantBudget_NotFound(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	err := s.SetTenantBudget(context.Background(), uuid.New(), decimal.NewFromInt(100), 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetUserBudgetUsage(t *testing.T) {
	ur := NewMockCLUserRepo()
	mb := decimal.NewFromInt(500)
	tq := int64(10000)
	ur.Create(context.Background(), &entity.User{
		Name: "U", Status: "active",
		MonthlyBudget:  &mb,
		BudgetUsedMonth: decimal.NewFromInt(200),
		TokenQuota:      &tq,
		TokensUsedMonth: 3000,
	})
	var uid uuid.UUID
	for id := range ur.users {
		uid = id
	}
	s := NewCostLimitService(NewMockCLTenantRepo(), ur, NewMockCLAPIKeyRepo())

	usage, err := s.GetUserBudgetUsage(context.Background(), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.MonthlyBudget == nil || !usage.MonthlyBudget.Equals(decimal.NewFromInt(500)) {
		t.Error("monthly budget mismatch")
	}
	if usage.TokensUsed != 3000 {
		t.Errorf("tokens used = %d, want 3000", usage.TokensUsed)
	}
}

func TestGetUserBudgetUsage_NotFound(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	_, err := s.GetUserBudgetUsage(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetAPIKeyCostUsage(t *testing.T) {
	kr := NewMockCLAPIKeyRepo()
	mcl := decimal.NewFromInt(500)
	kr.Create(context.Background(), &entity.APIKey{
		Name: "K", Status: "active",
		MonthlyCostLimit:  &mcl,
		MonthlyCostUsed:   decimal.NewFromInt(200),
		UsedTokensThisMonth: 5000,
	})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
	}
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), kr)

	usage, err := s.GetAPIKeyCostUsage(context.Background(), kid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.MonthlyCostLimit == nil || !usage.MonthlyCostLimit.Equals(decimal.NewFromInt(500)) {
		t.Error("monthly cost limit mismatch")
	}
	if usage.MonthlyTokensUsed != 5000 {
		t.Errorf("monthly tokens used = %d, want 5000", usage.MonthlyTokensUsed)
	}
}

func TestGetAPIKeyCostUsage_NotFound(t *testing.T) {
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), NewMockCLAPIKeyRepo())
	_, err := s.GetAPIKeyCostUsage(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCostSummary(t *testing.T) {
	tr := NewMockCLTenantRepo()
	ur := NewMockCLUserRepo()
	kr := NewMockCLAPIKeyRepo()
	s := NewCostLimitService(tr, ur, kr)

	tr.Create(context.Background(), &entity.Tenant{
		Name: "T", Slug: "t", Status: "active",
		Balance:         decimal.NewFromInt(500),
		BudgetUsedMonth: decimal.NewFromInt(300),
	})
	ur.Create(context.Background(), &entity.User{
		Name: "U", Status: "active",
		BudgetUsedMonth: decimal.NewFromInt(150),
	})
	kr.Create(context.Background(), &entity.APIKey{
		Name: "K", Status: "active",
		MonthlyCostUsed: decimal.NewFromInt(80),
	})

	var tid, uid, kid uuid.UUID
	for id := range tr.tenants {
		tid = id
	}
	for id := range ur.users {
		uid = id
	}
	for id := range kr.keys {
		kid = id
	}

	summary, err := s.GetCostSummary(context.Background(), kid, uid, tid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summary.TenantBalance.Equals(decimal.NewFromInt(500)) {
		t.Errorf("tenant balance = %v, want 500", summary.TenantBalance)
	}
	if !summary.TenantMonthlyUsed.Equals(decimal.NewFromInt(300)) {
		t.Errorf("tenant monthly used = %v, want 300", summary.TenantMonthlyUsed)
	}
	if !summary.UserMonthlyUsed.Equals(decimal.NewFromInt(150)) {
		t.Errorf("user monthly used = %v, want 150", summary.UserMonthlyUsed)
	}
	if !summary.APIKeyMonthlyCostUsed.Equals(decimal.NewFromInt(80)) {
		t.Errorf("apikey monthly cost = %v, want 80", summary.APIKeyMonthlyCostUsed)
	}
}

func TestRecordCost_DeductBalanceError(t *testing.T) {
	tr := &mockCLTenantDeductError{}
	ur := NewMockCLUserRepo()
	kr := NewMockCLAPIKeyRepo()
	s := NewCostLimitService(tr, ur, kr)
	err := s.RecordCost(context.Background(), uuid.New(), uuid.New(), uuid.New(), decimal.NewFromInt(10), 100)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRecordCost_IncrementBudgetError(t *testing.T) {
	tr := &mockCLTenantBudgetError{}
	ur := NewMockCLUserRepo()
	kr := NewMockCLAPIKeyRepo()
	s := NewCostLimitService(tr, ur, kr)
	err := s.RecordCost(context.Background(), uuid.New(), uuid.New(), uuid.New(), decimal.NewFromInt(10), 100)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRecordCost_UserMonthlyBudgetError(t *testing.T) {
	tr := NewMockCLTenantRepo()
	ur := &mockCLUserBudgetError{}
	kr := NewMockCLAPIKeyRepo()
	s := NewCostLimitService(tr, ur, kr)
	err := s.RecordCost(context.Background(), uuid.New(), uuid.New(), uuid.New(), decimal.NewFromInt(10), 100)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRecordCost_APIKeyError(t *testing.T) {
	tr := NewMockCLTenantRepo()
	ur := NewMockCLUserRepo()
	kr := &mockCLAPIKeyError{}
	s := NewCostLimitService(tr, ur, kr)
	err := s.RecordCost(context.Background(), uuid.New(), uuid.New(), uuid.New(), decimal.NewFromInt(10), 100)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Error-injecting mock tenant for DeductBalance failure
type mockCLTenantDeductError struct{ MockCLTenantRepo }

func (m *mockCLTenantDeductError) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return errors.New("balance deduction failed")
}

// Error-injecting mock tenant for IncrementBudgetUsed failure
type mockCLTenantBudgetError struct{ MockCLTenantRepo }

func (m *mockCLTenantBudgetError) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return errors.New("budget increment failed")
}

// Error-injecting mock user for budget error
type mockCLUserBudgetError struct{ MockCLUserRepo }

func (m *mockCLUserBudgetError) IncrementMonthlyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return errors.New("user budget error")
}

// Error-injecting mock APIKey for cost used error
type mockCLAPIKeyError struct{ MockCLAPIKeyRepo }

func (m *mockCLAPIKeyError) IncrementMonthlyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return errors.New("api key cost error")
}

func TestGetCostSummary_PartialIDs(t *testing.T) {
	kr := NewMockCLAPIKeyRepo()
	kr.Create(context.Background(), &entity.APIKey{Name: "K", Status: "active"})
	var kid uuid.UUID
	for id := range kr.keys {
		kid = id
	}
	s := NewCostLimitService(NewMockCLTenantRepo(), NewMockCLUserRepo(), kr)

	summary, err := s.GetCostSummary(context.Background(), kid, uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summary.TenantBalance.IsZero() {
		t.Error("tenant balance should be zero for nil UUID")
	}
	if !summary.UserMonthlyUsed.IsZero() {
		t.Error("user monthly should be zero for nil UUID")
	}
}