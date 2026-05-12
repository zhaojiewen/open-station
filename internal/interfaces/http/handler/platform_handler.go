package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
)

// PlatformHandler handles platform admin operations
type PlatformHandler struct {
	platformAuth         *auth.PlatformAuthService
	tenantAppService     *service.TenantApplicationService
	tenantRepo           repository.TenantRepository
}

// NewPlatformHandler creates a new platform handler
func NewPlatformHandler(
	platformAuth *auth.PlatformAuthService,
	tenantAppService *service.TenantApplicationService,
	tenantRepo repository.TenantRepository,
) *PlatformHandler {
	return &PlatformHandler{
		platformAuth:     platformAuth,
		tenantAppService: tenantAppService,
		tenantRepo:       tenantRepo,
	}
}

// Login handles platform admin login
func (h *PlatformHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	admin, token, err := h.platformAuth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"admin": gin.H{
			"id":    admin.ID,
			"email": admin.Email,
			"name":  admin.Name,
			"role":  admin.Role,
		},
		"token": token,
	})
}

// ListAdmins lists all platform admins
func (h *PlatformHandler) ListAdmins(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	admins, total, err := h.platformAuth.ListAdmins(c.Request.Context(), pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"admins": admins,
		"total":  total,
		"page":   pageInt,
	})
}

// CreateAdmin creates a new platform admin
func (h *PlatformHandler) CreateAdmin(c *gin.Context) {
	var req struct {
		Email       string   `json:"email" binding:"required"`
		Password    string   `json:"password" binding:"required,min=8"`
		Name        string   `json:"name" binding:"required"`
		Role        string   `json:"role" binding:"required"`
		Permissions []string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	actorID := middleware.GetPlatformAdminID(c)

	admin, err := h.platformAuth.CreateAdmin(
		c.Request.Context(),
		actorID,
		req.Email,
		req.Password,
		req.Name,
		req.Role,
		req.Permissions,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"admin": admin})
}

// GetAdmin gets a platform admin by ID
func (h *PlatformHandler) GetAdmin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	admin, err := h.platformAuth.ValidateSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"admin": admin})
}

// UpdateAdmin updates a platform admin
func (h *PlatformHandler) UpdateAdmin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	actorID := middleware.GetPlatformAdminID(c)

	if err := h.platformAuth.UpdateAdmin(c.Request.Context(), actorID, id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin updated"})
}

// DeleteAdmin deletes a platform admin
func (h *PlatformHandler) DeleteAdmin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	actorID := middleware.GetPlatformAdminID(c)

	if err := h.platformAuth.DeleteAdmin(c.Request.Context(), actorID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin deleted"})
}

// ListApplications lists tenant applications
func (h *PlatformHandler) ListApplications(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	apps, total, err := h.tenantAppService.List(c.Request.Context(), status, pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": apps,
		"total":        total,
		"page":         pageInt,
	})
}

// GetApplication gets a tenant application by ID
func (h *PlatformHandler) GetApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	app, err := h.tenantAppService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"application": app})
}

// ApproveApplication approves a tenant application
func (h *PlatformHandler) ApproveApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	c.ShouldBindJSON(&req)

	adminID := middleware.GetPlatformAdminID(c)

	tenant, err := h.tenantAppService.Approve(c.Request.Context(), id, adminID, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "application approved",
		"tenant_id": tenant.ID,
	})
}

// RejectApplication rejects a tenant application
func (h *PlatformHandler) RejectApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rejection reason required"})
		return
	}

	adminID := middleware.GetPlatformAdminID(c)

	if err := h.tenantAppService.Reject(c.Request.Context(), id, adminID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application rejected"})
}

// ListTenants lists all tenants
func (h *PlatformHandler) ListTenants(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	tenants, total, err := h.tenantRepo.List(c.Request.Context(), pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
		"total":   total,
		"page":    pageInt,
	})
}

// SuspendTenant suspends a tenant
func (h *PlatformHandler) SuspendTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	tenant.Status = "suspended"
	if err := h.tenantRepo.Update(c.Request.Context(), tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tenant suspended"})
}

// ActivateTenant activates a tenant
func (h *PlatformHandler) ActivateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	tenant.Status = "active"
	if err := h.tenantRepo.Update(c.Request.Context(), tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tenant activated"})
}