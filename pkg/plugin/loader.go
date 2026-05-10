package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// PluginLoader interface for loading plugins
type PluginLoader interface {
	// Load loads a plugin from source
	Load(source string, pluginType PluginType) (ProviderPlugin, PluginInfo, error)

	// Unload unloads a plugin
	Unload(pluginID string) error

	// GetConfig returns the config for a loaded plugin
	GetConfig(source string) map[string]interface{}

	// ListLoaded returns list of loaded plugin IDs
	ListLoaded() []string
}

// CompositeLoader handles both Go plugins and external adapters
type CompositeLoader struct {
	goLoader      *GoPluginLoader
	adapterLoader *AdapterLoader
	loadedPlugins map[string]string // plugin_id -> source
	mu            sync.RWMutex
	pluginDir     string
	configDir     string
}

// NewCompositeLoader creates a new composite loader
func NewCompositeLoader(pluginDir, configDir string, allowNative bool) *CompositeLoader {
	loader := &CompositeLoader{
		loadedPlugins: make(map[string]string),
		pluginDir:     pluginDir,
		configDir:     configDir,
	}

	if allowNative {
		loader.goLoader = NewGoPluginLoader(pluginDir)
	}

	loader.adapterLoader = NewAdapterLoader()

	return loader
}

// Load loads a plugin from source
func (l *CompositeLoader) Load(source string, pluginType PluginType) (ProviderPlugin, PluginInfo, error) {
	switch pluginType {
	case PluginTypeGo:
		if l.goLoader == nil {
			return nil, PluginInfo{}, fmt.Errorf("native plugins are disabled")
		}
		plugin, info, err := l.goLoader.Load(source)
		if err != nil {
			return nil, PluginInfo{}, err
		}

		l.mu.Lock()
		l.loadedPlugins[info.ID] = source
		l.mu.Unlock()

		return plugin, info, nil

	case PluginTypeAdapter:
		plugin, info, err := l.adapterLoader.Load(source)
		if err != nil {
			return nil, PluginInfo{}, err
		}

		l.mu.Lock()
		l.loadedPlugins[info.ID] = source
		l.mu.Unlock()

		return plugin, info, nil

	default:
		return nil, PluginInfo{}, fmt.Errorf("unsupported plugin type: %s", pluginType)
	}
}

// Unload unloads a plugin
func (l *CompositeLoader) Unload(pluginID string) error {
	l.mu.RLock()
	source, exists := l.loadedPlugins[pluginID]
	l.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not loaded", pluginID)
	}

	// Determine type and unload accordingly
	if l.goLoader != nil && filepath.Ext(source) == ".so" {
		if err := l.goLoader.Unload(pluginID); err != nil {
			return err
		}
	} else {
		if err := l.adapterLoader.Unload(pluginID); err != nil {
			return err
		}
	}

	l.mu.Lock()
	delete(l.loadedPlugins, pluginID)
	l.mu.Unlock()

	return nil
}

// GetConfig returns config for a plugin source
func (l *CompositeLoader) GetConfig(source string) map[string]interface{} {
	// Try to load config from config file
	if l.configDir != "" {
		configFile := filepath.Join(l.configDir, filepath.Base(source)+".json")
		if data, err := os.ReadFile(configFile); err == nil {
			config, err := ParsePluginConfig(data)
			if err == nil {
				return config
			}
		}
	}

	return make(map[string]interface{})
}

// ListLoaded returns list of loaded plugin IDs
func (l *CompositeLoader) ListLoaded() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	ids := make([]string, 0, len(l.loadedPlugins))
	for id := range l.loadedPlugins {
		ids = append(ids, id)
	}
	return ids
}

// LoadFromManifest loads plugin from a manifest file
func (l *CompositeLoader) LoadFromManifest(manifestPath string) (ProviderPlugin, PluginInfo, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, PluginInfo{}, fmt.Errorf("failed to read manifest: %w", err)
	}

	manifest, err := ParsePluginManifest(data)
	if err != nil {
		return nil, PluginInfo{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Determine plugin type and source
	if manifest.Type == PluginTypeGo && manifest.Main != "" {
		// Go plugin - source is the .so file path
		soPath := filepath.Join(l.pluginDir, manifest.ID+".so")
		return l.Load(soPath, PluginTypeGo)
	}

	// Adapter plugin - source is the adapter URL
	if manifest.AdapterURL != "" {
		return l.Load(manifest.AdapterURL, PluginTypeAdapter)
	}

	return nil, PluginInfo{}, fmt.Errorf("cannot determine plugin source from manifest")
}

// ScanPlugins scans the plugin directory for available plugins
func (l *CompositeLoader) ScanPlugins() ([]PluginManifest, error) {
	if l.pluginDir == "" {
		return nil, fmt.Errorf("plugin directory not configured")
	}

	manifests := make([]PluginManifest, 0)

	// Scan for manifest files
	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return manifests, nil // Directory doesn't exist, return empty
		}
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check for manifest.json in subdirectory
			manifestPath := filepath.Join(l.pluginDir, entry.Name(), "manifest.json")
			if data, err := os.ReadFile(manifestPath); err == nil {
				manifest, err := ParsePluginManifest(data)
				if err == nil {
					manifests = append(manifests, manifest)
				}
			}

			// Also check for .so file in subdirectory
			soPath := filepath.Join(l.pluginDir, entry.Name(), "plugin.so")
			if _, err := os.Stat(soPath); err == nil {
				// Found .so file, create a basic manifest
				manifests = append(manifests, PluginManifest{
					PluginInfo: PluginInfo{
						ID:       entry.Name(),
						Name:     entry.Name(),
						Type:     PluginTypeGo,
						Provider: entry.Name(),
					},
					Main: soPath,
				})
			}
		} else if filepath.Ext(entry.Name()) == ".so" {
			// Found .so file directly
			pluginID := filepath.Base(entry.Name()[:len(entry.Name())-3])
			manifests = append(manifests, PluginManifest{
				PluginInfo: PluginInfo{
					ID:       pluginID,
					Name:     pluginID,
					Type:     PluginTypeGo,
					Provider: pluginID,
				},
				Main: filepath.Join(l.pluginDir, entry.Name()),
			})
		}
	}

	return manifests, nil
}

// ParsePluginManifest parses a manifest JSON file
func ParsePluginManifest(data []byte) (PluginManifest, error) {
	var manifest PluginManifest
	if err := ParseJSON(data, &manifest); err != nil {
		return PluginManifest{}, err
	}

	// Validate required fields
	if manifest.ID == "" {
		return PluginManifest{}, fmt.Errorf("manifest missing id")
	}
	if manifest.Provider == "" {
		manifest.Provider = manifest.ID // Default provider to ID
	}

	return manifest, nil
}

// ParsePluginConfig parses a config JSON file
func ParsePluginConfig(data []byte) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := ParseJSON(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// ParseJSON helper for JSON parsing
func ParseJSON(data []byte, v interface{}) error {
	// Use encoding/json
	return fmt.Errorf("not implemented") // Will be implemented with encoding/json
}