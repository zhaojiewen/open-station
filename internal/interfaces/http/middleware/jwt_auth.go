package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/role"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// JWTServiceInterface defines the interface for JWT service (for middleware)
type JWTServiceInterface interface {
	ValidateToken(token string) (*auth.JWTClaims, error)
	IsTokenIDBlacklisted(tokenID uuid.UUID) bool
}

// UserAuthServiceInterface defines the interface for user auth service (for middleware)
type UserAuthServiceInterface interface {
	ValidateToken(ctx context.Context, token string) (*entity.User, *entity.UserTenant, *auth.JWTClaims, error)
}

// JWTAuthMiddleware JWT认证中间件
func JWTAuthMiddleware(jwtService JWTServiceInterface, userAuthService UserAuthServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 提取token
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization token provided"})
			c.Abort()
			return
		}

		// 2. 验证token
		ctx := c.Request.Context()
		user, userTenant, claims, err := userAuthService.ValidateToken(ctx, token)
		if err != nil {
			if apperrors.Is(err, apperrors.ErrSessionExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":        "token expired",
					"code":         "TOKEN_EXPIRED",
					"refresh_hint": "use refresh_token to get new access_token",
				})
			} else if apperrors.Is(err, apperrors.ErrTokenRevoked) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			}
			c.Abort()
			return
		}

		// 3. 检查设备匹配（可选）
		deviceID := c.GetHeader("X-Device-ID")
		if deviceID != "" && claims.DeviceID != "" && deviceID != claims.DeviceID {
			// 设备不匹配，作为异常信号但不强制拦截（移动用户可能切换设备）
			c.Set("device_mismatch", true)
		}

		// 4. 设置context
		c.Set("user_id", user.ID)
		c.Set("user", user)
		c.Set("tenant_id", claims.TenantID)
		c.Set("user_tenant", userTenant)
		c.Set("role", userTenant.Role)
		c.Set("email", user.Email)
		c.Set("token_id", claims.TokenID)
		c.Set("device_id", claims.DeviceID)

		c.Next()
	}
}

// OptionalJWTAuth 可选的JWT认证（不强制要求）
func OptionalJWTAuth(jwtService JWTServiceInterface, userAuthService UserAuthServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			// 无token，继续执行（不设置用户信息）
			c.Next()
			return
		}

		ctx := c.Request.Context()
		user, userTenant, claims, err := userAuthService.ValidateToken(ctx, token)
		if err != nil {
			// token无效，继续执行但不设置用户信息
			c.Next()
			return
		}

		// 设置context
		c.Set("user_id", user.ID)
		c.Set("user", user)
		c.Set("tenant_id", claims.TenantID)
		c.Set("user_tenant", userTenant)
		c.Set("role", userTenant.Role)

		c.Next()
	}
}

// RequireRole 要求特定角色
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userRole := role.(string)
		allowed := false
		for _, r := range allowedRoles {
			if userRole == r {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin 要求管理员角色
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(role.TenantRoleAdmin)
}

// RequireTenantMember 要求是该租户成员
func RequireTenantMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("user_tenant")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this tenant"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireTenantWrite blocks viewer-only users from write operations.
// Checks UserTenant.Role first (JWT auth), then falls back to user.Role (API key auth).
func RequireTenantWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		effectiveRole := ""

		if ut := GetUserTenant(c); ut != nil {
			effectiveRole = ut.Role
		} else if u := GetUserFromAuth(c); u != nil {
			effectiveRole = u.Role
		}

		if role.IsTenantViewer(effectiveRole) {
			c.JSON(http.StatusForbidden, gin.H{"error": "viewer role cannot perform this operation"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserFromAuth retrieves user from either API key auth or JWT auth context.
func GetUserFromAuth(c *gin.Context) *entity.User {
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*entity.User); ok {
			return u
		}
	}
	return nil
}

// extractToken 从请求中提取token
func extractToken(c *gin.Context) string {
	// 1. 从Authorization header提取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// 支持 "Bearer token" 格式
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		// 支持直接token
		return authHeader
	}

	// 2. 从query参数提取（较少使用）
	token := c.Query("token")
	if token != "" {
		return token
	}

	return ""
}

// GetUserTenant 获取当前用户的租户关联
func GetUserTenant(c *gin.Context) *entity.UserTenant {
	userTenant, exists := c.Get("user_tenant")
	if !exists {
		return nil
	}
	return userTenant.(*entity.UserTenant)
}

// GetRole 获取当前角色
func GetRole(c *gin.Context) string {
	role, exists := c.Get("role")
	if !exists {
		return ""
	}
	return role.(string)
}

// GetEmail 获取当前用户邮箱
func GetEmail(c *gin.Context) string {
	email, exists := c.Get("email")
	if !exists {
		return ""
	}
	return email.(string)
}

// GetDeviceID 获取设备ID
func GetDeviceID(c *gin.Context) string {
	deviceID, exists := c.Get("device_id")
	if !exists {
		return ""
	}
	return deviceID.(string)
}

// GetTokenID 获取TokenID
func GetTokenID(c *gin.Context) uuid.UUID {
	tokenID, exists := c.Get("token_id")
	if !exists {
		return uuid.Nil
	}
	return tokenID.(uuid.UUID)
}

// IsDeviceMismatch 检查设备是否不匹配
func IsDeviceMismatch(c *gin.Context) bool {
	mismatch, exists := c.Get("device_mismatch")
	if !exists {
		return false
	}
	return mismatch.(bool)
}