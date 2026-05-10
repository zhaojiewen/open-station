package deepseek

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

// DeepSeekPlugin implements ProviderPlugin for DeepSeek
type DeepSeekPlugin struct {
	*builtin.BasePlugin
}

// New creates a new DeepSeek plugin
func New() *DeepSeekPlugin {
	info := plugin.PluginInfo{
		ID:           "deepseek",
		Name:         "DeepSeek",
		Version:      "1.0.0",
		Type:         plugin.PluginTypeAdapter,
		Provider:     "deepseek",
		Description:  "DeepSeek API integration (DeepSeek V3, DeepSeek R1)",
		Author:       "Open Station Team",
		Repository:   "https://github.com/zhaojiewen/open-station",
		Capabilities: []string{"chat", "stream", "models"},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"base_url": map[string]interface{}{
					"type":        "string",
					"description": "DeepSeek API base URL",
					"default":     "https://api.deepseek.com/v1",
				},
				"api_key": map[string]interface{}{
					"type":        "string",
					"description": "DeepSeek API key",
					"required":    true,
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

	return &DeepSeekPlugin{
		BasePlugin: builtin.NewBasePlugin(info),
	}
}

// Initialize sets up the plugin
func (p *DeepSeekPlugin) Initialize(config map[string]interface{}) error {
	if _, ok := config["base_url"]; !ok {
		config["base_url"] = "https://api.deepseek.com/v1"
	}
	return p.BasePlugin.Initialize(config)
}

// ChatCompletion sends a synchronous chat request (OpenAI-compatible)
func (p *DeepSeekPlugin) ChatCompletion(ctx context.Context, req *plugin.ChatRequest) (*plugin.ChatResponse, error) {
	deepseekReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
	}

	if req.MaxTokens > 0 {
		deepseekReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		deepseekReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		deepseekReq["top_p"] = req.TopP
	}

	resp, err := p.doRequest(ctx, "POST", "/chat/completions", deepseekReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, p.parseError(resp, body)
	}

	var dsResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Created int64  `json:"created"`
		Choices []struct {
			Index        int `json:"index"`
			Message      struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &dsResp); err != nil {
		return nil, err
	}

	result := &plugin.ChatResponse{
		ID:      dsResp.ID,
		Model:   dsResp.Model,
		Created: dsResp.Created,
		Usage: plugin.UsageInfo{
			PromptTokens:     dsResp.Usage.PromptTokens,
			CompletionTokens: dsResp.Usage.CompletionTokens,
			TotalTokens:      dsResp.Usage.TotalTokens,
		},
	}

	for _, choice := range dsResp.Choices {
		result.Choices = append(result.Choices, plugin.ChatChoice{
			Index:        choice.Index,
			Message:      plugin.ChatMessage{Role: choice.Message.Role, Content: choice.Message.Content},
			FinishReason: choice.FinishReason,
		})
	}

	return result, nil
}

// StreamChatCompletion sends a streaming request
func (p *DeepSeekPlugin) StreamChatCompletion(ctx context.Context, req *plugin.ChatRequest) (plugin.StreamReader, error) {
	deepseekReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	if req.MaxTokens > 0 {
		deepseekReq["max_tokens"] = req.MaxTokens
	}

	resp, err := p.doRequest(ctx, "POST", "/chat/completions", deepseekReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, p.parseError(resp, body)
	}

	return &DeepSeekStreamReader{resp: resp, scanner: bufio.NewScanner(resp.Body)}, nil
}

// Embedding - DeepSeek doesn't have embedding models
func (p *DeepSeekPlugin) Embedding(ctx context.Context, req *plugin.EmbeddingRequest) (*plugin.EmbeddingResponse, error) {
	return nil, fmt.Errorf("deepseek does not support embeddings")
}

// ListModels returns available models
func (p *DeepSeekPlugin) ListModels(ctx context.Context) ([]plugin.ModelInfo, error) {
	models := []plugin.ModelInfo{
		{
			ID:           "deepseek-chat",
			Name:         "DeepSeek Chat",
			Provider:     "deepseek",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "deepseek-reasoner",
			Name:         "DeepSeek Reasoner (R1)",
			Provider:     "deepseek",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
	}
	return models, nil
}

// ValidateAPIKey checks API key format
func (p *DeepSeekPlugin) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("api key is empty")
	}
	if !strings.HasPrefix(apiKey, "sk-") {
		return fmt.Errorf("deepseek api key should start with 'sk-'")
	}
	return nil
}

// DeepSeekStreamReader implements StreamReader
type DeepSeekStreamReader struct {
	resp    *http.Response
	scanner *bufio.Scanner
	done    bool
}

// Recv reads next chunk
func (s *DeepSeekStreamReader) Recv() (*plugin.StreamChunk, error) {
	if s.done {
		return nil, io.EOF
	}

	for s.scanner.Scan() {
		line := s.scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			s.done = true
			return &plugin.StreamChunk{Done: true}, nil
		}

		var chunk struct {
			ID      string `json:"id"`
			Model   string `json:"model"`
			Created int64  `json:"created"`
			Choices []struct {
				Index   int `json:"index"`
				Delta   struct {
					Role    string `json:"role,omitempty"`
					Content string `json:"content,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason,omitempty"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		result := &plugin.StreamChunk{
			ID:      chunk.ID,
			Model:   chunk.Model,
			Created: chunk.Created,
		}

		for _, choice := range chunk.Choices {
			result.Choices = append(result.Choices, plugin.StreamChoice{
				Index:        choice.Index,
				Delta:        plugin.StreamDelta{Role: choice.Delta.Role, Content: choice.Delta.Content},
				FinishReason: choice.FinishReason,
			})
		}

		return result, nil
	}

	s.done = true
	return nil, io.EOF
}

// Close closes stream
func (s *DeepSeekStreamReader) Close() error {
	return s.resp.Body.Close()
}