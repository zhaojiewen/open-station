package metrics

import (
	"sync"
	"time"
)

// Metrics provides simple metrics collection without external dependencies.
// For production, replace with prometheus client or similar.
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal   map[string]int64
	httpRequestsActive  int64
	httpRequestLatency  map[string][]time.Duration

	// Provider metrics
	providerRequestsTotal   map[string]int64
	providerRequestsSuccess map[string]int64
	providerRequestsFailed  map[string]int64
	providerLatency         map[string][]time.Duration

	// Circuit breaker metrics
	circuitState      map[string]string
	circuitFailures   map[string]int64
	circuitSuccesses  map[string]int64

	// Connection pool metrics
	dbPoolOpen    int64
	dbPoolIdle    int64
	dbPoolInUse   int64
	redisPoolSize int64

	// Queue metrics
	queueSize       int64
	queueProcessed  int64
	queueDropped    int64
	queueLatency    []time.Duration

	// Billing metrics
	billingEventsProcessed int64
	billingEventsDropped   int64
	billingTotalCost       float64

	mu sync.RWMutex
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		httpRequestsTotal:      make(map[string]int64),
		httpRequestLatency:     make(map[string][]time.Duration),
		providerRequestsTotal:  make(map[string]int64),
		providerRequestsSuccess: make(map[string]int64),
		providerRequestsFailed:  make(map[string]int64),
		providerLatency:        make(map[string][]time.Duration),
		circuitState:           make(map[string]string),
		circuitFailures:        make(map[string]int64),
		circuitSuccesses:       make(map[string]int64),
	}
}

// HTTP Metrics

func (m *Metrics) RecordHTTPRequest(method, path string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := method + ":" + path
	m.httpRequestsTotal[key]++
	m.httpRequestLatency[key] = appendLatency(m.httpRequestLatency[key], latency)
}

func (m *Metrics) IncrementActiveRequests() {
	m.mu.Lock()
	m.httpRequestsActive++
	m.mu.Unlock()
}

func (m *Metrics) DecrementActiveRequests() {
	m.mu.Lock()
	m.httpRequestsActive--
	m.mu.Unlock()
}

func (m *Metrics) GetHTTPStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := int64(0)
	for _, v := range m.httpRequestsTotal {
		total += v
	}

	return map[string]interface{}{
		"total_requests":   total,
		"active_requests":  m.httpRequestsActive,
		"requests_by_path": m.httpRequestsTotal,
	}
}

// Provider Metrics

func (m *Metrics) RecordProviderRequest(provider string, success bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providerRequestsTotal[provider]++
	if success {
		m.providerRequestsSuccess[provider]++
	} else {
		m.providerRequestsFailed[provider]++
	}
	m.providerLatency[provider] = appendLatency(m.providerLatency[provider], latency)
}

func (m *Metrics) GetProviderStats(provider string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.providerRequestsTotal[provider]
	success := m.providerRequestsSuccess[provider]
	failed := m.providerRequestsFailed[provider]

	successRate := float64(0)
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}

	avgLatency := calculateAvgLatency(m.providerLatency[provider])

	return map[string]interface{}{
		"provider":        provider,
		"total_requests":  total,
		"success_count":   success,
		"failed_count":    failed,
		"success_rate":    successRate,
		"avg_latency_ms":  avgLatency,
	}
}

func (m *Metrics) GetAllProviderStats() map[string]map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]map[string]interface{})
	for provider := range m.providerRequestsTotal {
		stats := map[string]interface{}{
			"provider":       provider,
			"total_requests": m.providerRequestsTotal[provider],
			"success_count":  m.providerRequestsSuccess[provider],
			"failed_count":   m.providerRequestsFailed[provider],
		}
		if m.providerRequestsTotal[provider] > 0 {
			stats["success_rate"] = float64(m.providerRequestsSuccess[provider]) /
				float64(m.providerRequestsTotal[provider]) * 100
		}
		stats["avg_latency_ms"] = calculateAvgLatency(m.providerLatency[provider])
		result[provider] = stats
	}
	return result
}

// Circuit Breaker Metrics

func (m *Metrics) RecordCircuitState(provider, state string) {
	m.mu.Lock()
	m.circuitState[provider] = state
	m.mu.Unlock()
}

func (m *Metrics) RecordCircuitFailure(provider string) {
	m.mu.Lock()
	m.circuitFailures[provider]++
	m.mu.Unlock()
}

func (m *Metrics) RecordCircuitSuccess(provider string) {
	m.mu.Lock()
	m.circuitSuccesses[provider]++
	m.mu.Unlock()
}

func (m *Metrics) GetCircuitStats() map[string]map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]map[string]interface{})
	for provider, state := range m.circuitState {
		result[provider] = map[string]interface{}{
			"state":     state,
			"failures":  m.circuitFailures[provider],
			"successes": m.circuitSuccesses[provider],
		}
	}
	return result
}

// Connection Pool Metrics

func (m *Metrics) SetDBPoolStats(open, idle, inUse int64) {
	m.mu.Lock()
	m.dbPoolOpen = open
	m.dbPoolIdle = idle
	m.dbPoolInUse = inUse
	m.mu.Unlock()
}

func (m *Metrics) SetRedisPoolSize(size int64) {
	m.mu.Lock()
	m.redisPoolSize = size
	m.mu.Unlock()
}

func (m *Metrics) GetPoolStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"db_pool_open":   m.dbPoolOpen,
		"db_pool_idle":   m.dbPoolIdle,
		"db_pool_in_use": m.dbPoolInUse,
		"redis_pool_size": m.redisPoolSize,
	}
}

// Queue Metrics

func (m *Metrics) SetQueueSize(size int64) {
	m.mu.Lock()
	m.queueSize = size
	m.mu.Unlock()
}

func (m *Metrics) RecordQueueProcessed(latency time.Duration) {
	m.mu.Lock()
	m.queueProcessed++
	m.queueLatency = appendLatency(m.queueLatency, latency)
	m.mu.Unlock()
}

func (m *Metrics) RecordQueueDropped() {
	m.mu.Lock()
	m.queueDropped++
	m.mu.Unlock()
}

func (m *Metrics) GetQueueStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"current_size":  m.queueSize,
		"processed":     m.queueProcessed,
		"dropped":       m.queueDropped,
		"avg_latency_ms": calculateAvgLatency(m.queueLatency),
	}
}

// Billing Metrics

func (m *Metrics) RecordBillingEvent(processed bool, cost float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if processed {
		m.billingEventsProcessed++
		m.billingTotalCost += cost
	} else {
		m.billingEventsDropped++
	}
}

func (m *Metrics) GetBillingStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"events_processed": m.billingEventsProcessed,
		"events_dropped":   m.billingEventsDropped,
		"total_cost":       m.billingTotalCost,
	}
}

// Get all metrics for health/monitoring endpoint

func (m *Metrics) GetAllMetrics() map[string]interface{} {
	return map[string]interface{}{
		"http":           m.GetHTTPStats(),
		"providers":      m.GetAllProviderStats(),
		"circuit_breakers": m.GetCircuitStats(),
		"pools":          m.GetPoolStats(),
		"queue":          m.GetQueueStats(),
		"billing":        m.GetBillingStats(),
	}
}

// Helper functions

func appendLatency(latencies []time.Duration, l time.Duration) []time.Duration {
	// Keep last 100 latencies for calculating average
	if len(latencies) >= 100 {
		latencies = latencies[1:]
	}
	return append(latencies, l)
}

func calculateAvgLatency(latencies []time.Duration) float64 {
	if len(latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, l := range latencies {
		total += l
	}

	return float64(total.Milliseconds()) / float64(len(latencies))
}

// Global metrics instance
var globalMetrics *Metrics
var metricsOnce sync.Once

func GetGlobalMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = NewMetrics()
	})
	return globalMetrics
}