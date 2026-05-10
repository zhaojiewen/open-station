package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// PluginRegistry manages all registered plugins
type PluginRegistry struct {
	plugins    map[string]ProviderPlugin      // plugin_id -> plugin
	providers  map[string]ProviderPlugin      // provider_name -> plugin
	info       map[string]PluginInfo          // plugin_id -> info
	status     map[string]PluginStatus        // plugin_id -> status
	config     map[string]map[string]interface{} // plugin_id -> config
	mu         sync.RWMutex
	loader     PluginLoader
	validator  PluginValidator
	hooks      []PluginHook
	middleware []PluginMiddleware
	stats      map[string]*PluginStats
}

// PluginStats tracks runtime statistics for a plugin
type PluginStats struct {
	RequestCount   int64
	SuccessCount   int64
	ErrorCount     int64
	LastError      string
	LastErrorTime  time.Time
	LastSuccessTime time.Time
	AvgLatencyMs   float64
	HealthScore    int // 0-100
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(loader PluginLoader, validator PluginValidator) *PluginRegistry {
	return &PluginRegistry{
		plugins:   make(map[string]ProviderPlugin),
		providers: make(map[string]ProviderPlugin),
		info:      make(map[string]PluginInfo),
		status:    make(map[string]PluginStatus),
		config:    make(map[string]map[string]interface{}),
		stats:     make(map[string]*PluginStats),
		loader:    loader,
		validator: validator,
	}
}

// Register adds a plugin to the registry
func (r *PluginRegistry) Register(plugin ProviderPlugin) error {
	info := plugin.Info()

	// Validate plugin
	if r.validator != nil {
		if err := r.validator.Validate(info); err != nil {
			return fmt.Errorf("plugin validation failed: %w", err)
		}
	}

	// Check if already registered
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[info.ID]; exists {
		return fmt.Errorf("plugin %s already registered", info.ID)
	}

	// Initialize plugin if config exists
	if cfg, exists := r.config[info.ID]; exists {
		if err := plugin.Initialize(cfg); err != nil {
			return fmt.Errorf("plugin initialization failed: %w", err)
		}
	}

	// Register plugin
	r.plugins[info.ID] = plugin
	r.providers[info.Provider] = plugin
	r.info[info.ID] = info
	r.status[info.ID] = PluginStatusActive
	r.stats[info.ID] = &PluginStats{HealthScore: 100}

	// Call hooks
	for _, hook := range r.hooks {
		if err := hook.OnLoad(); err != nil {
			logger.Warn("plugin hook OnLoad failed", zap.Error(err))
		}
	}

	logger.Info("plugin registered",
		zap.String("plugin_id", info.ID),
		zap.String("provider", info.Provider),
		zap.String("version", info.Version),
		zap.String("type", string(info.Type)))

	return nil
}

// Unregister removes a plugin from the registry
func (r *PluginRegistry) Unregister(pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	// Shutdown plugin
	if err := plugin.Shutdown(); err != nil {
		logger.Warn("plugin shutdown failed", zap.Error(err))
	}

	info := r.info[pluginID]

	// Remove from registry
	delete(r.plugins, pluginID)
	delete(r.providers, info.Provider)
	delete(r.info, pluginID)
	delete(r.status, pluginID)
	delete(r.config, pluginID)
	delete(r.stats, pluginID)

	// Call hooks
	for _, hook := range r.hooks {
		hook.OnUnload()
	}

	logger.Info("plugin unregistered", zap.String("plugin_id", pluginID))

	return nil
}

// Get retrieves a plugin by ID
func (r *PluginRegistry) Get(pluginID string) (ProviderPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginID)
	}

	return plugin, nil
}

// GetByProvider retrieves plugin for a provider
func (r *PluginRegistry) GetByProvider(provider string) (ProviderPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", provider)
	}

	return plugin, nil
}

// List returns all registered plugins info
func (r *PluginRegistry) List() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PluginInfo, 0, len(r.info))
	for _, info := range r.info {
		result = append(result, info)
	}

	return result
}

// ListActive returns only active plugins info
func (r *PluginRegistry) ListActive() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PluginInfo, 0)
	for id, info := range r.info {
		if r.status[id] == PluginStatusActive {
			result = append(result, info)
		}
	}

	return result
}

// GetStatus returns the status of a plugin
func (r *PluginRegistry) GetStatus(pluginID string) (PluginStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status, exists := r.status[pluginID]
	if !exists {
		return "", fmt.Errorf("plugin %s not found", pluginID)
	}

	return status, nil
}

// SetStatus sets the status of a plugin
func (r *PluginRegistry) SetStatus(pluginID string, status PluginStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[pluginID]; !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	r.status[pluginID] = status

	logger.Info("plugin status changed",
		zap.String("plugin_id", pluginID),
		zap.String("status", string(status)))

	return nil
}

// SetConfig stores configuration for a plugin
func (r *PluginRegistry) SetConfig(pluginID string, config map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config[pluginID] = config

	// If plugin is already loaded, reinitialize
	if plugin, exists := r.plugins[pluginID]; exists {
		if err := plugin.Shutdown(); err != nil {
			logger.Warn("plugin shutdown failed during reconfigure", zap.Error(err))
		}
		if err := plugin.Initialize(config); err != nil {
			r.status[pluginID] = PluginStatusError
			return fmt.Errorf("plugin reinitialization failed: %w", err)
		}
	}

	return nil
}

// GetConfig returns configuration for a plugin
func (r *PluginRegistry) GetConfig(pluginID string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.config[pluginID]
	if !exists {
		return make(map[string]interface{}), nil
	}

	// Return a copy to prevent modification
	result := make(map[string]interface{})
	for k, v := range config {
		result[k] = v
	}
	return result, nil
}

// Load loads a plugin from source (file or URL)
func (r *PluginRegistry) Load(source string, pluginType PluginType) error {
	if r.loader == nil {
		return fmt.Errorf("plugin loader not configured")
	}

	plugin, info, err := r.loader.Load(source, pluginType)
	if err != nil {
		return fmt.Errorf("plugin load failed: %w", err)
	}

	// Store config from loaded plugin
	r.mu.Lock()
	r.config[info.ID] = r.loader.GetConfig(source)
	r.mu.Unlock()

	return r.Register(plugin)
}

// Unload unloads a plugin
func (r *PluginRegistry) Unload(pluginID string) error {
	if err := r.Unregister(pluginID); err != nil {
		return err
	}

	if r.loader != nil {
		return r.loader.Unload(pluginID)
	}

	return nil
}

// Reload reloads a plugin (for updates)
func (r *PluginRegistry) Reload(pluginID string) error {
	r.mu.RLock()
	info, exists := r.info[pluginID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	// Unload
	if err := r.Unload(pluginID); err != nil {
		return fmt.Errorf("unload failed: %w", err)
	}

	// Reload from source
	if r.loader != nil && info.AdapterURL != "" {
		// Get config before loading
		cfg, _ := r.GetConfig(pluginID)
		if err := r.Load(info.AdapterURL, info.Type); err != nil {
			return err
		}
		// Reinitialize with saved config
		plugin, err := r.Get(pluginID)
		if err == nil && cfg != nil {
			plugin.Initialize(cfg)
		}
		return nil
	}

	return fmt.Errorf("cannot reload plugin %s: no source available", pluginID)
}

// Validate validates a plugin
func (r *PluginRegistry) Validate(pluginID string) error {
	r.mu.RLock()
	info, exists := r.info[pluginID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	if r.validator != nil {
		return r.validator.Validate(info)
	}

	return nil
}

// RecordRequest records a request to a plugin
func (r *PluginRegistry) RecordRequest(pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if stats, exists := r.stats[pluginID]; exists {
		stats.RequestCount++
	}
}

// RecordSuccess records a successful request
func (r *PluginRegistry) RecordSuccess(pluginID string, latencyMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if stats, exists := r.stats[pluginID]; exists {
		stats.SuccessCount++
		stats.LastSuccessTime = time.Now()

		// Update average latency
		if stats.RequestCount > 1 {
			stats.AvgLatencyMs = (stats.AvgLatencyMs * float64(stats.RequestCount-1) + float64(latencyMs)) / float64(stats.RequestCount)
		} else {
			stats.AvgLatencyMs = float64(latencyMs)
		}

		// Update health score (increase on success)
		if stats.HealthScore < 100 {
			stats.HealthScore++
		}
	}
}

// RecordError records a failed request
func (r *PluginRegistry) RecordError(pluginID string, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if stats, exists := r.stats[pluginID]; exists {
		stats.ErrorCount++
		stats.LastError = errMsg
		stats.LastErrorTime = time.Now()

		// Decrease health score on error
		if stats.HealthScore > 0 {
			stats.HealthScore -= 5
		}

		// Check if we should mark as error status
		if stats.HealthScore < 20 {
			r.status[pluginID] = PluginStatusError
		}
	}
}

// GetStats returns statistics for a plugin
func (r *PluginRegistry) GetStats(pluginID string) (*PluginStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats, exists := r.stats[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginID)
	}

	return stats, nil
}

// GetAllStats returns statistics for all plugins
func (r *PluginRegistry) GetAllStats() map[string]*PluginStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*PluginStats)
	for id, stats := range r.stats {
		result[id] = stats
	}

	return result
}

// AddHook adds a lifecycle hook
func (r *PluginRegistry) AddHook(hook PluginHook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, hook)
}

// AddMiddleware adds a middleware
func (r *PluginRegistry) AddMiddleware(middleware PluginMiddleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware)
}

// ApplyMiddleware applies middleware to a request
func (r *PluginRegistry) ApplyMiddleware(ctx context.Context, req *ChatRequest) (*ChatRequest, error) {
	current := req
	for _, m := range r.middleware {
		next, err := m.PreRequest(ctx, current)
		if err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

// ChatCompletion sends a chat request through the registry
func (r *PluginRegistry) ChatCompletion(ctx context.Context, provider string, req *ChatRequest) (*ChatResponse, error) {
	plugin, err := r.GetByProvider(provider)
	if err != nil {
		return nil, err
	}

	// Check status
	status, err := r.GetStatus(plugin.Info().ID)
	if err != nil {
		return nil, err
	}
	if status != PluginStatusActive {
		return nil, fmt.Errorf("plugin %s is not active (status: %s)", plugin.Info().ID, status)
	}

	// Apply middleware
	req, err = r.ApplyMiddleware(ctx, req)
	if err != nil {
		return nil, err
	}

	// Record request
	r.RecordRequest(plugin.Info().ID)

	// Execute
	start := time.Now()
	resp, err := plugin.ChatCompletion(ctx, req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		r.RecordError(plugin.Info().ID, err.Error())
		return nil, err
	}

	r.RecordSuccess(plugin.Info().ID, latency)

	// Apply response middleware
	for _, m := range r.middleware {
		resp, err = m.PostResponse(ctx, resp)
		if err != nil {
			logger.Warn("middleware post response failed", zap.Error(err))
		}
	}

	return resp, nil
}

// StreamChatCompletion sends a streaming chat request
func (r *PluginRegistry) StreamChatCompletion(ctx context.Context, provider string, req *ChatRequest) (StreamReader, error) {
	plugin, err := r.GetByProvider(provider)
	if err != nil {
		return nil, err
	}

	// Check status
	status, err := r.GetStatus(plugin.Info().ID)
	if err != nil {
		return nil, err
	}
	if status != PluginStatusActive {
		return nil, fmt.Errorf("plugin %s is not active", plugin.Info().ID)
	}

	// Apply middleware
	req, err = r.ApplyMiddleware(ctx, req)
	if err != nil {
		return nil, err
	}

	// Record request
	r.RecordRequest(plugin.Info().ID)

	// Execute
	stream, err := plugin.StreamChatCompletion(ctx, req)
	if err != nil {
		r.RecordError(plugin.Info().ID, err.Error())
		return nil, err
	}

	// Wrap stream with middleware and stats
	return &middlewareStreamReader{
		stream:    stream,
		registry:  r,
		pluginID:  plugin.Info().ID,
		middleware: r.middleware,
		ctx:       ctx,
	}, nil
}

// middlewareStreamReader wraps a stream with middleware
type middlewareStreamReader struct {
	stream    StreamReader
	registry  *PluginRegistry
	pluginID  string
	middleware []PluginMiddleware
	ctx       context.Context
	done      bool
}

func (m *middlewareStreamReader) Recv() (*StreamChunk, error) {
	chunk, err := m.stream.Recv()
	if err != nil {
		m.registry.RecordError(m.pluginID, err.Error())
		return nil, err
	}

	// Apply middleware
	for _, mid := range m.middleware {
		chunk, err = mid.PostStreamChunk(m.ctx, chunk)
		if err != nil {
			logger.Warn("middleware stream chunk failed", zap.Error(err))
		}
	}

	// Record completion when done
	if chunk.Done {
		m.done = true
		m.registry.RecordSuccess(m.pluginID, 0)
	}

	return chunk, nil
}

func (m *middlewareStreamReader) Close() error {
	if !m.done {
		m.registry.RecordError(m.pluginID, "stream closed prematurely")
	}
	return m.stream.Close()
}

// HealthCheck checks health of a specific plugin
func (r *PluginRegistry) HealthCheck(ctx context.Context, pluginID string) error {
	plugin, err := r.Get(pluginID)
	if err != nil {
		return err
	}

	err = plugin.HealthCheck(ctx)
	if err != nil {
		r.RecordError(pluginID, err.Error())
		r.SetStatus(pluginID, PluginStatusError)
		return err
	}

	r.SetStatus(pluginID, PluginStatusActive)
	return nil
}

// HealthCheckAll checks health of all plugins
func (r *PluginRegistry) HealthCheckAll(ctx context.Context) map[string]error {
	r.mu.RLock()
	pluginIDs := make([]string, 0, len(r.plugins))
	for id := range r.plugins {
		pluginIDs = append(pluginIDs, id)
	}
	r.mu.RUnlock()

	results := make(map[string]error)
	for _, id := range pluginIDs {
		results[id] = r.HealthCheck(ctx, id)
	}

	return results
}

// HasProvider checks if a provider is registered
func (r *PluginRegistry) HasProvider(provider string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[provider] != nil
}

// GetProviders returns list of all registered providers
func (r *PluginRegistry) GetProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.providers))
	for provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}