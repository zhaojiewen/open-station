package glm

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

// GLMPlugin implements ProviderPlugin for GLM (Zhipu AI)
type GLMPlugin struct {
	*builtin.BasePlugin
}

// New creates a new GLM plugin
func New() *GLMPlugin {
	info := plugin.PluginInfo{
		ID:           "glm",
		Name:         "GLM (Zhipu AI)",
		Version:      "1.0.0",
		Type:         plugin.PluginTypeAdapter,
		Provider:     "glm",
		Description:  "Zhipu AI GLM API integration (GLM-4, GLM-4V)",
		Author:       "Open Station Team",
		Repository:   "https://github.com/zhaojiewen/open-station",
		Capabilities: []string{"chat", "stream", "embedding", "models"},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"base_url": map[string]interface{}{
					"type":        "string",
					"description": "GLM API base URL",
					"default":     "https://open.bigmodel.cn/api/paas/v4",
				},
				"api_key": map[string]interface{}{
					"type":        "string",
					"description": "Zhipu AI API key (JWT token)",
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

	return &GLMPlugin{
		BasePlugin: builtin.NewBasePlugin(info),
	}
}

// Initialize sets up the plugin
func (p *GLMPlugin) Initialize(config map[string]interface{}) error {
	if _, ok := config["base_url"]; !ok {
		config["base_url"] = "https://open.bigmodel.cn/api/paas/v4"
	}
	return p.BasePlugin.Initialize(config)
}

// ChatCompletion sends a synchronous chat request
func (p *GLMPlugin) ChatCompletion(ctx context.Context, req *plugin.ChatRequest) (*plugin.ChatResponse, error) {
	glmReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
	}

	if req.MaxTokens > 0 {
		glmReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		glmReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		glmReq["top_p"] = req.TopP
	}
	if len(req.Tools) > 0 {
		glmReq["tools"] = req.Tools
	}

	resp, err := p.doRequest(ctx, "POST", "/chat/completions", glmReq)
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

	var glmResp struct {
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

	if err := json.Unmarshal(body, &glmResp); err != nil {
		return nil, err
	}

	result := &plugin.ChatResponse{
		ID:      glmResp.ID,
		Model:   glmResp.Model,
		Created: glmResp.Created,
		Usage: plugin.UsageInfo{
			PromptTokens:     glmResp.Usage.PromptTokens,
			CompletionTokens: glmResp.Usage.CompletionTokens,
			TotalTokens:      glmResp.Usage.TotalTokens,
		},
	}

	for _, choice := range glmResp.Choices {
		result.Choices = append(result.Choices, plugin.ChatChoice{
			Index:        choice.Index,
			Message:      plugin.ChatMessage{Role: choice.Message.Role, Content: choice.Message.Content},
			FinishReason: choice.FinishReason,
		})
	}

	return result, nil
}

// StreamChatCompletion sends a streaming request
func (p *GLMPlugin) StreamChatCompletion(ctx context.Context, req *plugin.ChatRequest) (plugin.StreamReader, error) {
	glmReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	if req.MaxTokens > 0 {
		glmReq["max_tokens"] = req.MaxTokens
	}

	resp, err := p.doRequest(ctx, "POST", "/chat/completions", glmReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, p.parseError(resp, body)
	}

	return &GLMStreamReader{resp: resp, scanner: bufio.NewScanner(resp.Body)}, nil
}

// Embedding generates embeddings
func (p *GLMPlugin) Embedding(ctx context.Context, req *plugin.EmbeddingRequest) (*plugin.EmbeddingResponse, error) {
	glmReq := map[string]interface{}{
		"model": req.Model,
		"input": req.Input,
	}

	resp, err := p.doRequest(ctx, "POST", "/embeddings", glmReq)
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

	var glmResp struct {
		Model string `json:"model"`
		Data  []struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &glmResp); err != nil {
		return nil, err
	}

	result := &plugin.EmbeddingResponse{
		Model: glmResp.Model,
		Usage: plugin.UsageInfo{
			PromptTokens: glmResp.Usage.PromptTokens,
			TotalTokens:  glmResp.Usage.TotalTokens,
		},
	}

	for _, data := range glmResp.Data {
		result.Data = append(result.Data, plugin.EmbeddingData{
			Index:     data.Index,
			Embedding: data.Embedding,
		})
	}

	return result, nil
}

// ListModels returns available models
func (p *GLMPlugin) ListModels(ctx context.Context) ([]plugin.ModelInfo, error) {
	models := []plugin.ModelInfo{
		{
			ID:           "glm-4",
			Name:         "GLM-4",
			Provider:     "glm",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "glm-4-air",
			Name:         "GLM-4 Air",
			Provider:     "glm",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "glm-4-flash",
			Name:         "GLM-4 Flash",
			Provider:     "glm",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "glm-4v",
			Name:         "GLM-4V (Vision)",
			Provider:     "glm",
			Type:         "chat",
			Capabilities: []string{"chat", "stream", "vision"},
		},
		{
			ID:           "embedding-2",
			Name:         "Embedding-2",
			Provider:     "glm",
			Type:         "embedding",
			Capabilities: []string{"embedding"},
		},
	}
	return models, nil
}

// ValidateAPIKey checks API key format (GLM uses JWT tokens)
func (p *GLMPlugin) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("api key is empty")
	}
	// GLM API keys are JWT tokens, typically long strings
	if len(apiKey) < 50 {
		return fmt.Errorf("glm api key appears too short (should be JWT token)")
	}
	return nil
}

// GLMStreamReader implements StreamReader
type GLMStreamReader struct {
	resp    *http.Response
	scanner *bufio.Scanner
	done    bool
}

// Recv reads next chunk
func (s *GLMStreamReader) Recv() (*plugin.StreamChunk, error) {
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
func (s *GLMStreamReader) Close() error {
	return s.resp.Body.Close()
}