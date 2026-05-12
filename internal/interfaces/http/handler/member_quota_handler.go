package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
)

// MemberQuotaHandler handles member quota management requests
type MemberQuotaHandler struct {
	memberQuotaSvc *service.MemberQuotaService
}

// NewMemberQuotaHandler creates a new member quota handler
func NewMemberQuotaHandler(memberQuotaSvc *service.MemberQuotaService) *MemberQuotaHandler {
	return &MemberQuotaHandler{
		memberQuotaSvc: memberQuotaSvc,
	}
}

// ==================== Tenant Admin Routes ====================

// ListMemberQuotas handles GET /admin/member-quotas
func (h *MemberQuotaHandler) ListMemberQuotas(c *gin.Context) {
	// Get tenant ID from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	quotas, err := h.memberQuotaSvc.ListMemberQuotas(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"member_quotas": quotas})
}

// CreateMemberQuota handles POST /admin/member-quotas
func (h *MemberQuotaHandler) CreateMemberQuota(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	var req struct {
		UserID          uuid.UUID       `json:"user_id" binding:"required"`
		TokenQuotaLimit *int64          `json:"token_quota_limit"`
		CostLimit       *decimal.Decimal `json:"cost_limit"`
		CostLimitType   string          `json:"cost_limit_type"`
		MaxAPIKeys      *int            `json:"max_api_keys"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createReq := &service.MemberQuotaCreateRequest{
		TenantID:        tenantID.(uuid.UUID),
		UserID:          req.UserID,
		TokenQuotaLimit: req.TokenQuotaLimit,
		CostLimit:       req.CostLimit,
		CostLimitType:   req.CostLimitType,
		MaxAPIKeys:      req.MaxAPIKeys,
	}

	memberQuota, err := h.memberQuotaSvc.CreateMemberQuota(c.Request.Context(), createReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, memberQuota)
}

// GetMemberQuota handles GET /admin/member-quotas/:id
func (h *MemberQuotaHandler) GetMemberQuota(c *gin.Context) {
	quotaIDStr := c.Param("id")
	quotaID, err := uuid.Parse(quotaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota id"})
		return
	}

	memberQuota, err := h.memberQuotaSvc.GetMemberQuota(c.Request.Context(), quotaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member quota not found"})
		return
	}

	c.JSON(http.StatusOK, memberQuota)
}

// UpdateMemberQuota handles PUT /admin/member-quotas/:id
func (h *MemberQuotaHandler) UpdateMemberQuota(c *gin.Context) {
	quotaIDStr := c.Param("id")
	quotaID, err := uuid.Parse(quotaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota id"})
		return
	}

	var req struct {
		TokenQuotaLimit *int64          `json:"token_quota_limit"`
		CostLimit       *decimal.Decimal `json:"cost_limit"`
		CostLimitType   string          `json:"cost_limit_type"`
		MaxAPIKeys      *int            `json:"max_api_keys"`
		Status          string          `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateReq := &service.MemberQuotaUpdateRequest{
		TokenQuotaLimit: req.TokenQuotaLimit,
		CostLimit:       req.CostLimit,
		CostLimitType:   req.CostLimitType,
		MaxAPIKeys:      req.MaxAPIKeys,
		Status:          req.Status,
	}

	memberQuota, err := h.memberQuotaSvc.UpdateMemberQuota(c.Request.Context(), quotaID, updateReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, memberQuota)
}

// DeleteMemberQuota handles DELETE /admin/member-quotas/:id
func (h *MemberQuotaHandler) DeleteMemberQuota(c *gin.Context) {
	quotaIDStr := c.Param("id")
	quotaID, err := uuid.Parse(quotaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota id"})
		return
	}

	if err := h.memberQuotaSvc.DeleteMemberQuota(c.Request.Context(), quotaID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member quota deleted"})
}

// SetTokenLimit handles PUT /admin/member-quotas/:id/token-limit
func (h *MemberQuotaHandler) SetTokenLimit(c *gin.Context) {
	quotaIDStr := c.Param("id")
	quotaID, err := uuid.Parse(quotaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota id"})
		return
	}

	var req struct {
		Limit int64 `json:"limit" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.memberQuotaSvc.SetTokenQuotaLimit(c.Request.Context(), quotaID, req.Limit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "token limit updated"})
}

// SetCostLimit handles PUT /admin/member-quotas/:id/cost-limit
func (h *MemberQuotaHandler) SetCostLimit(c *gin.Context) {
	quotaIDStr := c.Param("id")
	quotaID, err := uuid.Parse(quotaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota id"})
		return
	}

	var req struct {
		Limit     decimal.Decimal `json:"limit" binding:"required"`
		LimitType string          `json:"limit_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.memberQuotaSvc.SetCostLimit(c.Request.Context(), quotaID, req.Limit, req.LimitType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cost limit updated"})
}

// GetMemberUsage handles GET /admin/member-quotas/:id/usage
func (h *MemberQuotaHandler) GetMemberUsage(c *gin.Context) {
	quotaIDStr := c.Param("id")
	quotaID, err := uuid.Parse(quotaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota id"})
		return
	}

	usage, err := h.memberQuotaSvc.GetMemberUsage(c.Request.Context(), quotaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member quota not found"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// ResetMemberQuota handles POST /admin/member-quotas/:id/reset
func (h *MemberQuotaHandler) ResetMemberQuota(c *gin.Context) {
	quotaIDStr := c.Param("id")
	quotaID, err := uuid.Parse(quotaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota id"})
		return
	}

	if err := h.memberQuotaSvc.ResetMemberQuota(c.Request.Context(), quotaID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member quota reset"})
}

// ==================== User Routes ====================

// GetMyMemberQuota handles GET /user/member-quota
func (h *MemberQuotaHandler) GetMyMemberQuota(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	memberQuota, err := h.memberQuotaSvc.GetMemberQuotaByUser(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no member quota found"})
		return
	}

	c.JSON(http.StatusOK, memberQuota)
}

// GetMyMemberUsage handles GET /user/member-usage
func (h *MemberQuotaHandler) GetMyMemberUsage(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	memberQuota, err := h.memberQuotaSvc.GetMemberQuotaByUser(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no member quota found"})
		return
	}

	usage, err := h.memberQuotaSvc.GetMemberUsage(c.Request.Context(), memberQuota.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "failed to get usage"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// ==================== Platform Admin Routes ====================

// ListAllMemberQuotas handles GET /platform/member-quotas
func (h *MemberQuotaHandler) ListAllMemberQuotas(c *gin.Context) {
	if !middleware.HasPlatformPermission(c, "tenant_view") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	quotas, total, err := h.memberQuotaSvc.ListAllMemberQuotas(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"member_quotas": quotas,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}