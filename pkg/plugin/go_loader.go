package plugin

import (
	"fmt"
	"plugin" // Go standard library plugin package
	"sync"
)

// GoPluginLoader loads compiled Go .so plugins
type GoPluginLoader struct {
	pluginDir string
	loaded    map[string]*plugin.Plugin // plugin_id -> loaded .so
	symbols   map[string]interface{}    // plugin_id -> symbol (ProviderPlugin)
	mu        sync.RWMutex
}

// NewGoPluginLoader creates a new Go plugin loader
func NewGoPluginLoader(pluginDir string) *GoPluginLoader {
	return &GoPluginLoader{
		pluginDir: pluginDir,
		loaded:    make(map[string]*plugin.Plugin),
		symbols:   make(map[string]interface{}),
	}
}

// Load loads a Go plugin from .so file
func (l *GoPluginLoader) Load(soPath string) (ProviderPlugin, PluginInfo, error) {
	// Open the .so file
	p, err := plugin.Open(soPath)
	if err != nil {
		return nil, PluginInfo{}, fmt.Errorf("failed to open plugin %s: %w", soPath, err)
	}

	// Look for "Plugin" symbol (the ProviderPlugin implementation)
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, PluginInfo{}, fmt.Errorf("plugin %s missing 'Plugin' symbol: %w", soPath, err)
	}

	// Cast to ProviderPlugin interface
	providerPlugin, ok := sym.(ProviderPlugin)
	if !ok {
		return nil, PluginInfo{}, fmt.Errorf("plugin %s 'Plugin' symbol does not implement ProviderPlugin", soPath)
	}

	// Get plugin info
	info := providerPlugin.Info()

	// Store loaded plugin
	l.mu.Lock()
	l.loaded[info.ID] = p
	l.symbols[info.ID] = providerPlugin
	l.mu.Unlock()

	return providerPlugin, info, nil
}

// Unload unloads a Go plugin (note: Go plugins cannot be truly unloaded)
func (l *GoPluginLoader) Unload(pluginID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.loaded[pluginID]; !exists {
		return fmt.Errorf("plugin %s not loaded", pluginID)
	}

	// Note: Go plugins cannot be unloaded once loaded
	// We just remove from our tracking
	delete(l.loaded, pluginID)
	delete(l.symbols, pluginID)

	return nil
}

// GetLoaded returns a loaded plugin by ID
func (l *GoPluginLoader) GetLoaded(pluginID string) (ProviderPlugin, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	sym, exists := l.symbols[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s not loaded", pluginID)
	}

	providerPlugin, ok := sym.(ProviderPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin %s symbol invalid", pluginID)
	}

	return providerPlugin, nil
}

// ListLoaded returns list of loaded plugin IDs
func (l *GoPluginLoader) ListLoaded() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	ids := make([]string, 0, len(l.loaded))
	for id := range l.loaded {
		ids = append(ids, id)
	}
	return ids
}

// IsLoaded checks if a plugin is loaded
func (l *GoPluginLoader) IsLoaded(pluginID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loaded[pluginID] != nil
}

// PluginBuilder helps build Go plugins that implement ProviderPlugin
type PluginBuilder struct {
	info   PluginInfo
	factory PluginFactory
}

// NewPluginBuilder creates a new plugin builder
func NewPluginBuilder(id, name, provider string) *PluginBuilder {
	return &PluginBuilder{
		info: PluginInfo{
			ID:       id,
			Name:     name,
			Type:     PluginTypeGo,
			Provider: provider,
			Version:  "1.0.0",
			Capabilities: []string{"chat", "stream", "embedding"},
		},
	}
}

// SetVersion sets the plugin version
func (b *PluginBuilder) SetVersion(version string) *PluginBuilder {
	b.info.Version = version
	return b
}

// SetDescription sets the description
func (b *PluginBuilder) SetDescription(desc string) *PluginBuilder {
	b.info.Description = desc
	return b
}

// SetAuthor sets the author
func (b *PluginBuilder) SetAuthor(author string) *PluginBuilder {
	b.info.Author = author
	return b
}

// SetCapabilities sets the capabilities
func (b *PluginBuilder) SetCapabilities(caps []string) *PluginBuilder {
	b.info.Capabilities = caps
	return b
}

// SetFactory sets the plugin factory
func (b *PluginBuilder) SetFactory(factory PluginFactory) *PluginBuilder {
	b.factory = factory
	return b
}

// SetConfigSchema sets the configuration schema
func (b *PluginBuilder) SetConfigSchema(schema map[string]interface{}) *PluginBuilder {
	b.info.ConfigSchema = schema
	return b
}

// Build returns the plugin info
func (b *PluginBuilder) BuildInfo() PluginInfo {
	return b.info
}

// Note: For Go plugins to be loaded, they must be compiled with:
// go build -buildmode=plugin -o plugin.so plugin.go
//
// Example plugin.go:
//
// package main
//
// import "github.com/zhaojiewen/open-station/pkg/plugin"
//
// type MyProviderPlugin struct {
//     info plugin.PluginInfo
//     config map[string]interface{}
// }
//
// func New() plugin.ProviderPlugin {
//     return &MyProviderPlugin{
//         info: plugin.PluginInfo{
//             ID: "myprovider",
//             Name: "My Provider",
//             Provider: "myprovider",
//             ...
//         },
//     }
// }
//
// // Exported symbol
// var Plugin = New()
//
// func (p *MyProviderPlugin) Info() plugin.PluginInfo { return p.info }
// func (p *MyProviderPlugin) Initialize(config map[string]interface{}) error { p.config = config; return nil }
// func (p *MyProviderPlugin) Shutdown() error { return nil }
// func (p *MyProviderPlugin) ChatCompletion(ctx, req) (*plugin.ChatResponse, error) { ... }
// ... implement all methods