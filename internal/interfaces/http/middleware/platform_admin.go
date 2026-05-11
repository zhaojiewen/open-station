package middleware

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// PlatformAdminMiddleware validates platform admin session
func PlatformAdminMiddleware(platformAuth *auth.PlatformAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": apperrors.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		// Extract admin ID from token (for simplicity, we use admin ID as token)
		// In production, this should be a proper session token validation
		token := strings.TrimPrefix(authHeader, "Bearer ")
		adminID, err := uuid.Parse(token)
		if err != nil {
			// Try to get admin ID from X-Platform-Admin-ID header (for development)
			adminIDStr := c.GetHeader("X-Platform-Admin-ID")
			if adminIDStr == "" {
				c.JSON(401, gin.H{"error": apperrors.ErrUnauthorized.Error()})
				c.Abort()
				return
			}
			adminID, err = uuid.Parse(adminIDStr)
			if err != nil {
				c.JSON(401, gin.H{"error": apperrors.ErrUnauthorized.Error()})
				c.Abort()
				return
			}
		}

		// Validate session
		admin, err := platformAuth.ValidateSession(c.Request.Context(), adminID)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Set admin info in context
		c.Set("platform_admin_id", admin.ID)
		c.Set("platform_admin", admin)

		c.Next()
	}
}

// SuperAdminMiddleware ensures the admin is a super admin
func SuperAdminMiddleware(platformAuth *auth.PlatformAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := GetPlatformAdminID(c)
		if adminID == uuid.Nil {
			c.JSON(401, gin.H{"error": apperrors.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		isSuper, err := platformAuth.IsSuperAdmin(c.Request.Context(), adminID)
		if err != nil || !isSuper {
			c.JSON(403, gin.H{"error": apperrors.ErrPlatformPermissionDenied.Error()})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PlatformPermissionMiddleware checks if admin has specific permission
func PlatformPermissionMiddleware(platformAuth *auth.PlatformAuthService, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := GetPlatformAdminID(c)
		if adminID == uuid.Nil {
			c.JSON(401, gin.H{"error": apperrors.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		// Super admin has all permissions
		isSuper, err := platformAuth.IsSuperAdmin(c.Request.Context(), adminID)
		if err == nil && isSuper {
			c.Next()
			return
		}

		hasPermission, err := platformAuth.CheckPermission(c.Request.Context(), adminID, permission)
		if err != nil || !hasPermission {
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
	// Super admin has all permissions
	if admin.Role == "super_admin" {
		return true
	}
	// Parse permissions from JSONB string
	var permissions []string
	if admin.Permissions != "" && admin.Permissions != "[]" {
		if err := json.Unmarshal([]byte(admin.Permissions), &permissions); err != nil {
			return false
		}
	}
	// Check if permission is in the admin's permissions list
	for _, p := range permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}