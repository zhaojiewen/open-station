package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	ratelimit "github.com/zhaojiewen/open-station/internal/infrastructure/persistence/redis"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

type RateLimitConfig struct {
	DefaultUserRPS    float64
	DefaultUserBurst  int
	DefaultTenantRPS  float64
	DefaultTenantBurst int
}

func RateLimitMiddleware(service *ratelimit.RateLimitService, cfg *RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKeyObj, exists := c.Get("api_key")
		if !exists {
			c.Next()
			return
		}

		apiKey := apiKeyObj.(*entity.APIKey)
		userObj, _ := c.Get("user")
		tenantObj, _ := c.Get("tenant")

		user := userObj.(*entity.User)
		tenant := tenantObj.(*entity.Tenant)

		userRPS := cfg.DefaultUserRPS
		userBurst := cfg.DefaultUserBurst
		if user.RateLimitRPS != nil {
			userRPS = float64(*user.RateLimitRPS)
		}
		if user.RateLimitBurst != nil {
			userBurst = *user.RateLimitBurst
		}

		if apiKey.RateLimitRPS != nil {
			userRPS = float64(*apiKey.RateLimitRPS)
		}
		if apiKey.RateLimitBurst != nil {
			userBurst = *apiKey.RateLimitBurst
		}

		if err := service.CheckAPIKeyLimit(c.Request.Context(), apiKey.ID.String(), userRPS, userBurst); err != nil {
			c.Header("X-RateLimit-Limit", strconv.Itoa(int(userRPS)))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   apperrors.ErrRateLimitExceeded.Code,
				"message": apperrors.ErrRateLimitExceeded.Message,
			})
			c.Abort()
			return
		}

		tenantRPS := float64(tenant.RateLimitRPS)
		tenantBurst := tenant.RateLimitBurst
		if err := service.CheckTenantLimit(c.Request.Context(), tenant.ID.String(), tenantRPS, tenantBurst); err != nil {
			c.Header("X-RateLimit-Limit", strconv.Itoa(int(tenantRPS)))
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   apperrors.ErrTenantLimitExceeded.Code,
				"message": apperrors.ErrTenantLimitExceeded.Message,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}