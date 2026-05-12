package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/role"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	ratelimit "github.com/zhaojiewen/open-station/internal/infrastructure/persistence/redis"
	"github.com/zhaojiewen/open-station/pkg/config"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

func AuthMiddleware(authService *auth.AuthService, safeService *ratelimit.SafeService, failedAuthCfg *config.FailedAuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   apperrors.ErrUnauthorized.Code,
				"message": apperrors.ErrUnauthorized.Message,
			})
			c.Abort()
			return
		}

		var apiKey string
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			apiKey = authHeader
		}

		key, user, tenant, err := authService.ValidateAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			if safeService != nil && failedAuthCfg != nil {
				clientIP := c.ClientIP()
				wasBlocked, recErr := safeService.RecordAuthFailure(
					c.Request.Context(), clientIP,
					failedAuthCfg.WindowS,
					failedAuthCfg.MaxAttempts,
					failedAuthCfg.BlockDurationS,
				)
				if recErr != nil {
					logger.Warn("safe: failed to record auth failure",
						zap.String("ip", clientIP), zap.Error(recErr))
				} else if wasBlocked {
					logger.Warn("safe: IP auto-blocked due to repeated auth failures",
						zap.String("ip", clientIP))
				}
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   apperrors.ErrInvalidAPIKey.Code,
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("api_key_id", key.ID)
		c.Set("api_key", key)
		c.Set("user_id", user.ID)
		c.Set("user", user)
		c.Set("tenant_id", tenant.ID)
		c.Set("tenant", tenant)

		if err := authService.UpdateAPIKeyLastUsed(c.Request.Context(), key.ID); err != nil {
		}

		if safeService != nil {
			if resetErr := safeService.ResetAuthFailures(c.Request.Context(), c.ClientIP()); resetErr != nil {
				logger.Warn("safe: failed to reset auth failures",
					zap.String("ip", c.ClientIP()), zap.Error(resetErr))
			}
		}

		c.Next()
	}
}

func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   apperrors.ErrUnauthorized.Code,
				"message": apperrors.ErrUnauthorized.Message,
			})
			c.Abort()
			return
		}

		// Check UserTenant.Role first (multi-tenant JWT auth path),
		// then fall back to user.Role (API key auth path).
		effectiveRole := user.Role
		if ut := GetUserTenant(c); ut != nil {
			effectiveRole = ut.Role
		}

		if !role.IsTenantAdmin(effectiveRole) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   apperrors.ErrForbidden.Code,
				"message": apperrors.ErrForbidden.Message,
			})
			c.Abort()
			return
		}

		tenantID := GetTenantID(c)
		if user.TenantID != tenantID {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   apperrors.ErrForbidden.Code,
				"message": apperrors.ErrForbidden.Message,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoginRateLimitMiddleware applies IP-based rate limiting to login/register endpoints.
func LoginRateLimitMiddleware(safeService *ratelimit.SafeService, rps, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if safeService == nil {
			c.Next()
			return
		}
		if err := safeService.CheckIPRateLimit(c.Request.Context(), c.ClientIP(), rps, burst); err != nil {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "too many requests",
				"message": "please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetAPIKeyID(c *gin.Context) uuid.UUID {
	id, _ := c.Get("api_key_id")
	return id.(uuid.UUID)
}

func GetUserID(c *gin.Context) uuid.UUID {
	id, _ := c.Get("user_id")
	return id.(uuid.UUID)
}

func GetTenantID(c *gin.Context) uuid.UUID {
	id, _ := c.Get("tenant_id")
	return id.(uuid.UUID)
}

func GetAPIKey(c *gin.Context) *entity.APIKey {
	key, _ := c.Get("api_key")
	return key.(*entity.APIKey)
}

func GetUser(c *gin.Context) *entity.User {
	user, exists := c.Get("user")
	if !exists || user == nil {
		return nil
	}
	return user.(*entity.User)
}

func GetTenant(c *gin.Context) *entity.Tenant {
	tenant, _ := c.Get("tenant")
	return tenant.(*entity.Tenant)
}