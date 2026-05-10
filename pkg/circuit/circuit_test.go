package circuit

import (
	"testing"
	"time"
)

func TestCircuitBreakerClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultSettings())

	// Should allow requests in closed state
	for i := 0; i < 10; i++ {
		if err := cb.Allow(); err != nil {
			t.Errorf("expected no error in closed state, got %v", err)
		}
	}

	// Record successes should keep it closed
	for i := 0; i < 5; i++ {
		cb.RecordSuccess()
	}

	if cb.State() != StateClosed {
		t.Errorf("expected state closed, got %v", cb.State())
	}
}

func TestCircuitBreakerOpenAfterFailures(t *testing.T) {
	settings := Settings{
		ResetTimeout:         100 * time.Millisecond,
		FailureThreshold:     3,
		SuccessThreshold:     2,
		HalfOpenMaxRequests:  2,
	}
	cb := NewCircuitBreaker(settings)

	// Record failures to open circuit
	for i := 0; i < settings.FailureThreshold; i++ {
		cb.RecordFailure()
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state open after %d failures, got %v", settings.FailureThreshold, cb.State())
	}

	// Should block requests
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreakerHalfOpenTransition(t *testing.T) {
	settings := Settings{
		ResetTimeout:         50 * time.Millisecond,
		FailureThreshold:     2,
		SuccessThreshold:     2,
		HalfOpenMaxRequests:  2,
	}
	cb := NewCircuitBreaker(settings)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Fatalf("expected open state")
	}

	// Wait for reset timeout
	time.Sleep(settings.ResetTimeout + 10*time.Millisecond)

	// Should transition to half-open
	if err := cb.Allow(); err != nil {
		t.Errorf("expected no error in half-open, got %v", err)
	}

	if cb.State() != StateHalfOpen {
		t.Errorf("expected half-open state, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpenToClosed(t *testing.T) {
	settings := Settings{
		ResetTimeout:         50 * time.Millisecond,
		FailureThreshold:     2,
		SuccessThreshold:     3,
		HalfOpenMaxRequests:  5,
	}
	cb := NewCircuitBreaker(settings)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(settings.ResetTimeout + 10*time.Millisecond)

	// Transition to half-open
	cb.Allow()

	// Record successes to close
	for i := 0; i < settings.SuccessThreshold; i++ {
		cb.RecordSuccess()
	}

	if cb.State() != StateClosed {
		t.Errorf("expected closed state after successes, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpenBackToOpen(t *testing.T) {
	settings := Settings{
		ResetTimeout:         50 * time.Millisecond,
		FailureThreshold:     2,
		SuccessThreshold:     3,
		HalfOpenMaxRequests:  5,
	}
	cb := NewCircuitBreaker(settings)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(settings.ResetTimeout + 10*time.Millisecond)

	// Transition to half-open
	cb.Allow()

	// Record failure to go back to open
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Errorf("expected open state after failure in half-open, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpenMaxRequests(t *testing.T) {
	settings := Settings{
		ResetTimeout:         50 * time.Millisecond,
		FailureThreshold:     2,
		SuccessThreshold:     3,
		HalfOpenMaxRequests:  2,
	}
	cb := NewCircuitBreaker(settings)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(settings.ResetTimeout + 10*time.Millisecond)

	// Call 1: Transitions from Open to HalfOpen, halfOpenReq stays 0
	if err := cb.Allow(); err != nil {
		t.Errorf("first request should be allowed (transitioning to half-open), got %v", err)
	}

	// Call 2: HalfOpen, halfOpenReq=0 -> increment to 1
	if err := cb.Allow(); err != nil {
		t.Errorf("second request should be allowed, got %v", err)
	}

	// Call 3: HalfOpen, halfOpenReq=1 -> increment to 2
	if err := cb.Allow(); err != nil {
		t.Errorf("third request should be allowed (reaching max), got %v", err)
	}

	// Call 4: HalfOpen, halfOpenReq=2 >= HalfOpenMaxRequests -> blocked
	if err := cb.Allow(); err != ErrTooManyRequests {
		t.Errorf("fourth request should be blocked with ErrTooManyRequests, got %v", err)
	}
}

func TestManagerGet(t *testing.T) {
	m := NewManager()

	cb1 := m.Get("openai")
	if cb1 == nil {
		t.Error("expected circuit breaker")
	}

	// Same provider should return same instance
	cb2 := m.Get("openai")
	if cb1 != cb2 {
		t.Error("expected same circuit breaker instance")
	}

	// Different provider should return different instance
	cb3 := m.Get("claude")
	if cb1 == cb3 {
		t.Error("expected different circuit breaker for different provider")
	}
}

func TestManagerAllStats(t *testing.T) {
	m := NewManager()

	// Create some circuit breakers and manipulate them
	cb1 := m.Get("openai")
	cb1.RecordFailure()
	cb1.RecordFailure()

	cb2 := m.Get("claude")
	cb2.RecordSuccess()

	stats := m.AllStats()
	if len(stats) != 2 {
		t.Errorf("expected 2 providers, got %d", len(stats))
	}

	if stats["openai"].Failures != 2 {
		t.Errorf("expected 2 failures for openai, got %d", stats["openai"].Failures)
	}

	if stats["claude"].State != StateClosed {
		t.Errorf("expected claude to be closed")
	}
}

func TestManagerReset(t *testing.T) {
	settings := Settings{
		ResetTimeout:         50 * time.Millisecond,
		FailureThreshold:     2,
		SuccessThreshold:     2,
		HalfOpenMaxRequests:  2,
	}
	m := NewManager()

	cb := m.GetOrCreate("openai", settings)
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Fatalf("expected open state")
	}

	m.Reset("openai")

	if cb.State() != StateClosed {
		t.Errorf("expected closed state after reset")
	}
}