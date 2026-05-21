package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"github.com/zhaojiewen/open-station/pkg/metrics"
	"go.uber.org/zap"
)

// BillingEvent represents a billing event to be processed asynchronously
type BillingEvent struct {
	TenantID             uuid.UUID
	UserID               uuid.UUID
	APIKeyID             uuid.UUID
	RequestID            string
	Provider             string
	ModelID              string
	PromptTokens         int64
	CompletionTokens     int64
	CacheReadTokens      int64
	CacheCreationTokens  int64
	LatencyMs            int
	StatusCode           int
	CreatedAt            time.Time
}

// TokenUpdateEvent represents a token usage update event
type TokenUpdateEvent struct {
	APIKeyID uuid.UUID
	Provider string
	Tokens   int64
	Cost     decimal.Decimal
}

// AsyncBillingQueue processes billing events asynchronously
type AsyncBillingQueue struct {
	billingEvents    chan BillingEvent
	tokenUpdates     chan TokenUpdateEvent
	billingService   *BillingService
	apiKeyRepo       repository.APIKeyRepository
	usageRepo        repository.UsageRepository
	wg               sync.WaitGroup
	stopCh           chan struct{}
	batchSize        int
	batchInterval    time.Duration
	pendingBillings  []BillingEvent
	pendingTokens    []TokenUpdateEvent
	billingMutex     sync.Mutex
	tokenMutex       sync.Mutex
	metrics          *metrics.Metrics
}

// NewAsyncBillingQueue creates a new async billing queue
func NewAsyncBillingQueue(
	billingService *BillingService,
	apiKeyRepo repository.APIKeyRepository,
	queueSize int,
	batchSize int,
	batchInterval time.Duration,
) *AsyncBillingQueue {
	return &AsyncBillingQueue{
		billingEvents:  make(chan BillingEvent, queueSize),
		tokenUpdates:   make(chan TokenUpdateEvent, queueSize),
		billingService: billingService,
		apiKeyRepo:     apiKeyRepo,
		usageRepo:      billingService.usageRepo,
		stopCh:         make(chan struct{}),
		batchSize:      batchSize,
		batchInterval:  batchInterval,
		metrics:        metrics.GetGlobalMetrics(),
	}
}

// Start begins processing billing events
func (q *AsyncBillingQueue) Start(workers int) {
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.billingWorker()
	}

	q.wg.Add(1)
	go q.batchProcessor()

	logger.Info("async billing queue started", zap.Int("workers", workers))
}

// Stop gracefully stops the queue
func (q *AsyncBillingQueue) Stop() {
	close(q.stopCh)
	q.wg.Wait()
	logger.Info("async billing queue stopped")
}

// EnqueueBilling adds a billing event to the queue (non-blocking)
func (q *AsyncBillingQueue) EnqueueBilling(event BillingEvent) bool {
	select {
	case q.billingEvents <- event:
		return true
	default:
		logger.Warn("billing queue full, dropping event",
			zap.String("request_id", event.RequestID),
		)
		q.metrics.RecordQueueDropped()
		q.metrics.RecordBillingEvent(false, 0)
		return false
	}
}

// EnqueueTokenUpdate adds a token update event to the queue (non-blocking)
func (q *AsyncBillingQueue) EnqueueTokenUpdate(event TokenUpdateEvent) bool {
	select {
	case q.tokenUpdates <- event:
		return true
	default:
		logger.Warn("token update queue full, dropping event",
			zap.String("api_key_id", event.APIKeyID.String()),
		)
		q.metrics.RecordQueueDropped()
		return false
	}
}

// billingWorker processes billing events
func (q *AsyncBillingQueue) billingWorker() {
	defer q.wg.Done()

	ctx := context.Background()

	for {
		select {
		case <-q.stopCh:
			return
		case event := <-q.billingEvents:
			start := time.Now()
			_, err := q.billingService.RecordUsage(ctx,
				event.TenantID,
				event.UserID,
				event.APIKeyID,
				event.RequestID,
				event.Provider,
				event.ModelID,
				event.PromptTokens,
				event.CompletionTokens,
				event.CacheReadTokens,
				event.CacheCreationTokens,
				event.LatencyMs,
				event.StatusCode,
			)
			latency := time.Since(start)
			q.metrics.RecordQueueProcessed(latency)

			if err != nil {
				logger.Error("failed to record usage async",
					zap.String("request_id", event.RequestID),
					zap.Error(err),
				)
				q.metrics.RecordBillingEvent(false, 0)
			} else {
				q.metrics.RecordBillingEvent(true, 0)
			}
		case event := <-q.tokenUpdates:
			err := q.apiKeyRepo.UpdateTokenUsage(ctx, event.APIKeyID, event.Tokens)
			if err != nil {
				logger.Error("failed to update token usage async",
					zap.String("api_key_id", event.APIKeyID.String()),
					zap.Error(err),
				)
			}
			if event.Provider != "" {
				if err := q.apiKeyRepo.UpdateProviderUsage(ctx, event.APIKeyID, event.Provider, event.Tokens, event.Cost); err != nil {
					logger.Error("failed to update provider usage async",
						zap.String("api_key_id", event.APIKeyID.String()),
						zap.String("provider", event.Provider),
						zap.Error(err),
					)
				}
			}
		}
	}
}

// batchProcessor handles batched processing for efficiency
func (q *AsyncBillingQueue) batchProcessor() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.batchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			q.flushPending()
			return
		case <-ticker.C:
			q.flushPending()
		}
	}
}

// flushPending processes any pending events using batch insert
func (q *AsyncBillingQueue) flushPending() {
	ctx := context.Background()

	// Flush billing events with batch insert
	q.billingMutex.Lock()
	if len(q.pendingBillings) > 0 {
		events := q.pendingBillings
		q.pendingBillings = nil
		q.billingMutex.Unlock()

		// Batch insert usage records
		q.processBatchBillings(ctx, events)
	} else {
		q.billingMutex.Unlock()
	}

	// Flush token updates
	q.tokenMutex.Lock()
	if len(q.pendingTokens) > 0 {
		events := q.pendingTokens
		q.pendingTokens = nil
		q.tokenMutex.Unlock()

		for _, event := range events {
			err := q.apiKeyRepo.UpdateTokenUsage(ctx, event.APIKeyID, event.Tokens)
			if err != nil {
				logger.Error("failed to flush token update",
					zap.String("api_key_id", event.APIKeyID.String()),
					zap.Error(err),
				)
			}
			if event.Provider != "" {
				if err := q.apiKeyRepo.UpdateProviderUsage(ctx, event.APIKeyID, event.Provider, event.Tokens, event.Cost); err != nil {
					logger.Error("failed to flush provider usage update",
						zap.String("api_key_id", event.APIKeyID.String()),
						zap.String("provider", event.Provider),
						zap.Error(err),
					)
				}
			}
		}
	} else {
		q.tokenMutex.Unlock()
	}
}

// processBatchBillings processes multiple billing events using batch database insert
func (q *AsyncBillingQueue) processBatchBillings(ctx context.Context, events []BillingEvent) {
	if len(events) == 0 {
		return
	}

	start := time.Now()

	// Group events for batch processing
	for _, event := range events {
		_, err := q.billingService.RecordUsage(ctx,
			event.TenantID,
			event.UserID,
			event.APIKeyID,
			event.RequestID,
			event.Provider,
			event.ModelID,
			event.PromptTokens,
			event.CompletionTokens,
			event.CacheReadTokens,
			event.CacheCreationTokens,
			event.LatencyMs,
			event.StatusCode,
		)
		if err != nil {
			logger.Error("failed to flush billing event",
				zap.String("request_id", event.RequestID),
				zap.Error(err),
			)
			q.metrics.RecordBillingEvent(false, 0)
		} else {
			q.metrics.RecordBillingEvent(true, 0)
		}
	}

	q.metrics.RecordQueueProcessed(time.Since(start))
}

// QueueBillingAsync is a convenience method for quick async billing
func (q *AsyncBillingQueue) QueueBillingAsync(
	tenantID, userID, apiKeyID uuid.UUID,
	requestID, provider, modelID string,
	promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens int64,
	latencyMs, statusCode int,
) {
	event := BillingEvent{
		TenantID:            tenantID,
		UserID:              userID,
		APIKeyID:            apiKeyID,
		RequestID:           requestID,
		Provider:            provider,
		ModelID:             modelID,
		PromptTokens:        promptTokens,
		CompletionTokens:    completionTokens,
		CacheReadTokens:     cacheReadTokens,
		CacheCreationTokens: cacheCreationTokens,
		LatencyMs:           latencyMs,
		StatusCode:          statusCode,
		CreatedAt:           time.Now(),
	}
	q.EnqueueBilling(event)
}

// QueueTokenUpdateAsync is a convenience method for quick async token update
func (q *AsyncBillingQueue) QueueTokenUpdateAsync(apiKeyID uuid.UUID, provider string, tokens int64, cost decimal.Decimal) {
	event := TokenUpdateEvent{
		APIKeyID: apiKeyID,
		Provider: provider,
		Tokens:   tokens,
		Cost:     cost,
	}
	q.EnqueueTokenUpdate(event)
}

// GetQueueStats returns current queue statistics
func (q *AsyncBillingQueue) GetQueueStats() map[string]interface{} {
	return q.metrics.GetQueueStats()
}