package anthropic

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

	"github.com/zhaojiewen/open-station/plugins/builtin"
	"github.com/zhaojiewen/open-station/pkg/plugin"
)

// AnthropicPlugin implements ProviderPlugin for Anthropic/Claude
type AnthropicPlugin struct {
	*builtin.BasePlugin
	version string // API version (e.g., "2023-06-01")
}

// New creates a new Anthropic plugin
func New() *AnthropicPlugin {
	info := plugin.PluginInfo{
		ID:           "anthropic",
		Name:         "Anthropic Claude",
		Version:      "1.0.0",
		Type:         plugin.PluginTypeAdapter,
		Provider:     "anthropic",
		Description:  "Anthropic Claude API integration (Claude 3.5, Claude 3)",
		Author:       "Open Station Team",
		Repository:   "https://github.com/zhaojiewen/open-station",
		Capabilities: []string{"chat", "stream", "models"},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"base_url": map[string]interface{}{
					"type":        "string",
					"description": "Anthropic API base URL",
					"default":     "https://api.anthropic.com/v1",
				},
				"api_key": map[string]interface{}{
					"type":        "string",
					"description": "Anthropic API key",
					"required":    true,
				},
				"version": map[string]interface{}{
					"type":        "string",
					"description": "Anthropic API version",
					"default":     "2023-06-01",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds",
					"default":     30,
				},
			},
			"required": []string{"api_key"},
		},
	}

	return &AnthropicPlugin{
		BasePlugin: builtin.NewBasePlugin(info),
		version:    "2023-06-01",
	}
}

// Initialize sets up the plugin with configuration
func (p *AnthropicPlugin) Initialize(config map[string]interface{}) error {
	// Set default base URL if not provided
	if _, ok := config["base_url"]; !ok {
		config["base_url"] = "https://api.anthropic.com/v1"
	}

	if version, ok := config["version"].(string); ok && version != "" {
		p.version = version
	}

	return p.BasePlugin.Initialize(config)
}

// doAnthropicRequest performs an HTTP request with Anthropic-specific headers
func (p *AnthropicPlugin) doAnthropicRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", p.version)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// ChatCompletion sends a synchronous chat request
func (p *AnthropicPlugin) ChatCompletion(ctx context.Context, req *plugin.ChatRequest) (*plugin.ChatResponse, error) {
	// Convert to Anthropic Messages API format
	anthropicReq := map[string]interface{}{
		"model": req.Model,
	}

	// Convert messages - Anthropic uses separate system field
	var systemMsg string
	var messages []map[string]interface{}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		} else {
			messages = append(messages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	if systemMsg != "" {
		anthropicReq["system"] = systemMsg
	}
	anthropicReq["messages"] = messages

	if req.MaxTokens > 0 {
		anthropicReq["max_tokens"] = req.MaxTokens
	} else {
		anthropicReq["max_tokens"] = 4096 // Default for Claude
	}

	// Add extra fields
	for k, v := range req.Extra {
		anthropicReq[k] = v
	}

	resp, err := p.doAnthropicRequest(ctx, "POST", "/messages", anthropicReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, p.parseError(resp, body)
	}

	// Parse Anthropic response
	var anthropicResp struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		Content      []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build response content
	var content string
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	// Convert to standard response
	result := &plugin.ChatResponse{
		ID:      anthropicResp.ID,
		Model:   anthropicResp.Model,
		Created: time.Now().Unix(),
		Usage: plugin.UsageInfo{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
		Choices: []plugin.ChatChoice{
			{
				Index:        0,
				Message:      plugin.ChatMessage{Role: "assistant", Content: content},
				FinishReason: anthropicResp.StopReason,
			},
		},
	}

	return result, nil
}

// StreamChatCompletion sends a streaming chat request
func (p *AnthropicPlugin) StreamChatCompletion(ctx context.Context, req *plugin.ChatRequest) (plugin.StreamReader, error) {
	// Convert to Anthropic Messages API format
	anthropicReq := map[string]interface{}{
		"model":  req.Model,
		"stream": true,
	}

	// Convert messages - Anthropic uses separate system field
	var systemMsg string
	var messages []map[string]interface{}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		} else {
			messages = append(messages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	if systemMsg != "" {
		anthropicReq["system"] = systemMsg
	}
	anthropicReq["messages"] = messages

	if req.MaxTokens > 0 {
		anthropicReq["max_tokens"] = req.MaxTokens
	} else {
		anthropicReq["max_tokens"] = 4096
	}

	resp, err := p.doAnthropicRequest(ctx, "POST", "/messages", anthropicReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, p.parseError(resp, body)
	}

	return &AnthropicStreamReader{resp: resp, scanner: bufio.NewScanner(resp.Body)}, nil
}

// Embedding generates embeddings - Claude doesn't have embedding models
func (p *AnthropicPlugin) Embedding(ctx context.Context, req *plugin.EmbeddingRequest) (*plugin.EmbeddingResponse, error) {
	return nil, fmt.Errorf("anthropic does not support embeddings")
}

// ListModels returns available models for this provider
func (p *AnthropicPlugin) ListModels(ctx context.Context) ([]plugin.ModelInfo, error) {
	// Anthropic doesn't have a models endpoint, return known models
	models := []plugin.ModelInfo{
		{
			ID:           "claude-3-5-sonnet-20241022",
			Name:         "Claude 3.5 Sonnet",
			Provider:     "anthropic",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "claude-3-5-haiku-20241022",
			Name:         "Claude 3.5 Haiku",
			Provider:     "anthropic",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "claude-3-opus-20240229",
			Name:         "Claude 3 Opus",
			Provider:     "anthropic",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "claude-3-sonnet-20240229",
			Name:         "Claude 3 Sonnet",
			Provider:     "anthropic",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "claude-3-haiku-20240307",
			Name:         "Claude 3 Haiku",
			Provider:     "anthropic",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
	}
	return models, nil
}

// ValidateAPIKey checks if an API key is valid format
func (p *AnthropicPlugin) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("api key is empty")
	}
	if !strings.HasPrefix(apiKey, "sk-ant-") {
		return fmt.Errorf("anthropic api key should start with 'sk-ant-'")
	}
	return nil
}

// AnthropicStreamReader implements StreamReader for Anthropic streaming
type AnthropicStreamReader struct {
	resp    *http.Response
	scanner *bufio.Scanner
	done    bool
	id      string
	model   string
}

// Recv reads the next chunk from the stream
func (s *AnthropicStreamReader) Recv() (*plugin.StreamChunk, error) {
	if s.done {
		return nil, io.EOF
	}

	for s.scanner.Scan() {
		line := s.scanner.Text()
		if line == "" {
			continue
		}

		var event struct {
			Type         string `json:"type"`
			Index        int    `json:"index,omitempty"`
			Delta        struct {
				Type string `json:"type,omitempty"`
				Text string `json:"text,omitempty"`
			} `json:"delta,omitempty"`
			Message      struct {
				ID      string `json:"id,omitempty"`
				Type    string `json:"type,omitempty"`
				Model   string `json:"model,omitempty"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content,omitempty"`
			} `json:"message,omitempty"`
		}

		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			s.id = event.Message.ID
			s.model = event.Message.Model
			continue

		case "content_block_start":
			continue

		case "content_block_delta":
			return &plugin.StreamChunk{
				ID:      s.id,
				Model:   s.model,
				Created: time.Now().Unix(),
				Choices: []plugin.StreamChoice{
					{
						Index: event.Index,
						Delta: plugin.StreamDelta{Content: event.Delta.Text},
					},
				},
			}, nil

		case "content_block_stop":
			continue

		case "message_stop":
			s.done = true
			return &plugin.StreamChunk{Done: true}, nil

		case "message_delta":
			continue
		}
	}

	if err := s.scanner.Err(); err != nil {
		return nil, err
	}

	s.done = true
	return nil, io.EOF
}

// Close closes the stream reader
func (s *AnthropicStreamReader) Close() error {
	return s.resp.Body.Close()
}