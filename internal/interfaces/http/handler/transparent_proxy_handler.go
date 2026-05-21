package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/pkg/config"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// TransparentProxyHandler proxies requests transparently to upstream providers.
// It reads the model from the request body to determine the provider, rewrites
// the URL and auth headers, and forwards the request/response as-is. Only usage
// data is extracted from responses for billing.
type TransparentProxyHandler struct {
	accountManager  *service.ProviderAccountManager
	billingService  *service.BillingService
	asyncBilling    *service.AsyncBillingQueue
	authService     *auth.AuthService
	providersConfig *config.ProvidersConfig
	httpClient      *http.Client
}

type upstreamConfig struct {
	APIKey  string
	BaseURL string
}

// apiTypeDefaultProvider maps API types to their default upstream provider.
var apiTypeDefaultProvider = map[string]string{
	"gpt":    "openai",
	"claude": "claude",
}

// NewTransparentProxyHandler creates a new TransparentProxyHandler.
func NewTransparentProxyHandler(
	accountManager *service.ProviderAccountManager,
	billingService *service.BillingService,
	asyncBilling *service.AsyncBillingQueue,
	authService *auth.AuthService,
	providersConfig *config.ProvidersConfig,
) *TransparentProxyHandler {
	return &TransparentProxyHandler{
		accountManager:  accountManager,
		billingService:  billingService,
		asyncBilling:    asyncBilling,
		authService:     authService,
		providersConfig: providersConfig,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     120 * time.Second,
			},
		},
	}
}

// HandleProxy is the main handler for transparent proxy requests.
func (h *TransparentProxyHandler) HandleProxy(c *gin.Context) {
	apiType := c.Param("api")
	remainingPath := c.Param("path")

	// 1. Get auth context from middleware
	apiKey := c.MustGet("api_key").(*entity.APIKey)
	tenantID := c.MustGet("tenant_id").(uuid.UUID)
	userID := c.MustGet("user_id").(uuid.UUID)

	// 2. Read request body
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "failed to read request body"}})
		return
	}

	// 3. Extract model from body
	model := extractModelFromBody(rawBody)
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "model is required"}})
		return
	}

	// 4. Determine provider from model name + API type
	defaultProvider, ok := apiTypeDefaultProvider[apiType]
	if !ok {
		defaultProvider = apiType
	}
	provider := ResolveProvider(apiKey.AllowedModels, defaultProvider)

	logger.Info("transparent proxy request",
		zap.String("api_type", apiType),
		zap.String("model", model),
		zap.String("provider", provider),
		zap.String("path", remainingPath),
		zap.String("tenant_id", tenantID.String()),
	)

	// 5. Permission checks
	if !h.authService.CheckProviderAccess(apiKey, provider) {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"type": "permission_error", "message": "provider not allowed"}})
		return
	}
	if !h.authService.CheckModelAccess(apiKey, model) {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"type": "permission_error", "message": "model not allowed"}})
		return
	}

	// 6. Balance check (user-level only, not tenant)
	if h.billingService != nil {
		balance, err := h.billingService.CheckBalance(c.Request.Context(), userID)
		if err == nil && balance.LessThanOrEqual(decimal.Zero) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"type": "permission_error", "message": "Insufficient balance"}})
			return
		}
	}

	// 7. Check dedicated provider settings
	useDedicated := false
	if user, _ := c.Get("user"); user != nil {
		if u, ok := user.(*entity.User); ok && u.UseDedicatedProvider {
			useDedicated = true
		}
	}
	if !useDedicated {
		if tenant, _ := c.Get("tenant"); tenant != nil {
			if t, ok := tenant.(*entity.Tenant); ok && t.UseDedicatedProvider {
				useDedicated = true
			}
		}
	}

	// 8. Get upstream config (dedicated account first if enabled, then public)
	upstream, err := h.getUpstreamConfig(c.Request.Context(), provider, tenantID, userID, useDedicated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": err.Error()}})
		return
	}

	// 8. Build upstream URL (strip API type prefix, use provider base URL)
	upstreamURL := strings.TrimRight(upstream.BaseURL, "/") + remainingPath

	// 9. Inject stream_options for GPT-format streaming requests to get usage data
	bodyToSend := rawBody
	if apiType == "gpt" && isStreamingRequest(rawBody) {
		bodyToSend = injectStreamOptions(rawBody)
	}

	// 10. Build upstream request
	upstreamReq, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, upstreamURL, bytes.NewReader(bodyToSend))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": "failed to build upstream request"}})
		return
	}

	// 11. Copy headers (excluding gateway auth)
	copyProxyHeaders(c.Request.Header, upstreamReq.Header)
	h.setProviderAuth(upstreamReq, provider, upstream.APIKey)

	// 12. Execute request
	start := time.Now()
	resp, err := h.httpClient.Do(upstreamReq)
	if err != nil {
		latency := int(time.Since(start).Milliseconds())
		h.recordBilling(tenantID, userID, apiKey.ID, provider, model, 0, 0, 0, 0, latency, 500)
		logger.Error("upstream request failed", zap.Error(err), zap.String("provider", provider))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": err.Error()}})
		return
	}
	defer resp.Body.Close()

	// 13. Handle non-200 upstream responses
	if resp.StatusCode >= 400 {
		h.handleUpstreamError(c, resp, apiKey, tenantID, userID, provider, model, start)
		return
	}

	// 14. Handle response based on streaming
	if isStreamingRequest(rawBody) {
		h.proxyStream(c, resp, apiKey, tenantID, userID, provider, model, start)
	} else {
		h.proxyNonStream(c, resp, apiKey, tenantID, userID, provider, model, start)
	}
}

// handleUpstreamError handles error responses from upstream providers.
func (h *TransparentProxyHandler) handleUpstreamError(c *gin.Context, resp *http.Response,
	key *entity.APIKey, tenantID, userID uuid.UUID, provider, actualModel string, start time.Time) {

	body, _ := io.ReadAll(resp.Body)
	latency := int(time.Since(start).Milliseconds())
	h.recordBilling(tenantID, userID, key.ID, provider, actualModel, 0, 0, 0, 0, latency, resp.StatusCode)

	// Copy upstream error headers and body
	for k, values := range resp.Header {
		for _, v := range values {
			c.Header(k, v)
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

// getUpstreamConfig resolves the upstream API key and base URL for a provider.
// When useDedicated is true: user dedicated > tenant dedicated > public pool.
// When useDedicated is false: public pool only.
func (h *TransparentProxyHandler) getUpstreamConfig(ctx context.Context, provider string, tenantID, userID uuid.UUID, useDedicated bool) (*upstreamConfig, error) {
	cfg := &upstreamConfig{}

	if h.accountManager != nil {
		var account *entity.ProviderAccount
		var err error

		if useDedicated {
			account, err = h.accountManager.GetActiveAccountWithDedicated(ctx, provider, tenantID, userID)
		} else {
			account, err = h.accountManager.GetActiveAccount(ctx, provider)
		}

		if err == nil && account != nil && account.APIKey != "" {
			cfg.APIKey = account.APIKey
			if account.BaseURL != "" {
				cfg.BaseURL = account.BaseURL
			}
		}
	}

	if cfg.APIKey == "" {
		pc := h.providersConfig.GetProvider(provider)
		if pc != nil && pc.APIKey != "" {
			cfg.APIKey = pc.APIKey
		}
	}
	if cfg.BaseURL == "" {
		pc := h.providersConfig.GetProvider(provider)
		if pc != nil {
			cfg.BaseURL = pc.BaseURL
		}
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("no API key available for provider: %s", provider)
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("no base URL for provider: %s", provider)
	}

	return cfg, nil
}

// setProviderAuth sets the authentication header for the upstream request.
func (h *TransparentProxyHandler) setProviderAuth(req *http.Request, provider string, apiKey string) {
	pc := h.providersConfig.GetProvider(provider)
	headerName := "Authorization"
	headerValue := "Bearer " + apiKey

	if pc != nil && pc.AuthHeaderName != "" {
		headerName = pc.AuthHeaderName
		headerValue = apiKey
	}

	req.Header.Set(headerName, headerValue)
}

// copyProxyHeaders copies headers from the client request to the upstream request,
// skipping gateway-specific auth headers.
func copyProxyHeaders(src http.Header, dst http.Header) {
	skipHeaders := map[string]bool{
		"Authorization": true,
		"X-Api-Key":     true,
	}

	for key, values := range src {
		if skipHeaders[key] {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

// proxyNonStream handles non-streaming proxy responses.
func (h *TransparentProxyHandler) proxyNonStream(c *gin.Context, resp *http.Response,
	key *entity.APIKey, tenantID, userID uuid.UUID, provider, actualModel string, start time.Time) {

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		latency := int(time.Since(start).Milliseconds())
		h.recordBilling(tenantID, userID, key.ID, provider, actualModel, 0, 0, 0, 0, latency, resp.StatusCode)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": "failed to read upstream response"}})
		return
	}

	latency := int(time.Since(start).Milliseconds())
	promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens := extractUsageFromBody(body)

	h.finalizeBilling(c, key, tenantID, userID, provider, actualModel,
		promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens, latency, resp.StatusCode)

	for k, values := range resp.Header {
		for _, v := range values {
			c.Header(k, v)
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

// proxyStream handles streaming proxy responses.
func (h *TransparentProxyHandler) proxyStream(c *gin.Context, resp *http.Response,
	key *entity.APIKey, tenantID, userID uuid.UUID, provider, actualModel string, start time.Time) {

	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(resp.StatusCode)

	var promptTokens, completionTokens int64
	var cacheReadTokens, cacheCreationTokens int64

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		c.Writer.WriteString(line + "\n")

		if line != "" && line != "data: [DONE]" && line != "[DONE]" {
			pt, ct, crt, cct := extractUsageFromLine(line)
			if pt > 0 {
				promptTokens = pt
			}
			if ct > 0 {
				completionTokens = ct
			}
			if crt > 0 {
				cacheReadTokens = crt
			}
			if cct > 0 {
				cacheCreationTokens = cct
			}
		}

		c.Writer.Flush()
	}

	latency := int(time.Since(start).Milliseconds())
	h.finalizeBilling(c, key, tenantID, userID, provider, actualModel,
		promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens, latency, resp.StatusCode)
}

// finalizeBilling records usage and updates API key stats, accounting for cache hit pricing.
// Both stream and non-stream paths use this to ensure consistent cache-aware billing.
func (h *TransparentProxyHandler) finalizeBilling(c *gin.Context,
	key *entity.APIKey, tenantID, userID uuid.UUID, provider, actualModel string,
	promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens int64,
	latency, statusCode int) {

	h.recordBilling(tenantID, userID, key.ID, provider, actualModel,
		promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens,
		latency, statusCode)

	h.authService.UpdateAPIKeyLastUsed(c.Request.Context(), key.ID)

	// Calculate equivalent tokens accounting for cache hit pricing differences.
	// Cache reads cost less than uncached prompts, so equivalent tokens reflects
	// the actual cost in terms of uncached prompt tokens.
	if promptTokens > 0 || completionTokens > 0 {
		equivTokens := promptTokens + completionTokens
		if h.billingService != nil {
			equivTokens = h.billingService.CalculateEquivalentTokens(c.Request.Context(), provider, actualModel,
				promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens)
			// Update per-provider usage tracking for differentiated statistics
			cost, _ := h.billingService.CalculateCost(c.Request.Context(), provider, actualModel,
				promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens)
			h.authService.UpdateAPIKeyProviderUsage(c.Request.Context(), key.ID, provider, equivTokens, cost)
		}
		h.authService.UpdateAPIKeyTokenUsage(c.Request.Context(), key.ID, equivTokens)
	}
}

// recordBilling records usage via async billing or sync fallback.
func (h *TransparentProxyHandler) recordBilling(tenantID, userID, apiKeyID uuid.UUID,
	provider, model string,
	promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens int64,
	latency, statusCode int) {

	requestID := uuid.New().String()
	if h.asyncBilling != nil {
		h.asyncBilling.QueueBillingAsync(
			tenantID, userID, apiKeyID,
			requestID, provider, model,
			promptTokens, completionTokens,
			cacheReadTokens, cacheCreationTokens,
			latency, statusCode,
		)
	} else if h.billingService != nil {
		h.billingService.RecordUsage(context.Background(),
			tenantID, userID, apiKeyID,
			requestID, provider, model,
			promptTokens, completionTokens,
			cacheReadTokens, cacheCreationTokens,
			latency, statusCode,
		)
	}
}

// extractModelFromBody extracts the model name from a JSON request body.
func extractModelFromBody(body []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Model
}

// isStreamingRequest checks if the request body contains "stream": true.
func isStreamingRequest(body []byte) bool {
	var req struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	return req.Stream
}

// injectStreamOptions adds stream_options.include_usage to GPT-format requests
// so the upstream returns usage data in streaming mode.
func injectStreamOptions(body []byte) []byte {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return body
	}

	if so, ok := req["stream_options"]; ok {
		if soMap, ok := so.(map[string]interface{}); ok {
			soMap["include_usage"] = true
			req["stream_options"] = soMap
		}
	} else {
		req["stream_options"] = map[string]interface{}{
			"include_usage": true,
		}
	}

	modified, err := json.Marshal(req)
	if err != nil {
		return body
	}
	return modified
}

// extractUsageFromBody parses usage info from a non-streaming response body.
func extractUsageFromBody(body []byte) (promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens int64) {
	var resp struct {
		Usage struct {
			PromptTokens             int `json:"prompt_tokens"`
			CompletionTokens         int `json:"completion_tokens"`
			TotalTokens              int `json:"total_tokens"`
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, 0, 0
	}

	if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
		return int64(resp.Usage.InputTokens), int64(resp.Usage.OutputTokens),
			int64(resp.Usage.CacheReadInputTokens), int64(resp.Usage.CacheCreationInputTokens)
	}

	return int64(resp.Usage.PromptTokens), int64(resp.Usage.CompletionTokens),
		int64(resp.Usage.CacheReadInputTokens), int64(resp.Usage.CacheCreationInputTokens)
}

// extractUsageFromLine tries to extract usage from an SSE line.
func extractUsageFromLine(line string) (promptTokens, completionTokens, cacheReadTokens, cacheCreationTokens int64) {
	data := line
	if strings.HasPrefix(line, "data: ") {
		data = strings.TrimPrefix(line, "data: ")
	}

	var chunk struct {
		Usage *struct {
			PromptTokens             int `json:"prompt_tokens"`
			CompletionTokens         int `json:"completion_tokens"`
			TotalTokens              int `json:"total_tokens"`
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage,omitempty"`
		Message *struct {
			Usage *struct {
				InputTokens              int `json:"input_tokens"`
				CacheReadInputTokens     int `json:"cache_read_input_tokens"`
				CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			} `json:"usage,omitempty"`
		} `json:"message,omitempty"`
		Delta *struct {
			Usage *struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage,omitempty"`
		} `json:"delta,omitempty"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return 0, 0, 0, 0
	}

	// OpenAI format: usage in top-level "usage" field (includes cache_* tokens)
	if chunk.Usage != nil {
		return int64(chunk.Usage.PromptTokens), int64(chunk.Usage.CompletionTokens),
			int64(chunk.Usage.CacheReadInputTokens), int64(chunk.Usage.CacheCreationInputTokens)
	}

	// Anthropic format: message_start has input_tokens + cache tokens
	if chunk.Message != nil && chunk.Message.Usage != nil && chunk.Message.Usage.InputTokens > 0 {
		return int64(chunk.Message.Usage.InputTokens), 0,
			int64(chunk.Message.Usage.CacheReadInputTokens), int64(chunk.Message.Usage.CacheCreationInputTokens)
	}

	// Anthropic format: message_delta has output_tokens
	if chunk.Delta != nil && chunk.Delta.Usage != nil && chunk.Delta.Usage.OutputTokens > 0 {
		return 0, int64(chunk.Delta.Usage.OutputTokens), 0, 0
	}

	return 0, 0, 0, 0
}
