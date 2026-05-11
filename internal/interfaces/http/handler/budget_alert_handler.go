package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
)

// BudgetAlertHandler handles budget alert endpoints
type BudgetAlertHandler struct {
	alertService *service.BudgetAlertService
}

// NewBudgetAlertHandler creates a new budget alert handler
func NewBudgetAlertHandler(alertService *service.BudgetAlertService) *BudgetAlertHandler {
	return &BudgetAlertHandler{
		alertService: alertService,
	}
}

// Create creates a new budget alert
func (h *BudgetAlertHandler) Create(c *gin.Context) {
	var req struct {
		Scope            string   `json:"scope" binding:"required"`
		ScopeID          string   `json:"scope_id" binding:"required"`
		AlertType        string   `json:"alert_type" binding:"required"`
		ThresholdPercent int      `json:"threshold_percent" binding:"required"`
		NotifyEmails     []string `json:"notify_emails"`
		NotifySlack      string   `json:"notify_slack"`
		NotifyWebhook    string   `json:"notify_webhook"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	scopeID, err := uuid.Parse(req.ScopeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope_id"})
		return
	}

	// Validate scope belongs to tenant (for tenant admin)
	tenantID := middleware.GetTenantID(c)
	if req.Scope == "tenant" && scopeID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot create alert for other tenant"})
		return
	}

	alert, err := h.alertService.Create(
		c.Request.Context(),
		req.Scope,
		scopeID,
		req.AlertType,
		req.ThresholdPercent,
		req.NotifyEmails,
		req.NotifySlack,
		req.NotifyWebhook,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alert": alert})
}

// List lists budget alerts for the tenant
func (h *BudgetAlertHandler) List(c *gin.Context) {
	scope := c.DefaultQuery("scope", "")
	scopeIDStr := c.DefaultQuery("scope_id", "")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	// If scope and scope_id provided, get alerts for that specific scope
	if scope != "" && scopeIDStr != "" {
		scopeID, err := uuid.Parse(scopeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope_id"})
			return
		}

		alerts, err := h.alertService.GetByScope(c.Request.Context(), scope, scopeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"alerts": alerts})
		return
	}

	// Otherwise, list all alerts with pagination
	alerts, total, err := h.alertService.List(c.Request.Context(), pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts":    alerts,
		"total":     total,
		"page":      pageInt,
		"page_size": pageSizeInt,
	})
}

// Get gets a budget alert by ID
func (h *BudgetAlertHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	alert, err := h.alertService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alert": alert})
}

// Update updates a budget alert
func (h *BudgetAlertHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		ThresholdPercent int      `json:"threshold_percent" binding:"required"`
		NotifyEmails     []string `json:"notify_emails"`
		NotifySlack      string   `json:"notify_slack"`
		NotifyWebhook    string   `json:"notify_webhook"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.alertService.Update(
		c.Request.Context(),
		id,
		req.ThresholdPercent,
		req.NotifyEmails,
		req.NotifySlack,
		req.NotifyWebhook,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert updated"})
}

// Enable enables a budget alert
func (h *BudgetAlertHandler) Enable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.alertService.Enable(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert enabled"})
}

// Disable disables a budget alert
func (h *BudgetAlertHandler) Disable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.alertService.Disable(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert disabled"})
}

// Delete deletes a budget alert
func (h *BudgetAlertHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.alertService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert deleted"})
}