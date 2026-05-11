package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
)

// CreditApplicationHandler handles credit application requests
type CreditApplicationHandler struct {
	creditAppSvc *service.CreditApplicationService
}

// NewCreditApplicationHandler creates a new credit application handler
func NewCreditApplicationHandler(creditAppSvc *service.CreditApplicationService) *CreditApplicationHandler {
	return &CreditApplicationHandler{
		creditAppSvc: creditAppSvc,
	}
}

// ==================== Tenant Routes ====================

// ApplyForCredit handles POST /tenant/credit-application
func (h *CreditApplicationHandler) ApplyForCredit(c *gin.Context) {
	// Get tenant ID from context (set by auth middleware)
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	var req struct {
		RequestedLimit  decimal.Decimal `json:"requested_limit" binding:"required"`
		Reason          string          `json:"reason"`
		SettlementCycle string          `json:"settlement_cycle"` // monthly, weekly, threshold, custom
		ThresholdAmount *decimal.Decimal `json:"threshold_amount"`
		SettlementDay   *int            `json:"settlement_day"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	applicationReq := &service.CreditApplicationRequest{
		RequestedLimit:  req.RequestedLimit,
		Reason:          req.Reason,
		SettlementCycle: req.SettlementCycle,
		ThresholdAmount: req.ThresholdAmount,
		SettlementDay:   req.SettlementDay,
	}

	application, err := h.creditAppSvc.ApplyForCredit(c.Request.Context(), tenantID.(uuid.UUID), applicationReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, application)
}

// GetApplication handles GET /tenant/credit-application
func (h *CreditApplicationHandler) GetApplication(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	application, err := h.creditAppSvc.GetTenantApplication(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no credit application found"})
		return
	}

	c.JSON(http.StatusOK, application)
}

// UpdateApplication handles PUT /tenant/credit-application
func (h *CreditApplicationHandler) UpdateApplication(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	// Get existing application
	application, err := h.creditAppSvc.GetTenantApplication(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no credit application found"})
		return
	}

	var req struct {
		RequestedLimit  decimal.Decimal `json:"requested_limit"`
		Reason          string          `json:"reason"`
		SettlementCycle string          `json:"settlement_cycle"`
		ThresholdAmount *decimal.Decimal `json:"threshold_amount"`
		SettlementDay   *int            `json:"settlement_day"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateReq := &service.CreditApplicationUpdateRequest{
		RequestedLimit:  req.RequestedLimit,
		Reason:          req.Reason,
		SettlementCycle: req.SettlementCycle,
		ThresholdAmount: req.ThresholdAmount,
		SettlementDay:   req.SettlementDay,
	}

	updatedApp, err := h.creditAppSvc.UpdateApplication(c.Request.Context(), application.ID, updateReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedApp)
}

// CancelApplication handles DELETE /tenant/credit-application
func (h *CreditApplicationHandler) CancelApplication(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	// Get existing application
	application, err := h.creditAppSvc.GetTenantApplication(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no credit application found"})
		return
	}

	if err := h.creditAppSvc.CancelApplication(c.Request.Context(), application.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application cancelled"})
}

// ==================== Platform Admin Routes ====================

// ListApplications handles GET /platform/credit-applications
func (h *CreditApplicationHandler) ListApplications(c *gin.Context) {
	// Check platform admin permission
	if !middleware.HasPlatformPermission(c, "credit_review") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if status != "" {
		applications, total, err := h.creditAppSvc.ListApplicationsByStatus(c.Request.Context(), status, page, pageSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"applications": applications,
			"total":        total,
			"page":         page,
			"page_size":    pageSize,
		})
		return
	}

	applications, total, err := h.creditAppSvc.ListApplications(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": applications,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
	})
}

// GetApplicationDetail handles GET /platform/credit-applications/:id
func (h *CreditApplicationHandler) GetApplicationDetail(c *gin.Context) {
	if !middleware.HasPlatformPermission(c, "credit_review") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	applicationIDStr := c.Param("id")
	applicationID, err := uuid.Parse(applicationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application id"})
		return
	}

	application, err := h.creditAppSvc.GetApplication(c.Request.Context(), applicationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	c.JSON(http.StatusOK, application)
}

// ReviewApplication handles POST /platform/credit-applications/:id/review
func (h *CreditApplicationHandler) ReviewApplication(c *gin.Context) {
	if !middleware.HasPlatformPermission(c, "credit_review") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	applicationIDStr := c.Param("id")
	applicationID, err := uuid.Parse(applicationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application id"})
		return
	}

	// Get reviewer ID from context
	reviewerID, exists := c.Get("platform_admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "reviewer not found"})
		return
	}

	var req struct {
		Status        string          `json:"status" binding:"required"` // approved, rejected
		ApprovedLimit decimal.Decimal `json:"approved_limit"`
		ReviewNotes   string          `json:"review_notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var application *entity.CreditApplication

	if req.Status == "approved" {
		if req.ApprovedLimit.LessThanOrEqual(decimal.Zero) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "approved limit must be positive"})
			return
		}
		approvalReq := &service.ApprovalRequest{
			ApprovedLimit: req.ApprovedLimit,
			ReviewNotes:   req.ReviewNotes,
		}
		application, err = h.creditAppSvc.ApproveApplication(c.Request.Context(), applicationID, reviewerID.(uuid.UUID), approvalReq)
	} else if req.Status == "rejected" {
		application, err = h.creditAppSvc.RejectApplication(c.Request.Context(), applicationID, reviewerID.(uuid.UUID), req.ReviewNotes)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status, must be approved or rejected"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, application)
}

// AdjustCreditLimit handles PUT /platform/tenants/:id/credit
func (h *CreditApplicationHandler) AdjustCreditLimit(c *gin.Context) {
	if !middleware.HasPlatformPermission(c, "credit_manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	var req struct {
		CreditLimit decimal.Decimal `json:"credit_limit" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.creditAppSvc.AdjustCreditLimit(c.Request.Context(), tenantID, req.CreditLimit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id":    tenantID,
		"credit_limit": req.CreditLimit.String(),
	})
}

// GetPendingCount handles GET /platform/credit-applications/pending-count
func (h *CreditApplicationHandler) GetPendingCount(c *gin.Context) {
	if !middleware.HasPlatformPermission(c, "credit_review") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	count, err := h.creditAppSvc.GetPendingCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pending_count": count})
}