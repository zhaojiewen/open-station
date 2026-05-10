package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// AdapterLoader loads external adapter plugins (HTTP/gRPC)
type AdapterLoader struct {
	adapters map[string]*AdapterClient
	mu       sync.RWMutex
	timeout  time.Duration
}

// NewAdapterLoader creates a new adapter loader
func NewAdapterLoader() *AdapterLoader {
	return &AdapterLoader{
		adapters: make(map[string]*AdapterClient),
		timeout:  120 * time.Second,
	}
}

// Load loads an adapter from URL
func (l *AdapterLoader) Load(url string) (ProviderPlugin, PluginInfo, error) {
	// Create adapter client
	client := NewAdapterClient(url, l.timeout)

	// Fetch plugin info from adapter
	info, err := client.FetchInfo()
	if err != nil {
		return nil, PluginInfo{}, fmt.Errorf("failed to fetch adapter info: %w", err)
	}

	l.mu.Lock()
	l.adapters[info.ID] = client
	l.mu.Unlock()

	return client, info, nil
}

// Unload unloads an adapter
func (l *AdapterLoader) Unload(pluginID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if client, exists := l.adapters[pluginID]; exists {
		client.Close()
		delete(l.adapters, pluginID)
	}

	return nil
}

// AdapterClient implements ProviderPlugin for external HTTP adapters
type AdapterClient struct {
	baseURL   string
	timeout   time.Duration
	httpClient *http.Client
	info      PluginInfo
	mu        sync.RWMutex
}

// NewAdapterClient creates a new adapter client
func NewAdapterClient(url string, timeout time.Duration) *AdapterClient {
	return &AdapterClient{
		baseURL: strings.TrimSuffix(url, "/"),
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// FetchInfo retrieves plugin info from adapter
func (c *AdapterClient) FetchInfo() (PluginInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/info")
	if err != nil {
		return PluginInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PluginInfo{}, fmt.Errorf("adapter returned status %d", resp.StatusCode)
	}

	var info PluginInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return PluginInfo{}, err
	}

	c.mu.Lock()
	c.info = info
	c.mu.Unlock()

	return info, nil
}

// Info returns plugin metadata
func (c *AdapterClient) Info() PluginInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.info
}

// Initialize sets up the adapter with configuration
func (c *AdapterClient) Initialize(config map[string]interface{}) error {
	// Send config to adapter
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/initialize", "application/json", strings.NewReader(string(jsonConfig)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("initialize failed: %s", string(body))
	}

	return nil
}

// Shutdown shuts down the adapter
func (c *AdapterClient) Shutdown() error {
	resp, err := c.httpClient.Post(c.baseURL+"/shutdown", "application/json", nil)
	if err != nil {
		logger.Warn("adapter shutdown request failed", zap.Error(err))
		return nil // Don't fail on shutdown error
	}
	defer resp.Body.Close()
	return nil
}

// ChatCompletion sends a chat request to the adapter
func (c *AdapterClient) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", strings.NewReader(string(jsonReq)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, c.ParseError(fmt.Errorf("adapter error: %s", string(body)))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, err
	}

	return &chatResp, nil
}

// StreamChatCompletion sends a streaming chat request
func (c *AdapterClient) StreamChatCompletion(ctx context.Context, req *ChatRequest) (StreamReader, error) {
	req.Stream = true
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions/stream", strings.NewReader(string(jsonReq)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("adapter stream error: %s", string(body))
	}

	return NewAdapterStreamReader(resp.Body), nil
}

// Embedding sends an embedding request
func (c *AdapterClient) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", strings.NewReader(string(jsonReq)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding error: %s", string(body))
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, err
	}

	return &embResp, nil
}

// ListModels returns available models
func (c *AdapterClient) ListModels(ctx context.Context) ([]ModelInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/models")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: status %d", resp.StatusCode)
	}

	var models []ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, err
	}

	return models, nil
}

// ValidateAPIKey checks if an API key is valid format
func (c *AdapterClient) ValidateAPIKey(apiKey string) error {
	// Basic validation - adapters may override
	if apiKey == "" {
		return fmt.Errorf("api key is empty")
	}
	return nil
}

// ParseError maps adapter errors to standard errors
func (c *AdapterClient) ParseError(err error) *PluginError {
	errMsg := err.Error()

	// Common error patterns
	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "429") {
		return &PluginError{
			Type:      ErrorTypeRateLimit,
			Message:   errMsg,
			Retryable: true,
			WaitTime:  60,
		}
	}

	if strings.Contains(errMsg, "quota") || strings.Contains(errMsg, "insufficient") {
		return &PluginError{
			Type:      ErrorTypeQuota,
			Message:   errMsg,
			Retryable: false,
		}
	}

	if strings.Contains(errMsg, "auth") || strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") {
		return &PluginError{
			Type:      ErrorTypeAuth,
			Message:   errMsg,
			Retryable: false,
		}
	}

	if strings.Contains(errMsg, "timeout") {
		return &PluginError{
			Type:      ErrorTypeTimeout,
			Message:   errMsg,
			Retryable: true,
			WaitTime:  5,
		}
	}

	if strings.Contains(errMsg, "model") || strings.Contains(errMsg, "not found") {
		return &PluginError{
			Type:      ErrorTypeModel,
			Message:   errMsg,
			Retryable: false,
		}
	}

	return &PluginError{
		Type:      ErrorTypeInternal,
		Message:   errMsg,
		Retryable: false,
	}
}

// HealthCheck checks adapter health
func (c *AdapterClient) HealthCheck(ctx context.Context) error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("adapter health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// GetCapabilities returns adapter capabilities
func (c *AdapterClient) GetCapabilities() []string {
	return c.info.Capabilities
}

// Close closes the adapter client
func (c *AdapterClient) Close() {
	c.httpClient.CloseIdleConnections()
}

// AdapterStreamReader implements StreamReader for adapter streaming
type AdapterStreamReader struct {
	body    io.ReadCloser
	decoder *json.Decoder
	done    bool
}

// NewAdapterStreamReader creates a new adapter stream reader
func NewAdapterStreamReader(body io.ReadCloser) *AdapterStreamReader {
	return &AdapterStreamReader{
		body:    body,
		decoder: json.NewDecoder(body),
	}
}

// Recv receives a stream chunk
func (r *AdapterStreamReader) Recv() (*StreamChunk, error) {
	if r.done {
		return nil, io.EOF
	}

	var chunk StreamChunk
	if err := r.decoder.Decode(&chunk); err != nil {
		if err == io.EOF {
			r.done = true
			return nil, io.EOF
		}
		return nil, err
	}

	if chunk.Done {
		r.done = true
	}

	return &chunk, nil
}

// Close closes the stream reader
func (r *AdapterStreamReader) Close() error {
	return r.body.Close()
}