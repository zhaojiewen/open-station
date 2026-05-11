package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/pkg/errors"
)

// UserAuthServiceInterface defines the interface for user auth service
type UserAuthServiceInterface interface {
	Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error)
	Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error)
	RegisterTenant(ctx context.Context, req *auth.RegisterTenantRequest) (*auth.RegisterTenantResponse, error)
	ValidateToken(ctx context.Context, token string) (*entity.User, *entity.UserTenant, *auth.JWTClaims, error)
	SwitchTenant(ctx context.Context, userID, tenantID uuid.UUID, currentToken string) (string, error)
	Logout(ctx context.Context, token string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	RefreshToken(ctx context.Context, refreshToken string, deviceID string) (string, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
	GetUserTenants(ctx context.Context, userID uuid.UUID) ([]entity.UserTenant, error)
	VerifyEmail(ctx context.Context, token string) (*entity.User, error)
	ResendVerification(ctx context.Context, email string) error
}

// JWTServiceInterface defines the interface for JWT service
type JWTServiceInterface interface {
	GenerateToken(userID, tenantID uuid.UUID, email, role, deviceID string) (accessToken, refreshToken string, tokenID uuid.UUID, err error)
	ValidateToken(token string) (*auth.JWTClaims, error)
	InvalidateToken(token string) error
	InvalidateTokenByID(tokenID uuid.UUID, expiry time.Duration) error
	RefreshToken(refreshToken string, tenantID uuid.UUID, role string) (string, error)
	IsTokenIDBlacklisted(tokenID uuid.UUID) bool
	GetAccessTokenExpiry() time.Duration
	GetRefreshTokenExpiry() time.Duration
}

// AuthHandler 认证Handler
type AuthHandler struct {
	userAuthService UserAuthServiceInterface
	jwtService      JWTServiceInterface
}

// NewAuthHandler 创建认证Handler
func NewAuthHandler(userAuthService UserAuthServiceInterface, jwtService JWTServiceInterface) *AuthHandler {
	return &AuthHandler{
		userAuthService: userAuthService,
		jwtService:      jwtService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// 获取IP和UserAgent
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	deviceID := c.GetHeader("X-Device-ID") // 客户端可提供设备ID

	loginReq := &auth.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		IP:        ip,
		UserAgent: userAgent,
		DeviceID:  deviceID,
	}

	ctx := c.Request.Context()
	resp, err := h.userAuthService.Login(ctx, loginReq)
	if err != nil {
		// Check specific errors first
		if errors.Is(err, errors.ErrTooManyAttempts) {
			// 返回封禁剩余时间 - 需要从login security获取
			// 这里简化处理，返回固定时间
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":           err.Error(),
				"retry_after":     900, // 15分钟
			})
			return
		}
		if errors.Is(err, errors.ErrEmailNotVerified) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "email not verified",
				"code":          "VERIFY_004",
				"require_verify": true,
			})
			return
		}
		if errors.IsLoginError(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建响应
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    resp.User.ID,
			"email": resp.User.Email,
			"name":  resp.User.Name,
			"role":  resp.User.Role,
		},
		"tenants":            h.buildTenantsResponseFromEntity(resp.UserTenants),
		"current_tenant_id":  resp.DefaultTenantID,
		"access_token":       resp.AccessToken,
		"refresh_token":      resp.RefreshToken,
		"expires_at":         resp.ExpiresAt,
		"is_anomaly":         resp.IsAnomaly,
		"anomaly_type":       resp.AnomalyType,
	})
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// Register 个人注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	ctx := c.Request.Context()
	registerReq := &auth.RegisterRequest{
		Email:     req.Email,
		Password:  req.Password,
		Name:      req.Name,
		IP:        ip,
		UserAgent: userAgent,
	}

	resp, err := h.userAuthService.Register(ctx, registerReq)
	if err != nil {
		if errors.IsRegisterError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if email verification is required
	if resp.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":    resp.User.ID,
				"email": resp.User.Email,
				"name":  resp.User.Name,
			},
			"message":        "registration successful, please verify your email",
			"require_verify": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    resp.User.ID,
			"email": resp.User.Email,
			"name":  resp.User.Name,
		},
		"tenant_id":     resp.UserTenant.TenantID,
		"access_token":  resp.AccessToken,
		"refresh_token": resp.RefreshToken,
		"expires_at":    resp.ExpiresAt,
		"require_verify": false,
	})
}

// RegisterTenantRequest 企业注册请求
type RegisterTenantRequest struct {
	TenantName string `json:"tenant_name" binding:"required"`
	TenantSlug string `json:"tenant_slug" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	Name       string `json:"name" binding:"required"`
}

// RegisterTenant 企业注册
func (h *AuthHandler) RegisterTenant(c *gin.Context) {
	var req RegisterTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	ctx := c.Request.Context()
	registerReq := &auth.RegisterTenantRequest{
		TenantName: req.TenantName,
		TenantSlug: req.TenantSlug,
		Email:      req.Email,
		Password:   req.Password,
		Name:       req.Name,
		IP:         ip,
		UserAgent:  userAgent,
	}

	resp, err := h.userAuthService.RegisterTenant(ctx, registerReq)
	if err != nil {
		if errors.Is(err, errors.ErrEmailNotVerified) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":          "email not verified, please verify your existing account first",
				"code":           "VERIFY_004",
				"require_verify": true,
			})
			return
		}
		if errors.IsRegisterError(err) || errors.IsTenantError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if email verification is required
	if resp.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{
			"tenant": gin.H{
				"id":     resp.Tenant.ID,
				"name":   resp.Tenant.Name,
				"slug":   resp.Tenant.Slug,
				"status": resp.Tenant.Status,
				"plan":   resp.Tenant.Plan,
			},
			"user": gin.H{
				"id":    resp.User.ID,
				"email": resp.User.Email,
				"name":  resp.User.Name,
				"role":  resp.UserTenant.Role,
			},
			"message":        "registration successful, please verify your email to activate the account",
			"require_verify": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant": gin.H{
			"id":     resp.Tenant.ID,
			"name":   resp.Tenant.Name,
			"slug":   resp.Tenant.Slug,
			"status": resp.Tenant.Status,
			"plan":   resp.Tenant.Plan,
		},
		"user": gin.H{
			"id":    resp.User.ID,
			"email": resp.User.Email,
			"name":  resp.User.Name,
			"role":  resp.UserTenant.Role,
		},
		"access_token":  resp.AccessToken,
		"refresh_token": resp.RefreshToken,
		"expires_at":    resp.ExpiresAt,
	})
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	token := h.extractToken(c)
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no token provided"})
		return
	}

	ctx := c.Request.Context()
	if err := h.userAuthService.Logout(ctx, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// RefreshTokenRequest 刷新token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	DeviceID     string `json:"device_id"` // 可选
}

// RefreshToken 刷新access token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx := c.Request.Context()
	newAccessToken, err := h.userAuthService.RefreshToken(ctx, req.RefreshToken, req.DeviceID)
	if err != nil {
		if errors.IsLoginError(err) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": newAccessToken,
		"expires_at":   time.Now().Add(h.jwtService.GetAccessTokenExpiry()),
	})
}

// GetProfile 获取用户信息
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := GetUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	tenants, err := h.userAuthService.GetUserTenants(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userEntity := user.(*entity.User)
	tenantID := GetTenantIDFromContext(c)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        userEntity.ID,
			"email":     userEntity.Email,
			"name":      userEntity.Name,
			"role":      userEntity.Role,
			"status":    userEntity.Status,
			"user_mode": userEntity.UserMode,
		},
		"tenants":           h.buildTenantsResponseFromEntity(tenants),
		"current_tenant_id": tenantID,
	})
}

// GetTenants 获取用户所有租户
func (h *AuthHandler) GetTenants(c *gin.Context) {
	userID := GetUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	tenants, err := h.userAuthService.GetUserTenants(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": h.buildTenantsResponseFromEntity(tenants),
	})
}

// SwitchTenantRequest 切换租户请求
type SwitchTenantRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
}

// SwitchTenant 切换当前租户
func (h *AuthHandler) SwitchTenant(c *gin.Context) {
	var req SwitchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	userID := GetUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	currentToken := h.extractToken(c)
	ctx := c.Request.Context()

	newAccessToken, err := h.userAuthService.SwitchTenant(ctx, userID, tenantID, currentToken)
	if err != nil {
		if errors.IsUserError(err) || errors.IsTenantError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":    newAccessToken,
		"current_tenant_id": tenantID,
		"expires_at":      time.Now().Add(h.jwtService.GetAccessTokenExpiry()),
	})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userID := GetUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	if err := h.userAuthService.ChangePassword(ctx, userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.IsLoginError(err) || errors.IsRegisterError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully, please login again"})
}

// LogoutAll 登出所有设备
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := GetUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	if err := h.userAuthService.LogoutAll(ctx, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 同时注销当前token
	currentToken := h.extractToken(c)
	if currentToken != "" {
		h.jwtService.InvalidateToken(currentToken)
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out from all devices"})
}

// VerifyEmailRequest 验证邮箱请求
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyEmail 验证邮箱
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: token required"})
		return
	}

	ctx := c.Request.Context()
	user, err := h.userAuthService.VerifyEmail(ctx, req.Token)
	if err != nil {
		if errors.Is(err, errors.ErrInvalidVerificationToken) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification token"})
			return
		}
		if errors.Is(err, errors.ErrVerificationTokenExpired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "verification token expired, please request a new one"})
			return
		}
		if errors.Is(err, errors.ErrEmailAlreadyVerified) {
			c.JSON(http.StatusOK, gin.H{"message": "email already verified"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "email verified successfully",
		"user_id":       user.ID,
		"email":         user.Email,
		"verified_at":   user.EmailVerifiedAt,
	})
}

// ResendVerificationRequest 重发验证邮件请求
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResendVerification 重发验证邮件
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: email required"})
		return
	}

	ctx := c.Request.Context()
	err := h.userAuthService.ResendVerification(ctx, req.Email)
	if err != nil {
		if errors.Is(err, errors.ErrUserNotFound) {
			// 不暴露用户是否存在，返回成功
			c.JSON(http.StatusOK, gin.H{"message": "if the email exists and is not verified, a verification email will be sent"})
			return
		}
		if errors.Is(err, errors.ErrEmailAlreadyVerified) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email already verified"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send verification email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification email sent successfully"})
}

// helper functions

func (h *AuthHandler) extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// 支持 "Bearer token" 和直接token
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return authHeader
}

func (h *AuthHandler) buildTenantsResponseFromEntity(tenants []entity.UserTenant) []gin.H {
	result := make([]gin.H, 0, len(tenants))
	for _, t := range tenants {
		result = append(result, gin.H{
			"tenant_id":  t.TenantID,
			"role":       t.Role,
			"status":     t.Status,
			"is_default": t.IsDefault,
			"joined_at":  t.JoinedAt,
		})
	}
	return result
}

// GetUserIDFromContext 从context获取用户ID
func GetUserIDFromContext(c *gin.Context) uuid.UUID {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	return userID.(uuid.UUID)
}

// GetTenantIDFromContext 从context获取租户ID
func GetTenantIDFromContext(c *gin.Context) uuid.UUID {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil
	}
	return tenantID.(uuid.UUID)
}