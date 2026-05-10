package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/zhaojiewen/open-station/plugins/builtin"
	"github.com/zhaojiewen/open-station/pkg/plugin"
)

// OpenAIPlugin implements ProviderPlugin for OpenAI
type OpenAIPlugin struct {
	*builtin.BasePlugin
}

// New creates a new OpenAI plugin
func New() *OpenAIPlugin {
	info := plugin.PluginInfo{
		ID:           "openai",
		Name:         "OpenAI",
		Version:      "1.0.0",
		Type:         plugin.PluginTypeAdapter,
		Provider:     "openai",
		Description:  "OpenAI API integration (GPT-4, GPT-3.5, DALL-E)",
		Author:       "Open Station Team",
		Repository:   "https://github.com/zhaojiewen/open-station",
		Capabilities: []string{"chat", "stream", "embedding", "models"},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"base_url": map[string]interface{}{
					"type":        "string",
					"description": "OpenAI API base URL",
					"default":     "https://api.openai.com/v1",
				},
				"api_key": map[string]interface{}{
					"type":        "string",
					"description": "OpenAI API key",
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

	return &OpenAIPlugin{
		BasePlugin: builtin.NewBasePlugin(info),
	}
}

// Initialize sets up the plugin with configuration
func (p *OpenAIPlugin) Initialize(config map[string]interface{}) error {
	// Set default base URL if not provided
	if _, ok := config["base_url"]; !ok {
		config["base_url"] = "https://api.openai.com/v1"
	}
	return p.BasePlugin.Initialize(config)
}

// ChatCompletion sends a synchronous chat request
func (p *OpenAIPlugin) ChatCompletion(ctx context.Context, req *plugin.ChatRequest) (*plugin.ChatResponse, error) {
	// Convert to OpenAI request format
	openaiReq := map[string]interface{}{
		"model": req.Model,
		"messages": req.Messages,
	}

	if req.MaxTokens > 0 {
		openaiReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		openaiReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		openaiReq["top_p"] = req.TopP
	}
	if len(req.Tools) > 0 {
		openaiReq["tools"] = req.Tools
	}

	// Add extra fields
	for k, v := range req.Extra {
		openaiReq[k] = v
	}

	resp, err := p.DoRequest(ctx, "POST", "/chat/completions", openaiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, p.ParseErrorResponse(resp, body)
	}

	// Parse OpenAI response
	var openaiResp struct {
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

	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to standard response
	result := &plugin.ChatResponse{
		ID:      openaiResp.ID,
		Model:   openaiResp.Model,
		Created: openaiResp.Created,
		Usage: plugin.UsageInfo{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}

	for _, choice := range openaiResp.Choices {
		result.Choices = append(result.Choices, plugin.ChatChoice{
			Index:        choice.Index,
			Message:      plugin.ChatMessage{Role: choice.Message.Role, Content: choice.Message.Content},
			FinishReason: choice.FinishReason,
		})
	}

	return result, nil
}

// StreamChatCompletion sends a streaming chat request
func (p *OpenAIPlugin) StreamChatCompletion(ctx context.Context, req *plugin.ChatRequest) (plugin.StreamReader, error) {
	// Convert to OpenAI request format
	openaiReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	if req.MaxTokens > 0 {
		openaiReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		openaiReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		openaiReq["top_p"] = req.TopP
	}

	resp, err := p.doRequest(ctx, "POST", "/chat/completions", openaiReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, p.parseError(resp, body)
	}

	return &OpenAIStreamReader{resp: resp, scanner: bufio.NewScanner(resp.Body)}, nil
}

// Embedding generates embeddings
func (p *OpenAIPlugin) Embedding(ctx context.Context, req *plugin.EmbeddingRequest) (*plugin.EmbeddingResponse, error) {
	openaiReq := map[string]interface{}{
		"model": req.Model,
		"input": req.Input,
	}

	resp, err := p.doRequest(ctx, "POST", "/embeddings", openaiReq)
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

	var openaiResp struct {
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

	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := &plugin.EmbeddingResponse{
		Model: openaiResp.Model,
		Usage: plugin.UsageInfo{
			PromptTokens: openaiResp.Usage.PromptTokens,
			TotalTokens:  openaiResp.Usage.TotalTokens,
		},
	}

	for _, data := range openaiResp.Data {
		result.Data = append(result.Data, plugin.EmbeddingData{
			Index:     data.Index,
			Embedding: data.Embedding,
		})
	}

	return result, nil
}

// ListModels returns available models for this provider
func (p *OpenAIPlugin) ListModels(ctx context.Context) ([]plugin.ModelInfo, error) {
	resp, err := p.doRequest(ctx, "GET", "/models", nil)
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

	var openaiResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	models := []plugin.ModelInfo{}
	for _, m := range openaiResp.Data {
		modelType := "chat"
		if strings.Contains(m.ID, "embedding") || strings.Contains(m.ID, "text-embedding") {
			modelType = "embedding"
		} else if strings.Contains(m.ID, "dall-e") || strings.Contains(m.ID, "whisper") {
			modelType = m.ID
		}

		models = append(models, plugin.ModelInfo{
			ID:          m.ID,
			Name:        m.ID,
			Provider:    "openai",
			Type:        modelType,
			Capabilities: p.GetCapabilities(),
		})
	}

	return models, nil
}

// ValidateAPIKey checks if an API key is valid format
func (p *OpenAIPlugin) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("api key is empty")
	}
	if !strings.HasPrefix(apiKey, "sk-") {
		return fmt.Errorf("openai api key should start with 'sk-'")
	}
	return nil
}

// OpenAIStreamReader implements StreamReader for OpenAI streaming
type OpenAIStreamReader struct {
	resp    *http.Response
	scanner *bufio.Scanner
	done    bool
}

// Recv reads the next chunk from the stream
func (s *OpenAIStreamReader) Recv() (*plugin.StreamChunk, error) {
	if s.done {
		return nil, io.EOF
	}

	for s.scanner.Scan() {
		line := s.scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
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
				Index        int `json:"index"`
				Delta        struct {
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

	if err := s.scanner.Err(); err != nil {
		return nil, err
	}

	s.done = true
	return nil, io.EOF
}

// Close closes the stream reader
func (s *OpenAIStreamReader) Close() error {
	return s.resp.Body.Close()
}