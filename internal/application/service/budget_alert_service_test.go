package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
)

// MockBudgetAlertRepo for testing
type MockBudgetAlertRepo struct {
	alerts map[uuid.UUID]*entity.BudgetAlert
}

func NewMockBudgetAlertRepo() *MockBudgetAlertRepo {
	return &MockBudgetAlertRepo{alerts: make(map[uuid.UUID]*entity.BudgetAlert)}
}

func (m *MockBudgetAlertRepo) Create(ctx context.Context, alert *entity.BudgetAlert) error {
	alert.ID = uuid.New()
	m.alerts[alert.ID] = alert
	return nil
}

func (m *MockBudgetAlertRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.BudgetAlert, error) {
	if a, ok := m.alerts[id]; ok {
		return a, nil
	}
	return nil, errors.New("not found")
}

func (m *MockBudgetAlertRepo) GetByScope(ctx context.Context, scope string, scopeID uuid.UUID) ([]entity.BudgetAlert, error) {
	var result []entity.BudgetAlert
	for _, a := range m.alerts {
		if a.Scope == scope && a.ScopeID == scopeID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *MockBudgetAlertRepo) GetByScopeAndType(ctx context.Context, scope string, scopeID uuid.UUID, alertType string) (*entity.BudgetAlert, error) {
	for _, a := range m.alerts {
		if a.Scope == scope && a.ScopeID == scopeID && a.AlertType == alertType {
			return a, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *MockBudgetAlertRepo) Update(ctx context.Context, alert *entity.BudgetAlert) error {
	m.alerts[alert.ID] = alert
	return nil
}

func (m *MockBudgetAlertRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.alerts, id)
	return nil
}

func (m *MockBudgetAlertRepo) List(ctx context.Context, page, pageSize int) ([]entity.BudgetAlert, int64, error) {
	var result []entity.BudgetAlert
	for _, a := range m.alerts {
		result = append(result, *a)
	}
	return result, int64(len(result)), nil
}

func (m *MockBudgetAlertRepo) ListEnabled(ctx context.Context) ([]entity.BudgetAlert, error) {
	var result []entity.BudgetAlert
	for _, a := range m.alerts {
		if a.IsEnabled {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *MockBudgetAlertRepo) Enable(ctx context.Context, id uuid.UUID) error {
	if a, ok := m.alerts[id]; ok {
		a.IsEnabled = true
		return nil
	}
	return errors.New("not found")
}

func (m *MockBudgetAlertRepo) Disable(ctx context.Context, id uuid.UUID) error {
	if a, ok := m.alerts[id]; ok {
		a.IsEnabled = false
		return nil
	}
	return errors.New("not found")
}

func (m *MockBudgetAlertRepo) MarkTriggered(ctx context.Context, id uuid.UUID) error {
	return nil
}

var _ repository.BudgetAlertRepository = (*MockBudgetAlertRepo)(nil)

// MockBATenantRepo for budget alert tests
type MockBATenantRepo struct {
	tenants map[uuid.UUID]*entity.Tenant
}

func NewMockBATenantRepo() *MockBATenantRepo {
	return &MockBATenantRepo{tenants: make(map[uuid.UUID]*entity.Tenant)}
}

func (m *MockBATenantRepo) Create(ctx context.Context, tenant *entity.Tenant) error {
	tenant.ID = uuid.New()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockBATenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *MockBATenantRepo) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	return nil, errors.New("not found")
}

func (m *MockBATenantRepo) Update(ctx context.Context, tenant *entity.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockBATenantRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockBATenantRepo) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}

func (m *MockBATenantRepo) ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}

func (m *MockBATenantRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockBATenantRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *MockBATenantRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockBATenantRepo) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockBATenantRepo) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }

func (m *MockBATenantRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}

func (m *MockBATenantRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockBATenantRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error { return nil }

var _ repository.TenantRepository = (*MockBATenantRepo)(nil)

// MockBAUserRepo for budget alert tests
type MockBAUserRepo struct {
	users map[uuid.UUID]*entity.User
}

func NewMockBAUserRepo() *MockBAUserRepo {
	return &MockBAUserRepo{users: make(map[uuid.UUID]*entity.User)}
}

func (m *MockBAUserRepo) Create(ctx context.Context, user *entity.User) error {
	user.ID = uuid.New()
	m.users[user.ID] = user
	return nil
}

func (m *MockBAUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (m *MockBAUserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return nil, errors.New("not found")
}

func (m *MockBAUserRepo) GetByVerificationToken(ctx context.Context, token string) (*entity.User, error) {
	return nil, errors.New("not found")
}

func (m *MockBAUserRepo) Update(ctx context.Context, user *entity.User) error { return nil }
func (m *MockBAUserRepo) Delete(ctx context.Context, id uuid.UUID) error     { return nil }
func (m *MockBAUserRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error) {
	return nil, 0, nil
}

func (m *MockBAUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockBAUserRepo) IncrementMonthlyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockBAUserRepo) IncrementDailyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockBAUserRepo) ResetMonthlyBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockBAUserRepo) ResetDailyBudgetUsed(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *MockBAUserRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, int64, error) {
	return decimal.Zero, decimal.Zero, 0, nil
}
func (m *MockBAUserRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockBAUserRepo) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error  { return nil }
func (m *MockBAUserRepo) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error  { return nil }
func (m *MockBAUserRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (m *MockBAUserRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockBAUserRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

var _ repository.UserRepository = (*MockBAUserRepo)(nil)

// MockBAAPIKeyRepo for budget alert tests
type MockBAAPIKeyRepo struct {
	keys map[uuid.UUID]*entity.APIKey
}

func NewMockBAAPIKeyRepo() *MockBAAPIKeyRepo {
	return &MockBAAPIKeyRepo{keys: make(map[uuid.UUID]*entity.APIKey)}
}

func (m *MockBAAPIKeyRepo) Create(ctx context.Context, apiKey *entity.APIKey) error { return nil }
func (m *MockBAAPIKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.APIKey, error) {
	if k, ok := m.keys[id]; ok {
		return k, nil
	}
	return nil, errors.New("not found")
}
func (m *MockBAAPIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*entity.APIKey, error) {
	return nil, nil
}
func (m *MockBAAPIKeyRepo) GetWithRelations(ctx context.Context, keyHash string) (*entity.APIKey, *entity.User, *entity.Tenant, error) {
	return nil, nil, nil, nil
}
func (m *MockBAAPIKeyRepo) GetByKeyPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error) {
	return nil, nil
}
func (m *MockBAAPIKeyRepo) Update(ctx context.Context, apiKey *entity.APIKey) error { return nil }
func (m *MockBAAPIKeyRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *MockBAAPIKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *MockBAAPIKeyRepo) List(ctx context.Context, userID uuid.UUID) ([]entity.APIKey, error) {
	return nil, nil
}
func (m *MockBAAPIKeyRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.APIKey, error) {
	return nil, nil
}
func (m *MockBAAPIKeyRepo) ListAll(ctx context.Context) ([]entity.APIKey, error) { return nil, nil }
func (m *MockBAAPIKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *MockBAAPIKeyRepo) UpdateTokenUsage(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockBAAPIKeyRepo) IncrementMonthlyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockBAAPIKeyRepo) IncrementDailyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockBAAPIKeyRepo) ResetMonthlyCostUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockBAAPIKeyRepo) ResetDailyCostUsed(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *MockBAAPIKeyRepo) GetCostUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, int64, int64, error) {
	return decimal.Zero, decimal.Zero, 0, 0, nil
}
func (m *MockBAAPIKeyRepo) IncrementDailyTokens(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockBAAPIKeyRepo) ResetDailyTokens(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockBAAPIKeyRepo) UpdateProviderUsage(ctx context.Context, id uuid.UUID, provider string, tokens int64, cost decimal.Decimal) error {
	return nil
}

var _ repository.APIKeyRepository = (*MockBAAPIKeyRepo)(nil)

// Store created IDs on mock so checkAPIKeyAlerts can look up the tenant info
func (m *MockBAAPIKeyRepo) StoreKey(id uuid.UUID, key *entity.APIKey) {
	m.keys[id] = key
}

func TestNewBudgetAlertService(t *testing.T) {
	svc := NewBudgetAlertService(
		NewMockBudgetAlertRepo(),
		NewMockBATenantRepo(),
		NewMockBAUserRepo(),
		NewMockBAAPIKeyRepo(),
		nil,
	)
	if svc == nil {
		t.Fatal("service should not be nil")
	}
}

func TestCreateBudgetAlert(t *testing.T) {
	svc := NewBudgetAlertService(
		NewMockBudgetAlertRepo(),
		NewMockBATenantRepo(),
		NewMockBAUserRepo(),
		NewMockBAAPIKeyRepo(),
		nil,
	)

	alert, err := svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 80,
		[]string{"admin@test.com"}, "", "")
	if err != nil {
		t.Fatalf("should create: %v", err)
	}
	if alert.Scope != "tenant" {
		t.Fatalf("expected tenant scope, got %s", alert.Scope)
	}
	if alert.ThresholdPercent != 80 {
		t.Fatalf("expected 80 threshold, got %d", alert.ThresholdPercent)
	}
	if !alert.IsEnabled {
		t.Fatal("should be enabled by default")
	}
}

func TestCreateBudgetAlert_InvalidScope(t *testing.T) {
	svc := NewBudgetAlertService(
		NewMockBudgetAlertRepo(),
		NewMockBATenantRepo(),
		NewMockBAUserRepo(),
		NewMockBAAPIKeyRepo(),
		nil,
	)

	_, err := svc.Create(context.Background(), "invalid", uuid.New(), "budget_80", 80, nil, "", "")
	if err == nil {
		t.Fatal("should fail for invalid scope")
	}
}

func TestCreateBudgetAlert_InvalidThreshold(t *testing.T) {
	svc := NewBudgetAlertService(
		NewMockBudgetAlertRepo(),
		NewMockBATenantRepo(),
		NewMockBAUserRepo(),
		NewMockBAAPIKeyRepo(),
		nil,
	)

	_, err := svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 0, nil, "", "")
	if err == nil {
		t.Fatal("should fail for threshold 0")
	}
	_, err = svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 101, nil, "", "")
	if err == nil {
		t.Fatal("should fail for threshold 101")
	}
}

func TestGetBudgetAlertByID(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	svc := NewBudgetAlertService(alertRepo, NewMockBATenantRepo(), NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), nil)

	created, _ := svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 80, nil, "", "")

	retrieved, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should retrieve: %v", err)
	}
	if retrieved.ID != created.ID {
		t.Fatal("IDs should match")
	}
}

func TestListBudgetAlerts(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	svc := NewBudgetAlertService(alertRepo, NewMockBATenantRepo(), NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), nil)

	svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 80, nil, "", "")
	svc.Create(context.Background(), "user", uuid.New(), "budget_90", 90, nil, "", "")

	alerts, count, err := svc.List(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("should list: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(alerts))
	}
}

func TestGetBudgetAlertsByScope(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	svc := NewBudgetAlertService(alertRepo, NewMockBATenantRepo(), NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), nil)

	scopeID := uuid.New()
	svc.Create(context.Background(), "tenant", scopeID, "budget_80", 80, nil, "", "")
	svc.Create(context.Background(), "tenant", scopeID, "budget_100", 100, nil, "", "")
	svc.Create(context.Background(), "user", uuid.New(), "budget_90", 90, nil, "", "")

	alerts, err := svc.GetByScope(context.Background(), "tenant", scopeID)
	if err != nil {
		t.Fatalf("should get by scope: %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 in scope, got %d", len(alerts))
	}
}

func TestUpdateBudgetAlert(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	svc := NewBudgetAlertService(alertRepo, NewMockBATenantRepo(), NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), nil)

	created, _ := svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 80, nil, "", "")

	err := svc.Update(context.Background(), created.ID, 95, []string{"new@test.com"}, "slack-url", "webhook-url")
	if err != nil {
		t.Fatalf("should update: %v", err)
	}

	updated, _ := svc.GetByID(context.Background(), created.ID)
	if updated.ThresholdPercent != 95 {
		t.Fatalf("expected 95, got %d", updated.ThresholdPercent)
	}
}

func TestEnableBudgetAlert(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	svc := NewBudgetAlertService(alertRepo, NewMockBATenantRepo(), NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), nil)

	created, _ := svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 80, nil, "", "")
	svc.Disable(context.Background(), created.ID)

	err := svc.Enable(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should enable: %v", err)
	}

	alert, _ := svc.GetByID(context.Background(), created.ID)
	if !alert.IsEnabled {
		t.Fatal("should be enabled")
	}
}

func TestDisableBudgetAlert(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	svc := NewBudgetAlertService(alertRepo, NewMockBATenantRepo(), NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), nil)

	created, _ := svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 80, nil, "", "")

	err := svc.Disable(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should disable: %v", err)
	}

	alert, _ := svc.GetByID(context.Background(), created.ID)
	if alert.IsEnabled {
		t.Fatal("should be disabled")
	}
}

func TestDeleteBudgetAlert(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	svc := NewBudgetAlertService(alertRepo, NewMockBATenantRepo(), NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), nil)

	created, _ := svc.Create(context.Background(), "tenant", uuid.New(), "budget_80", 80, nil, "", "")

	err := svc.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should delete: %v", err)
	}

	_, err = svc.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Fatal("should not find deleted alert")
	}
}

func TestCheckAndTriggerAlerts(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	userRepo := NewMockBAUserRepo()
	apiKeyRepo := NewMockBAAPIKeyRepo()

	budgetLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active",
		BudgetUsedMonth:     decimal.NewFromInt(800),
		MonthlyBudgetLimit:  &budgetLimit,
		BillingEmail:        "billing@test.com",
	}
	tenantRepo.Create(context.Background(), tenant)

	monthlyBudget := decimal.NewFromInt(500)
	user := &entity.User{
		Name: "TestUser", Email: "user@test.com", Status: "active",
		TenantID:        tenant.ID,
		MonthlyBudget:   &monthlyBudget,
		BudgetUsedMonth: decimal.NewFromInt(400),
	}
	userRepo.Create(context.Background(), user)

	monthlyCostLimit := decimal.NewFromInt(100)
	apiKey := &entity.APIKey{
		Name: "TestKey", Status: "active",
		TenantID:            tenant.ID,
		MonthlyCostLimit:    &monthlyCostLimit,
		MonthlyCostUsed:     decimal.NewFromInt(85),
		AlertThreshold1:     80,
		AlertThreshold2:     90,
	}
	apiKeyRepo.StoreKey(uuid.New(), apiKey)

	notificationSvc := NewNotificationService("", "", "", "", 0)
	svc := NewBudgetAlertService(alertRepo, tenantRepo, userRepo, apiKeyRepo, notificationSvc)

	// This should not panic
	svc.CheckAndTriggerAlerts(context.Background(), uuid.New(), user.ID, tenant.ID)
}

func TestCheckTenantAlerts_Triggered(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	notificationSvc := NewNotificationService("", "", "", "", 0)

	budgetLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active",
		BudgetUsedMonth:    decimal.NewFromInt(950),
		MonthlyBudgetLimit: &budgetLimit,
		BillingEmail:       "billing@test.com",
	}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewBudgetAlertService(alertRepo, tenantRepo, NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), notificationSvc)

	// Create alert at 80%
	svc.Create(context.Background(), "tenant", tenant.ID, "budget_80", 80,
		[]string{"alert@test.com"}, "slack-url", "")

	// checkTenantAlerts is unexported, test through CheckAndTriggerAlerts
	svc.CheckAndTriggerAlerts(context.Background(), uuid.New(), uuid.New(), tenant.ID)
}

func TestCheckTenantAlerts_NotTriggered(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	notificationSvc := NewNotificationService("", "", "", "", 0)

	budgetLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active",
		BudgetUsedMonth:    decimal.NewFromInt(300),
		MonthlyBudgetLimit: &budgetLimit,
	}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewBudgetAlertService(alertRepo, tenantRepo, NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), notificationSvc)

	svc.Create(context.Background(), "tenant", tenant.ID, "budget_80", 80, nil, "", "")

	// Should not panic or trigger (below 80%)
	svc.CheckAndTriggerAlerts(context.Background(), uuid.New(), uuid.New(), tenant.ID)
}

func TestCheckUserAlerts(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	userRepo := NewMockBAUserRepo()
	notificationSvc := NewNotificationService("", "", "", "", 0)

	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active", BillingEmail: "billing@test.com",
	}
	tenantRepo.Create(context.Background(), tenant)

	monthlyBudget := decimal.NewFromInt(500)
	user := &entity.User{
		Name: "TestUser", Email: "user@test.com", Status: "active",
		TenantID:        tenant.ID,
		MonthlyBudget:   &monthlyBudget,
		BudgetUsedMonth: decimal.NewFromInt(450),
	}
	userRepo.Create(context.Background(), user)

	svc := NewBudgetAlertService(alertRepo, tenantRepo, userRepo, NewMockBAAPIKeyRepo(), notificationSvc)

	svc.Create(context.Background(), "user", user.ID, "budget_90", 90,
		[]string{"alert@test.com"}, "", "")

	// 450/500 = 90%, should trigger
	svc.CheckAndTriggerAlerts(context.Background(), uuid.New(), user.ID, tenant.ID)
}

func TestCheckUserAlerts_DailyHigher(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	userRepo := NewMockBAUserRepo()
	notificationSvc := NewNotificationService("", "", "", "", 0)

	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active", BillingEmail: "billing@test.com",
	}
	tenantRepo.Create(context.Background(), tenant)

	monthlyBudget := decimal.NewFromInt(1000)
	dailyBudget := decimal.NewFromInt(200)
	user := &entity.User{
		Name: "TestUser", Email: "user@test.com", Status: "active",
		TenantID:        tenant.ID,
		MonthlyBudget:   &monthlyBudget,
		BudgetUsedMonth: decimal.NewFromInt(500), // 50%
		DailyBudget:     &dailyBudget,
		BudgetUsedToday: decimal.NewFromInt(180), // 90%
	}
	userRepo.Create(context.Background(), user)

	svc := NewBudgetAlertService(alertRepo, tenantRepo, userRepo, NewMockBAAPIKeyRepo(), notificationSvc)

	svc.Create(context.Background(), "user", user.ID, "budget_80", 80,
		[]string{"alert@test.com"}, "", "")

	// Uses max(50%, 90%) = 90%, should trigger at threshold 80
	svc.CheckAndTriggerAlerts(context.Background(), uuid.New(), user.ID, tenant.ID)
}

func TestCheckAPIKeyAlerts_MonthlyTriggered(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	apiKeyRepo := NewMockBAAPIKeyRepo()
	notificationSvc := NewNotificationService("", "", "", "", 0)

	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active", BillingEmail: "billing@test.com",
	}
	tenantRepo.Create(context.Background(), tenant)

	monthlyLimit := decimal.NewFromInt(100)
	apiKeyID := uuid.New()
	apiKey := &entity.APIKey{
		Name: "TestKey", Status: "active",
		TenantID:          tenant.ID,
		MonthlyCostLimit:  &monthlyLimit,
		MonthlyCostUsed:   decimal.NewFromInt(85),
		AlertThreshold1:   80,
		AlertThreshold2:   90,
		AlertThreshold3:   100,
	}
	apiKeyRepo.StoreKey(apiKeyID, apiKey)

	svc := NewBudgetAlertService(alertRepo, tenantRepo, NewMockBAUserRepo(), apiKeyRepo, notificationSvc)

	// Should trigger and send email to tenant billing email
	svc.CheckAndTriggerAlerts(context.Background(), apiKeyID, uuid.New(), tenant.ID)
}

func TestCheckAPIKeyAlerts_DailyTriggered(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	apiKeyRepo := NewMockBAAPIKeyRepo()
	notificationSvc := NewNotificationService("", "", "", "", 0)

	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active", BillingEmail: "billing@test.com",
	}
	tenantRepo.Create(context.Background(), tenant)

	dailyLimit := decimal.NewFromInt(50)
	apiKeyID := uuid.New()
	apiKey := &entity.APIKey{
		Name: "TestKey", Status: "active",
		TenantID:       tenant.ID,
		DailyCostLimit: &dailyLimit,
		DailyCostUsed:  decimal.NewFromInt(50), // 100%
	}
	apiKeyRepo.StoreKey(apiKeyID, apiKey)

	svc := NewBudgetAlertService(alertRepo, tenantRepo, NewMockBAUserRepo(), apiKeyRepo, notificationSvc)

	// Should trigger daily alert at 100%
	svc.CheckAndTriggerAlerts(context.Background(), apiKeyID, uuid.New(), tenant.ID)
}

func TestCheckAPIKeyAlerts_NoBillingEmail(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	apiKeyRepo := NewMockBAAPIKeyRepo()
	notificationSvc := NewNotificationService("", "", "", "", 0)

	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active", BillingEmail: "",
	}
	tenantRepo.Create(context.Background(), tenant)

	monthlyLimit := decimal.NewFromInt(100)
	apiKeyID := uuid.New()
	apiKey := &entity.APIKey{
		Name: "TestKey", Status: "active",
		TenantID:         tenant.ID,
		MonthlyCostLimit: &monthlyLimit,
		MonthlyCostUsed:  decimal.NewFromInt(100),
		AlertThreshold1:  80,
	}
	apiKeyRepo.StoreKey(apiKeyID, apiKey)

	svc := NewBudgetAlertService(alertRepo, tenantRepo, NewMockBAUserRepo(), apiKeyRepo, notificationSvc)

	// Should not panic when no billing email
	svc.CheckAndTriggerAlerts(context.Background(), apiKeyID, uuid.New(), tenant.ID)
}

func TestTriggerAlert_WithAllChannels(t *testing.T) {
	alertRepo := NewMockBudgetAlertRepo()
	tenantRepo := NewMockBATenantRepo()
	notificationSvc := NewNotificationService("", "", "", "", 0)

	budgetLimit := decimal.NewFromInt(1000)
	tenant := &entity.Tenant{
		Name: "TestCorp", Status: "active",
		BudgetUsedMonth:    decimal.NewFromInt(950),
		MonthlyBudgetLimit: &budgetLimit,
		BillingEmail:       "billing@test.com",
	}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewBudgetAlertService(alertRepo, tenantRepo, NewMockBAUserRepo(), NewMockBAAPIKeyRepo(), notificationSvc)

	// Create alert with all notification channels
	svc.Create(context.Background(), "tenant", tenant.ID, "budget_80", 80,
		[]string{"admin@test.com"}, "https://hooks.slack.com/test", "https://webhook.test.com")

	// CheckAndTriggerAlerts will trigger due to 95% usage against 80% threshold
	svc.CheckAndTriggerAlerts(context.Background(), uuid.New(), uuid.New(), tenant.ID)
}