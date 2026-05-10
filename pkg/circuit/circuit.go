package circuit

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen      = errors.New("circuit breaker is open")
	ErrTooManyRequests  = errors.New("too many requests in half-open state")
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota   // Normal operation
	StateOpen                   // Failing, requests blocked
	StateHalfOpen               // Testing if recovered
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Settings configures the circuit breaker
type Settings struct {
	// Time after which the circuit transitions from open to half-open
	ResetTimeout time.Duration

	// Number of consecutive failures to open the circuit
	FailureThreshold int

	// Number of consecutive successes in half-open to close the circuit
	SuccessThreshold int

	// Max requests allowed in half-open state
	HalfOpenMaxRequests int

	// Timeout for individual requests
	RequestTimeout time.Duration
}

// DefaultSettings provides sensible defaults for LLM providers
func DefaultSettings() Settings {
	return Settings{
		ResetTimeout:         30 * time.Second,
		FailureThreshold:     5,
		SuccessThreshold:     3,
		HalfOpenMaxRequests:  3,
		RequestTimeout:       60 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern for provider failover
type CircuitBreaker struct {
	settings     Settings
	state        State
	failures     int
	successes    int
	halfOpenReq  int
	lastFailTime time.Time
	mu           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with the given settings
func NewCircuitBreaker(settings Settings) *CircuitBreaker {
	return &CircuitBreaker{
		settings: settings,
		state:    StateClosed,
	}
}

// Allow checks if a request should be allowed
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailTime) > cb.settings.ResetTimeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.failures = 0
			cb.halfOpenReq = 0
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		if cb.halfOpenReq >= cb.settings.HalfOpenMaxRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenReq++
		return nil
	}

	return nil
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	switch cb.state {
	case StateClosed:
		// Nothing to do

	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.settings.SuccessThreshold {
			cb.state = StateClosed
			cb.halfOpenReq = 0
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes = 0
	cb.lastFailTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.settings.FailureThreshold {
			cb.state = StateOpen
		}

	case StateHalfOpen:
		cb.state = StateOpen
		cb.halfOpenReq = 0
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns current statistics
func (cb *CircuitBreaker) Stats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return Stats{
		State:            cb.state,
		Failures:         cb.failures,
		Successes:        cb.successes,
		HalfOpenRequests: cb.halfOpenReq,
		LastFailTime:     cb.lastFailTime,
	}
}

// Stats holds circuit breaker statistics
type Stats struct {
	State            State
	Failures         int
	Successes        int
	HalfOpenRequests int
	LastFailTime     time.Time
}

// Manager manages multiple circuit breakers per provider
type Manager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewManager creates a new circuit breaker manager
func NewManager() *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Get returns the circuit breaker for a provider, creating if needed
func (m *Manager) Get(provider string) *CircuitBreaker {
	m.mu.RLock()
	if cb, ok := m.breakers[provider]; ok {
		m.mu.RUnlock()
		return cb
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after acquiring write lock
	if cb, ok := m.breakers[provider]; ok {
		return cb
	}

	cb := NewCircuitBreaker(DefaultSettings())
	m.breakers[provider] = cb
	return cb
}

// GetOrCreate returns a circuit breaker with custom settings
func (m *Manager) GetOrCreate(provider string, settings Settings) *CircuitBreaker {
	m.mu.RLock()
	if cb, ok := m.breakers[provider]; ok {
		m.mu.RUnlock()
		return cb
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, ok := m.breakers[provider]; ok {
		return cb
	}

	cb := NewCircuitBreaker(settings)
	m.breakers[provider] = cb
	return cb
}

// AllStats returns statistics for all circuit breakers
func (m *Manager) AllStats() map[string]Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]Stats)
	for name, cb := range m.breakers {
		result[name] = cb.Stats()
	}
	return result
}

// Reset resets a specific circuit breaker
func (m *Manager) Reset(provider string) {
	m.mu.RLock()
	if cb, ok := m.breakers[provider]; ok {
		m.mu.RUnlock()
		cb.mu.Lock()
		cb.state = StateClosed
		cb.failures = 0
		cb.successes = 0
		cb.halfOpenReq = 0
		cb.mu.Unlock()
	} else {
		m.mu.RUnlock()
	}
}

// ResetAll resets all circuit breakers
func (m *Manager) ResetAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cb := range m.breakers {
		cb.mu.Lock()
		cb.state = StateClosed
		cb.failures = 0
		cb.successes = 0
		cb.halfOpenReq = 0
		cb.mu.Unlock()
	}
}