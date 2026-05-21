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

// MockMQMemberQuotaRepo for member quota tests
type MockMQMemberQuotaRepo struct {
	quotas  map[uuid.UUID]*entity.MemberQuota
	byUser  map[uuid.UUID]*entity.MemberQuota
}

func NewMockMQMemberQuotaRepo() *MockMQMemberQuotaRepo {
	return &MockMQMemberQuotaRepo{
		quotas: make(map[uuid.UUID]*entity.MemberQuota),
		byUser: make(map[uuid.UUID]*entity.MemberQuota),
	}
}

func (m *MockMQMemberQuotaRepo) Create(ctx context.Context, quota *entity.MemberQuota) error {
	quota.ID = uuid.New()
	m.quotas[quota.ID] = quota
	m.byUser[quota.UserID] = quota
	return nil
}

func (m *MockMQMemberQuotaRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.MemberQuota, error) {
	if q, ok := m.quotas[id]; ok {
		return q, nil
	}
	return nil, errors.New("not found")
}

func (m *MockMQMemberQuotaRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.MemberQuota, error) {
	if q, ok := m.byUser[userID]; ok {
		return q, nil
	}
	return nil, errors.New("not found")
}

func (m *MockMQMemberQuotaRepo) GetByTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) (*entity.MemberQuota, error) {
	if q, ok := m.byUser[userID]; ok {
		if q.TenantID == tenantID {
			return q, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *MockMQMemberQuotaRepo) Update(ctx context.Context, quota *entity.MemberQuota) error {
	m.quotas[quota.ID] = quota
	m.byUser[quota.UserID] = quota
	return nil
}

func (m *MockMQMemberQuotaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.quotas, id)
	return nil
}

func (m *MockMQMemberQuotaRepo) List(ctx context.Context, page, pageSize int) ([]entity.MemberQuota, int64, error) {
	var result []entity.MemberQuota
	for _, q := range m.quotas {
		result = append(result, *q)
	}
	return result, int64(len(result)), nil
}

func (m *MockMQMemberQuotaRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.MemberQuota, int64, error) {
	var result []entity.MemberQuota
	for _, q := range m.quotas {
		if q.TenantID == tenantID {
			result = append(result, *q)
		}
	}
	return result, int64(len(result)), nil
}

func (m *MockMQMemberQuotaRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockMQMemberQuotaRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error { return nil }

func (m *MockMQMemberQuotaRepo) GetTokenUsage(ctx context.Context, id uuid.UUID) (int64, int64, error) {
	return 0, 0, nil
}

func (m *MockMQMemberQuotaRepo) IncrementCostUsed(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error {
	return nil
}

func (m *MockMQMemberQuotaRepo) ResetCostUsed(ctx context.Context, id uuid.UUID) error { return nil }

func (m *MockMQMemberQuotaRepo) GetCostUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, error) {
	return decimal.Zero, decimal.Zero, nil
}

func (m *MockMQMemberQuotaRepo) GetTotalTokensUsedByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *MockMQMemberQuotaRepo) GetTotalCostUsedByTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *MockMQMemberQuotaRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}

func (m *MockMQMemberQuotaRepo) SetExceeded(ctx context.Context, id uuid.UUID, reason string) error {
	return nil
}

func (m *MockMQMemberQuotaRepo) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockMQMemberQuotaRepo) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockMQMemberQuotaRepo) GetActiveAPIKeysCount(ctx context.Context, id uuid.UUID) (int, error) {
	return 0, nil
}

var _ repository.MemberQuotaRepository = (*MockMQMemberQuotaRepo)(nil)

// MockMQTenantRepo for member quota tests
type MockMQTenantRepo struct {
	tenants map[uuid.UUID]*entity.Tenant
}

func NewMockMQTenantRepo() *MockMQTenantRepo {
	return &MockMQTenantRepo{tenants: make(map[uuid.UUID]*entity.Tenant)}
}

func (m *MockMQTenantRepo) Create(ctx context.Context, tenant *entity.Tenant) error {
	tenant.ID = uuid.New()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockMQTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *MockMQTenantRepo) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	return nil, errors.New("not found")
}

func (m *MockMQTenantRepo) Update(ctx context.Context, tenant *entity.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *MockMQTenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.tenants, id)
	return nil
}

func (m *MockMQTenantRepo) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	var result []entity.Tenant
	for _, t := range m.tenants {
		result = append(result, *t)
	}
	return result, int64(len(result)), nil
}

func (m *MockMQTenantRepo) ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error) {
	return nil, 0, nil
}

func (m *MockMQTenantRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockMQTenantRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *MockMQTenantRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockMQTenantRepo) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

func (m *MockMQTenantRepo) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }

func (m *MockMQTenantRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}

func (m *MockMQTenantRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}

func (m *MockMQTenantRepo) ResetTokensUsed(ctx context.Context, id uuid.UUID) error { return nil }

var _ repository.TenantRepository = (*MockMQTenantRepo)(nil)

// MockMQUserRepo for member quota tests
type MockMQUserRepo struct {
	users map[uuid.UUID]*entity.User
}

func NewMockMQUserRepo() *MockMQUserRepo {
	return &MockMQUserRepo{users: make(map[uuid.UUID]*entity.User)}
}

func (m *MockMQUserRepo) Create(ctx context.Context, user *entity.User) error {
	user.ID = uuid.New()
	m.users[user.ID] = user
	return nil
}

func (m *MockMQUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (m *MockMQUserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return nil, errors.New("not found")
}

func (m *MockMQUserRepo) GetByVerificationToken(ctx context.Context, token string) (*entity.User, error) {
	return nil, errors.New("not found")
}

func (m *MockMQUserRepo) Update(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockMQUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.users, id)
	return nil
}

func (m *MockMQUserRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error) {
	return nil, 0, nil
}

func (m *MockMQUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockMQUserRepo) IncrementMonthlyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockMQUserRepo) IncrementDailyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockMQUserRepo) ResetMonthlyBudgetUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockMQUserRepo) ResetDailyBudgetUsed(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *MockMQUserRepo) GetBudgetUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, int64, error) {
	return decimal.Zero, decimal.Zero, 0, nil
}
func (m *MockMQUserRepo) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockMQUserRepo) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error  { return nil }
func (m *MockMQUserRepo) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error  { return nil }
func (m *MockMQUserRepo) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (m *MockMQUserRepo) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockMQUserRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}

var _ repository.UserRepository = (*MockMQUserRepo)(nil)

func TestNewMemberQuotaService(t *testing.T) {
	svc := NewMemberQuotaService(
		NewMockMQMemberQuotaRepo(),
		NewMockMQTenantRepo(),
		NewMockMQUserRepo(),
	)
	if svc == nil {
		t.Fatal("service should not be nil")
	}
}

func TestCreateMemberQuota_Success(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	tokenLimit := int64(100000)
	req := &MemberQuotaCreateRequest{
		TenantID:        tenant.ID,
		UserID:          user.ID,
		TokenQuotaLimit: &tokenLimit,
	}

	mq, err := svc.CreateMemberQuota(context.Background(), req)
	if err != nil {
		t.Fatalf("should create: %v", err)
	}
	if mq.Status != "active" {
		t.Fatalf("expected active, got %s", mq.Status)
	}
	if *mq.TokenQuotaLimit != 100000 {
		t.Fatalf("expected 100000 token limit, got %d", *mq.TokenQuotaLimit)
	}

	// Verify user updated with member quota reference
	updatedUser, _ := userRepo.GetByID(context.Background(), user.ID)
	if updatedUser.UserMode != "member" {
		t.Fatalf("expected user_mode member, got %s", updatedUser.UserMode)
	}
	if updatedUser.MemberQuotaID == nil || *updatedUser.MemberQuotaID != mq.ID {
		t.Fatal("user should have MemberQuotaID set")
	}
}

func TestCreateMemberQuota_NotOrganization(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "Individual", Type: "individual", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	_, err := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID,
		UserID:   uuid.New(),
	})
	if err == nil {
		t.Fatal("should fail for non-organization")
	}
}

func TestCreateMemberQuota_UserNotInTenant(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	otherTenantID := uuid.New()
	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: otherTenantID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	_, err := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID,
		UserID:   user.ID,
	})
	if err == nil {
		t.Fatal("should fail when user not in tenant")
	}
}

func TestCreateMemberQuota_Duplicate(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	req := &MemberQuotaCreateRequest{TenantID: tenant.ID, UserID: user.ID}
	_, err := svc.CreateMemberQuota(context.Background(), req)
	if err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	_, err = svc.CreateMemberQuota(context.Background(), req)
	if err == nil {
		t.Fatal("should fail for duplicate")
	}
}

func TestGetMemberQuota(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID,
		UserID:   user.ID,
	})

	retrieved, err := svc.GetMemberQuota(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should retrieve: %v", err)
	}
	if retrieved.ID != created.ID {
		t.Fatal("IDs should match")
	}
}

func TestGetMemberQuotaByUser(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID,
		UserID:   user.ID,
	})

	mq, err := svc.GetMemberQuotaByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("should retrieve: %v", err)
	}
	if mq.UserID != user.ID {
		t.Fatal("user IDs should match")
	}
}

func TestListMemberQuotas(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user1 := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user1)

	user2 := &entity.User{Name: "Member2", Email: "m2@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user2)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user1.ID,
	})
	svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user2.ID,
	})

	quotas, err := svc.ListMemberQuotas(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("should list: %v", err)
	}
	if len(quotas) != 2 {
		t.Fatalf("expected 2 quotas, got %d", len(quotas))
	}
}

func TestUpdateMemberQuota(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user.ID,
	})

	newTokenLimit := int64(50000)
	costLimit := decimal.NewFromInt(100)
	status := "suspended"
	maxKeys := 5
	updated, err := svc.UpdateMemberQuota(context.Background(), created.ID, &MemberQuotaUpdateRequest{
		TokenQuotaLimit: &newTokenLimit,
		CostLimit:       &costLimit,
		CostLimitType:   "monthly",
		Status:          status,
		MaxAPIKeys:      &maxKeys,
	})
	if err != nil {
		t.Fatalf("should update: %v", err)
	}
	if *updated.TokenQuotaLimit != 50000 {
		t.Fatalf("expected 50000, got %d", *updated.TokenQuotaLimit)
	}
	if !updated.CostLimit.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("expected 100, got %s", updated.CostLimit)
	}
	if updated.Status != "suspended" {
		t.Fatalf("expected suspended, got %s", updated.Status)
	}
}

func TestSetTokenQuotaLimit(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user.ID,
	})

	err := svc.SetTokenQuotaLimit(context.Background(), created.ID, 99999)
	if err != nil {
		t.Fatalf("should set: %v", err)
	}

	mq, _ := svc.GetMemberQuota(context.Background(), created.ID)
	if *mq.TokenQuotaLimit != 99999 {
		t.Fatalf("expected 99999, got %d", *mq.TokenQuotaLimit)
	}
}

func TestSetCostLimit(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user.ID,
	})

	limit := decimal.NewFromInt(200)
	err := svc.SetCostLimit(context.Background(), created.ID, limit, "daily")
	if err != nil {
		t.Fatalf("should set: %v", err)
	}

	mq, _ := svc.GetMemberQuota(context.Background(), created.ID)
	if !mq.CostLimit.Equal(limit) {
		t.Fatalf("expected %s, got %s", limit, mq.CostLimit)
	}
	if mq.CostLimitType != "daily" {
		t.Fatalf("expected daily, got %s", mq.CostLimitType)
	}
}

func TestResetMemberQuota(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user.ID,
	})

	err := svc.ResetMemberQuota(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should reset: %v", err)
	}

	mq, _ := svc.GetMemberQuota(context.Background(), created.ID)
	if mq.TokensUsed != 0 {
		t.Fatalf("expected 0 tokens used, got %d", mq.TokensUsed)
	}
	if !mq.CostUsed.Equal(decimal.Zero) {
		t.Fatalf("expected 0 cost used, got %s", mq.CostUsed)
	}
	if mq.Status != "active" {
		t.Fatalf("expected active, got %s", mq.Status)
	}
}

func TestDeleteMemberQuota(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user.ID,
	})

	err := svc.DeleteMemberQuota(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should delete: %v", err)
	}

	// Verify user member quota reference removed
	updatedUser, _ := userRepo.GetByID(context.Background(), user.ID)
	if updatedUser.MemberQuotaID != nil {
		t.Fatal("MemberQuotaID should be nil after deletion")
	}
}

func TestGetMemberUsage(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tokenLimit := int64(1000)
	tenant := &entity.Tenant{
		Name: "OrgCorp", Type: "organization", Status: "active",
		TokenLimit: &tokenLimit,
	}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	tokenLimit64 := int64(5000)
	costLimit := decimal.NewFromInt(100)
	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID:        tenant.ID,
		UserID:          user.ID,
		TokenQuotaLimit: &tokenLimit64,
		CostLimit:       &costLimit,
		CostLimitType:   "monthly",
	})

	info, err := svc.GetMemberUsage(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should get usage: %v", err)
	}
	if info.TokenQuotaLimit != 5000 {
		t.Fatalf("expected 5000 limit, got %d", info.TokenQuotaLimit)
	}
	if info.TokenRemaining != 5000 {
		t.Fatalf("expected 5000 remaining, got %d", info.TokenRemaining)
	}
	if !info.CostRemaining.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("expected 100 remaining cost, got %s", info.CostRemaining)
	}
	if info.TenantTokenLimit != 1000 {
		t.Fatalf("expected 1000 tenant token limit, got %d", info.TenantTokenLimit)
	}
}

func TestSuspendMember(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user.ID,
	})

	err := svc.SuspendMember(context.Background(), created.ID, "Quota exceeded")
	if err != nil {
		t.Fatalf("should suspend: %v", err)
	}

	mq, _ := svc.GetMemberQuota(context.Background(), created.ID)
	if mq.Status != "suspended" {
		t.Fatalf("expected suspended, got %s", mq.Status)
	}
	if mq.ExceededAt == nil {
		t.Fatal("expected ExceededAt to be set")
	}
	if mq.ExceededReason != "Quota exceeded" {
		t.Fatalf("expected reason, got %s", mq.ExceededReason)
	}
}

func TestActivateMember(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	created, _ := svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user.ID,
	})

	svc.SuspendMember(context.Background(), created.ID, "test")

	err := svc.ActivateMember(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("should activate: %v", err)
	}

	mq, _ := svc.GetMemberQuota(context.Background(), created.ID)
	if mq.Status != "active" {
		t.Fatalf("expected active, got %s", mq.Status)
	}
	if mq.ExceededAt != nil {
		t.Fatal("expected ExceededAt to be nil")
	}
}

func TestListAllMemberQuotas(t *testing.T) {
	mqRepo := NewMockMQMemberQuotaRepo()
	tenantRepo := NewMockMQTenantRepo()
	userRepo := NewMockMQUserRepo()

	tenant := &entity.Tenant{Name: "OrgCorp", Type: "organization", Status: "active"}
	tenantRepo.Create(context.Background(), tenant)

	user := &entity.User{Name: "Member1", Email: "m1@test.com", TenantID: tenant.ID}
	userRepo.Create(context.Background(), user)

	svc := NewMemberQuotaService(mqRepo, tenantRepo, userRepo)

	svc.CreateMemberQuota(context.Background(), &MemberQuotaCreateRequest{
		TenantID: tenant.ID, UserID: user.ID,
	})

	quotas, count, err := svc.ListAllMemberQuotas(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("should list: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
	if len(quotas) != 1 {
		t.Fatalf("expected 1 quota, got %d", len(quotas))
	}
}