package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

func init() {
	logger.Log = zap.NewNop()
}

// MockAsyncAPIKeyRepo for async billing tests
type MockAsyncAPIKeyRepo struct {
	tokenUpdates map[uuid.UUID]int64
}

func NewMockAsyncAPIKeyRepo() *MockAsyncAPIKeyRepo {
	return &MockAsyncAPIKeyRepo{tokenUpdates: make(map[uuid.UUID]int64)}
}

func (m *MockAsyncAPIKeyRepo) Create(ctx context.Context, apiKey *entity.APIKey) error { return nil }
func (m *MockAsyncAPIKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.APIKey, error) {
	return nil, nil
}
func (m *MockAsyncAPIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*entity.APIKey, error) {
	return nil, nil
}
func (m *MockAsyncAPIKeyRepo) GetWithRelations(ctx context.Context, keyHash string) (*entity.APIKey, *entity.User, *entity.Tenant, error) {
	return nil, nil, nil, nil
}
func (m *MockAsyncAPIKeyRepo) GetByKeyPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error) {
	return nil, nil
}
func (m *MockAsyncAPIKeyRepo) Update(ctx context.Context, apiKey *entity.APIKey) error { return nil }
func (m *MockAsyncAPIKeyRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *MockAsyncAPIKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *MockAsyncAPIKeyRepo) List(ctx context.Context, userID uuid.UUID) ([]entity.APIKey, error) {
	return nil, nil
}
func (m *MockAsyncAPIKeyRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.APIKey, error) {
	return nil, nil
}
func (m *MockAsyncAPIKeyRepo) ListAll(ctx context.Context) ([]entity.APIKey, error) { return nil, nil }
func (m *MockAsyncAPIKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *MockAsyncAPIKeyRepo) UpdateTokenUsage(ctx context.Context, id uuid.UUID, tokens int64) error {
	m.tokenUpdates[id] = m.tokenUpdates[id] + tokens
	return nil
}
func (m *MockAsyncAPIKeyRepo) IncrementMonthlyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockAsyncAPIKeyRepo) IncrementDailyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return nil
}
func (m *MockAsyncAPIKeyRepo) ResetMonthlyCostUsed(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockAsyncAPIKeyRepo) ResetDailyCostUsed(ctx context.Context, id uuid.UUID) error   { return nil }
func (m *MockAsyncAPIKeyRepo) GetCostUsage(ctx context.Context, id uuid.UUID) (decimal.Decimal, decimal.Decimal, int64, int64, error) {
	return decimal.Zero, decimal.Zero, 0, 0, nil
}
func (m *MockAsyncAPIKeyRepo) IncrementDailyTokens(ctx context.Context, id uuid.UUID, tokens int64) error {
	return nil
}
func (m *MockAsyncAPIKeyRepo) ResetDailyTokens(ctx context.Context, id uuid.UUID) error { return nil }

var _ repository.APIKeyRepository = (*MockAsyncAPIKeyRepo)(nil)

func createMockBillingService() *BillingService {
	tenantRepo := NewMockTenantRepo()
	tenantRepo.Create(context.Background(), &entity.Tenant{
		Name: "test", Slug: "test", Status: "active",
		Balance: decimal.NewFromInt(10000),
	})

	modelRepo := NewMockModelRepo()
	modelRepo.Create(context.Background(), &entity.Model{
		Provider:        "test-provider",
		ModelID:         "test-model",
		PromptPrice:     decimal.NewFromFloat(0.01),
		CompletionPrice: decimal.NewFromFloat(0.02),
	})

	return NewBillingService(
		tenantRepo,
		NewMockUsageRepo(),
		NewMockBillRepo(),
		NewMockRechargeRepo(),
		modelRepo,
	)
}

func TestNewAsyncBillingQueue(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 100, 50, time.Second)

	if q == nil {
		t.Fatal("queue should not be nil")
	}
	if q.batchSize != 50 {
		t.Fatalf("expected batchSize 50, got %d", q.batchSize)
	}
}

func TestAsyncBillingQueue_StartStop(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 10, 10, 100*time.Millisecond)

	q.Start(2)
	time.Sleep(50 * time.Millisecond)
	q.Stop()
}

func TestAsyncBillingQueue_EnqueueBilling(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 100, 10, time.Second)

	event := BillingEvent{
		TenantID:         uuid.New(),
		UserID:           uuid.New(),
		APIKeyID:         uuid.New(),
		RequestID:        "req-1",
		Provider:         "test",
		ModelID:          "test-model",
		PromptTokens:     100,
		CompletionTokens: 50,
		LatencyMs:        200,
		StatusCode:       200,
	}

	if !q.EnqueueBilling(event) {
		t.Fatal("should enqueue successfully")
	}
}

func TestAsyncBillingQueue_EnqueueBilling_Full(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 1, 10, time.Second)

	event := BillingEvent{RequestID: "req-1"}
	q.EnqueueBilling(event)

	if q.EnqueueBilling(BillingEvent{RequestID: "req-2"}) {
		t.Fatal("should return false when queue is full")
	}
}

func TestAsyncBillingQueue_EnqueueTokenUpdate(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 100, 10, time.Second)

	if !q.EnqueueTokenUpdate(TokenUpdateEvent{APIKeyID: uuid.New(), Tokens: 100}) {
		t.Fatal("should enqueue token update successfully")
	}
}

func TestAsyncBillingQueue_EnqueueTokenUpdate_Full(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 1, 10, time.Second)

	q.EnqueueTokenUpdate(TokenUpdateEvent{APIKeyID: uuid.New(), Tokens: 1})

	if q.EnqueueTokenUpdate(TokenUpdateEvent{APIKeyID: uuid.New(), Tokens: 2}) {
		t.Fatal("should return false when token queue is full")
	}
}

func TestAsyncBillingQueue_QueueBillingAsync(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 100, 10, time.Second)

	q.QueueBillingAsync(uuid.New(), uuid.New(), uuid.New(),
		"req-1", "openai", "gpt-4", 100, 50, 200, 200)
}

func TestAsyncBillingQueue_QueueTokenUpdateAsync(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 100, 10, time.Second)

	q.QueueTokenUpdateAsync(uuid.New(), 500)
}

func TestAsyncBillingQueue_GetQueueStats(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 100, 10, time.Second)

	stats := q.GetQueueStats()
	if stats == nil {
		t.Fatal("stats should not be nil")
	}
}

func TestAsyncBillingQueue_BillingWorker(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 10, 10, 100*time.Millisecond)

	q.Start(1)
	defer q.Stop()

	q.QueueBillingAsync(uuid.New(), uuid.New(), uuid.New(),
		"req-worker", "openai", "gpt-4", 100, 50, 200, 200)

	time.Sleep(100 * time.Millisecond)
}

func TestAsyncBillingQueue_BatchProcessor(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 10, 10, 50*time.Millisecond)

	q.Start(1)
	time.Sleep(100 * time.Millisecond)
	q.Stop()
}

func TestAsyncBillingQueue_ProcessBatchBillings_Empty(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 10, 10, time.Second)

	q.processBatchBillings(context.Background(), nil)
	q.processBatchBillings(context.Background(), []BillingEvent{})
}

func TestAsyncBillingQueue_ProcessBatchBillings_WithEvents(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 10, 10, time.Second)

	events := []BillingEvent{
		{
			TenantID:         uuid.New(),
			UserID:           uuid.New(),
			APIKeyID:         uuid.New(),
			RequestID:        "batch-1",
			Provider:         "test",
			ModelID:          "test-model",
			PromptTokens:     10,
			CompletionTokens: 5,
			LatencyMs:        100,
			StatusCode:       200,
		},
	}

	q.processBatchBillings(context.Background(), events)
}

func TestAsyncBillingQueue_EnqueueBilling_IgnoreDropped(t *testing.T) {
	bs := createMockBillingService()
	akRepo := NewMockAsyncAPIKeyRepo()
	q := NewAsyncBillingQueue(bs, akRepo, 1, 10, time.Second)

	// Fill the queue
	q.EnqueueBilling(BillingEvent{RequestID: "fill-1"})

	// Dropped but should not panic
	result := q.EnqueueBilling(BillingEvent{RequestID: "drop-1"})
	if result {
		t.Fatal("expected false for dropped event")
	}
}