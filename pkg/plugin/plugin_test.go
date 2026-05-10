package plugin

import (
	"context"
	"testing"

	"github.com/zhaojiewen/open-station/pkg/logger"
)

func init() {
	// Initialize logger for tests
	_ = logger.Init("info", "console", "stdout")
}

// MockPlugin implements ProviderPlugin for testing
type MockPlugin struct {
	info   PluginInfo
	config map[string]interface{}
}

func NewMockPlugin(id, provider string) *MockPlugin {
	return &MockPlugin{
		info: PluginInfo{
			ID:           id,
			Name:         id + " Plugin",
			Version:      "1.0.0",
			Type:         PluginTypeAdapter,
			Provider:     provider,
			Capabilities: []string{"chat", "stream", "embedding"},
		},
	}
}

func (m *MockPlugin) Info() PluginInfo { return m.info }
func (m *MockPlugin) Initialize(config map[string]interface{}) error {
	m.config = config
	return nil
}
func (m *MockPlugin) Shutdown() error { return nil }
func (m *MockPlugin) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		ID:    "test-id",
		Model: req.Model,
		Choices: []ChatChoice{
			{Index: 0, Message: ChatMessage{Role: "assistant", Content: "test response"}},
		},
	}, nil
}
func (m *MockPlugin) StreamChatCompletion(ctx context.Context, req *ChatRequest) (StreamReader, error) {
	return &MockStreamReader{}, nil
}
func (m *MockPlugin) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return &EmbeddingResponse{
		Data: []EmbeddingData{{Index: 0, Embedding: []float64{0.1, 0.2, 0.3}}},
	}, nil
}
func (m *MockPlugin) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return []ModelInfo{{ID: "test-model", Provider: m.info.Provider}}, nil
}
func (m *MockPlugin) ValidateAPIKey(apiKey string) error { return nil }
func (m *MockPlugin) ParseError(err error) *PluginError {
	return &PluginError{Type: ErrorTypeInternal, Message: err.Error()}
}
func (m *MockPlugin) HealthCheck(ctx context.Context) error { return nil }
func (m *MockPlugin) GetCapabilities() []string { return m.info.Capabilities }

// MockStreamReader implements StreamReader for testing
type MockStreamReader struct {
	chunks []*StreamChunk
	index  int
}

func NewMockStreamReader() *MockStreamReader {
	return &MockStreamReader{
		chunks: []*StreamChunk{
			{ID: "test", Model: "test-model", Choices: []StreamChoice{{Delta: StreamDelta{Content: "Hello"}}}},
			{ID: "test", Model: "test-model", Choices: []StreamChoice{{Delta: StreamDelta{Content: " world"}}}, Done: true},
		},
	}
}

func (m *MockStreamReader) Recv() (*StreamChunk, error) {
	if m.index >= len(m.chunks) {
		return nil, nil
	}
	chunk := m.chunks[m.index]
	m.index++
	return chunk, nil
}
func (m *MockStreamReader) Close() error { return nil }

// Tests

func TestPluginInfo(t *testing.T) {
	info := PluginInfo{
		ID:           "test",
		Name:         "Test Plugin",
		Version:      "1.0.0",
		Type:         PluginTypeAdapter,
		Provider:     "test",
		Capabilities: []string{"chat", "stream"},
	}

	if info.ID != "test" {
		t.Errorf("expected ID test, got %s", info.ID)
	}
	if info.Type != PluginTypeAdapter {
		t.Errorf("expected type adapter, got %s", info.Type)
	}
}

func TestPluginRegistry_Register(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	plugin := NewMockPlugin("test-plugin", "test")
	err := registry.Register(plugin)
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	// Verify registered
	p, err := registry.Get("test-plugin")
	if err != nil {
		t.Fatalf("failed to get plugin: %v", err)
	}
	if p.Info().ID != "test-plugin" {
		t.Errorf("wrong plugin ID")
	}
}

func TestPluginRegistry_RegisterDuplicate(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	plugin := NewMockPlugin("test-plugin", "test")
	registry.Register(plugin)

	// Try to register again
	err := registry.Register(plugin)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestPluginRegistry_GetByProvider(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	plugin := NewMockPlugin("test-plugin", "test-provider")
	registry.Register(plugin)

	p, err := registry.GetByProvider("test-provider")
	if err != nil {
		t.Fatalf("failed to get by provider: %v", err)
	}
	if p.Info().Provider != "test-provider" {
		t.Errorf("wrong provider")
	}
}

func TestPluginRegistry_List(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	registry.Register(NewMockPlugin("plugin1", "provider1"))
	registry.Register(NewMockPlugin("plugin2", "provider2"))

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(list))
	}
}

func TestPluginRegistry_Unregister(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	plugin := NewMockPlugin("test-plugin", "test")
	registry.Register(plugin)

	err := registry.Unregister("test-plugin")
	if err != nil {
		t.Fatalf("failed to unregister: %v", err)
	}

	// Verify removed
	_, err = registry.Get("test-plugin")
	if err == nil {
		t.Error("expected error after unregister")
	}
}

func TestPluginRegistry_SetStatus(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	plugin := NewMockPlugin("test-plugin", "test")
	registry.Register(plugin)

	err := registry.SetStatus("test-plugin", PluginStatusInactive)
	if err != nil {
		t.Fatalf("failed to set status: %v", err)
	}

	status, err := registry.GetStatus("test-plugin")
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if status != PluginStatusInactive {
		t.Errorf("expected inactive status")
	}
}

func TestPluginRegistry_SetConfig(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	plugin := NewMockPlugin("test-plugin", "test")
	registry.Register(plugin)

	config := map[string]interface{}{
		"api_key": "test-key",
		"timeout": 60,
	}

	err := registry.SetConfig("test-plugin", config)
	if err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	// Get config
	cfg, err := registry.GetConfig("test-plugin")
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}
	if cfg["api_key"] != "test-key" {
		t.Errorf("wrong api_key in config")
	}
}

func TestPluginRegistry_ChatCompletion(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	plugin := NewMockPlugin("test-plugin", "test-provider")
	registry.Register(plugin)

	req := &ChatRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "Hello"}},
	}

	resp, err := registry.ChatCompletion(context.Background(), "test-provider", req)
	if err != nil {
		t.Fatalf("chat completion failed: %v", err)
	}

	if resp.Model != "test-model" {
		t.Errorf("wrong model in response")
	}
}

func TestPluginRegistry_Stats(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	plugin := NewMockPlugin("test-plugin", "test")
	registry.Register(plugin)

	// Record some requests
	registry.RecordRequest("test-plugin")
	registry.RecordSuccess("test-plugin", 100)
	registry.RecordError("test-plugin", "test error")

	stats, err := registry.GetStats("test-plugin")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.RequestCount != 1 {
		t.Errorf("expected 1 request, got %d", stats.RequestCount)
	}
	if stats.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", stats.SuccessCount)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", stats.ErrorCount)
	}
}

func TestPluginRegistry_HasProvider(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	if registry.HasProvider("test") {
		t.Error("should not have provider before registration")
	}

	registry.Register(NewMockPlugin("test-plugin", "test"))

	if !registry.HasProvider("test") {
		t.Error("should have provider after registration")
	}
}

func TestPluginRegistry_GetProviders(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	registry := NewPluginRegistry(nil, validator)

	registry.Register(NewMockPlugin("plugin1", "provider1"))
	registry.Register(NewMockPlugin("plugin2", "provider2"))

	providers := registry.GetProviders()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}
}

func TestDefaultValidator_Validate(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")

	// Valid plugin info
	info := PluginInfo{
		ID:           "test-plugin",
		Name:         "Test Plugin",
		Version:      "1.0.0",
		Type:         PluginTypeAdapter,
		Provider:     "test",
		Capabilities: []string{"chat"},
	}

	err := validator.Validate(info)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Invalid - missing ID
	infoNoID := PluginInfo{Name: "Test", Version: "1.0.0"}
	err = validator.Validate(infoNoID)
	if err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestDefaultValidator_Blocklist(t *testing.T) {
	validator := NewDefaultValidator("1.0.0")
	validator.SetBlocklist([]string{"blocked-plugin"})

	info := PluginInfo{
		ID:       "blocked-plugin",
		Name:     "Blocked",
		Version:  "1.0.0",
		Type:     PluginTypeAdapter,
		Provider: "blocked",
	}

	err := validator.Validate(info)
	if err == nil {
		t.Error("expected error for blocklisted plugin")
	}
}

func TestConfigBuilder(t *testing.T) {
	config := NewConfigBuilder().
		SetAPIKey("test-key").
		SetBaseURL("https://api.test.com").
		SetTimeout(60).
		Build()

	if config["api_key"] != "test-key" {
		t.Errorf("wrong api_key")
	}
	if config["base_url"] != "https://api.test.com" {
		t.Errorf("wrong base_url")
	}
	if config["timeout"] != 60 {
		t.Errorf("wrong timeout")
	}
}

func TestConfigSchemaBuilder(t *testing.T) {
	schema := NewConfigSchemaBuilder().
		AddString("api_key", true, "API key").
		AddNumber("timeout", false, "Timeout", 120).
		AddBoolean("debug", false, "Enable debug", false).
		Build()

	props := schema["properties"].(map[string]interface{})
	if props["api_key"] == nil {
		t.Error("api_key property missing")
	}

	required := schema["required"].([]string)
	if len(required) != 1 || required[0] != "api_key" {
		t.Errorf("wrong required fields")
	}
}

func TestMergeConfigs(t *testing.T) {
	base := map[string]interface{}{
		"api_key": "key1",
		"timeout": 30,
	}

	override := map[string]interface{}{
		"timeout": 60,
		"debug":   true,
	}

	merged := MergeConfigs(base, override)

	if merged["api_key"] != "key1" {
		t.Error("api_key should be preserved")
	}
	if merged["timeout"] != 60 {
		t.Error("timeout should be overridden")
	}
	if merged["debug"] != true {
		t.Error("debug should be added")
	}
}

func TestMaskSensitiveConfig(t *testing.T) {
	config := map[string]interface{}{
		"api_key":    "sk-1234567890abcdef",
		"timeout":    60,
		"secret_key": "mysecret",
	}

	sensitive := []string{"api_key"}
	masked := MaskSensitiveConfig(config, sensitive)

	// api_key should be masked
	if masked["api_key"] == "sk-1234567890abcdef" {
		t.Error("api_key should be masked")
	}

	// timeout should not be masked
	if masked["timeout"] != 60 {
		t.Error("timeout should not be masked")
	}

	// secret_key should be auto-detected as sensitive
	if masked["secret_key"] == "mysecret" {
		t.Error("secret_key should be auto-masked")
	}
}

func TestMarketplace_New(t *testing.T) {
	m := NewMarketplace("")
	if m == nil {
		t.Error("failed to create marketplace")
	}
}

func TestMarketplace_AddAvailable(t *testing.T) {
	m := NewMarketplace("")

	plugin := AvailablePlugin{
		ID:       "test-plugin",
		Name:     "Test Plugin",
		Version:  "1.0.0",
		Type:     PluginTypeAdapter,
		Provider: "test",
	}

	err := m.AddAvailable(plugin)
	if err != nil {
		t.Fatalf("failed to add plugin: %v", err)
	}

	available := m.ListAvailable()
	if len(available) != 1 {
		t.Errorf("expected 1 available plugin")
	}
}

func TestMarketplace_Install(t *testing.T) {
	m := NewMarketplace("")
	m.AddAvailable(AvailablePlugin{
		ID:       "test-plugin",
		Name:     "Test",
		Version:  "1.0.0",
		Type:     PluginTypeAdapter,
		Provider: "test",
	})

	config := map[string]interface{}{"api_key": "test-key"}
	installed, err := m.Install("test-plugin", config)
	if err != nil {
		t.Fatalf("failed to install: %v", err)
	}

	if installed.ID != "test-plugin" {
		t.Errorf("wrong installed plugin ID")
	}

	// Check is installed
	if !m.IsInstalled("test-plugin") {
		t.Error("plugin should be marked as installed")
	}
}

func TestMarketplace_Search(t *testing.T) {
	m := NewMarketplace("")
	m.AddAvailable(AvailablePlugin{
		ID:           "openai",
		Name:         "OpenAI",
		Provider:     "openai",
		Capabilities: []string{"chat", "embedding"},
	})
	m.AddAvailable(AvailablePlugin{
		ID:           "anthropic",
		Name:         "Claude",
		Provider:     "anthropic",
		Capabilities: []string{"chat"},
	})

	// Search by name
	results := m.Search("OpenAI")
	if len(results) != 1 {
		t.Errorf("expected 1 result for OpenAI search")
	}

	// Search by capability
	results = m.Search("embedding")
	if len(results) != 1 {
		t.Errorf("expected 1 result for embedding search")
	}
}

func TestMarketplace_ByProvider(t *testing.T) {
	m := NewMarketplace("")
	m.AddAvailable(AvailablePlugin{ID: "p1", Provider: "openai"})
	m.AddAvailable(AvailablePlugin{ID: "p2", Provider: "anthropic"})
	m.AddAvailable(AvailablePlugin{ID: "p3", Provider: "openai"})

	results := m.ByProvider("openai")
	if len(results) != 2 {
		t.Errorf("expected 2 openai plugins")
	}
}

func TestPluginError(t *testing.T) {
	err := &PluginError{
		Type:      ErrorTypeRateLimit,
		Message:   "rate limit exceeded",
		Retryable: true,
		WaitTime:  60,
	}

	if err.Error() != "rate limit exceeded" {
		t.Errorf("wrong error message")
	}
	if !err.Retryable {
		t.Error("should be retryable")
	}
}

func TestIsValidID(t *testing.T) {
	if !isValidID("test-plugin") {
		t.Error("test-plugin should be valid")
	}
	if !isValidID("Test_Plugin_123") {
		t.Error("Test_Plugin_123 should be valid")
	}
	if isValidID("test plugin!") {
		t.Error("test plugin! should be invalid")
	}
}