package loadbalancer

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// StrategyType defines the load balancing strategy type
type StrategyType string

const (
	// StrategyPriority - Priority-based selection (default, current implementation)
	// Selects account with lowest priority number (highest priority)
	StrategyPriority StrategyType = "priority"

	// StrategyRoundRobin - Round-robin selection
	// Distributes requests evenly across all available accounts
	StrategyRoundRobin StrategyType = "round_robin"

	// StrategyWeightedRoundRobin - Weighted round-robin selection
	// Distributes requests based on account weight (configured per account)
	StrategyWeightedRoundRobin StrategyType = "weighted_round_robin"

	// StrategyLeastConnections - Least connections selection
	// Selects account with the least active connections
	StrategyLeastConnections StrategyType = "least_connections"

	// StrategyLeastResponseTime - Least response time selection
	// Selects account with the best average response time
	StrategyLeastResponseTime StrategyType = "least_response_time"

	// StrategyHealthScore - Health score-based selection
	// Selects account with the highest health score
	StrategyHealthScore StrategyType = "health_score"

	// StrategyRandom - Random selection
	// Randomly selects from available accounts
	StrategyRandom StrategyType = "random"

	// StrategyAdaptive - Adaptive selection (AI-driven)
	// Dynamically adjusts based on multiple factors (health, latency, connections)
	StrategyAdaptive StrategyType = "adaptive"
)

// AccountStats tracks runtime statistics for an account
type AccountStats struct {
	ID             string
	ActiveConns    int64         // Current active connections
	TotalRequests  int64         // Total requests processed
	SuccessCount   int64         // Successful requests
	FailureCount   int64         // Failed requests
	TotalLatency   int64         // Total latency in milliseconds (for average calculation)
	AvgLatencyMs   float64       // Average latency
	LastUsed       time.Time     // Last usage timestamp
	LastError      time.Time     // Last error timestamp
	HealthScore    int           // Health score (0-100)
	Weight         int           // Weight for weighted strategies (default: 1)
	CurrentWeight  int           // Current effective weight for WRR
	SuccessRate    float64       // Success rate (0.0-1.0)
}

// LoadBalancer manages account selection with configurable strategies
type LoadBalancer struct {
	strategy    StrategyType
	accountStats sync.Map // map[string]*AccountStats (account ID -> stats)
	roundRobinIndex sync.Map // map[string]int64 (provider -> current index)
	mu          sync.RWMutex
	cooldown    time.Duration // Switch cooldown period
}

// LoadBalancerConfig configures the load balancer
type LoadBalancerConfig struct {
	Strategy          StrategyType   `json:"strategy" yaml:"strategy"`
	CooldownDuration  time.Duration  `json:"cooldown_duration" yaml:"cooldown_duration"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
}

// DefaultLoadBalancerConfig returns default configuration
func DefaultLoadBalancerConfig() LoadBalancerConfig {
	return LoadBalancerConfig{
		Strategy:           StrategyPriority,
		CooldownDuration:   10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(config LoadBalancerConfig) *LoadBalancer {
	return &LoadBalancer{
		strategy: config.Strategy,
		cooldown: config.CooldownDuration,
	}
}

// SetStrategy changes the load balancing strategy
func (lb *LoadBalancer) SetStrategy(strategy StrategyType) {
	lb.mu.Lock()
	lb.strategy = strategy
	lb.mu.Unlock()

	logger.Info("Load balancer strategy changed", zap.String("strategy", string(strategy)))
}

// GetStrategy returns current strategy
func (lb *LoadBalancer) GetStrategy() StrategyType {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.strategy
}

// SelectAccount selects an account based on current strategy
func (lb *LoadBalancer) SelectAccount(ctx context.Context, accounts []*entity.ProviderAccount, provider string) (*entity.ProviderAccount, error) {
	if len(accounts) == 0 {
		return nil, ErrNoAvailableAccounts
	}

	// Filter usable accounts
	usable := lb.filterUsableAccounts(accounts)
	if len(usable) == 0 {
		return nil, ErrNoAvailableAccounts
	}

	lb.mu.RLock()
	strategy := lb.strategy
	lb.mu.RUnlock()

	switch strategy {
	case StrategyPriority:
		return lb.selectByPriority(usable), nil
	case StrategyRoundRobin:
		return lb.selectByRoundRobin(usable, provider), nil
	case StrategyWeightedRoundRobin:
		return lb.selectByWeightedRoundRobin(usable, provider), nil
	case StrategyLeastConnections:
		return lb.selectByLeastConnections(usable), nil
	case StrategyLeastResponseTime:
		return lb.selectByLeastResponseTime(usable), nil
	case StrategyHealthScore:
		return lb.selectByHealthScore(usable), nil
	case StrategyRandom:
		return lb.selectByRandom(usable), nil
	case StrategyAdaptive:
		return lb.selectByAdaptive(usable), nil
	default:
		return lb.selectByPriority(usable), nil
	}
}

// filterUsableAccounts filters accounts that are usable
func (lb *LoadBalancer) filterUsableAccounts(accounts []*entity.ProviderAccount) []*entity.ProviderAccount {
	usable := make([]*entity.ProviderAccount, 0)
	for _, acc := range accounts {
		if lb.isAccountUsable(acc) {
			usable = append(usable, acc)
		}
	}
	return usable
}

// isAccountUsable checks if an account is usable
func (lb *LoadBalancer) isAccountUsable(account *entity.ProviderAccount) bool {
	if account.Status != "active" && account.Status != "limited" {
		return false
	}

	// Check monthly limit
	if account.MonthlyLimit != nil && account.UsedThisMonth.GreaterThanOrEqual(*account.MonthlyLimit) {
		return false
	}

	// Check error count threshold
	if account.ErrorCount >= 5 {
		return false
	}

	// Check recent errors (cool down after recent failure)
	if account.LastErrorAt != nil && time.Since(*account.LastErrorAt) < lb.cooldown {
		return false
	}

	return true
}

// Strategy implementations

// selectByPriority selects the account with highest priority (lowest priority number)
func (lb *LoadBalancer) selectByPriority(accounts []*entity.ProviderAccount) *entity.ProviderAccount {
	// Already sorted by priority in database query
	return accounts[0]
}

// selectByRoundRobin selects accounts in round-robin fashion
func (lb *LoadBalancer) selectByRoundRobin(accounts []*entity.ProviderAccount, provider string) *entity.ProviderAccount {
	key := provider
	indexPtr, _ := lb.roundRobinIndex.LoadOrStore(key, new(int64))
	index := atomic.AddInt64(indexPtr.(*int64), 1)

	// Wrap around
	selectedIndex := int(index) % len(accounts)
	return accounts[selectedIndex]
}

// selectByWeightedRoundRobin selects accounts based on weight
func (lb *LoadBalancer) selectByWeightedRoundRobin(accounts []*entity.ProviderAccount, provider string) *entity.ProviderAccount {
	// Get weights for each account
	totalWeight := 0
	weights := make([]int, len(accounts))

	for i, acc := range accounts {
		stats := lb.getAccountStats(acc.ID.String())
		weight := stats.Weight
		if weight <= 0 {
			weight = 1 // Default weight
		}

		// Adjust weight based on health score (healthier accounts get more weight)
		if stats.HealthScore > 0 {
			weight = weight * stats.HealthScore / 100
			if weight < 1 {
				weight = 1
			}
		}

		weights[i] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return accounts[0]
	}

	// Generate random weight selection
	r := rand.Intn(totalWeight)
	currentWeight := 0

	for i, weight := range weights {
		currentWeight += weight
		if r < currentWeight {
			return accounts[i]
		}
	}

	return accounts[0]
}

// selectByLeastConnections selects the account with least active connections
func (lb *LoadBalancer) selectByLeastConnections(accounts []*entity.ProviderAccount) *entity.ProviderAccount {
	minConns := int64(-1)
	selected := accounts[0]

	for _, acc := range accounts {
		stats := lb.getAccountStats(acc.ID.String())
		conns := stats.ActiveConns

		if minConns == -1 || conns < minConns {
			minConns = conns
			selected = acc
		}
	}

	return selected
}

// selectByLeastResponseTime selects the account with best average response time
func (lb *LoadBalancer) selectByLeastResponseTime(accounts []*entity.ProviderAccount) *entity.ProviderAccount {
	minLatency := float64(-1)
	selected := accounts[0]

	for _, acc := range accounts {
		stats := lb.getAccountStats(acc.ID.String())

		// If no data yet, prefer this account (new account gets chance)
		if stats.TotalRequests == 0 {
			return acc
		}

		avgLatency := stats.AvgLatencyMs
		if minLatency == -1 || avgLatency < minLatency {
			minLatency = avgLatency
			selected = acc
		}
	}

	return selected
}

// selectByHealthScore selects the account with highest health score
func (lb *LoadBalancer) selectByHealthScore(accounts []*entity.ProviderAccount) *entity.ProviderAccount {
	maxScore := -1
	selected := accounts[0]

	for _, acc := range accounts {
		stats := lb.getAccountStats(acc.ID.String())

		// Calculate health score
		score := lb.calculateHealthScore(acc, stats)

		if score > maxScore {
			maxScore = score
			selected = acc
		}
	}

	return selected
}

// selectByRandom randomly selects an account
func (lb *LoadBalancer) selectByRandom(accounts []*entity.ProviderAccount) *entity.ProviderAccount {
	if len(accounts) == 1 {
		return accounts[0]
	}
	return accounts[rand.Intn(len(accounts))]
}

// selectByAdaptive uses AI-driven adaptive selection based on multiple factors
func (lb *LoadBalancer) selectByAdaptive(accounts []*entity.ProviderAccount) *entity.ProviderAccount {
	bestScore := float64(-1)
	selected := accounts[0]

	for _, acc := range accounts {
		stats := lb.getAccountStats(acc.ID.String())
		score := lb.calculateAdaptiveScore(acc, stats)

		if score > bestScore {
			bestScore = score
			selected = acc
		}
	}

	return selected
}

// calculateAdaptiveScore calculates a composite score for adaptive selection
func (lb *LoadBalancer) calculateAdaptiveScore(account *entity.ProviderAccount, stats *AccountStats) float64 {
	// Weights for different factors
	const (
		healthWeight    = 0.3
		latencyWeight   = 0.25
		successWeight   = 0.25
		connectionWeight = 0.15
		loadWeight      = 0.05
	)

	// Health score (0-100) normalized to 0-1
	healthScore := float64(stats.HealthScore) / 100.0

	// Latency score (inverse - lower latency = higher score)
	// Assume max acceptable latency is 5000ms
	latencyScore := 1.0 - (stats.AvgLatencyMs / 5000.0)
	if latencyScore < 0 {
		latencyScore = 0
	}

	// Success rate score (already 0-1)
	successScore := stats.SuccessRate

	// Connection score (inverse - fewer connections = higher score)
	// Assume max connections per account is 100
	connScore := 1.0 - (float64(stats.ActiveConns) / 100.0)
	if connScore < 0 {
		connScore = 0
	}

	// Load score (based on monthly limit usage)
	loadScore := 1.0
	if account.MonthlyLimit != nil && !account.MonthlyLimit.IsZero() {
		usageRate := account.UsedThisMonth.Div(*account.MonthlyLimit).InexactFloat64()
		loadScore = 1.0 - usageRate
		if loadScore < 0 {
			loadScore = 0
		}
	}

	// Composite score
	composite := healthScore*healthWeight +
		latencyScore*latencyWeight +
		successScore*successWeight +
		connScore*connectionWeight +
		loadScore*loadWeight

	return composite
}

// calculateHealthScore calculates health score for an account
func (lb *LoadBalancer) calculateHealthScore(account *entity.ProviderAccount, stats *AccountStats) int {
	score := 100

	// Deduct for errors
	score -= account.ErrorCount * 10

	// Deduct for usage rate
	if account.MonthlyLimit != nil && !account.MonthlyLimit.IsZero() {
		usageRate := account.UsedThisMonth.Div(*account.MonthlyLimit).InexactFloat64()
		if usageRate > 0.8 {
			score -= 30
		} else if usageRate > 0.5 {
			score -= 10
		}
	}

	// Deduct for recent errors
	if account.LastErrorAt != nil && time.Since(*account.LastErrorAt) < 5*time.Minute {
		score -= 20
	}

	// Deduct for low success rate
	if stats.SuccessRate < 0.5 {
		score -= 20
	} else if stats.SuccessRate < 0.8 {
		score -= 10
	}

	if score < 0 {
		score = 0
	}

	return score
}

// Stats tracking methods

// getAccountStats gets stats for an account, creating if not exists
func (lb *LoadBalancer) getAccountStats(accountID string) *AccountStats {
	statsPtr, _ := lb.accountStats.LoadOrStore(accountID, &AccountStats{
		ID:          accountID,
		Weight:      1,
		HealthScore: 100,
		SuccessRate: 1.0,
	})

	return statsPtr.(*AccountStats)
}

// RecordRequestStart records the start of a request
func (lb *LoadBalancer) RecordRequestStart(accountID string) {
	stats := lb.getAccountStats(accountID)
	atomic.AddInt64(&stats.ActiveConns, 1)
	atomic.AddInt64(&stats.TotalRequests, 1)
	stats.LastUsed = time.Now()
}

// RecordRequestSuccess records a successful request
func (lb *LoadBalancer) RecordRequestSuccess(accountID string, latencyMs int64) {
	stats := lb.getAccountStats(accountID)
	atomic.AddInt64(&stats.SuccessCount, 1)
	atomic.AddInt64(&stats.TotalLatency, latencyMs)

	// Update average latency
	totalReq := atomic.LoadInt64(&stats.TotalRequests)
	if totalReq > 0 {
		totalLat := atomic.LoadInt64(&stats.TotalLatency)
		stats.AvgLatencyMs = float64(totalLat) / float64(totalReq)
	}

	// Update success rate
	stats.SuccessRate = float64(stats.SuccessCount) / float64(totalReq)

	// Update health score based on success
	if stats.SuccessRate > 0.9 {
		stats.HealthScore = min(100, stats.HealthScore + 1)
	}
}

// RecordRequestFailure records a failed request
func (lb *LoadBalancer) RecordRequestFailure(accountID string) {
	stats := lb.getAccountStats(accountID)
	atomic.AddInt64(&stats.FailureCount, 1)
	stats.LastError = time.Now()

	// Update success rate
	totalReq := atomic.LoadInt64(&stats.TotalRequests)
	if totalReq > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(totalReq)
	}

	// Reduce health score
	stats.HealthScore = max(0, stats.HealthScore - 5)
}

// RecordRequestEnd records the end of a request (connection closed)
func (lb *LoadBalancer) RecordRequestEnd(accountID string) {
	stats := lb.getAccountStats(accountID)
	atomic.AddInt64(&stats.ActiveConns, -1)
}

// SetAccountWeight sets the weight for an account (for weighted strategies)
func (lb *LoadBalancer) SetAccountWeight(accountID string, weight int) {
	stats := lb.getAccountStats(accountID)
	stats.Weight = weight
}

// GetAccountStats returns current stats for an account
func (lb *LoadBalancer) GetAccountStats(accountID string) map[string]interface{} {
	stats := lb.getAccountStats(accountID)

	return map[string]interface{}{
		"account_id":       accountID,
		"active_connections": stats.ActiveConns,
		"total_requests":   stats.TotalRequests,
		"success_count":    stats.SuccessCount,
		"failure_count":    stats.FailureCount,
		"avg_latency_ms":   stats.AvgLatencyMs,
		"success_rate":     stats.SuccessRate,
		"health_score":     stats.HealthScore,
		"weight":           stats.Weight,
		"last_used":        stats.LastUsed,
		"last_error":       stats.LastError,
	}
}

// GetAllStats returns stats for all accounts
func (lb *LoadBalancer) GetAllStats() map[string]interface{} {
	result := make(map[string]interface{})

	lb.accountStats.Range(func(key, value interface{}) bool {
		accountID := key.(string)
		result[accountID] = lb.GetAccountStats(accountID)
		return true
	})

	return result
}

// ResetStats resets stats for an account
func (lb *LoadBalancer) ResetStats(accountID string) {
	lb.accountStats.Store(accountID, &AccountStats{
		ID:          accountID,
		Weight:      1,
		HealthScore: 100,
		SuccessRate: 1.0,
	})
}

// Error definitions
var (
	ErrNoAvailableAccounts = errors.New("no available accounts for selection")
	ErrStrategyNotSupported = errors.New("load balancing strategy not supported")
)