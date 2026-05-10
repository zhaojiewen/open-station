package loadbalancer

import (
	"fmt"
	"time"
)

// StrategyFactory creates load balancers with preset configurations
type StrategyFactory struct{}

// NewStrategyFactory creates a new strategy factory
func NewStrategyFactory() *StrategyFactory {
	return &StrategyFactory{}
}

// Create creates a load balancer with the specified strategy
func (f *StrategyFactory) Create(strategy StrategyType) *LoadBalancer {
	config := LoadBalancerConfig{
		Strategy:          strategy,
		CooldownDuration:  10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
	return NewLoadBalancer(config)
}

// CreateWithConfig creates a load balancer with custom configuration
func (f *StrategyFactory) CreateWithConfig(config LoadBalancerConfig) *LoadBalancer {
	return NewLoadBalancer(config)
}

// CreateDefault creates a load balancer with default (priority) strategy
func (f *StrategyFactory) CreateDefault() *LoadBalancer {
	return f.Create(StrategyPriority)
}

// CreateRoundRobin creates a load balancer with round-robin strategy
func (f *StrategyFactory) CreateRoundRobin() *LoadBalancer {
	return f.Create(StrategyRoundRobin)
}

// CreateWeighted creates a load balancer with weighted round-robin strategy
func (f *StrategyFactory) CreateWeighted() *LoadBalancer {
	return f.Create(StrategyWeightedRoundRobin)
}

// CreateLeastConnections creates a load balancer with least connections strategy
func (f *StrategyFactory) CreateLeastConnections() *LoadBalancer {
	return f.Create(StrategyLeastConnections)
}

// CreateLeastLatency creates a load balancer with least response time strategy
func (f *StrategyFactory) CreateLeastLatency() *LoadBalancer {
	return f.Create(StrategyLeastResponseTime)
}

// CreateHealthScore creates a load balancer with health score strategy
func (f *StrategyFactory) CreateHealthScore() *LoadBalancer {
	return f.Create(StrategyHealthScore)
}

// CreateRandom creates a load balancer with random selection strategy
func (f *StrategyFactory) CreateRandom() *LoadBalancer {
	return f.Create(StrategyRandom)
}

// CreateAdaptive creates a load balancer with adaptive AI-driven strategy
func (f *StrategyFactory) CreateAdaptive() *LoadBalancer {
	return f.Create(StrategyAdaptive)
}

// StrategyInfo provides information about a strategy
type StrategyInfo struct {
	Name        StrategyType
	Description string
	BestFor     string
	Pros        []string
	Cons        []string
}

// GetStrategyInfo returns detailed information about each strategy
func GetStrategyInfo() map[StrategyType]StrategyInfo {
	return map[StrategyType]StrategyInfo{
		StrategyPriority: {
			Name:        StrategyPriority,
			Description: "Selects account based on configured priority order",
			BestFor:     "When you have preferred accounts and backups",
			Pros:        []string{"Simple configuration", "Deterministic behavior", "Easy to understand"},
			Cons:        []string{"No load distribution", "Preferred account gets all traffic"},
		},
		StrategyRoundRobin: {
			Name:        StrategyRoundRobin,
			Description: "Distributes requests evenly across all accounts in rotation",
			BestFor:     "Load balancing across equal-capacity accounts",
			Pros:        []string{"Fair distribution", "Even load spread", "Simple"},
			Cons:        []string{"Ignores account capacity", "No health consideration"},
		},
		StrategyWeightedRoundRobin: {
			Name:        StrategyWeightedRoundRobin,
			Description: "Distributes based on account weight and health score",
			BestFor:     "Accounts with different capacities/performance",
			Pros:        []string{"Flexible weight config", "Accounts with better health get more traffic"},
			Cons:        []string{"Requires weight tuning", "More complex setup"},
		},
		StrategyLeastConnections: {
			Name:        StrategyLeastConnections,
			Description: "Selects account with fewest active connections",
			BestFor:     "Long-running requests (streaming, large payloads)",
			Pros:        []string{"Real-time load balancing", "Prevents overload"},
			Cons:        []string{"Connection tracking overhead", "Short requests might skew"},
		},
		StrategyLeastResponseTime: {
			Name:        StrategyLeastResponseTime,
			Description: "Selects account with lowest average latency",
			BestFor:     "Performance-sensitive applications",
			Pros:        []string{"Optimizes for speed", "Auto-adapts to performance"},
			Cons:        []string{"Cold start issue", "Needs traffic to calibrate"},
		},
		StrategyHealthScore: {
			Name:        StrategyHealthScore,
			Description: "Selects account with highest computed health score",
			BestFor:     "Reliability-focused production systems",
			Pros:        []string{"Multi-factor evaluation", "Error-aware selection"},
			Cons:        []string{"Complex scoring", "May avoid new accounts"},
		},
		StrategyRandom: {
			Name:        StrategyRandom,
			Description: "Randomly selects from available accounts",
			BestFor:     "Testing, simple setups, equal accounts",
			Pros:        []string{"Simplest implementation", "No state tracking"},
			Cons:        []string{"Uneven distribution possible", "No optimization"},
		},
		StrategyAdaptive: {
			Name:        StrategyAdaptive,
			Description: "AI-driven selection based on multiple weighted factors",
			BestFor:     "Production systems requiring optimal performance",
			Pros:        []string{"Multi-factor optimization", "Dynamic adjustment", "Best overall score"},
			Cons:        []string{"Most complex", "Needs calibration time", "Higher overhead"},
		},
	}
}

// GetAvailableStrategies returns list of all available strategies
func GetAvailableStrategies() []StrategyType {
	return []StrategyType{
		StrategyPriority,
		StrategyRoundRobin,
		StrategyWeightedRoundRobin,
		StrategyLeastConnections,
		StrategyLeastResponseTime,
		StrategyHealthScore,
		StrategyRandom,
		StrategyAdaptive,
	}
}

// ValidateStrategy checks if a strategy type is valid
func ValidateStrategy(strategy StrategyType) error {
	valid := GetAvailableStrategies()
	for _, s := range valid {
		if s == strategy {
			return nil
		}
	}
	return fmt.Errorf("invalid strategy: %s (valid options: %v)", strategy, valid)
}

// ParseStrategy parses a string into StrategyType
func ParseStrategy(s string) (StrategyType, error) {
	strategy := StrategyType(s)
	if err := ValidateStrategy(strategy); err != nil {
		return "", err
	}
	return strategy, nil
}

// RecommendedStrategy returns recommended strategy for different scenarios
type Scenario string

const (
	ScenarioProduction      Scenario = "production"
	ScenarioDevelopment     Scenario = "development"
	ScenarioHighVolume      Scenario = "high_volume"
	ScenarioLongStreaming   Scenario = "long_streaming"
	ScenarioMixedCapacity   Scenario = "mixed_capacity"
	ScenarioReliabilityFirst Scenario = "reliability_first"
)

// GetRecommendedStrategy returns recommended strategy for a scenario
func GetRecommendedStrategy(scenario Scenario) StrategyType {
	switch scenario {
	case ScenarioProduction:
		return StrategyAdaptive // Best overall performance
	case ScenarioDevelopment:
		return StrategyPriority // Simple, predictable
	case ScenarioHighVolume:
		return StrategyLeastConnections // Prevent overload
	case ScenarioLongStreaming:
		return StrategyLeastConnections // Balance long connections
	case ScenarioMixedCapacity:
		return StrategyWeightedRoundRobin // Different capacities
	case ScenarioReliabilityFirst:
		return StrategyHealthScore // Maximize reliability
	default:
		return StrategyPriority
	}
}