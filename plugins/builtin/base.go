package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zhaojiewen/open-station/pkg/plugin"
)

// BasePlugin provides common functionality for built-in provider plugins
type BasePlugin struct {
	Info         plugin.PluginInfo
	Config       map[string]interface{}
	Client       *http.Client
	BaseURL      string
	APIKey       string
	Timeout      time.Duration
	Capabilities []string
}

// NewBasePlugin creates a new base plugin with common settings
func NewBasePlugin(info plugin.PluginInfo) *BasePlugin {
	return &BasePlugin{
		Info: info,
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		Timeout:      30 * time.Second,
		Capabilities: info.Capabilities,
	}
}

// GetInfo returns plugin metadata
func (p *BasePlugin) GetInfo() plugin.PluginInfo {
	return p.Info
}

// Initialize sets up the plugin with configuration
func (p *BasePlugin) Initialize(config map[string]interface{}) error {
	p.Config = config

	if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
		p.BaseURL = baseURL
	}

	if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
		p.APIKey = apiKey
	}

	if timeout, ok := config["timeout"].(int); ok && timeout > 0 {
		p.Timeout = time.Duration(timeout) * time.Second
		p.Client.Timeout = p.Timeout
	} else if timeoutFloat, ok := config["timeout"].(float64); ok && timeoutFloat > 0 {
		p.Timeout = time.Duration(timeoutFloat) * time.Second
		p.Client.Timeout = p.Timeout
	}

	return nil
}

// Shutdown cleans up plugin resources
func (p *BasePlugin) Shutdown() error {
	p.Client.CloseIdleConnections()
	return nil
}

// GetCapabilities returns supported capabilities
func (p *BasePlugin) GetCapabilities() []string {
	return p.Capabilities
}

// DoRequest performs an HTTP request to the provider
func (p *BasePlugin) DoRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := p.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// ParseErrorResponse parses HTTP error responses
func (p *BasePlugin) ParseErrorResponse(resp *http.Response, body []byte) *plugin.PluginError {
	errType := plugin.ErrorTypeInternal
	retryable := false
	waitTime := 0

	switch resp.StatusCode {
	case 401:
		errType = plugin.ErrorTypeAuth
	case 403:
		errType = plugin.ErrorTypeAuth
	case 404:
		errType = plugin.ErrorTypeModel
	case 429:
		errType = plugin.ErrorTypeRateLimit
		retryable = true
		waitTime = 60
	case 500, 502, 503:
		errType = plugin.ErrorTypeInternal
		retryable = true
		waitTime = 5
	case 504:
		errType = plugin.ErrorTypeTimeout
		retryable = true
		waitTime = 10
	}

	return &plugin.PluginError{
		Type:      errType,
		Message:   string(body),
		Code:      fmt.Sprintf("%d", resp.StatusCode),
		Retryable: retryable,
		WaitTime:  waitTime,
		Provider:  p.Info.Provider,
	}
}

// ValidateAPIKey checks if an API key is valid format
func (p *BasePlugin) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("api key is empty")
	}
	if len(apiKey) < 10 {
		return fmt.Errorf("api key too short")
	}
	return nil
}

// ParseError maps provider-specific errors to standard errors
func (p *BasePlugin) ParseError(err error) *plugin.PluginError {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for common error patterns
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
		return &plugin.PluginError{
			Type:      plugin.ErrorTypeTimeout,
			Message:   errMsg,
			Retryable: true,
			WaitTime:  10,
			Provider:  p.Info.Provider,
		}
	}

	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "network") {
		return &plugin.PluginError{
			Type:      plugin.ErrorTypeNetwork,
			Message:   errMsg,
			Retryable: true,
			WaitTime:  5,
			Provider:  p.Info.Provider,
		}
	}

	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "too many requests") {
		return &plugin.PluginError{
			Type:      plugin.ErrorTypeRateLimit,
			Message:   errMsg,
			Retryable: true,
			WaitTime:  60,
			Provider:  p.Info.Provider,
		}
	}

	return &plugin.PluginError{
		Type:      plugin.ErrorTypeInternal,
		Message:   errMsg,
		Retryable: false,
		Provider:  p.Info.Provider,
	}
}

// HealthCheck verifies plugin is working
func (p *BasePlugin) HealthCheck(ctx context.Context) error {
	if p.APIKey == "" {
		return fmt.Errorf("api key not configured")
	}
	if p.BaseURL == "" {
		return fmt.Errorf("base url not configured")
	}
	// Simple connectivity check - try to reach the models endpoint
	resp, err := p.DoRequest(ctx, "GET", "/models", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("provider returned status %d", resp.StatusCode)
	}

	return nil
}

// GetConfig returns current configuration
func (p *BasePlugin) GetConfig() map[string]interface{} {
	return p.Config
}

// Info returns plugin metadata (implements ProviderPlugin interface)
func (p *BasePlugin) Info() plugin.PluginInfo {
	return p.Info
}