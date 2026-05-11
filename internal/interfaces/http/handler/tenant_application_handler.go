package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/zhaojiewen/open-station/internal/application/service"
)

// TenantApplicationHandler handles tenant application endpoints
type TenantApplicationHandler struct {
	appService *service.TenantApplicationService
}

// NewTenantApplicationHandler creates a new tenant application handler
func NewTenantApplicationHandler(appService *service.TenantApplicationService) *TenantApplicationHandler {
	return &TenantApplicationHandler{
		appService: appService,
	}
}

// Submit handles tenant application submission
func (h *TenantApplicationHandler) Submit(c *gin.Context) {
	var req struct {
		CompanyName    string `json:"company_name" binding:"required"`
		CompanySlug    string `json:"company_slug" binding:"required"`
		ContactEmail   string `json:"contact_email" binding:"required,email"`
		ContactPhone   string `json:"contact_phone"`
		ContactName    string `json:"contact_name" binding:"required"`
		RequestedPlan  string `json:"requested_plan"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Default plan to free if not specified
	if req.RequestedPlan == "" {
		req.RequestedPlan = "free"
	}

	app, err := h.appService.Submit(
		c.Request.Context(),
		req.CompanyName,
		req.CompanySlug,
		req.ContactEmail,
		req.ContactPhone,
		req.ContactName,
		req.RequestedPlan,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"application": gin.H{
			"id":          app.ID,
			"company_name": app.CompanyName,
			"status":      app.Status,
			"created_at":  app.CreatedAt,
		},
	})
}

// GetStatus returns the status of a tenant application
func (h *TenantApplicationHandler) GetStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application id"})
		return
	}

	app, err := h.appService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"application": gin.H{
			"id":              app.ID,
			"company_name":    app.CompanyName,
			"status":          app.Status,
			"requested_plan":  app.RequestedPlan,
			"created_at":      app.CreatedAt,
			"reviewed_at":     app.ReviewedAt,
			"review_notes":    app.ReviewNotes,
			"rejection_reason": app.RejectionReason,
			"tenant_id":       app.TenantID,
		},
	})
}