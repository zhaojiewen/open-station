package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/role"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// PlatformAdminMiddleware validates platform admin session token.
func PlatformAdminMiddleware(platformAuth *auth.PlatformAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": apperrors.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		admin, err := platformAuth.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("platform_admin_id", admin.ID)
		c.Set("platform_admin", admin)

		c.Next()
	}
}

// SuperAdminMiddleware ensures the admin is a super admin
func SuperAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := GetPlatformAdmin(c)
		if admin == nil {
			c.JSON(401, gin.H{"error": apperrors.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		if !role.IsSuperAdmin(admin.Role) {
			c.JSON(403, gin.H{"error": apperrors.ErrPlatformPermissionDenied.Error()})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PlatformPermissionMiddleware checks if admin has specific permission
func PlatformPermissionMiddleware(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := GetPlatformAdmin(c)
		if admin == nil {
			c.JSON(401, gin.H{"error": apperrors.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		if !role.HasPlatformPermission(admin.Role, admin.Permissions, permission) {
			c.JSON(403, gin.H{"error": apperrors.ErrPlatformPermissionDenied.Error()})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper functions

func GetPlatformAdminID(c *gin.Context) uuid.UUID {
	id, exists := c.Get("platform_admin_id")
	if !exists {
		return uuid.Nil
	}
	return id.(uuid.UUID)
}

func GetPlatformAdmin(c *gin.Context) *entity.PlatformAdmin {
	admin, exists := c.Get("platform_admin")
	if !exists {
		return nil
	}
	return admin.(*entity.PlatformAdmin)
}

// HasPlatformPermission checks if the platform admin in context has a specific permission
// This is a helper function for use inside handlers (not a middleware)
func HasPlatformPermission(c *gin.Context, permission string) bool {
	admin := GetPlatformAdmin(c)
	if admin == nil {
		return false
	}
	return role.HasPlatformPermission(admin.Role, admin.Permissions, permission)
}