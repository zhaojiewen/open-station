package handler

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/role"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/internal/infrastructure/proxy"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// Anthropic Messages API 兼容处理器

// AnthropicMessagesRequest - Anthropic Messages API请求格式
type AnthropicMessagesRequest struct {
	Model       string                 `json:"model"`
	MaxTokens   int                    `json:"max_tokens"`
	Messages    []AnthropicMessage     `json:"messages"`
	System      string                 `json:"system,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Tools       []AnthropicTool        `json:"tools,omitempty"`
	ToolChoice  interface{}            `json:"tool_choice,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	// For tool use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`

	// For tool result
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`

	// For images
	Source *AnthropicImageSource `json:"source,omitempty"`
}

type AnthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// AnthropicMessagesResponse - Anthropic Messages API响应格式
type AnthropicMessagesResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Model        string                  `json:"model"`
	Content      []AnthropicContentBlock `json:"content"`
	StopReason   string                  `json:"stop_reason,omitempty"`
	StopSequence string                  `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage          `json:"usage"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamEvent - 流式响应事件
type AnthropicStreamEvent struct {
	Type         string                   `json:"type"`
	Index        int                      `json:"index,omitempty"`
	Message      *AnthropicMessagesResponse `json:"message,omitempty"`
	ContentBlock *AnthropicContentBlock   `json:"content_block,omitempty"`
	Delta        *AnthropicDelta          `json:"delta,omitempty"`
	Usage        *AnthropicUsage          `json:"usage,omitempty"`
}

type AnthropicDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// AnthropicHandler - Anthropic兼容处理器
type AnthropicHandler struct {
	proxyService   *proxy.ProxyService
	authService    *auth.AuthService
	billingService *service.BillingService
	modelMapping   map[string]string // Claude model ID -> Provider mapping
}

func NewAnthropicHandler(proxyService *proxy.ProxyService, authService *auth.AuthService, billingService *service.BillingService) *AnthropicHandler {
	// Claude模型到Provider的映射
	modelMapping := map[string]string{
		// Claude 4系列 -> 使用claude provider
		"claude-opus-4-7":    "claude",
		"claude-opus-4-6":    "claude",
		"claude-opus-4-5":    "claude",
		"claude-opus-4-1":    "claude",
		"claude-sonnet-4-6":  "claude",
		"claude-sonnet-4-5":  "claude",
		"claude-haiku-4-5":   "claude",
		// Claude 3系列
		"claude-3-5-sonnet": "claude",
		"claude-3-5-haiku":  "claude",
		"claude-3-opus":     "claude",
		"claude-3-sonnet":   "claude",
		"claude-3-haiku":    "claude",
		// 别名映射
		"claude-opus-4-7-20250514":   "claude",
		"claude-sonnet-4-6-20250514": "claude",
		"claude-haiku-4-5-20251001":  "claude",
	}

	return &AnthropicHandler{
		proxyService:   proxyService,
		authService:    authService,
		billingService: billingService,
		modelMapping:   modelMapping,
	}
}

// Messages - 处理 Anthropic Messages API 请求
func (h *AnthropicHandler) Messages(c *gin.Context) {
	// 1. 认证
	authHeader := c.GetHeader("Authorization")
	apiKeyHeader := c.GetHeader("X-Api-Key")

	var apiKey string
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		apiKey = strings.TrimPrefix(authHeader, "Bearer ")
	} else if apiKeyHeader != "" {
		apiKey = apiKeyHeader
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "authentication_error",
				"message": "Invalid API key",
			},
		})
		return
	}

	key, user, tenant, err := h.authService.ValidateAPIKey(c.Request.Context(), apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "authentication_error",
				"message": err.Error(),
			},
		})
		return
	}

	// 2. 解析请求
	var anthropicReq AnthropicMessagesRequest
	if err := c.ShouldBindJSON(&anthropicReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": err.Error(),
			},
		})
		return
	}

	// 验证必填字段
	if anthropicReq.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "model is required",
			},
		})
		return
	}

	if anthropicReq.MaxTokens <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "max_tokens must be positive",
			},
		})
		return
	}

	if len(anthropicReq.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "messages is required and must be non-empty",
			},
		})
		return
	}

	logger.Info("anthropic messages request",
		zap.String("model", anthropicReq.Model),
		zap.Int("max_tokens", anthropicReq.MaxTokens),
		zap.Int("messages", len(anthropicReq.Messages)),
		zap.Bool("stream", anthropicReq.Stream),
		zap.String("tenant_id", tenant.ID.String()),
	)

	// 3. 权限检查
	if !h.authService.CheckPermission(key, role.PermChat) {
		c.JSON(http.StatusForbidden, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "permission_error",
				"message": "API key does not have chat permission",
			},
		})
		return
	}

	// 4. 确定Provider和实际模型
	provider := h.getProviderForModel(anthropicReq.Model)
	actualModel := anthropicReq.Model

	// 支持通过前缀访问其他Provider: openai-gpt-4o, deepseek-v4-flash等
	if strings.Contains(anthropicReq.Model, "-") {
		parts := strings.SplitN(anthropicReq.Model, "-", 2)
		if len(parts) == 2 {
			possibleProvider := parts[0]
			validProviders := []string{"openai", "claude", "deepseek", "glm", "gemini"}
			for _, vp := range validProviders {
				if possibleProvider == vp {
					provider = possibleProvider
					actualModel = parts[1]
					break
				}
			}
		}
	}

	// 5. Provider和模型访问权限检查
	if !h.authService.CheckProviderAccess(key, provider) {
		c.JSON(http.StatusForbidden, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "permission_error",
				"message": fmt.Sprintf("API key does not have access to provider: %s", provider),
			},
		})
		return
	}

	if !h.authService.CheckModelAccess(key, actualModel) {
		c.JSON(http.StatusForbidden, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "permission_error",
				"message": fmt.Sprintf("API key does not have access to model: %s", actualModel),
			},
		})
		return
	}

	// 6. 余额检查（如果有billing服务）
	if h.billingService != nil {
		balance, err := h.billingService.CheckBalance(c.Request.Context(), tenant.ID)
		if err != nil {
			logger.Warn("failed to check balance", zap.Error(err))
		} else if balance.LessThanOrEqual(decimal.Zero) {
			c.JSON(http.StatusForbidden, gin.H{
				"type":  "error",
				"error": gin.H{
					"type":    "permission_error",
					"message": "Insufficient balance",
				},
			})
			return
		}
	}

	// 7. 转换为内部统一格式
	proxyReq := h.convertToProxyRequest(provider, actualModel, &anthropicReq)

	// 8. 调用代理服务
	start := time.Now()
	requestID := uuid.New().String()

	// 9. 根据是否流式请求分别处理
	if anthropicReq.Stream {
		h.handleStreamRequest(c, proxyReq, key, user, tenant, requestID, provider, actualModel, start, anthropicReq.Model)
		return
	}

	// 非流式请求处理
	resp, err := h.proxyService.ChatCompletion(c.Request.Context(), proxyReq)
	if err != nil {
		latency := int(time.Since(start).Milliseconds())
		logger.Error("proxy error", zap.Error(err), zap.String("request_id", requestID))

		// 记录失败请求
		h.recordUsage(c, tenant.ID, user.ID, key.ID, requestID, provider, actualModel, 0, 0, latency, 500, err.Error())

		// 根据错误类型返回不同的响应
		errMsg := err.Error()
		errType := "api_error"
		statusCode := http.StatusInternalServerError

		if strings.Contains(errMsg, "timeout") {
			errType = "timeout_error"
			statusCode = http.StatusGatewayTimeout
		} else if strings.Contains(errMsg, "rate limit") {
			errType = "rate_limit_error"
			statusCode = http.StatusTooManyRequests
		} else if strings.Contains(errMsg, "unsupported provider") {
			errType = "invalid_request_error"
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    errType,
				"message": errMsg,
			},
		})
		return
	}

	// 10. 记录使用量
	latency := int(time.Since(start).Milliseconds())
	h.recordUsage(c, tenant.ID, user.ID, key.ID, requestID, provider, actualModel,
		int64(resp.Usage.PromptTokens), int64(resp.Usage.CompletionTokens), latency, 200, "")

	// 11. 转换为Anthropic格式响应
	anthropicResp := h.convertToAnthropicResponse(resp, anthropicReq.Model)

	c.JSON(http.StatusOK, anthropicResp)
}

// handleStreamRequest - 处理流式请求
func (h *AnthropicHandler) handleStreamRequest(c *gin.Context, proxyReq *proxy.ProxyRequest,
	key *entity.APIKey, user *entity.User, tenant *entity.Tenant, requestID, provider, actualModel string,
	start time.Time, originalModel string) {

	// 获取流式响应
	streamReader, err := h.proxyService.StreamChatCompletion(c.Request.Context(), proxyReq)
	if err != nil {
		latency := int(time.Since(start).Milliseconds())
		logger.Error("stream proxy error", zap.Error(err), zap.String("request_id", requestID))
		h.recordUsage(c, tenant.ID, user.ID, key.ID, requestID, provider, actualModel, 0, 0, latency, 500, err.Error())

		c.JSON(http.StatusInternalServerError, gin.H{
			"type":  "error",
			"error": gin.H{
				"type":    "api_error",
				"message": err.Error(),
			},
		})
		return
	}
	defer streamReader.Close()

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 发送message_start事件
	messageID := "msg_" + uuid.New().String()
	h.sendSSEEvent(c, AnthropicStreamEvent{
		Type: "message_start",
		Message: &AnthropicMessagesResponse{
			ID:      messageID,
			Type:    "message",
			Role:    "assistant",
			Model:   originalModel,
			Content: []AnthropicContentBlock{},
			Usage:   AnthropicUsage{InputTokens: 0, OutputTokens: 0},
		},
	})

	// 发送content_block_start事件
	h.sendSSEEvent(c, AnthropicStreamEvent{
		Type:  "content_block_start",
		Index: 0,
		ContentBlock: &AnthropicContentBlock{
			Type: "text",
			Text: "",
		},
	})

	// 读取流式数据
	scanner := bufio.NewScanner(streamReader)
	totalOutputTokens := 0

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// 解析OpenAI流式响应
		var chunk proxy.StreamChunk
		if err := proxy.ParseStreamChunk(data, &chunk); err != nil {
			logger.Warn("failed to parse stream chunk", zap.Error(err))
			continue
		}

		// 转换并发送Anthropic格式事件
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				h.sendSSEEvent(c, AnthropicStreamEvent{
					Type:  "content_block_delta",
					Index: 0,
					Delta: &AnthropicDelta{
						Type: "text_delta",
						Text: delta.Content,
					},
				})
				totalOutputTokens++
			}

			// 处理结束
			if chunk.Choices[0].FinishReason != "" {
				stopReason := "end_turn"
				switch chunk.Choices[0].FinishReason {
				case "stop":
					stopReason = "end_turn"
				case "length":
					stopReason = "max_tokens"
				}

				h.sendSSEEvent(c, AnthropicStreamEvent{
					Type:  "content_block_delta",
					Index: 0,
					Delta: &AnthropicDelta{
						Type:       "input_json_delta",
						StopReason: stopReason,
					},
				})
			}
		}

		// 处理usage
		if chunk.Usage != nil {
			h.sendSSEEvent(c, AnthropicStreamEvent{
				Type: "message_delta",
				Usage: &AnthropicUsage{
					OutputTokens: chunk.Usage.CompletionTokens,
				},
			})
		}
	}

	// 发送content_block_stop事件
	h.sendSSEEvent(c, AnthropicStreamEvent{
		Type:  "content_block_stop",
		Index: 0,
	})

	// 发送message_stop事件
	h.sendSSEEvent(c, AnthropicStreamEvent{
		Type: "message_stop",
	})

	// 记录使用量
	latency := int(time.Since(start).Milliseconds())
	// 流式请求的token估算（简化处理）
	h.recordUsage(c, tenant.ID, user.ID, key.ID, requestID, provider, actualModel, 0, int64(totalOutputTokens), latency, 200, "")

	c.SSEvent("message_stop", "")
}

// sendSSEEvent - 发送SSE事件
func (h *AnthropicHandler) sendSSEEvent(c *gin.Context, event AnthropicStreamEvent) {
	c.SSEvent(event.Type, event)
}

// convertToProxyRequest - Anthropic格式转内部统一格式
func (h *AnthropicHandler) convertToProxyRequest(provider, model string, anthropicReq *AnthropicMessagesRequest) *proxy.ProxyRequest {
	messages := make([]proxy.Message, 0)

	// 处理system prompt
	if anthropicReq.System != "" {
		messages = append(messages, proxy.Message{
			Role:    "system",
			Content: anthropicReq.System,
		})
	}

	// 转换messages
	for _, msg := range anthropicReq.Messages {
		content := h.convertContent(msg.Content)
		messages = append(messages, proxy.Message{
			Role:    msg.Role,
			Content: content,
		})
	}

	// 构建参数
	parameters := map[string]interface{}{
		"max_tokens": anthropicReq.MaxTokens,
	}

	return &proxy.ProxyRequest{
		Provider:   provider,
		Model:      model,
		Messages:   messages,
		Parameters: parameters,
		Stream:     anthropicReq.Stream,
	}
}

// convertContent - 转换Anthropic content格式
func (h *AnthropicHandler) convertContent(content interface{}) interface{} {
	// 如果是字符串，直接返回
	if str, ok := content.(string); ok {
		return str
	}

	// 如果是数组（多模态内容）
	if blocks, ok := content.([]interface{}); ok {
		convertedBlocks := make([]interface{}, 0)
		for _, block := range blocks {
			if b, ok := block.(map[string]interface{}); ok {
				blockType, _ := b["type"].(string)
				switch blockType {
				case "text":
					text, _ := b["text"].(string)
					convertedBlocks = append(convertedBlocks, map[string]interface{}{
						"type": "text",
						"text": text,
					})
				case "image":
					// 转换图片格式
					source, _ := b["source"].(map[string]interface{})
					if source != nil {
						mediaType, _ := source["media_type"].(string)
						data, _ := source["data"].(string)
						convertedBlocks = append(convertedBlocks, map[string]interface{}{
							"type": "image_url",
							"image_url": map[string]interface{}{
								"url": fmt.Sprintf("data:%s;base64,%s", mediaType, data),
							},
						})
					}
				default:
					convertedBlocks = append(convertedBlocks, block)
				}
			}
		}
		return convertedBlocks
	}

	return content
}

// convertToAnthropicResponse - 内部格式转Anthropic响应格式
func (h *AnthropicHandler) convertToAnthropicResponse(proxyResp *proxy.ProxyResponse, originalModel string) *AnthropicMessagesResponse {
	// 构建content blocks
	contentBlocks := make([]AnthropicContentBlock, 0)

	// 从choices中提取内容
	for _, choice := range proxyResp.Choices {
		// 转换content
		if str, ok := choice.Message.Content.(string); ok {
			contentBlocks = append(contentBlocks, AnthropicContentBlock{
				Type: "text",
				Text: str,
			})
		} else if blocks, ok := choice.Message.Content.([]interface{}); ok {
			for _, block := range blocks {
				if b, ok := block.(map[string]interface{}); ok {
					blockType, _ := b["type"].(string)
					switch blockType {
					case "text":
						text, _ := b["text"].(string)
						contentBlocks = append(contentBlocks, AnthropicContentBlock{
							Type: "text",
							Text: text,
						})
					}
				}
			}
		}
	}

	// 转换stop_reason
	stopReason := "end_turn"
	if len(proxyResp.Choices) > 0 && proxyResp.Choices[0].FinishReason != "" {
		switch proxyResp.Choices[0].FinishReason {
		case "stop":
			stopReason = "end_turn"
		case "length":
			stopReason = "max_tokens"
		case "tool_calls":
			stopReason = "tool_use"
		default:
			stopReason = proxyResp.Choices[0].FinishReason
		}
	}

	return &AnthropicMessagesResponse{
		ID:         proxyResp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      originalModel,
		Content:    contentBlocks,
		StopReason: stopReason,
		Usage: AnthropicUsage{
			InputTokens:  proxyResp.Usage.PromptTokens,
			OutputTokens: proxyResp.Usage.CompletionTokens,
		},
	}
}

// getProviderForModel - 根据模型ID确定Provider
func (h *AnthropicHandler) getProviderForModel(model string) string {
	// 检查映射表
	for modelPrefix, provider := range h.modelMapping {
		if strings.HasPrefix(model, modelPrefix) || model == modelPrefix {
			return provider
		}
	}

	// 默认返回claude（因为是Anthropic格式请求）
	return "claude"
}

// recordUsage - 记录使用量（完整版，支持计费）
func (h *AnthropicHandler) recordUsage(c *gin.Context, tenantID, userID, apiKeyID uuid.UUID,
	requestID, provider, model string, promptTokens, completionTokens int64,
	latency int, statusCode int, errorMsg string) {

	// 更新API Key最后使用时间和token用量
	h.authService.UpdateAPIKeyLastUsed(c.Request.Context(), apiKeyID)
	h.authService.UpdateAPIKeyTokenUsage(c.Request.Context(), apiKeyID, promptTokens+completionTokens)

	// 如果有billing服务，记录完整的使用量和计费
	if h.billingService != nil && statusCode == 200 {
		_, err := h.billingService.RecordUsage(
			c.Request.Context(),
			tenantID,
			userID,
			apiKeyID,
			requestID,
			provider,
			model,
			promptTokens,
			completionTokens,
			latency,
			statusCode,
		)
		if err != nil {
			logger.Error("failed to record billing usage", zap.Error(err))
		}
	}

	logger.Info("usage recorded",
		zap.String("request_id", requestID),
		zap.String("provider", provider),
		zap.String("model", model),
		zap.Int64("prompt_tokens", promptTokens),
		zap.Int64("completion_tokens", completionTokens),
		zap.Int("latency_ms", latency),
	)
}

// Models - 返回可用模型列表（动态）
func (h *AnthropicHandler) Models(c *gin.Context) {
	// 返回Anthropic格式的模型列表
	// 默认列表（如果无法从数据库获取）
	models := []gin.H{
		{"id": "claude-opus-4-7", "display_name": "Claude Opus 4.7", "type": "message"},
		{"id": "claude-sonnet-4-6", "display_name": "Claude Sonnet 4.6", "type": "message"},
		{"id": "claude-haiku-4-5", "display_name": "Claude Haiku 4.5", "type": "message"},
		{"id": "claude-opus-4-6", "display_name": "Claude Opus 4.6", "type": "message"},
		{"id": "claude-sonnet-4-5", "display_name": "Claude Sonnet 4.5", "type": "message"},
		// 其他Provider模型（通过前缀访问）
		{"id": "openai-gpt-4o", "display_name": "GPT-4o (via OpenAI)", "type": "message"},
		{"id": "openai-gpt-4o-mini", "display_name": "GPT-4o Mini (via OpenAI)", "type": "message"},
		{"id": "deepseek-v4-flash", "display_name": "DeepSeek V4 Flash", "type": "message"},
		{"id": "deepseek-v4-pro", "display_name": "DeepSeek V4 Pro", "type": "message"},
		{"id": "glm-4.7", "display_name": "GLM-4.7", "type": "message"},
		{"id": "glm-4-flash", "display_name": "GLM-4 Flash (Free)", "type": "message"},
		{"id": "gemini-2.5-flash", "display_name": "Gemini 2.5 Flash", "type": "message"},
	}

	c.JSON(http.StatusOK, gin.H{
		"type": "models",
		"data": models,
	})
}

