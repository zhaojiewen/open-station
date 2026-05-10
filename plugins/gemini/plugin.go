package gemini

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

// GeminiPlugin implements ProviderPlugin for Google Gemini
type GeminiPlugin struct {
	*builtin.BasePlugin
}

// New creates a new Gemini plugin
func New() *GeminiPlugin {
	info := plugin.PluginInfo{
		ID:           "gemini",
		Name:         "Google Gemini",
		Version:      "1.0.0",
		Type:         plugin.PluginTypeAdapter,
		Provider:     "gemini",
		Description:  "Google Gemini API integration (Gemini Pro, Gemini Flash)",
		Author:       "Open Station Team",
		Repository:   "https://github.com/zhaojiewen/open-station",
		Capabilities: []string{"chat", "stream", "embedding", "models"},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"base_url": map[string]interface{}{
					"type":        "string",
					"description": "Gemini API base URL",
					"default":     "https://generativelanguage.googleapis.com/v1beta",
				},
				"api_key": map[string]interface{}{
					"type":        "string",
					"description": "Google API key",
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

	return &GeminiPlugin{
		BasePlugin: builtin.NewBasePlugin(info),
	}
}

// Initialize sets up the plugin with configuration
func (p *GeminiPlugin) Initialize(config map[string]interface{}) error {
	if _, ok := config["base_url"]; !ok {
		config["base_url"] = "https://generativelanguage.googleapis.com/v1beta"
	}
	return p.BasePlugin.Initialize(config)
}

// ChatCompletion sends a synchronous chat request
func (p *GeminiPlugin) ChatCompletion(ctx context.Context, req *plugin.ChatRequest) (*plugin.ChatResponse, error) {
	// Convert to Gemini request format
	geminiReq := map[string]interface{}{
		"contents": []interface{}{},
	}

	// Convert messages to Gemini format
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			// Gemini uses systemInstruction for system messages
			geminiReq["systemInstruction"] = map[string]interface{}{
				"parts": []map[string]interface{}{
					{"text": msg.Content},
				},
			}
			continue
		}

		geminiReq["contents"] = append(geminiReq["contents"].([]interface{}), map[string]interface{}{
			"role":  role,
			"parts": []map[string]interface{}{{"text": msg.Content}},
		})
	}

	// Add generation config
	genConfig := map[string]interface{}{}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		genConfig["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		genConfig["topP"] = req.TopP
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	// Gemini API requires API key in URL
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, req.Model, p.apiKey)

	var reqBody io.Reader
	data, _ := json.Marshal(geminiReq)
	reqBody = bytes.NewReader(data)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, p.parseError(resp, body)
	}

	// Parse Gemini response
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build response content
	var content string
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		content += part.Text
	}

	result := &plugin.ChatResponse{
		ID:      fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
		Model:   req.Model,
		Created: time.Now().Unix(),
		Usage: plugin.UsageInfo{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
		Choices: []plugin.ChatChoice{
			{
				Index:        0,
				Message:      plugin.ChatMessage{Role: "assistant", Content: content},
				FinishReason: geminiResp.Candidates[0].FinishReason,
			},
		},
	}

	return result, nil
}

// StreamChatCompletion sends a streaming chat request
func (p *GeminiPlugin) StreamChatCompletion(ctx context.Context, req *plugin.ChatRequest) (plugin.StreamReader, error) {
	// Convert to Gemini request format
	geminiReq := map[string]interface{}{
		"contents": []interface{}{},
	}

	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			geminiReq["systemInstruction"] = map[string]interface{}{
				"parts": []map[string]interface{}{
					{"text": msg.Content},
				},
			}
			continue
		}

		geminiReq["contents"] = append(geminiReq["contents"].([]interface{}), map[string]interface{}{
			"role":  role,
			"parts": []map[string]interface{}{{"text": msg.Content}},
		})
	}

	genConfig := map[string]interface{}{}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		genConfig["temperature"] = req.Temperature
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", p.baseURL, req.Model, p.apiKey)

	data, _ := json.Marshal(geminiReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, p.parseError(resp, body)
	}

	return &GeminiStreamReader{resp: resp, scanner: bufio.NewScanner(resp.Body), model: req.Model}, nil
}

// Embedding generates embeddings
func (p *GeminiPlugin) Embedding(ctx context.Context, req *plugin.EmbeddingRequest) (*plugin.EmbeddingResponse, error) {
	geminiReq := map[string]interface{}{
		"model": "models/" + req.Model,
		"content": map[string]interface{}{
			"parts": []interface{}{},
		},
	}

	for _, input := range req.Input {
		geminiReq["content"].(map[string]interface{})["parts"] = append(
			geminiReq["content"].(map[string]interface{})["parts"].([]interface{}),
			map[string]interface{}{"text": input},
		)
	}

	url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", p.baseURL, req.Model, p.apiKey)

	data, _ := json.Marshal(geminiReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
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

	var geminiResp struct {
		Embedding struct {
			Values []float64 `json:"values"`
		} `json:"embedding"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, err
	}

	return &plugin.EmbeddingResponse{
		Model: req.Model,
		Data: []plugin.EmbeddingData{
			{Index: 0, Embedding: geminiResp.Embedding.Values},
		},
	}, nil
}

// ListModels returns available models
func (p *GeminiPlugin) ListModels(ctx context.Context) ([]plugin.ModelInfo, error) {
	models := []plugin.ModelInfo{
		{
			ID:           "gemini-1.5-pro",
			Name:         "Gemini 1.5 Pro",
			Provider:     "gemini",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "gemini-1.5-flash",
			Name:         "Gemini 1.5 Flash",
			Provider:     "gemini",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "gemini-2.0-flash",
			Name:         "Gemini 2.0 Flash",
			Provider:     "gemini",
			Type:         "chat",
			Capabilities: []string{"chat", "stream"},
		},
		{
			ID:           "text-embedding-004",
			Name:         "Text Embedding 004",
			Provider:     "gemini",
			Type:         "embedding",
			Capabilities: []string{"embedding"},
		},
	}
	return models, nil
}

// ValidateAPIKey checks if an API key is valid
func (p *GeminiPlugin) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("api key is empty")
	}
	if len(apiKey) < 20 {
		return fmt.Errorf("api key too short")
	}
	return nil
}

// GeminiStreamReader implements StreamReader for Gemini streaming
type GeminiStreamReader struct {
	resp    *http.Response
	scanner *bufio.Scanner
	done    bool
	model   string
}

// Recv reads the next chunk
func (s *GeminiStreamReader) Recv() (*plugin.StreamChunk, error) {
	if s.done {
		return nil, io.EOF
	}

	for s.scanner.Scan() {
		line := s.scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Candidates) == 0 {
			continue
		}

		var content string
		for _, part := range chunk.Candidates[0].Content.Parts {
			content += part.Text
		}

		result := &plugin.StreamChunk{
			ID:      fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
			Model:   s.model,
			Created: time.Now().Unix(),
			Choices: []plugin.StreamChoice{
				{Index: 0, Delta: plugin.StreamDelta{Content: content}},
			},
		}

		if chunk.Candidates[0].FinishReason != "" {
			result.Choices[0].FinishReason = chunk.Candidates[0].FinishReason
			s.done = true
		}

		return result, nil
	}

	if err := s.scanner.Err(); err != nil {
		return nil, err
	}

	s.done = true
	return nil, io.EOF
}

// Close closes the stream
func (s *GeminiStreamReader) Close() error {
	return s.resp.Body.Close()
}