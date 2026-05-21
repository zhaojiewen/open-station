package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/pkg/crypto"
)

// ProviderAccountService 管理 Provider 多账户配置和切换
type ProviderAccountService struct {
	repo            repository.ProviderAccountRepository
	encryptKey      []byte
	recoveryTimers  map[uuid.UUID]*time.Timer
	recoveryMutex   sync.Mutex
	stopCh          chan struct{}
}

func NewProviderAccountService(repo repository.ProviderAccountRepository, encryptionKeyHex string) *ProviderAccountService {
	var encryptKey []byte
	if encryptionKeyHex != "" {
		key, err := hex.DecodeString(encryptionKeyHex)
		if err == nil {
			encryptKey = key
		}
	}
	return &ProviderAccountService{
		repo:           repo,
		encryptKey:     encryptKey,
		recoveryTimers: make(map[uuid.UUID]*time.Timer),
		stopCh:         make(chan struct{}),
	}
}

// Stop cancels all pending recovery timers
func (s *ProviderAccountService) Stop() {
	close(s.stopCh)
	s.recoveryMutex.Lock()
	for id, timer := range s.recoveryTimers {
		timer.Stop()
		delete(s.recoveryTimers, id)
	}
	s.recoveryMutex.Unlock()
}

// CreateAccount 创建新的 Provider 账户
func (s *ProviderAccountService) CreateAccount(ctx context.Context, provider, name, apiKey, baseURL string, priority int, monthlyLimit *decimal.Decimal) (*entity.ProviderAccount, error) {
	// 验证 provider
	validProviders := []string{"openai", "anthropic", "deepseek", "glm"}
	isValid := false
	for _, p := range validProviders {
		if p == provider {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, fmt.Errorf("invalid provider: %s, must be one of: %v", provider, validProviders)
	}

	// 检查是否已有账户
	existing, _ := s.repo.GetByProvider(ctx, provider)
	isDefault := len(existing) == 0 // 第一个账户自动设为默认

	encryptedAPIKey := apiKey
	if s.encryptKey != nil && apiKey != "" {
		enc, err := crypto.EncryptString(apiKey, s.encryptKey)
		if err == nil {
			encryptedAPIKey = enc
		}
	}

	account := &entity.ProviderAccount{
		ID:           uuid.New(),
		Provider:     provider,
		Name:         name,
		APIKey:       encryptedAPIKey,
		BaseURL:      baseURL,
		Priority:     priority,
		Status:       "active",
		IsDefault:    isDefault,
		MonthlyLimit: monthlyLimit,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to create provider account: %w", err)
	}

	return account, nil
}

// GetAccount 获取账户详情
func (s *ProviderAccountService) GetAccount(ctx context.Context, id uuid.UUID) (*entity.ProviderAccount, error) {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.decryptAPIKey(account)
	return account, nil
}

// ListAccounts 列出所有账户
func (s *ProviderAccountService) ListAccounts(ctx context.Context, provider string, page, pageSize int) ([]entity.ProviderAccount, int64, error) {
	if provider != "" {
		accounts, err := s.repo.GetByProvider(ctx, provider)
		if err != nil {
			return nil, 0, err
		}
		for i := range accounts {
			s.decryptAPIKey(&accounts[i])
		}
		return accounts, int64(len(accounts)), nil
	}
	accounts, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range accounts {
		s.decryptAPIKey(&accounts[i])
	}
	return accounts, total, nil
}

// UpdateAccount 更新账户
func (s *ProviderAccountService) UpdateAccount(ctx context.Context, id uuid.UUID, name, apiKey, baseURL string, priority int, monthlyLimit *decimal.Decimal) (*entity.ProviderAccount, error) {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != "" {
		account.Name = name
	}
	if apiKey != "" {
		if s.encryptKey != nil {
			enc, err := crypto.EncryptString(apiKey, s.encryptKey)
			if err == nil {
				apiKey = enc
			}
		}
		account.APIKey = apiKey
	}
	if baseURL != "" {
		account.BaseURL = baseURL
	}
	if priority >= 0 {
		account.Priority = priority
	}
	if monthlyLimit != nil {
		account.MonthlyLimit = monthlyLimit
	}
	account.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, account); err != nil {
		return nil, err
	}

	return account, nil
}

// SetDefaultAccount 设置默认账户
func (s *ProviderAccountService) SetDefaultAccount(ctx context.Context, provider string, id uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if account.Provider != provider {
		return fmt.Errorf("account %s does not belong to provider %s", id, provider)
	}

	if account.Status != "active" {
		return fmt.Errorf("cannot set non-active account as default")
	}

	return s.repo.SetDefault(ctx, provider, id)
}

// EnableAccount 启用账户
func (s *ProviderAccountService) EnableAccount(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, "active")
}

// DisableAccount 禁用账户
func (s *ProviderAccountService) DisableAccount(ctx context.Context, id uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if account.IsDefault {
		// 如果禁用的是默认账户，自动选择下一个活跃账户作为默认
		next, err := s.repo.GetNextAvailable(ctx, account.Provider, id)
		if err == nil {
			s.repo.SetDefault(ctx, account.Provider, next.ID)
		}
	}

	return s.repo.UpdateStatus(ctx, id, "disabled")
}

// DeleteAccount 删除账户
func (s *ProviderAccountService) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if account.IsDefault {
		// 如果删除的是默认账户，自动选择下一个活跃账户作为默认
		next, err := s.repo.GetNextAvailable(ctx, account.Provider, id)
		if err == nil {
			s.repo.SetDefault(ctx, account.Provider, next.ID)
		}
	}

	return s.repo.Delete(ctx, id)
}

// GetActiveAccount 获取活跃账户（用于请求）
// 返回优先级最高且状态正常的账户
func (s *ProviderAccountService) GetActiveAccount(ctx context.Context, provider string) (*entity.ProviderAccount, error) {
	// 首先尝试获取默认账户
	account, err := s.repo.GetDefaultByProvider(ctx, provider)
	if err == nil && account.Status == "active" {
		s.decryptAPIKey(account)
		// 检查是否超过月度限额
		if account.MonthlyLimit != nil && account.UsedThisMonth.GreaterThanOrEqual(*account.MonthlyLimit) {
			// 标记为 exhausted
			s.repo.UpdateStatus(ctx, account.ID, "exhausted")
			// 切换到下一个可用账户
			return s.switchToNextAvailable(ctx, provider, account.ID)
		}
		// 检查连续错误次数
		if account.ErrorCount >= 5 {
			s.repo.UpdateStatus(ctx, account.ID, "limited")
			return s.switchToNextAvailable(ctx, provider, account.ID)
		}
		return account, nil
	}

	// 获取所有活跃账户
	accounts, err := s.repo.GetActiveByProvider(ctx, provider)
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("no active account available for provider: %s", provider)
	}

	// 返回优先级最高的账户
	for i := range accounts {
		acc := &accounts[i]
		if acc.Status == "active" {
		s.decryptAPIKey(acc)
			// 检查限额和错误计数
			if acc.MonthlyLimit != nil && acc.UsedThisMonth.GreaterThanOrEqual(*acc.MonthlyLimit) {
				continue
			}
			if acc.ErrorCount >= 5 {
				continue
			}
			return acc, nil
		}
	}

	return nil, fmt.Errorf("no available account for provider: %s (all limited or exhausted)", provider)
}

// switchToNextAvailable 切换到下一个可用账户
func (s *ProviderAccountService) switchToNextAvailable(ctx context.Context, provider string, excludeID uuid.UUID) (*entity.ProviderAccount, error) {
	next, err := s.repo.GetNextAvailable(ctx, provider, excludeID)
	if err != nil {
		return nil, fmt.Errorf("no alternative account available for provider: %s", provider)
	}
	s.decryptAPIKey(next)

	// 设置为默认
	s.repo.SetDefault(ctx, provider, next.ID)

	return next, nil
}

// RecordSuccess 记录成功请求
func (s *ProviderAccountService) RecordSuccess(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error {
	return s.repo.IncrementUsage(ctx, id, cost)
}

// RecordError 记录失败请求
func (s *ProviderAccountService) RecordError(ctx context.Context, id uuid.UUID, errMsg string) error {
	err := s.repo.RecordError(ctx, id, errMsg)

	// 检查是否需要切换账户
	account, _ := s.repo.GetByID(ctx, id)
	if account != nil && account.ErrorCount >= 5 {
		// 标记为 limited
		s.repo.UpdateStatus(ctx, id, "limited")

		// 切换到下一个账户
		if account.IsDefault {
			s.switchToNextAvailable(ctx, account.Provider, id)
		}
	}

	return err
}

// HandleRateLimit 处理 rate limit 错误
func (s *ProviderAccountService) HandleRateLimit(ctx context.Context, id uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 标记为 limited
	s.repo.UpdateStatus(ctx, id, "limited")

	// 切换到下一个账户
	if account.IsDefault {
		_, _ = s.switchToNextAvailable(ctx, account.Provider, id)

		// Cancel any existing timer for this account
		s.recoveryMutex.Lock()
		if existingTimer, exists := s.recoveryTimers[id]; exists {
			existingTimer.Stop()
			delete(s.recoveryTimers, id)
		}

		// Schedule recovery with tracked timer
		timer := time.AfterFunc(5*time.Minute, func() {
			select {
			case <-s.stopCh:
				return
			default:
				recoverCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				s.repo.UpdateStatus(recoverCtx, id, "active")
				s.recoveryMutex.Lock()
				delete(s.recoveryTimers, id)
				s.recoveryMutex.Unlock()
			}
		})
		s.recoveryTimers[id] = timer
		s.recoveryMutex.Unlock()
	}

	return nil
}

// HandleInsufficientQuota 处理余额不足
func (s *ProviderAccountService) HandleInsufficientQuota(ctx context.Context, id uuid.UUID) error {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 标记为 exhausted
	s.repo.UpdateStatus(ctx, id, "exhausted")

	// 切换到下一个账户
	if account.IsDefault {
		s.switchToNextAvailable(ctx, account.Provider, id)
	}

	return nil
}

// ResetMonthlyUsage 重置月度用量（每月初执行）
func (s *ProviderAccountService) ResetMonthlyUsage(ctx context.Context) error {
	return s.repo.ResetMonthlyUsage(ctx)
}

// GetProviderStatus 获取 Provider 状态摘要
func (s *ProviderAccountService) GetProviderStatus(ctx context.Context, provider string) (map[string]interface{}, error) {
	accounts, err := s.repo.GetByProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	var activeCount, limitedCount, exhaustedCount, disabledCount int
	var defaultAccount *entity.ProviderAccount
	totalUsed := decimal.Zero

	for _, acc := range accounts {
		switch acc.Status {
		case "active":
			activeCount++
		case "limited":
			limitedCount++
		case "exhausted":
			exhaustedCount++
		case "disabled":
			disabledCount++
		}

		if acc.IsDefault {
			defaultAccount = &acc
		}

		totalUsed = totalUsed.Add(acc.UsedThisMonth)
	}

	status := "healthy"
	if activeCount == 0 {
		status = "critical"
	} else if activeCount < len(accounts)/2 {
		status = "warning"
	}

	result := map[string]interface{}{
		"provider":        provider,
		"total_accounts":  len(accounts),
		"active":          activeCount,
		"limited":         limitedCount,
		"exhausted":       exhaustedCount,
		"disabled":        disabledCount,
		"status":          status,
		"total_used":      totalUsed.StringFixed(2),
		"default_account": nil,
	}

	if defaultAccount != nil {
		result["default_account"] = map[string]interface{}{
			"id":            defaultAccount.ID.String(),
			"name":          defaultAccount.Name,
			"status":        defaultAccount.Status,
			"used_this_month": defaultAccount.UsedThisMonth.StringFixed(2),
			"error_count":   defaultAccount.ErrorCount,
		}
	}

	return result, nil
}

// GetAllProvidersStatus 获取所有 Provider 状态
func (s *ProviderAccountService) GetAllProvidersStatus(ctx context.Context) (map[string]interface{}, error) {
	providers := []string{"openai", "anthropic", "deepseek", "glm"}
	result := make(map[string]interface{})

	for _, p := range providers {
		status, err := s.GetProviderStatus(ctx, p)
		if err != nil {
			result[p] = map[string]interface{}{
				"provider": p,
				"status":   "not_configured",
				"accounts": 0,
			}
		} else {
			result[p] = status
		}
	}

	return result, nil
}

// CheckAccountHealth 检查账户健康状态
func (s *ProviderAccountService) CheckAccountHealth(ctx context.Context, id uuid.UUID) (map[string]interface{}, error) {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.decryptAPIKey(account)

	health := "healthy"
	issues := []string{}

	// 检查错误计数
	if account.ErrorCount >= 3 {
		health = "warning"
		issues = append(issues, fmt.Sprintf("high error count: %d", account.ErrorCount))
	}

	// 检查月度限额
	if account.MonthlyLimit != nil {
		usagePercent := account.UsedThisMonth.Div(*account.MonthlyLimit).Mul(decimal.NewFromInt(100))
		if usagePercent.GreaterThanOrEqual(decimal.NewFromInt(80)) {
			if usagePercent.GreaterThanOrEqual(decimal.NewFromInt(100)) {
				health = "critical"
				issues = append(issues, "monthly limit exceeded")
			} else {
				if health == "healthy" {
					health = "warning"
				}
				issues = append(issues, fmt.Sprintf("approaching monthly limit: %s%%", usagePercent.StringFixed(0)))
			}
		}
	}

	// 检查最近错误
	if account.LastErrorAt != nil && time.Since(*account.LastErrorAt) < 5*time.Minute {
		if health == "healthy" {
			health = "warning"
		}
		issues = append(issues, fmt.Sprintf("recent error: %s", truncateErrorPtr(account.LastError, 50)))
	}

	return map[string]interface{}{
		"id":            id.String(),
		"provider":      account.Provider,
		"name":          account.Name,
		"status":        account.Status,
		"health":        health,
		"issues":        issues,
		"error_count":   account.ErrorCount,
		"used_this_month": account.UsedThisMonth.StringFixed(2),
	}, nil
}

func truncateError(err string, maxLen int) string {
	if len(err) <= maxLen {
		return err
	}
	return err[:maxLen] + "..."
}

func truncateErrorPtr(err *string, maxLen int) string {
	if err == nil {
		return ""
	}
	return truncateError(*err, maxLen)
}

// IsRateLimitError 判断是否是 rate limit 错误
func (s *ProviderAccountService) IsRateLimitError(errMsg string) bool {
	lowerErrMsg := strings.ToLower(errMsg)
	rateLimitKeywords := []string{
		"rate limit",
		"rate_limit",
		"too many requests",
		"429",
		"quota exceeded",
		"requests per minute",
		"tpm",
		"rpm",
	}

	for _, keyword := range rateLimitKeywords {
		if strings.Contains(lowerErrMsg, keyword) {
			return true
		}
	}
	return false
}

// IsQuotaError 判断是否是余额不足错误
func (s *ProviderAccountService) IsQuotaError(errMsg string) bool {
	lowerErrMsg := strings.ToLower(errMsg)
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
		if strings.Contains(lowerErrMsg, keyword) {
			return true
		}
	}
	return false
}

func (s *ProviderAccountService) decryptAPIKey(account *entity.ProviderAccount) {
	if s.encryptKey != nil && account.APIKey != "" {
		dec, err := crypto.DecryptString(account.APIKey, s.encryptKey)
		if err == nil {
			account.APIKey = dec
		}
	}
}

// ==================== 独享账号 ====================

// CreateDedicatedAccount 创建独享Provider账号（租户或用户）
func (s *ProviderAccountService) CreateDedicatedAccount(ctx context.Context, provider, name, apiKey, baseURL string, tenantID, userID uuid.UUID) (*entity.ProviderAccount, error) {
	encryptedAPIKey := apiKey
	if s.encryptKey != nil && apiKey != "" {
		enc, err := crypto.EncryptString(apiKey, s.encryptKey)
		if err == nil {
			encryptedAPIKey = enc
		}
	}

	account := &entity.ProviderAccount{
		ID:        uuid.New(),
		Provider:  provider,
		Name:      name,
		APIKey:    encryptedAPIKey,
		BaseURL:   baseURL,
		Status:    "active",
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if tenantID != uuid.Nil {
		account.TenantID = &tenantID
	}
	if userID != uuid.Nil {
		account.UserID = &userID
	}

	if err := s.repo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to create dedicated account: %w", err)
	}
	return account, nil
}

// UpdateDedicatedAccount 更新独享账号
func (s *ProviderAccountService) UpdateDedicatedAccount(ctx context.Context, id uuid.UUID, provider, name, apiKey, baseURL string) (*entity.ProviderAccount, error) {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	encryptedAPIKey := apiKey
	if s.encryptKey != nil && apiKey != "" {
		enc, err := crypto.EncryptString(apiKey, s.encryptKey)
		if err == nil {
			encryptedAPIKey = enc
		}
	}

	account.Provider = provider
	account.Name = name
	account.APIKey = encryptedAPIKey
	account.BaseURL = baseURL
	account.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

// ListDedicatedByTenant 列出租户独享账号
func (s *ProviderAccountService) ListDedicatedByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.ProviderAccount, error) {
	return s.repo.ListDedicatedByTenant(ctx, tenantID)
}

// ListDedicatedByUser 列出用户独享账号
func (s *ProviderAccountService) ListDedicatedByUser(ctx context.Context, userID uuid.UUID) ([]entity.ProviderAccount, error) {
	return s.repo.ListDedicatedByUser(ctx, userID)
}

// SetTenantUseDedicated 设置租户是否使用独享账号
func (s *ProviderAccountService) SetTenantUseDedicated(ctx context.Context, tenantID uuid.UUID, enabled bool) error {
	return s.repo.UpdateUseDedicatedTenant(ctx, tenantID, enabled)
}

// SetUserUseDedicated 设置用户是否使用独享账号
func (s *ProviderAccountService) SetUserUseDedicated(ctx context.Context, userID uuid.UUID, enabled bool) error {
	return s.repo.UpdateUseDedicatedUser(ctx, userID, enabled)
}