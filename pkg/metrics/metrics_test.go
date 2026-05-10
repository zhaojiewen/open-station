package metrics

import (
	"testing"
	"time"
)

func TestMetricsRecordHTTPRequest(t *testing.T) {
	m := NewMetrics()

	m.RecordHTTPRequest("GET", "/v1/models", 100*time.Millisecond)
	m.RecordHTTPRequest("POST", "/v1/chat/completions", 200*time.Millisecond)
	m.RecordHTTPRequest("POST", "/v1/chat/completions", 150*time.Millisecond)

	stats := m.GetHTTPStats()
	if stats["total_requests"] != int64(3) {
		t.Errorf("expected 3 total requests, got %v", stats["total_requests"])
	}

	reqByPath := stats["requests_by_path"].(map[string]int64)
	if reqByPath["POST:/v1/chat/completions"] != 2 {
		t.Errorf("expected 2 POST requests to chat/completions, got %v", reqByPath["POST:/v1/chat/completions"])
	}
}

func TestMetricsActiveRequests(t *testing.T) {
	m := NewMetrics()

	m.IncrementActiveRequests()
	m.IncrementActiveRequests()
	m.IncrementActiveRequests()

	stats := m.GetHTTPStats()
	if stats["active_requests"] != int64(3) {
		t.Errorf("expected 3 active requests, got %v", stats["active_requests"])
	}

	m.DecrementActiveRequests()
	m.DecrementActiveRequests()

	stats = m.GetHTTPStats()
	if stats["active_requests"] != int64(1) {
		t.Errorf("expected 1 active request after decrement, got %v", stats["active_requests"])
	}
}

func TestMetricsProviderRequest(t *testing.T) {
	m := NewMetrics()

	m.RecordProviderRequest("openai", true, 100*time.Millisecond)
	m.RecordProviderRequest("openai", true, 150*time.Millisecond)
	m.RecordProviderRequest("openai", false, 200*time.Millisecond)
	m.RecordProviderRequest("claude", true, 300*time.Millisecond)

	stats := m.GetProviderStats("openai")
	if stats["total_requests"] != int64(3) {
		t.Errorf("expected 3 openai requests, got %v", stats["total_requests"])
	}

	if stats["success_count"] != int64(2) {
		t.Errorf("expected 2 successes, got %v", stats["success_count"])
	}

	if stats["failed_count"] != int64(1) {
		t.Errorf("expected 1 failure, got %v", stats["failed_count"])
	}

	// Success rate should be ~66.67%
	if stats["success_rate"].(float64) < 66.0 || stats["success_rate"].(float64) > 67.0 {
		t.Errorf("expected ~66.67%% success rate, got %v", stats["success_rate"])
	}
}

func TestMetricsCircuitBreaker(t *testing.T) {
	m := NewMetrics()

	m.RecordCircuitState("openai", "closed")
	m.RecordCircuitState("claude", "open")
	m.RecordCircuitFailure("openai")
	m.RecordCircuitFailure("openai")
	m.RecordCircuitSuccess("claude")

	stats := m.GetCircuitStats()
	openaiStats, exists := stats["openai"]
	if !exists {
		t.Error("expected openai stats")
		return
	}

	if openaiStats["state"] != "closed" {
		t.Errorf("expected closed state, got %v", openaiStats["state"])
	}

	if openaiStats["failures"].(int64) != int64(2) {
		t.Errorf("expected 2 failures, got %v", openaiStats["failures"])
	}
}

func TestMetricsQueue(t *testing.T) {
	m := NewMetrics()

	m.SetQueueSize(1000)
	m.RecordQueueProcessed(50*time.Millisecond)
	m.RecordQueueProcessed(30*time.Millisecond)
	m.RecordQueueDropped()
	m.RecordQueueDropped()

	stats := m.GetQueueStats()
	if stats["current_size"] != int64(1000) {
		t.Errorf("expected queue size 1000, got %v", stats["current_size"])
	}

	if stats["processed"] != int64(2) {
		t.Errorf("expected 2 processed, got %v", stats["processed"])
	}

	if stats["dropped"] != int64(2) {
		t.Errorf("expected 2 dropped, got %v", stats["dropped"])
	}
}

func TestMetricsBilling(t *testing.T) {
	m := NewMetrics()

	m.RecordBillingEvent(true, 0.01)
	m.RecordBillingEvent(true, 0.02)
	m.RecordBillingEvent(false, 0)
	m.RecordBillingEvent(true, 0.03)

	stats := m.GetBillingStats()
	if stats["events_processed"] != int64(3) {
		t.Errorf("expected 3 processed, got %v", stats["events_processed"])
	}

	if stats["events_dropped"] != int64(1) {
		t.Errorf("expected 1 dropped, got %v", stats["events_dropped"])
	}

	if stats["total_cost"].(float64) != 0.06 {
		t.Errorf("expected total cost 0.06, got %v", stats["total_cost"])
	}
}

func TestMetricsPoolStats(t *testing.T) {
	m := NewMetrics()

	m.SetDBPoolStats(50, 20, 30)
	m.SetRedisPoolSize(100)

	stats := m.GetPoolStats()
	if stats["db_pool_open"] != int64(50) {
		t.Errorf("expected db_pool_open 50, got %v", stats["db_pool_open"])
	}

	if stats["db_pool_idle"] != int64(20) {
		t.Errorf("expected db_pool_idle 20, got %v", stats["db_pool_idle"])
	}

	if stats["redis_pool_size"] != int64(100) {
		t.Errorf("expected redis_pool_size 100, got %v", stats["redis_pool_size"])
	}
}

func TestMetricsGetAllMetrics(t *testing.T) {
	m := NewMetrics()

	// Record some data
	m.RecordHTTPRequest("POST", "/test", 100*time.Millisecond)
	m.RecordProviderRequest("testprovider", true, 50*time.Millisecond)
	m.RecordCircuitState("testprovider", "closed")
	m.SetQueueSize(100)

	all := m.GetAllMetrics()

	if all["http"] == nil {
		t.Error("expected http metrics")
	}
	if all["providers"] == nil {
		t.Error("expected provider metrics")
	}
	if all["circuit_breakers"] == nil {
		t.Error("expected circuit breaker metrics")
	}
	if all["queue"] == nil {
		t.Error("expected queue metrics")
	}
}

func TestGetGlobalMetrics(t *testing.T) {
	m1 := GetGlobalMetrics()
	m2 := GetGlobalMetrics()

	if m1 != m2 {
		t.Error("expected same global metrics instance")
	}
}

func TestLatencyCalculation(t *testing.T) {
	m := NewMetrics()

	// Record multiple latencies
	for i := 0; i < 10; i++ {
		m.RecordProviderRequest("test", true, time.Duration(100+i*10)*time.Millisecond)
	}

	stats := m.GetProviderStats("test")
	avgLatency := stats["avg_latency_ms"].(float64)

	// Average should be around 145ms (100+110+120+...+190)/10
	if avgLatency < 140 || avgLatency > 150 {
		t.Errorf("expected avg latency around 145ms, got %v", avgLatency)
	}
}

func TestLatencyRollingWindow(t *testing.T) {
	m := NewMetrics()

	// Record more than 100 latencies
	for i := 0; i < 150; i++ {
		m.RecordProviderRequest("test", true, 100*time.Millisecond)
	}

	stats := m.GetProviderStats("test")
	// Should only keep last 100
	avgLatency := stats["avg_latency_ms"].(float64)
	if avgLatency != 100.0 {
		t.Errorf("expected 100ms average (rolling window), got %v", avgLatency)
	}
}