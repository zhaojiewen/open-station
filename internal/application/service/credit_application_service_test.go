package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// MockCreditApplicationRepo for testing
type MockCreditApplicationRepo struct {
	apps   map[uuid.UUID]*entity.CreditApplication
	byTenant map[uuid.UUID]*entity.CreditApplication
}

func NewMockCreditApplicationRepo() *MockCreditApplicationRepo {
	return &MockCreditApplicationRepo{
		apps:   make(map[uuid.UUID]*entity.CreditApplication),
		byTenant: make(map[uuid.UUID]*entity.CreditApplication),
	}
}

func (m *MockCreditApplicationRepo) Create(ctx context.Context, app *entity.CreditApplication) error {
	app.ID = uuid.New()
	m.apps[app.ID] = app
	m.byTenant[app.TenantID] = app
	return nil
}

func (m *MockCreditApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error) {
	if a, ok := m.apps[id]; ok {
		return a, nil
	}
	return nil, errors.New("not found")
}

func (m *MockCreditApplicationRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.CreditApplication, error) {
	if a, ok := m.byTenant[tenantID]; ok {
		return a, nil
	}
	return nil, errors.New("not found")
}

func (m *MockCreditApplicationRepo) GetLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.CreditApplication, error) {
	if a, ok := m.byTenant[tenantID]; ok {
		return a, nil
	}
	return nil, errors.New("not found")
}

func (m *MockCreditApplicationRepo) Update(ctx context.Context, app *entity.CreditApplication) error {
	m.apps[app.ID] = app
	m.byTenant[app.TenantID] = app
	return nil
}

func (m *MockCreditApplicationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.apps, id)
	return nil
}

func (m *MockCreditApplicationRepo) List(ctx context.Context, page, pageSize int) ([]entity.CreditApplication, int64, error) {
	var result []entity.CreditApplication
	for _, a := range m.apps {
		result = append(result, *a)
	}
	return result, int64(len(result)), nil
}

func (m *MockCreditApplicationRepo) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.CreditApplication, int64, error) {
	var result []entity.CreditApplication
	for _, a := range m.apps {
		if a.Status == status {
			result = append(result, *a)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockCreditApplicationRepo) Approve(ctx context.Context, id uuid.UUID, approvedLimit decimal.Decimal, reviewedBy uuid.UUID, reviewNotes string) error {
	if a, ok := m.apps[id]; ok {
		a.Status = "approved"
		return nil
	}
	return errors.New("not found")
}

func (m *MockCreditApplicationRepo) Reject(ctx context.Context, id uuid.UUID, reviewedBy uuid.UUID, reviewNotes string) error {
	if a, ok := m.apps[id]; ok {
		a.Status = "rejected"
		return nil
	}
	return errors.New("not found")
}

func (m *MockCreditApplicationRepo) GetPendingCount(ctx context.Context) (int64, error) {
	var count int64
	for _, a := range m.apps {
		if a.Status == "pending" {
			count++
		}
	}
	return count, nil
}

var _ repository.CreditApplicationRepository = (*MockCreditApplicationRepo)(nil)

// MockCATenantRepo for credit application tests
type MockCATenantRepo struct {
	tenants map[uuid.UUID]*entity.Tenant
}

func NewMockCATenantRepo() *MockCATenantRepo {
	return &MockCATenantRepo{tenants: make(map[uuid.UUID]*entity.Tenant)}
}

func (m *MockCATenantRepo) Create(ctx context.Context, tenant *entity.Tenant) error {
	tenant.ID = uuid.New()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockCATenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *MockCATenantRepo) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	return nil, errors.New("not found")
}

func (m *MockCATenantRepo) Update(ctx context.Context, tenant *entity.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockCATenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.tenants, id)
	return nil
}

func (m *MockCATenantRepo) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	var result []entity.Tenant
	for _, t := range m.tenants {
		result = append(result, *t)
	}
	return result, int64(len(result)), nil
}

func (m *MockCATenantRepo) ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}

func (m *MockCATenantRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockCATenantRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *MockCATenantRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockCATenantRepo) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockCATenantRepo) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }

func (m *MockCATenantRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}

func (m *MockCATenantRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockCATenantRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error { return nil }

var _ repository.TenantRepository = (*MockCATenantRepo)(nil)

func TestNewCreditApplicationService(t *testing.T) {
	svc := NewCreditApplicationService(
		NewMockCreditApplicationRepo(),
		NewMockCATenantRepo(),
		nil,
	)
	if svc == nil {
		t.Fatal("service should not be nil")
	}
}

func TestApplyForCredit_Success(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		Reason:          "Need credit for API usage",
		SettlementCycle: "monthly",
	}

	app, err := svc.ApplyForCredit(context.Background(), tenant.ID, req)
	if err != nil {
		t.Fatalf("should succeed: %v", err)
	}
	if app.Status != "pending" {
		t.Fatalf("expected pending, got %s", app.Status)
	}
	if !app.RequestedLimit.Equal(decimal.NewFromInt(5000)) {
		t.Fatalf("expected 5000 limit, got %s", app.RequestedLimit)
	}

	// Verify tenant credit status updated
	updatedTenant, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if updatedTenant.CreditStatus != "pending" {
		t.Fatalf("expected tenant credit_status pending, got %s", updatedTenant.CreditStatus)
	}
}

func TestApplyForCredit_NotOrganization(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "Individual", Type: "individual", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit: decimal.NewFromInt(1000),
	}

	_, err := svc.ApplyForCredit(context.Background(), tenant.ID, req)
	if err == nil {
		t.Fatal("should fail for non-organization tenant")
	}
	if !errors.Is(err, apperrors.ErrInvalidQuotaType) {
		t.Fatalf("expected ErrInvalidQuotaType, got %v", err)
	}
}

func TestApplyForCredit_TenantNotFound(t *testing.T) {
	svc := NewCreditApplicationService(
		NewMockCreditApplicationRepo(),
		NewMockCATenantRepo(),
		nil,
	)

	req := &CreditApplicationRequest{RequestedLimit: decimal.NewFromInt(1000)}
	_, err := svc.ApplyForCredit(context.Background(), uuid.New(), req)
	if err == nil {
		t.Fatal("should fail for non-existent tenant")
	}
}

func TestApplyForCredit_Duplicate(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}

	// First application
	_, err := svc.ApplyForCredit(context.Background(), tenant.ID, req)
	if err != nil {
		t.Fatalf("first apply should succeed: %v", err)
	}

	// Set tenant credit_status back to pending for duplicate check
	tenant.CreditStatus = "none"
	tenantRepo.Update(context.Background(), tenant)

	// Second application should fail due to existing pending
	_, err = svc.ApplyForCredit(context.Background(), tenant.ID, req)
	if err == nil {
		t.Fatal("should fail for duplicate application")
	}
}

func TestGetApplication(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	created, _ := svc.ApplyForCredit(context.Background(), tenant.ID, req)

	retrieved, err := svc.GetApplication(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should retrieve: %v", err)
	}
	if retrieved.ID != created.ID {
		t.Fatal("IDs should match")
	}
}

func TestGetTenantApplication(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	svc.ApplyForCredit(context.Background(), tenant.ID, req)

	app, err := svc.GetTenantApplication(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("should retrieve: %v", err)
	}
	if app.TenantID != tenant.ID {
		t.Fatal("tenant IDs should match")
	}
}

func TestUpdateApplication(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	created, _ := svc.ApplyForCredit(context.Background(), tenant.ID, req)

	newLimit := decimal.NewFromInt(8000)
	settlementDay := 15
	thresholdAmount := decimal.NewFromInt(2000)
	updateReq := &CreditApplicationUpdateRequest{
		RequestedLimit:  newLimit,
		Reason:          "Updated reason",
		SettlementCycle: "weekly",
		ThresholdAmount: &thresholdAmount,
		SettlementDay:   &settlementDay,
	}

	updated, err := svc.UpdateApplication(context.Background(), created.ID, updateReq)
	if err != nil {
		t.Fatalf("should update: %v", err)
	}
	if !updated.RequestedLimit.Equal(newLimit) {
		t.Fatalf("expected limit %s, got %s", newLimit, updated.RequestedLimit)
	}
	if updated.Reason != "Updated reason" {
		t.Fatalf("expected reason 'Updated reason', got '%s'", updated.Reason)
	}
	if updated.SettlementCycle != "weekly" {
		t.Fatalf("expected weekly, got %s", updated.SettlementCycle)
	}
}

func TestUpdateApplication_NotPending(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	created, _ := svc.ApplyForCredit(context.Background(), tenant.ID, req)

	// Approve it
	reviewerID := uuid.New()
	svc.ApproveApplication(context.Background(), created.ID, reviewerID, &ApprovalRequest{
		ApprovedLimit: decimal.NewFromInt(5000),
	})

	updateReq := &CreditApplicationUpdateRequest{Reason: "should fail"}
	_, err := svc.UpdateApplication(context.Background(), created.ID, updateReq)
	if err == nil {
		t.Fatal("should fail for non-pending application")
	}
}

func TestCancelApplication(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	created, _ := svc.ApplyForCredit(context.Background(), tenant.ID, req)

	err := svc.CancelApplication(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should cancel: %v", err)
	}

	// Verify tenant credit status reset
	updatedTenant, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if updatedTenant.CreditStatus != "none" {
		t.Fatalf("expected credit_status none, got %s", updatedTenant.CreditStatus)
	}
}

func TestListApplications(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	svc.ApplyForCredit(context.Background(), tenant.ID, req)

	apps, count, err := svc.ListApplications(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("should list: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
}

func TestListApplicationsByStatus(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	svc.ApplyForCredit(context.Background(), tenant.ID, req)

	apps, count, err := svc.ListApplicationsByStatus(context.Background(), "pending", 1, 10)
	if err != nil {
		t.Fatalf("should list: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 pending, got %d", count)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}

	apps, count, err = svc.ListApplicationsByStatus(context.Background(), "approved", 1, 10)
	if err != nil {
		t.Fatalf("should list: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 approved, got %d", count)
	}
}

func TestApproveApplication(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	appReq := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
		SettlementDay:   intPtr(1),
	}
	created, _ := svc.ApplyForCredit(context.Background(), tenant.ID, appReq)

	reviewerID := uuid.New()
	approved, err := svc.ApproveApplication(context.Background(), created.ID, reviewerID, &ApprovalRequest{
		ApprovedLimit: decimal.NewFromInt(3000),
		ReviewNotes:   "Looks good",
	})
	if err != nil {
		t.Fatalf("should approve: %v", err)
	}
	if approved.Status != "approved" {
		t.Fatalf("expected approved, got %s", approved.Status)
	}

	// Verify tenant credit fields
	updatedTenant, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if updatedTenant.CreditStatus != "approved" {
		t.Fatalf("expected tenant credit_status approved, got %s", updatedTenant.CreditStatus)
	}
	if !updatedTenant.CreditLimit.Equal(decimal.NewFromInt(3000)) {
		t.Fatalf("expected credit_limit 3000, got %s", updatedTenant.CreditLimit)
	}
	if updatedTenant.SettlementCycle != "monthly" {
		t.Fatalf("expected monthly settlement, got %s", updatedTenant.SettlementCycle)
	}
}

func TestRejectApplication(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	appReq := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	created, _ := svc.ApplyForCredit(context.Background(), tenant.ID, appReq)

	reviewerID := uuid.New()
	rejected, err := svc.RejectApplication(context.Background(), created.ID, reviewerID, "Insufficient history")
	if err != nil {
		t.Fatalf("should reject: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("expected rejected, got %s", rejected.Status)
	}

	// Verify tenant credit status
	updatedTenant, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if updatedTenant.CreditStatus != "rejected" {
		t.Fatalf("expected credit_status rejected, got %s", updatedTenant.CreditStatus)
	}
	if updatedTenant.CreditRejectReason != "Insufficient history" {
		t.Fatalf("expected reject reason, got %s", updatedTenant.CreditRejectReason)
	}
}

func TestAdjustCreditLimit(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	creditLimit := decimal.NewFromInt(5000)
	tenant := &entity.Tenant{
		Name: "TestCorp", Type: "organization", Status: "active",
		CreditStatus: "approved", CreditLimit: &creditLimit,
	}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	newLimit := decimal.NewFromInt(10000)
	err := svc.AdjustCreditLimit(context.Background(), tenant.ID, newLimit)
	if err != nil {
		t.Fatalf("should adjust: %v", err)
	}

	updatedTenant, _ := tenantRepo.GetByID(context.Background(), tenant.ID)
	if !updatedTenant.CreditLimit.Equal(newLimit) {
		t.Fatalf("expected limit %s, got %s", newLimit, updatedTenant.CreditLimit)
	}
}

func TestAdjustCreditLimit_NotApproved(t *testing.T) {
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{
		Name: "TestCorp", Type: "organization", Status: "active",
		CreditStatus: "none",
	}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(NewMockCreditApplicationRepo(), tenantRepo, nil)

	err := svc.AdjustCreditLimit(context.Background(), tenant.ID, decimal.NewFromInt(1000))
	if err == nil {
		t.Fatal("should fail for non-approved tenant")
	}
	if !errors.Is(err, apperrors.ErrCreditNotApproved) {
		t.Fatalf("expected ErrCreditNotApproved, got %v", err)
	}
}

func TestGetPendingCount(t *testing.T) {
	appRepo := NewMockCreditApplicationRepo()
	tenantRepo := NewMockCATenantRepo()

	tenant := &entity.Tenant{Name: "TestCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewCreditApplicationService(appRepo, tenantRepo, nil)

	req := &CreditApplicationRequest{
		RequestedLimit:  decimal.NewFromInt(5000),
		SettlementCycle: "monthly",
	}
	svc.ApplyForCredit(context.Background(), tenant.ID, req)

	count, err := svc.GetPendingCount(context.Background())
	if err != nil {
		t.Fatalf("should get count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 pending, got %d", count)
	}
}

func intPtr(i int) *int {
	return &i
}