package plugin

import (
	"context"
	"time"
)

// PluginType defines how the plugin is loaded
type PluginType string

const (
	PluginTypeGo      PluginType = "go"      // Compiled Go .so plugin
	PluginTypeAdapter PluginType = "adapter" // External HTTP/gRPC adapter
)

// PluginStatus defines the current state of a plugin
type PluginStatus string

const (
	PluginStatusActive   PluginStatus = "active"
	PluginStatusInactive PluginStatus = "inactive"
	PluginStatusError    PluginStatus = "error"
	PluginStatusLoading  PluginStatus = "loading"
)

// ErrorType defines standardized error categories from providers
type ErrorType string

const (
	ErrorTypeRateLimit ErrorType = "rate_limit"
	ErrorTypeQuota     ErrorType = "quota_exceeded"
	ErrorTypeAuth      ErrorType = "authentication"
	ErrorTypeTimeout   ErrorType = "timeout"
	ErrorTypeModel     ErrorType = "model_not_found"
	ErrorTypeInternal  ErrorType = "internal"
	ErrorTypeNetwork   ErrorType = "network"
	ErrorTypeInvalid   ErrorType = "invalid_request"
)

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	ID              string                 `json:"id"`              // Unique identifier (e.g., "openai", "anthropic")
	Name            string                 `json:"name"`            // Display name
	Version         string                 `json:"version"`         // Plugin version
	Type            PluginType             `json:"type"`            // go or adapter
	Provider        string                 `json:"provider"`        // Provider this plugin handles
	Description     string                 `json:"description"`     // Plugin description
	Author          string                 `json:"author"`          // Plugin author
	Repository      string                 `json:"repository"`      // Source repository URL
	Capabilities    []string               `json:"capabilities"`    // ["chat", "stream", "embedding", "models"]
	ConfigSchema    map[string]interface{} `json:"config_schema"`   // Configuration schema
	Dependencies    []string               `json:"dependencies"`    // Required dependencies
	AdapterURL      string                 `json:"adapter_url"`     // For external adapters
	AdapterProtocol string                 `json:"adapter_protocol"` // "http" or "grpc"
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string                   `json:"model"`
	Messages    []ChatMessage            `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	TopP        float64                  `json:"top_p,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	Tools       []ToolDefinition         `json:"tools,omitempty"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
	Extra       map[string]interface{}   `json:"extra,omitempty"` // Provider-specific fields
}

// ChatMessage represents a single message in a chat
type ChatMessage struct {
	Role    string                 `json:"role"`    // system, user, assistant
	Content string                 `json:"content"` // Text content
	Name    string                 `json:"name,omitempty"`
	Images  []ImageContent         `json:"images,omitempty"` // For vision models
	Extra   map[string]interface{} `json:"extra,omitempty"`  // Provider-specific fields
}

// ImageContent represents an image in a message
type ImageContent struct {
	Type     string `json:"type"`     // "url" or "base64"
	URL      string `json:"url,omitempty"`
	Base64   string `json:"base64,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// ToolDefinition represents a tool/function definition
type ToolDefinition struct {
	Type     string                 `json:"type"`     // "function"
	Function FunctionDefinition     `json:"function"`
}

// FunctionDefinition represents a function definition
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string            `json:"id"`
	Model   string            `json:"model"`
	Choices []ChatChoice      `json:"choices"`
	Usage   UsageInfo         `json:"usage"`
	Created int64             `json:"created"`
	Extra   map[string]interface{} `json:"extra,omitempty"` // Provider-specific fields
}

// ChatChoice represents a choice in the response
type ChatChoice struct {
	Index        int          `json:"index"`
	Message      ChatMessage  `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// UsageInfo contains token usage information
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	ID      string           `json:"id"`
	Model   string           `json:"model"`
	Choices []StreamChoice   `json:"choices"`
	Created int64            `json:"created"`
	Done    bool             `json:"done"` // Indicates stream end
}

// StreamChoice represents a choice in a stream chunk
type StreamChoice struct {
	Index        int               `json:"index"`
	Delta        StreamDelta        `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

// StreamDelta represents the delta content in a stream
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"` // Texts to embed
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Model   string            `json:"model"`
	Data    []EmbeddingData   `json:"data"`
	Usage   UsageInfo         `json:"usage"`
}

// EmbeddingData represents a single embedding
type EmbeddingData struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Provider    string            `json:"provider"`
	Type        string            `json:"type"` // "chat", "embedding", "image"
	MaxTokens   int               `json:"max_tokens"`
	Pricing     ModelPricing      `json:"pricing"`
	Capabilities []string         `json:"capabilities"`
	Created     time.Time         `json:"created"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// ModelPricing represents pricing information
type ModelPricing struct {
	PromptPrice     float64 `json:"prompt_price"`     // Per 1K tokens
	CompletionPrice float64 `json:"completion_price"` // Per 1K tokens
	Currency        string  `json:"currency"`
}

// PluginError represents an error from a plugin
type PluginError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Code       string    `json:"code,omitempty"`       // Provider-specific code
	Retryable  bool      `json:"retryable"`
	WaitTime   int       `json:"wait_time,omitempty"`  // Seconds to wait before retry
	Provider   string    `json:"provider,omitempty"`
}

func (e *PluginError) Error() string {
	return e.Message
}

// StreamReader interface for streaming responses
type StreamReader interface {
	Recv() (*StreamChunk, error)
	Close() error
}

// StreamWriter interface for writing streaming responses
type StreamWriter interface {
	Write(chunk *StreamChunk) error
	Close() error
}

// ProviderPlugin is the main interface for provider implementations
type ProviderPlugin interface {
	// Info returns plugin metadata
	Info() PluginInfo

	// Initialize sets up the plugin with configuration
	Initialize(config map[string]interface{}) error

	// Shutdown cleans up plugin resources
	Shutdown() error

	// ChatCompletion sends a synchronous chat request
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChatCompletion sends a streaming chat request
	StreamChatCompletion(ctx context.Context, req *ChatRequest) (StreamReader, error)

	// Embedding generates embeddings
	Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// ListModels returns available models for this provider
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// ValidateAPIKey checks if an API key is valid format
	ValidateAPIKey(apiKey string) error

	// ParseError maps provider-specific errors to standard errors
	ParseError(err error) *PluginError

	// HealthCheck verifies plugin is working
	HealthCheck(ctx context.Context) error

	// GetCapabilities returns supported capabilities
	GetCapabilities() []string
}

// PluginHook defines hooks for plugin lifecycle events
type PluginHook interface {
	// OnLoad called when plugin is loaded
	OnLoad() error

	// OnActivate called when plugin is activated
	OnActivate() error

	// OnDeactivate called when plugin is deactivated
	OnDeactivate() error

	// OnUnload called when plugin is unloaded
	OnUnload() error

	// OnError called when plugin encounters an error
	OnError(err error)
}

// PluginMiddleware defines middleware for request/response processing
type PluginMiddleware interface {
	// PreRequest called before request is sent to provider
	PreRequest(ctx context.Context, req *ChatRequest) (*ChatRequest, error)

	// PostResponse called after response is received
	PostResponse(ctx context.Context, resp *ChatResponse) (*ChatResponse, error)

	// PreStream called before streaming request
	PreStream(ctx context.Context, req *ChatRequest) (*ChatRequest, error)

	// PostStreamChunk called after each stream chunk
	PostStreamChunk(ctx context.Context, chunk *StreamChunk) (*StreamChunk, error)
}

// PluginFactory creates provider plugins
type PluginFactory interface {
	Create(config map[string]interface{}) (ProviderPlugin, error)
}

// PluginManifest for plugin metadata file
type PluginManifest struct {
	PluginInfo
	Main    string `json:"main"`    // Entry point function for Go plugins
	Module  string `json:"module"`  // Go module path
	SHA256  string `json:"sha256"`  // SHA256 checksum for validation
	MinVersion string `json:"min_version"` // Minimum open-station version
}