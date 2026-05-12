package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/internal/domain/role"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
)

// UserApplicationHandler handles user application endpoints
type UserApplicationHandler struct {
	appService     *service.UserApplicationService
	authService    *auth.AuthService
	userRepo       repository.UserRepository
	tenantRepo     repository.TenantRepository
}

// NewUserApplicationHandler creates a new user application handler
func NewUserApplicationHandler(
	appService *service.UserApplicationService,
	authService *auth.AuthService,
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
) *UserApplicationHandler {
	return &UserApplicationHandler{
		appService:  appService,
		authService: authService,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
	}
}

// SubmitRequest handles user request to join tenant (public endpoint)
func (h *UserApplicationHandler) SubmitRequest(c *gin.Context) {
	var req struct {
		TenantID      string `json:"tenant_id" binding:"required"`
		Email         string `json:"email" binding:"required,email"`
		Name          string `json:"name" binding:"required"`
		RequestedRole string `json:"requested_role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	// Default role to member
	if req.RequestedRole == "" {
		req.RequestedRole = role.TenantRoleMember
	}
	if err := role.RequireRequestableRole(req.RequestedRole); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, err := h.appService.SubmitRequest(c.Request.Context(), tenantID, req.Email, req.Name, req.RequestedRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"application": gin.H{
			"id":         app.ID,
			"tenant_id":  app.TenantID,
			"email":      app.Email,
			"status":     app.Status,
			"created_at": app.CreatedAt,
		},
	})
}

// AcceptInvitation handles invitation acceptance (public endpoint)
func (h *UserApplicationHandler) AcceptInvitation(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := h.appService.AcceptInvitation(c.Request.Context(), req.Token, req.Name, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"tenant_id":  user.TenantID,
		},
		"message": "invitation accepted, user created",
	})
}

// VerifyInvitation verifies an invitation token (public endpoint)
func (h *UserApplicationHandler) VerifyInvitation(c *gin.Context) {
	token := c.Param("token")

	app, err := h.appService.GetByToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid invitation token"})
		return
	}

	// Check status and expiration
	if app.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invitation already used"})
		return
	}

	if app.ExpiresAt != nil && app.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invitation expired"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invitation": gin.H{
			"email":        app.Email,
			"tenant_id":    app.TenantID,
			"requested_role": app.RequestedRole,
			"expires_at":   app.ExpiresAt,
		},
	})
}

// AdminListApplications lists applications for tenant admin
func (h *UserApplicationHandler) AdminListApplications(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	status := c.DefaultQuery("status", "all")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	apps, total, err := h.appService.List(c.Request.Context(), tenantID, status, pageInt, pageSizeInt)
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

// AdminApproveRequest approves a user request
func (h *UserApplicationHandler) AdminApproveRequest(c *gin.Context) {
	_ = middleware.GetTenantID(c) // Just to verify tenant access
	userID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required (min 8 characters)"})
		return
	}

	user, err := h.appService.ApproveRequest(c.Request.Context(), id, userID, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
		"message": "request approved, user created",
	})
}

// AdminRejectRequest rejects a user request
func (h *UserApplicationHandler) AdminRejectRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.appService.RejectRequest(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "request rejected"})
}

// AdminSendInvitation sends an invitation to a user
func (h *UserApplicationHandler) AdminSendInvitation(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	var req struct {
		Email         string `json:"email" binding:"required,email"`
		Name          string `json:"name"`
		RequestedRole string `json:"requested_role"`
		ExpiresIn     int    `json:"expires_in"` // seconds
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Default role to member
	if req.RequestedRole == "" {
		req.RequestedRole = role.TenantRoleMember
	}
	if err := role.RequireRequestableRole(req.RequestedRole); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app, err := h.appService.SendInvitation(c.Request.Context(), tenantID, req.Email, req.Name, req.RequestedRole, userID, req.ExpiresIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invitation": gin.H{
			"id":          app.ID,
			"email":       app.Email,
			"invite_token": app.InviteToken,
			"expires_at":  app.ExpiresAt,
		},
		"message": "invitation sent",
	})
}

// AdminCreateUser directly creates a user
func (h *UserApplicationHandler) AdminCreateUser(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Name     string `json:"name" binding:"required"`
		Role     string `json:"role" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := role.RequireTenantRole(req.Role); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := h.appService.CreateDirect(c.Request.Context(), tenantID, req.Email, req.Name, req.Role, req.Password, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
		"message": "user created",
	})
}

// AdminCancelInvitation cancels an invitation
func (h *UserApplicationHandler) AdminCancelInvitation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.appService.CancelInvitation(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invitation cancelled"})
}