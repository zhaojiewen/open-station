package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
)

// APIKeyHandler handles API key management requests
type APIKeyHandler struct {
	authService *auth.AuthService
}

func NewAPIKeyHandler(authService *auth.AuthService) *APIKeyHandler {
	return &APIKeyHandler{
		authService: authService,
	}
}

func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req struct {
		UserID            string   `json:"user_id" binding:"required"`
		Name              string   `json:"name"`
		Permissions       []string `json:"permissions"`
		AllowedModels     []string `json:"allowed_models"`
		AllowedProviders  []string `json:"allowed_providers"`
		ExpiresAt         string   `json:"expires_at"`
		MonthlyTokenLimit int64    `json:"monthly_token_limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
		return
	}

	permissions := req.Permissions
	if len(permissions) == 0 {
		permissions = []string{"chat", "embeddings"}
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	var tokenLimit *int64
	if req.MonthlyTokenLimit > 0 {
		tokenLimit = &req.MonthlyTokenLimit
	}

	apiKey, key, err := h.authService.CreateAPIKey(
		c.Request.Context(),
		userID,
		user.TenantID,
		req.Name,
		permissions,
		req.AllowedModels,
		req.AllowedProviders,
		expiresAt,
		tokenLimit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         apiKey.ID,
		"key":        key,
		"key_prefix": apiKey.KeyPrefix,
		"name":       apiKey.Name,
		"created_at": apiKey.CreatedAt,
		"message":    "API key created successfully. Please save the key as it will not be shown again.",
	})
}

func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	keys, err := h.authService.ListAPIKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range keys {
		keys[i].KeyHash = ""
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	keyIDStr := c.Param("id")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	if err := h.authService.RevokeAPIKey(c.Request.Context(), keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
}

func (h *APIKeyHandler) CreateMyAPIKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	tenantID := c.MustGet("tenant_id").(uuid.UUID)

	var req struct {
		Name              string   `json:"name"`
		Permissions       []string `json:"permissions"`
		AllowedModels     []string `json:"allowed_models"`
		AllowedProviders  []string `json:"allowed_providers"`
		ExpiresAt         string   `json:"expires_at"`
		MonthlyTokenLimit int64    `json:"monthly_token_limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permissions := req.Permissions
	if len(permissions) == 0 {
		permissions = []string{"chat", "embeddings"}
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	var tokenLimit *int64
	if req.MonthlyTokenLimit > 0 {
		tokenLimit = &req.MonthlyTokenLimit
	}

	apiKey, key, err := h.authService.CreateAPIKey(
		c.Request.Context(),
		userID,
		tenantID,
		req.Name,
		permissions,
		req.AllowedModels,
		req.AllowedProviders,
		expiresAt,
		tokenLimit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         apiKey.ID,
		"key":        key,
		"key_prefix": apiKey.KeyPrefix,
		"name":       apiKey.Name,
		"created_at": apiKey.CreatedAt,
		"message":    "API key created successfully. Please save the key as it will not be shown again.",
	})
}

func (h *APIKeyHandler) ListMyAPIKeys(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	keys, err := h.authService.ListAPIKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range keys {
		keys[i].KeyHash = ""
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}