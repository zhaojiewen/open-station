package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// ProviderAccountHandler Provider 账户管理 Handler（实时监控和切换）
type ProviderAccountHandler struct {
	accountManager *service.ProviderAccountManager
	accountService *service.ProviderAccountService
}

// NewProviderAccountHandler 创建账户管理 Handler
func NewProviderAccountHandler(accountManager *service.ProviderAccountManager, accountService *service.ProviderAccountService) *ProviderAccountHandler {
	return &ProviderAccountHandler{
		accountManager: accountManager,
		accountService: accountService,
	}
}

// GetProviderStatus 获取 Provider 实时状态
// GET /admin/providers/:provider/status
func (h *ProviderAccountHandler) GetProviderStatus(c *gin.Context) {
	provider := c.Param("provider")

	if h.accountManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"provider": provider,
			"status":   "not_configured",
			"message":  "Account manager not initialized",
		})
		return
	}

	// 获取缓存统计
	cacheStats := h.accountManager.GetCacheStats()

	// 获取 Provider 状态
	status, err := h.accountService.GetProviderStatus(c.Request.Context(), provider)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"provider": provider,
			"status":   "not_configured",
			"message":  err.Error(),
		})
		return
	}

	// 合并缓存信息
	result := gin.H{
		"provider":     provider,
		"status_info":  status,
		"cache_info":   cacheStats,
		"current_time": gin.H{},
	}

	// 添加当前使用的账户信息
	if accounts, ok := cacheStats["accounts"].(map[string]interface{}); ok {
		if cachedAccount, ok := accounts[provider]; ok {
			result["current_account"] = cachedAccount
		}
	}

	c.JSON(http.StatusOK, result)
}

// GetAllProvidersStatus 获取所有 Provider 实时状态
// GET /admin/providers/status
func (h *ProviderAccountHandler) GetAllProvidersStatus(c *gin.Context) {
	if h.accountManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "not_configured",
			"message": "Account manager not initialized",
		})
		return
	}

	// 获取所有 Provider 状态
	allStatus, err := h.accountService.GetAllProvidersStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取缓存统计
	cacheStats := h.accountManager.GetCacheStats()

	c.JSON(http.StatusOK, gin.H{
		"providers":     allStatus,
		"cache_stats":   cacheStats,
		"recommendation": h.generateOverallRecommendation(allStatus),
	})
}

// SwitchAccount 实时切换到指定账户
// POST /admin/providers/:provider/switch
func (h *ProviderAccountHandler) SwitchAccount(c *gin.Context) {
	provider := c.Param("provider")

	var req struct {
		AccountID string `json:"account_id" binding:"required"`
		Reason    string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required"})
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id format"})
		return
	}

	// 执行切换
	err = h.accountManager.SwitchAccount(c.Request.Context(), provider, accountID)
	if err != nil {
		logger.Error("Failed to switch account",
			zap.String("provider", provider),
			zap.String("account_id", req.AccountID),
			zap.Error(err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    err.Error(),
			"provider": provider,
			"account_id": req.AccountID,
		})
		return
	}

	// 获取新账户状态
	newStatus, err := h.accountManager.GetAccountStatus(c.Request.Context(), accountID)
	if err != nil {
		newStatus = map[string]interface{}{"id": req.AccountID}
	}

	logger.Info("Account switched successfully via API",
		zap.String("provider", provider),
		zap.String("account_id", req.AccountID),
		zap.String("reason", req.Reason))

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Account switched successfully",
		"provider":      provider,
		"new_account":   newStatus,
		"switch_reason": req.Reason,
		"timestamp":     gin.H{},
	})
}

// GetAccountDetail 获取账户详细状态和健康度
// GET /admin/providers/accounts/:account_id
func (h *ProviderAccountHandler) GetAccountDetail(c *gin.Context) {
	accountIDStr := c.Param("account_id")

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id format"})
		return
	}

	// 获取详细状态
	status, err := h.accountManager.GetAccountStatus(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account": status,
	})
}

// ForceRefreshCache 强制刷新账户缓存
// POST /admin/providers/cache/refresh
func (h *ProviderAccountHandler) ForceRefreshCache(c *gin.Context) {
	if h.accountManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Account manager not initialized",
		})
		return
	}

	err := h.accountManager.RefreshCache(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cacheStats := h.accountManager.GetCacheStats()

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Cache refreshed successfully",
		"cache_stats":  cacheStats,
		"timestamp":    gin.H{},
	})
}

// GetCacheStats 获取缓存统计信息
// GET /admin/providers/cache/stats
func (h *ProviderAccountHandler) GetCacheStats(c *gin.Context) {
	if h.accountManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "not_initialized",
		})
		return
	}

	stats := h.accountManager.GetCacheStats()

	c.JSON(http.StatusOK, gin.H{
		"cache": stats,
	})
}

// generateOverallRecommendation 生成整体建议
func (h *ProviderAccountHandler) generateOverallRecommendation(allStatus map[string]interface{}) string {
	criticalCount := 0
	warningCount := 0
	healthyCount := 0

	for _, status := range allStatus {
		if statusMap, ok := status.(map[string]interface{}); ok {
			if statusStr, ok := statusMap["status"].(string); ok {
				switch statusStr {
				case "critical":
					criticalCount++
				case "warning":
					warningCount++
				case "healthy":
					healthyCount++
				}
			}
		}
	}

	if criticalCount > 0 {
		return "⚠️ CRITICAL: Some providers have no available accounts. Immediate action required."
	} else if warningCount > 0 {
		return "⚠️ WARNING: Some providers have limited account availability. Consider adding backup accounts."
	} else if healthyCount == len(allStatus) {
		return "✅ HEALTHY: All providers have sufficient account coverage."
	} else {
		return "📊 INFO: Provider status varies. Monitor closely."
	}
}

// GetSwitchHistory 获取账户切换历史（未来实现）
// GET /admin/providers/:provider/history
func (h *ProviderAccountHandler) GetSwitchHistory(c *gin.Context) {
	// provider := c.Param("provider")

	// TODO: 实现切换历史记录存储和查询
	c.JSON(http.StatusOK, gin.H{
		"provider": c.Param("provider"),
		"history":  []interface{}{},
		"message":  "Switch history feature will be implemented soon",
	})
}

// ManualRecoverAccount 手动恢复账户状态
// POST /admin/providers/accounts/:account_id/recover
func (h *ProviderAccountHandler) ManualRecoverAccount(c *gin.Context) {
	accountIDStr := c.Param("account_id")

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id format"})
		return
	}

	// 启用账户
	err = h.accountService.EnableAccount(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 刷新缓存
	h.accountManager.RefreshCache(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Account recovered successfully",
		"account_id": accountIDStr,
	})
}

// GetRealTimeMetrics 获取实时指标（用于监控仪表板）
// GET /admin/providers/metrics
func (h *ProviderAccountHandler) GetRealTimeMetrics(c *gin.Context) {
	if h.accountManager == nil || h.accountService == nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "not_initialized",
		})
		return
	}

	// 获取所有 Provider 状态
	allStatus, _ := h.accountService.GetAllProvidersStatus(c.Request.Context())
	cacheStats := h.accountManager.GetCacheStats()

	// 计算总体指标
	totalAccounts := 0
	activeAccounts := 0
	limitedAccounts := 0
	exhaustedAccounts := 0

	for _, status := range allStatus {
		if statusMap, ok := status.(map[string]interface{}); ok {
			if total, ok := statusMap["total_accounts"].(int); ok {
				totalAccounts += total
			}
			if active, ok := statusMap["active"].(int); ok {
				activeAccounts += active
			}
			if limited, ok := statusMap["limited"].(int); ok {
				limitedAccounts += limited
			}
			if exhausted, ok := statusMap["exhausted"].(int); ok {
				exhaustedAccounts += exhausted
			}
		}
	}

	// 构建实时指标
	metrics := gin.H{
		"summary": gin.H{
			"total_accounts":    totalAccounts,
			"active_accounts":   activeAccounts,
			"limited_accounts":  limitedAccounts,
			"exhausted_accounts": exhaustedAccounts,
			"health_rate":       float64(activeAccounts) / float64(totalAccounts) * 100,
		},
		"providers":        allStatus,
		"cache_info":       cacheStats,
		"recommendation":   h.generateOverallRecommendation(allStatus),
		"last_updated":     gin.H{},
	}

	c.JSON(http.StatusOK, metrics)
}