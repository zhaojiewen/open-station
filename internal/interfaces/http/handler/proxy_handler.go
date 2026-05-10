package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/internal/infrastructure/proxy"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// ProxyHandler handles proxy requests to AI providers
type ProxyHandler struct {
	proxyService   *proxy.ProxyService
	billingService *service.BillingService
	asyncBilling   *service.AsyncBillingQueue
	authService    *auth.AuthService
}

func NewProxyHandler(proxyService *proxy.ProxyService, billingService *service.BillingService, asyncBilling *service.AsyncBillingQueue, authService *auth.AuthService) *ProxyHandler {
	return &ProxyHandler{
		proxyService:   proxyService,
		billingService: billingService,
		asyncBilling:   asyncBilling,
		authService:    authService,
	}
}

func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
	apiKey := c.MustGet("api_key").(*entity.APIKey)

	var req proxy.ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   apperrors.ErrInvalidRequest.Code,
			"message": "invalid request body: " + err.Error(),
		})
		return
	}

	if !h.authService.CheckModelAccess(apiKey, req.Model) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   apperrors.ErrModelNotSupported.Code,
			"message": "model not allowed for this API key",
		})
		return
	}

	if !h.authService.CheckProviderAccess(apiKey, req.Provider) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   apperrors.ErrProviderNotEnabled.Code,
			"message": "provider not allowed for this API key",
		})
		return
	}

	if !h.authService.CheckTokenLimit(apiKey) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "LIMIT_001",
			"message": "monthly token limit exceeded",
		})
		return
	}

	start := time.Now()
	requestID := h.proxyService.GenerateRequestID()

	resp, err := h.proxyService.ChatCompletion(c.Request.Context(), &req)
	if err != nil {
		latency := int(time.Since(start).Milliseconds())
		if h.asyncBilling != nil {
			h.asyncBilling.QueueBillingAsync(
				c.MustGet("tenant_id").(uuid.UUID),
				c.MustGet("user_id").(uuid.UUID),
				apiKey.ID,
				requestID,
				req.Provider,
				req.Model,
				0, 0,
				latency,
				500,
			)
		} else {
			h.billingService.RecordUsage(c.Request.Context(),
				c.MustGet("tenant_id").(uuid.UUID),
				c.MustGet("user_id").(uuid.UUID),
				apiKey.ID,
				requestID,
				req.Provider,
				req.Model,
				0, 0,
				latency,
				500,
			)
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   apperrors.ErrProviderError.Code,
			"message": err.Error(),
		})
		return
	}

	latency := int(time.Since(start).Milliseconds())

	if h.asyncBilling != nil {
		h.asyncBilling.QueueBillingAsync(
			c.MustGet("tenant_id").(uuid.UUID),
			c.MustGet("user_id").(uuid.UUID),
			apiKey.ID,
			requestID,
			req.Provider,
			req.Model,
			int64(resp.Usage.PromptTokens),
			int64(resp.Usage.CompletionTokens),
			latency,
			200,
		)
		h.asyncBilling.QueueTokenUpdateAsync(apiKey.ID, int64(resp.Usage.TotalTokens))
	} else {
		h.billingService.RecordUsage(c.Request.Context(),
			c.MustGet("tenant_id").(uuid.UUID),
			c.MustGet("user_id").(uuid.UUID),
			apiKey.ID,
			requestID,
			req.Provider,
			req.Model,
			int64(resp.Usage.PromptTokens),
			int64(resp.Usage.CompletionTokens),
			latency,
			200,
		)
		h.authService.UpdateAPIKeyTokenUsage(c.Request.Context(), apiKey.ID, int64(resp.Usage.TotalTokens))
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ProxyHandler) Embeddings(c *gin.Context) {
	apiKey := c.MustGet("api_key").(*entity.APIKey)

	var req proxy.ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   apperrors.ErrInvalidRequest.Code,
			"message": "invalid request body: " + err.Error(),
		})
		return
	}

	if !h.authService.CheckModelAccess(apiKey, req.Model) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   apperrors.ErrModelNotSupported.Code,
			"message": "model not allowed for this API key",
		})
		return
	}

	embedding, err := h.proxyService.Embedding(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   apperrors.ErrProviderError.Code,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data": []gin.H{
			{
				"object":    "embedding",
				"embedding": embedding,
				"index":     0,
			},
		},
		"model": req.Model,
	})
}