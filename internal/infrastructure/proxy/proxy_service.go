package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/pkg/config"
	"github.com/zhaojiewen/open-station/pkg/errors"
)

// Shared HTTP transport with optimized connection pooling for high-throughput proxy
var (
	sharedTransport     *http.Transport
	sharedTransportOnce sync.Once
)

func getSharedTransport() *http.Transport {
	sharedTransportOnce.Do(func() {
		sharedTransport = &http.Transport{
			// High-throughput settings for LLM gateway
			MaxIdleConns:          500,              // Total idle connections across all hosts
			MaxIdleConnsPerHost:   100,              // Per-host idle (critical for same provider)
			MaxConnsPerHost:       200,              // Max total connections per host
			IdleConnTimeout:       120 * time.Second, // Longer timeout for long-lived connections
			ResponseHeaderTimeout: 30 * time.Second, // Prevent hanging on slow responses
			ExpectContinueTimeout: 1 * time.Second,
			DisableCompression:    false,
			ForceAttemptHTTP2:     true,
			// Enable TCP keepalive for connection health
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}
	})
	return sharedTransport
}

func newHTTPClientWithPool(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: getSharedTransport(),
	}
}

type ProxyRequest struct {
	Provider   string                 `json:"provider"`
	Model      string                 `json:"model"`
	Messages   []Message              `json:"messages"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Stream     bool                   `json:"stream,omitempty"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
	Name    string      `json:"name,omitempty"`
}

type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type ProxyResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamReader - 流式响应读取器接口
type StreamReader interface {
	io.ReadCloser
}

// StreamChunk - 流式响应块
type StreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// ParseStreamChunk - 解析流式响应块
func ParseStreamChunk(data string, chunk *StreamChunk) error {
	return json.Unmarshal([]byte(data), chunk)
}

// ProviderClient interface
type ProviderClient interface {
	ChatCompletion(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)
	StreamChatCompletion(ctx context.Context, req *ProxyRequest) (StreamReader, error)
	Embedding(ctx context.Context, req *ProxyRequest) ([]float64, error)
}

// OpenAIClient
type OpenAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewOpenAIClient(cfg *config.ProviderConfig) *OpenAIClient {
	return &OpenAIClient{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		httpClient: newHTTPClientWithPool(cfg.Timeout),
	}
}

func (c *OpenAIClient) ChatCompletion(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	openaiReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   req.Stream,
	}

	for k, v := range req.Parameters {
		openaiReq[k] = v
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	var openaiResp ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &openaiResp, nil
}

func (c *OpenAIClient) StreamChatCompletion(ctx context.Context, req *ProxyRequest) (StreamReader, error) {
	openaiReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	for k, v := range req.Parameters {
		openaiReq[k] = v
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	return resp.Body, nil
}

func (c *OpenAIClient) Embedding(ctx context.Context, req *ProxyRequest) ([]float64, error) {
	openaiReq := map[string]interface{}{
		"model": req.Model,
		"input": req.Messages[0].Content,
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	var embeddingResp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embeddingResp.Data[0].Embedding, nil
}

// ClaudeClient
type ClaudeClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClaudeClient(cfg *config.ProviderConfig) *ClaudeClient {
	return &ClaudeClient{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		httpClient: newHTTPClientWithPool(cfg.Timeout),
	}
}

func (c *ClaudeClient) ChatCompletion(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	claudeReq := buildClaudeRequestMap(req)

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	var claudeResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	content := ""
	for _, c := range claudeResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &ProxyResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   claudeResp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: claudeResp.StopReason,
			},
		},
		Usage: Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}, nil
}

func (c *ClaudeClient) StreamChatCompletion(ctx context.Context, req *ProxyRequest) (StreamReader, error) {
	claudeReq := buildClaudeRequestMap(req)
	claudeReq["stream"] = true

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	return resp.Body, nil
}

func (c *ClaudeClient) Embedding(ctx context.Context, req *ProxyRequest) ([]float64, error) {
	return nil, fmt.Errorf("Claude does not support embeddings")
}

// DeepSeekClient
type DeepSeekClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewDeepSeekClient(cfg *config.ProviderConfig) *DeepSeekClient {
	return &DeepSeekClient{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		httpClient: newHTTPClientWithPool(cfg.Timeout),
	}
}

func (c *DeepSeekClient) ChatCompletion(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	deepseekReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   req.Stream,
	}

	for k, v := range req.Parameters {
		deepseekReq[k] = v
	}

	body, err := json.Marshal(deepseekReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	var deepseekResp ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&deepseekResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &deepseekResp, nil
}

func (c *DeepSeekClient) StreamChatCompletion(ctx context.Context, req *ProxyRequest) (StreamReader, error) {
	deepseekReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	for k, v := range req.Parameters {
		deepseekReq[k] = v
	}

	body, err := json.Marshal(deepseekReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	return resp.Body, nil
}

func (c *DeepSeekClient) Embedding(ctx context.Context, req *ProxyRequest) ([]float64, error) {
	return nil, fmt.Errorf("DeepSeek does not support embeddings API")
}

// GLMClient
type GLMClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewGLMClient(cfg *config.ProviderConfig) *GLMClient {
	return &GLMClient{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		httpClient: newHTTPClientWithPool(cfg.Timeout),
	}
}

func (c *GLMClient) ChatCompletion(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	glmReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   req.Stream,
	}

	for k, v := range req.Parameters {
		glmReq[k] = v
	}

	body, err := json.Marshal(glmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	var glmResp ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&glmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &glmResp, nil
}

func (c *GLMClient) StreamChatCompletion(ctx context.Context, req *ProxyRequest) (StreamReader, error) {
	glmReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	for k, v := range req.Parameters {
		glmReq[k] = v
	}

	body, err := json.Marshal(glmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	return resp.Body, nil
}

func (c *GLMClient) Embedding(ctx context.Context, req *ProxyRequest) ([]float64, error) {
	glmReq := map[string]interface{}{
		"model": req.Model,
		"input": req.Messages[0].Content,
	}

	body, err := json.Marshal(glmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		_, _ = io.ReadAll(resp.Body)
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	var embeddingResp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embeddingResp.Data[0].Embedding, nil
}

// ProxyService
type ProxyService struct {
	openaiClient   *OpenAIClient
	claudeClient   *ClaudeClient
	deepseekClient *DeepSeekClient
	glmClient      *GLMClient
	clients        map[string]ProviderClient
}

func NewProxyService(cfg *config.ProvidersConfig) *ProxyService {
	openaiClient := NewOpenAIClient(&cfg.OpenAI)
	claudeClient := NewClaudeClient(&cfg.Claude)
	deepseekClient := NewDeepSeekClient(&cfg.DeepSeek)
	glmClient := NewGLMClient(&cfg.GLM)

	return &ProxyService{
		openaiClient:   openaiClient,
		claudeClient:   claudeClient,
		deepseekClient: deepseekClient,
		glmClient:      glmClient,
		clients: map[string]ProviderClient{
			"openai":   openaiClient,
			"claude":   claudeClient,
			"deepseek": deepseekClient,
			"glm":      glmClient,
		},
	}
}

func (s *ProxyService) ChatCompletion(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	client, ok := s.clients[req.Provider]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}

	return client.ChatCompletion(ctx, req)
}

func (s *ProxyService) StreamChatCompletion(ctx context.Context, req *ProxyRequest) (StreamReader, error) {
	client, ok := s.clients[req.Provider]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}

	req.Stream = true
	return client.StreamChatCompletion(ctx, req)
}

func (s *ProxyService) Embedding(ctx context.Context, req *ProxyRequest) ([]float64, error) {
	client, ok := s.clients[req.Provider]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}

	return client.Embedding(ctx, req)
}

func (s *ProxyService) GenerateRequestID() string {
	return uuid.New().String()
}

// HTTPClientWrapper 用于动态账户的 HTTP 客户端包装器
type HTTPClientWrapper struct {
	httpClient *http.Client
}

// NewHTTPClientWrapper 创建 HTTP 客户端包装器
func NewHTTPClientWrapper(timeout time.Duration) *HTTPClientWrapper {
	return &HTTPClientWrapper{
		httpClient: newHTTPClientWithPool(timeout),
	}
}

// ChatCompletion 使用指定配置执行 ChatCompletion 请求
func (w *HTTPClientWrapper) ChatCompletion(ctx context.Context, cfg *config.ProviderConfig, req *ProxyRequest) (*ProxyResponse, error) {
	isClaude := req.Provider == "anthropic" || req.Provider == "claude"

	var body []byte
	var endpoint string

	if isClaude {
		endpoint = "/messages"
		body = w.buildClaudeRequest(req)
	} else {
		endpoint = "/chat/completions"
		openaiReq := map[string]interface{}{
			"model":    req.Model,
			"messages": req.Messages,
		}
		for k, v := range req.Parameters {
			openaiReq[k] = v
		}
		var err error
		body, err = json.Marshal(openaiReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if isClaude {
		httpReq.Header.Set("x-api-key", cfg.APIKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	if isClaude {
		return w.parseClaudeResponse(resp)
	}

	var proxyResp ProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&proxyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &proxyResp, nil
}

// buildClaudeRequestMap builds a Claude API request map from ProxyRequest
func buildClaudeRequestMap(req *ProxyRequest) map[string]interface{} {
	systemPrompt := ""
	messages := make([]map[string]interface{}, 0)

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				systemPrompt = content
			}
		} else {
			content := msg.Content
			if str, ok := content.(string); ok {
				content = []map[string]interface{}{
					{"type": "text", "text": str},
				}
			}
			messages = append(messages, map[string]interface{}{
				"role":    msg.Role,
				"content": content,
			})
		}
	}

	claudeReq := map[string]interface{}{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": 4096,
	}

	if systemPrompt != "" {
		claudeReq["system"] = systemPrompt
	}

	for k, v := range req.Parameters {
		if k == "max_tokens" {
			claudeReq["max_tokens"] = v
		} else {
			claudeReq[k] = v
		}
	}

	return claudeReq
}

func (w *HTTPClientWrapper) buildClaudeRequestMap(req *ProxyRequest) map[string]interface{} {
	return buildClaudeRequestMap(req)
}

func (w *HTTPClientWrapper) buildClaudeRequest(req *ProxyRequest) []byte {
	claudeReq := buildClaudeRequestMap(req)
	body, _ := json.Marshal(claudeReq)
	return body
}

// StreamChatCompletion 使用指定配置执行流式请求
func (w *HTTPClientWrapper) StreamChatCompletion(ctx context.Context, cfg *config.ProviderConfig, req *ProxyRequest) (StreamReader, error) {
	isClaude := req.Provider == "anthropic" || req.Provider == "claude"

	var body []byte
	var endpoint string

	if isClaude {
		endpoint = "/messages"
		bodyMap := w.buildClaudeRequestMap(req)
		bodyMap["stream"] = true
		body, _ = json.Marshal(bodyMap)
	} else {
		endpoint = "/chat/completions"
		openaiReq := map[string]interface{}{
			"model":    req.Model,
			"messages": req.Messages,
			"stream":   true,
		}
		for k, v := range req.Parameters {
			openaiReq[k] = v
		}
		var err error
		body, err = json.Marshal(openaiReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if isClaude {
		httpReq.Header.Set("x-api-key", cfg.APIKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, errors.NewAppError("PROV_001", "provider returned an error", nil)
	}

	return resp.Body, nil
}

// parseClaudeResponse 解析 Claude/Anthropic 响应
func (w *HTTPClientWrapper) parseClaudeResponse(resp *http.Response) (*ProxyResponse, error) {
	var claudeResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode Claude response: %w", err)
	}

	content := ""
	for _, c := range claudeResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &ProxyResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   claudeResp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: claudeResp.StopReason,
			},
		},
		Usage: Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}, nil
}