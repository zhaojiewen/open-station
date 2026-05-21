package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/pkg/loadbalancer"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// ProviderAccountManager 增强 Provider 账户管理，支持实时切换和智能负载均衡
type ProviderAccountManager struct {
	repo           repository.ProviderAccountRepository
	accountCache   map[string]*entity.ProviderAccount // 当前活跃账户缓存
	cacheMutex     sync.RWMutex
	switchCooldown time.Duration // 切换冷却时间，防止频繁切换
	lastSwitchTime map[string]time.Time
	loadBalancer   *loadbalancer.LoadBalancer // Load balancer for account selection
}

// NewProviderAccountManager creates enhanced account manager with load balancer
func NewProviderAccountManager(repo repository.ProviderAccountRepository) *ProviderAccountManager {
	return &ProviderAccountManager{
		repo:           repo,
		accountCache:   make(map[string]*entity.ProviderAccount),
		lastSwitchTime: make(map[string]time.Time),
		switchCooldown: 10 * time.Second,
		loadBalancer:   loadbalancer.NewLoadBalancer(loadbalancer.DefaultLoadBalancerConfig()),
	}
}

// NewProviderAccountManagerWithStrategy creates account manager with specified load balancer strategy
func NewProviderAccountManagerWithStrategy(repo repository.ProviderAccountRepository, strategy loadbalancer.StrategyType) *ProviderAccountManager {
	factory := loadbalancer.NewStrategyFactory()
	return &ProviderAccountManager{
		repo:           repo,
		accountCache:   make(map[string]*entity.ProviderAccount),
		lastSwitchTime: make(map[string]time.Time),
		switchCooldown: 10 * time.Second,
		loadBalancer:   factory.Create(strategy),
	}
}

// NewProviderAccountManagerWithConfig creates account manager with custom load balancer config
func NewProviderAccountManagerWithConfig(repo repository.ProviderAccountRepository, config loadbalancer.LoadBalancerConfig) *ProviderAccountManager {
	return &ProviderAccountManager{
		repo:           repo,
		accountCache:   make(map[string]*entity.ProviderAccount),
		lastSwitchTime: make(map[string]time.Time),
		switchCooldown: config.CooldownDuration,
		loadBalancer:   loadbalancer.NewLoadBalancer(config),
	}
}

// SetLoadBalancerStrategy changes the load balancing strategy
func (m *ProviderAccountManager) SetLoadBalancerStrategy(strategy loadbalancer.StrategyType) {
	m.loadBalancer.SetStrategy(strategy)
	logger.Info("Load balancer strategy updated", zap.String("strategy", string(strategy)))
}

// GetLoadBalancerStrategy returns current load balancing strategy
func (m *ProviderAccountManager) GetLoadBalancerStrategy() loadbalancer.StrategyType {
	return m.loadBalancer.GetStrategy()
}

// GetLoadBalancerStats returns load balancer statistics
func (m *ProviderAccountManager) GetLoadBalancerStats() map[string]interface{} {
	return m.loadBalancer.GetAllStats()
}

// GetActiveAccountWithDedicated 获取活跃账户，优先使用独享账号。
// 优先级：用户独享 > 租户独享 > 公共账号（负载均衡）
func (m *ProviderAccountManager) GetActiveAccountWithDedicated(ctx context.Context, provider string, tenantID, userID uuid.UUID) (*entity.ProviderAccount, error) {
	// 1. 检查用户独享账号
	account, err := m.repo.GetDedicatedByUser(ctx, userID, provider)
	if err == nil && account != nil && m.isAccountUsable(account) {
		return account, nil
	}

	// 2. 检查租户独享账号
	account, err = m.repo.GetDedicatedByTenant(ctx, tenantID, provider)
	if err == nil && account != nil && m.isAccountUsable(account) {
		return account, nil
	}

	// 3. 回退到公共账号
	return m.GetActiveAccount(ctx, provider)
}

// GetActiveAccount 获取当前活跃账户（支持实时切换）
func (m *ProviderAccountManager) GetActiveAccount(ctx context.Context, provider string) (*entity.ProviderAccount, error) {
	// 1. 检查缓存中的账户
	m.cacheMutex.RLock()
	cachedAccount, exists := m.accountCache[provider]
	m.cacheMutex.RUnlock()

	if exists && cachedAccount != nil && cachedAccount.Status == "active" {
		// 验证缓存账户是否仍然可用
		if m.isAccountUsable(cachedAccount) {
			logger.Info("Using cached active account",
				zap.String("provider", provider),
				zap.String("account_name", cachedAccount.Name),
				zap.String("account_id", cachedAccount.ID.String()))
			return cachedAccount, nil
		}
		// 缓存账户不可用，需要切换
		logger.Warn("Cached account is not usable, switching",
			zap.String("provider", provider),
			zap.String("account_id", cachedAccount.ID.String()),
			zap.String("status", cachedAccount.Status))
	}

	// 2. 从数据库获取最佳账户
	account, err := m.selectBestAccount(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("no available account for provider %s: %w", provider, err)
	}

	// 3. 更新缓存
	m.cacheMutex.Lock()
	m.accountCache[provider] = account
	m.cacheMutex.Unlock()

	logger.Info("Selected new active account",
		zap.String("provider", provider),
		zap.String("account_name", account.Name),
		zap.String("account_id", account.ID.String()),
		zap.Int("priority", account.Priority))

	return account, nil
}

// selectBestAccount 智能选择最佳账户（基于负载均衡策略）
func (m *ProviderAccountManager) selectBestAccount(ctx context.Context, provider string) (*entity.ProviderAccount, error) {
	// 获取所有活跃账户
	accounts, err := m.repo.GetActiveByProvider(ctx, provider)
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("no active accounts for provider %s", provider)
	}

	// Convert to slice of pointers for load balancer
	accountPtrs := make([]*entity.ProviderAccount, len(accounts))
	for i := range accounts {
		accountPtrs[i] = &accounts[i]
	}

	// Use load balancer to select best account based on configured strategy
	selected, err := m.loadBalancer.SelectAccount(ctx, accountPtrs, provider)
	if err != nil {
		// Fallback to priority-based selection if load balancer fails
		for _, acc := range accounts {
			if m.isAccountUsable(&acc) {
				return &acc, nil
			}
		}
		return nil, fmt.Errorf("all accounts for provider %s are unusable", provider)
	}

	// Record selection for stats tracking
	m.loadBalancer.RecordRequestStart(selected.ID.String())

	logger.Info("Account selected by load balancer",
		zap.String("provider", provider),
		zap.String("strategy", string(m.loadBalancer.GetStrategy())),
		zap.String("account_id", selected.ID.String()),
		zap.String("account_name", selected.Name))

	return selected, nil
}

// RecordRequestSuccess records successful request to an account
func (m *ProviderAccountManager) RecordRequestSuccess(accountID uuid.UUID, latencyMs int64) {
	m.loadBalancer.RecordRequestSuccess(accountID.String(), latencyMs)
	m.loadBalancer.RecordRequestEnd(accountID.String())
}

// RecordRequestFailure records failed request to an account
func (m *ProviderAccountManager) RecordRequestFailure(accountID uuid.UUID) {
	m.loadBalancer.RecordRequestFailure(accountID.String())
	m.loadBalancer.RecordRequestEnd(accountID.String())
}

// SetAccountWeight sets weight for an account (used in weighted strategies)
func (m *ProviderAccountManager) SetAccountWeight(accountID uuid.UUID, weight int) {
	m.loadBalancer.SetAccountWeight(accountID.String(), weight)
}

// isAccountUsable 检查账户是否可用
func (m *ProviderAccountManager) isAccountUsable(account *entity.ProviderAccount) bool {
	// 1. 检查状态
	if account.Status != "active" {
		return false
	}

	// 2. 检查月度限额
	if account.MonthlyLimit != nil && account.UsedThisMonth.GreaterThanOrEqual(*account.MonthlyLimit) {
		return false
	}

	// 3. 检查连续错误次数
	if account.ErrorCount >= 5 {
		return false
	}

	// 4. 检查最近错误时间（5分钟内有错误则暂时不使用）
	if account.LastErrorAt != nil && time.Since(*account.LastErrorAt) < 5*time.Minute {
		return false
	}

	return true
}

// SwitchAccount 实时切换到指定账户（手动切换）
func (m *ProviderAccountManager) SwitchAccount(ctx context.Context, provider string, accountID uuid.UUID) error {
	// 检查冷却时间
	m.cacheMutex.RLock()
	lastSwitch, exists := m.lastSwitchTime[provider]
	m.cacheMutex.RUnlock()

	if exists && time.Since(lastSwitch) < m.switchCooldown {
		return fmt.Errorf("account switch cooldown active, please wait %v", m.switchCooldown-time.Since(lastSwitch))
	}

	// 获取目标账户
	account, err := m.repo.GetByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	if account.Provider != provider {
		return fmt.Errorf("account %s does not belong to provider %s", accountID, provider)
	}

	if account.Status != "active" {
		return fmt.Errorf("account %s is not active (status: %s)", accountID, account.Status)
	}

	// 设置为默认账户
	err = m.repo.SetDefault(ctx, provider, accountID)
	if err != nil {
		return fmt.Errorf("failed to set default account: %w", err)
	}

	// 更新缓存
	m.cacheMutex.Lock()
	m.accountCache[provider] = account
	m.lastSwitchTime[provider] = time.Now()
	m.cacheMutex.Unlock()

	logger.Info("Account switched successfully",
		zap.String("provider", provider),
		zap.String("account_id", accountID.String()),
		zap.String("account_name", account.Name))

	// 发布切换事件（用于通知其他组件）
	m.publishSwitchEvent(provider, account)

	return nil
}

// HandleAccountFailure 处理账户失败，实时切换到备用账户
func (m *ProviderAccountManager) HandleAccountFailure(ctx context.Context, provider string, failedAccountID uuid.UUID, errMsg string) (*entity.ProviderAccount, error) {
	logger.Warn("Handling account failure",
		zap.String("provider", provider),
		zap.String("failed_account_id", failedAccountID.String()),
		zap.String("error", errMsg))

	// 1. 更新失败账户状态
	account, err := m.repo.GetByID(ctx, failedAccountID)
	if err != nil {
		return nil, err
	}

	// 根据错误类型更新状态
	newStatus := "active"
	if m.IsRateLimitError(errMsg) {
		newStatus = "limited"
	} else if m.IsQuotaError(errMsg) {
		newStatus = "exhausted"
	} else if account.ErrorCount >= 4 { // 这次失败后将达到5次
		newStatus = "limited"
	}

	// 更新状态
	m.repo.UpdateStatus(ctx, failedAccountID, newStatus)
	m.repo.RecordError(ctx, failedAccountID, errMsg)

	// 2. 立即切换到备用账户
	nextAccount, err := m.repo.GetNextAvailable(ctx, provider, failedAccountID)
	if err != nil {
		logger.Error("No backup account available",
			zap.String("provider", provider),
			zap.Error(err))
		return nil, fmt.Errorf("no backup account available for provider %s", provider)
	}

	// 3. 设置为默认账户
	m.repo.SetDefault(ctx, provider, nextAccount.ID)

	// 4. 更新缓存
	m.cacheMutex.Lock()
	m.accountCache[provider] = nextAccount
	m.cacheMutex.Unlock()

	logger.Info("Switched to backup account",
		zap.String("provider", provider),
		zap.String("new_account_id", nextAccount.ID.String()),
		zap.String("new_account_name", nextAccount.Name),
		zap.Int("priority", nextAccount.Priority))

	// 发布切换事件
	m.publishSwitchEvent(provider, nextAccount)

	return nextAccount, nil
}

// publishSwitchEvent 发布账户切换事件（用于通知和监控）
func (m *ProviderAccountManager) publishSwitchEvent(provider string, newAccount *entity.ProviderAccount) {
	event := AccountSwitchEvent{
		Provider:    provider,
		AccountID:   newAccount.ID,
		AccountName: newAccount.Name,
		Priority:    newAccount.Priority,
		Timestamp:   time.Now(),
	}

	// 将事件记录到日志（未来可以发送到消息队列或监控系统）
	logger.Info("Account switch event",
		zap.String("provider", event.Provider),
		zap.String("account_id", event.AccountID.String()),
		zap.String("account_name", event.AccountName),
		zap.Time("timestamp", event.Timestamp))

	// TODO: 发送到消息队列（如 Redis Pub/Sub、Kafka 等）
	// TODO: 发送到监控系统（如 Prometheus、Grafana 等）
}

// AccountSwitchEvent 账户切换事件
type AccountSwitchEvent struct {
	Provider    string
	AccountID   uuid.UUID
	AccountName string
	Priority    int
	Timestamp   time.Time
}

// GetAccountStatus 获取账户详细状态（用于监控）
func (m *ProviderAccountManager) GetAccountStatus(ctx context.Context, accountID uuid.UUID) (map[string]interface{}, error) {
	account, err := m.repo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// 计算健康度分数（0-100）
	healthScore := m.calculateHealthScore(account)

	// 计算使用率
	usageRate := 0.0
	if account.MonthlyLimit != nil {
		usageRate = account.UsedThisMonth.Div(*account.MonthlyLimit).Mul(decimal.NewFromInt(100)).InexactFloat64()
	}

	// 预计剩余容量
	remainingQuota := "unlimited"
	if account.MonthlyLimit != nil {
		remaining := account.MonthlyLimit.Sub(account.UsedThisMonth)
		remainingQuota = remaining.StringFixed(2)
	}

	return map[string]interface{}{
		"id":                 account.ID.String(),
		"provider":           account.Provider,
		"name":               account.Name,
		"status":             account.Status,
		"health_score":       healthScore,
		"priority":           account.Priority,
		"is_default":         account.IsDefault,
		"monthly_limit":      account.MonthlyLimit.StringFixed(2),
		"used_this_month":    account.UsedThisMonth.StringFixed(2),
		"usage_rate":         fmt.Sprintf("%.2f%%", usageRate),
		"remaining_quota":    remainingQuota,
		"error_count":        account.ErrorCount,
		"success_rate":       m.calculateSuccessRate(account),
		"last_used":          account.LastUsedAt,
		"last_error":         account.LastError,
		"last_error_time":    account.LastErrorAt,
		"is_currently_used":  m.isCurrentlyUsed(account.Provider, account.ID),
		"recommendation":     m.getAccountRecommendation(account),
	}, nil
}

// calculateHealthScore 计算账户健康度分数（0-100）
func (m *ProviderAccountManager) calculateHealthScore(account *entity.ProviderAccount) int {
	score := 100

	// 状态影响
	if account.Status != "active" {
		return 0 // 非 active 状态直接返回 0
	}

	// 错误次数影响（每次错误减10分）
	score -= account.ErrorCount * 10
	if score < 0 {
		score = 0
	}

	// 月度限额使用影响（使用超过50%减10分，超过80%减30分）
	if account.MonthlyLimit != nil {
		usageRate := account.UsedThisMonth.Div(*account.MonthlyLimit)
		if usageRate.GreaterThanOrEqual(decimal.NewFromFloat(0.8)) {
			score -= 30
		} else if usageRate.GreaterThanOrEqual(decimal.NewFromFloat(0.5)) {
			score -= 10
		}
	}

	// 最近错误时间影响（5分钟内有错误减20分）
	if account.LastErrorAt != nil && time.Since(*account.LastErrorAt) < 5*time.Minute {
		score -= 20
	}

	// 成功率影响
	successRate := m.calculateSuccessRate(account)
	if successRate < 0.5 {
		score -= 20
	} else if successRate < 0.8 {
		score -= 10
	}

	if score < 0 {
		score = 0
	}

	return score
}

// calculateSuccessRate 计算成功率
func (m *ProviderAccountManager) calculateSuccessRate(account *entity.ProviderAccount) float64 {
	if account.TotalRequests == 0 {
		return 1.0 // 没有请求时默认100%成功率
	}
	return float64(account.TotalSuccess) / float64(account.TotalRequests)
}

// isCurrentlyUsed 检查账户是否当前正在使用
func (m *ProviderAccountManager) isCurrentlyUsed(provider string, accountID uuid.UUID) bool {
	m.cacheMutex.RLock()
	currentAccount := m.accountCache[provider]
	m.cacheMutex.RUnlock()

	return currentAccount != nil && currentAccount.ID == accountID
}

// getAccountRecommendation 获取账户建议
func (m *ProviderAccountManager) getAccountRecommendation(account *entity.ProviderAccount) string {
	healthScore := m.calculateHealthScore(account)

	if healthScore >= 80 {
		return "healthy - continue using"
	} else if healthScore >= 60 {
		return "fair - monitor closely"
	} else if healthScore >= 40 {
		return "warning - consider switching to backup"
	} else if healthScore >= 20 {
		return "poor - switch to backup account"
	} else {
		return "critical - do not use"
	}
}

// IsRateLimitError 判断是否是 rate limit 错误
func (m *ProviderAccountManager) IsRateLimitError(errMsg string) bool {
	rateLimitKeywords := []string{
		"rate limit",
		"rate_limit",
		"too many requests",
		"429",
		"quota exceeded",
		"requests per minute",
		"TPM",
		"RPM",
	}

	for _, keyword := range rateLimitKeywords {
		if containsIgnoreCase(errMsg, keyword) {
			return true
		}
	}
	return false
}

// IsQuotaError 判断是否是余额不足错误
func (m *ProviderAccountManager) IsQuotaError(errMsg string) bool {
	quotaKeywords := []string{
		"insufficient_quota",
		"insufficient quota",
		"quota exceeded",
		"billing_hard_limit_reached",
		"no remaining credits",
		"out of credits",
		"credit balance",
	}

	for _, keyword := range quotaKeywords {
		if containsIgnoreCase(errMsg, keyword) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// RefreshCache 刷新账户缓存（用于定期更新）
func (m *ProviderAccountManager) RefreshCache(ctx context.Context) error {
	providers := []string{"openai", "anthropic", "deepseek", "glm"}

	for _, provider := range providers {
		account, err := m.repo.GetDefaultByProvider(ctx, provider)
		if err == nil && account.Status == "active" {
			m.cacheMutex.Lock()
			m.accountCache[provider] = account
			m.cacheMutex.Unlock()
		} else {
			// 清除缓存
			m.cacheMutex.Lock()
			delete(m.accountCache, provider)
			m.cacheMutex.Unlock()
		}
	}

	logger.Info("Account cache refreshed")
	return nil
}

// PreloadAccounts 预加载账户缓存（启动时执行）
func (m *ProviderAccountManager) PreloadAccounts(ctx context.Context) error {
	return m.RefreshCache(ctx)
}

// GetCacheStats 获取缓存统计信息
func (m *ProviderAccountManager) GetCacheStats() map[string]interface{} {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	stats := make(map[string]interface{})
	for provider, account := range m.accountCache {
		stats[provider] = map[string]interface{}{
			"account_id":   account.ID.String(),
			"account_name": account.Name,
			"status":       account.Status,
			"priority":     account.Priority,
		}
	}

	return map[string]interface{}{
		"cached_providers": len(m.accountCache),
		"accounts":         stats,
		"last_switch":      m.lastSwitchTime,
	}
}