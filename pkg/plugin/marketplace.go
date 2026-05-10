package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// Marketplace handles plugin discovery from local configuration
type Marketplace struct {
	availablePlugins map[string]AvailablePlugin
	installedPlugins map[string]InstalledPlugin
	configFile       string
	mu               sync.RWMutex
}

// AvailablePlugin represents a plugin available for installation
type AvailablePlugin struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Type            PluginType             `json:"type"`
	Provider        string                 `json:"provider"`
	Description     string                 `json:"description"`
	Author          string                 `json:"author"`
	Capabilities    []string               `json:"capabilities"`
	AdapterURL      string                 `json:"adapter_url,omitempty"`
	AdapterProtocol string                 `json:"adapter_protocol,omitempty"`
	DownloadURL     string                 `json:"download_url,omitempty"`
	SHA256          string                 `json:"sha256,omitempty"`
	ConfigSchema    map[string]interface{} `json:"config_schema"`
}

// InstalledPlugin represents an installed plugin
type InstalledPlugin struct {
	AvailablePlugin
	InstallDate string                 `json:"install_date"`
	InstallFrom string                 `json:"install_from"`
	Config      map[string]interface{} `json:"config"`
	Status      PluginStatus           `json:"status"`
}

// MarketplaceConfig represents the marketplace configuration file
type MarketplaceConfig struct {
	Plugins map[string]AvailablePlugin `json:"plugins"`
}

// NewMarketplace creates a new marketplace from config file
func NewMarketplace(configFile string) *Marketplace {
	m := &Marketplace{
		availablePlugins: make(map[string]AvailablePlugin),
		installedPlugins: make(map[string]InstalledPlugin),
		configFile:       configFile,
	}

	// Load available plugins from config
	m.loadConfig()

	return m
}

// NewMarketplaceEmpty creates a new empty marketplace for programmatic registration
func NewMarketplaceEmpty() *Marketplace {
	return &Marketplace{
		availablePlugins: make(map[string]AvailablePlugin),
		installedPlugins: make(map[string]InstalledPlugin),
		configFile:       "",
	}
}

// RegisterAvailable adds a plugin to the available plugins list
func (m *Marketplace) RegisterAvailable(plugin AvailablePlugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.availablePlugins[plugin.ID] = plugin
}

// loadConfig loads the marketplace configuration
func (m *Marketplace) loadConfig() error {
	if m.configFile == "" {
		return nil
	}

	data, err := os.ReadFile(m.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("marketplace config file not found, starting empty")
			return nil
		}
		return fmt.Errorf("failed to read marketplace config: %w", err)
	}

	var config MarketplaceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse marketplace config: %w", err)
	}

	m.mu.Lock()
	m.availablePlugins = config.Plugins
	m.mu.Unlock()

	logger.Info("marketplace loaded", zap.Int("plugins", len(m.availablePlugins)))

	return nil
}

// ListAvailable returns all available plugins
func (m *Marketplace) ListAvailable() []AvailablePlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]AvailablePlugin, 0, len(m.availablePlugins))
	for _, plugin := range m.availablePlugins {
		list = append(list, plugin)
	}

	return list
}

// ListInstalled returns all installed plugins
func (m *Marketplace) ListInstalled() []InstalledPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]InstalledPlugin, 0, len(m.installedPlugins))
	for _, plugin := range m.installedPlugins {
		list = append(list, plugin)
	}

	return list
}

// GetAvailable returns an available plugin by ID
func (m *Marketplace) GetAvailable(pluginID string) (*AvailablePlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.availablePlugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found in marketplace", pluginID)
	}

	return &plugin, nil
}

// GetInstalled returns an installed plugin by ID
func (m *Marketplace) GetInstalled(pluginID string) (*InstalledPlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.installedPlugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s not installed", pluginID)
	}

	return &plugin, nil
}

// IsInstalled checks if a plugin is installed
func (m *Marketplace) IsInstalled(pluginID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.installedPlugins[pluginID].ID != ""
}

// Install installs a plugin from marketplace
func (m *Marketplace) Install(pluginID string, config map[string]interface{}) (*InstalledPlugin, error) {
	// Get available plugin
	available, err := m.GetAvailable(pluginID)
	if err != nil {
		return nil, err
	}

	// Validate config against schema
	if available.ConfigSchema != nil {
		schema, err := parseConfigSchema(available.ConfigSchema)
		if err == nil {
			if err := ValidateConfig(config, schema); err != nil {
				return nil, fmt.Errorf("config validation failed: %w", err)
			}
		}
	}

	// Create installed plugin
	installed := InstalledPlugin{
		AvailablePlugin: *available,
		InstallDate:     getCurrentDate(),
		InstallFrom:     "marketplace",
		Config:          config,
		Status:          PluginStatusInactive,
	}

	m.mu.Lock()
	m.installedPlugins[pluginID] = installed
	m.mu.Unlock()

	logger.Info("plugin installed from marketplace",
		zap.String("plugin_id", pluginID),
		zap.String("version", available.Version))

	return &installed, nil
}

// Uninstall removes an installed plugin
func (m *Marketplace) Uninstall(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.installedPlugins[pluginID]; !exists {
		return fmt.Errorf("plugin %s not installed", pluginID)
	}

	delete(m.installedPlugins, pluginID)

	logger.Info("plugin uninstalled", zap.String("plugin_id", pluginID))

	return nil
}

// Configure updates configuration for an installed plugin
func (m *Marketplace) Configure(pluginID string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	installed, exists := m.installedPlugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s not installed", pluginID)
	}

	// Merge with existing config
	installed.Config = MergeConfigs(installed.Config, config)
	m.installedPlugins[pluginID] = installed

	logger.Info("plugin configured", zap.String("plugin_id", pluginID))

	return nil
}

// SetStatus sets the status of an installed plugin
func (m *Marketplace) SetStatus(pluginID string, status PluginStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	installed, exists := m.installedPlugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s not installed", pluginID)
	}

	installed.Status = status
	m.installedPlugins[pluginID] = installed

	logger.Info("plugin status changed",
		zap.String("plugin_id", pluginID),
		zap.String("status", string(status)))

	return nil
}

// Search searches for plugins by name, provider, or capability
func (m *Marketplace) Search(query string) []AvailablePlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queryLower := strings.ToLower(query)
	results := make([]AvailablePlugin, 0)

	for _, plugin := range m.availablePlugins {
		// Search in name, provider, description
		if strings.Contains(strings.ToLower(plugin.Name), queryLower) ||
			strings.Contains(strings.ToLower(plugin.Provider), queryLower) ||
			strings.Contains(strings.ToLower(plugin.Description), queryLower) {
			results = append(results, plugin)
			continue
		}

		// Search in capabilities
		for _, cap := range plugin.Capabilities {
			if strings.Contains(strings.ToLower(cap), queryLower) {
				results = append(results, plugin)
				break
			}
		}
	}

	return results
}

// ByCapability returns plugins with a specific capability
func (m *Marketplace) ByCapability(capability string) []AvailablePlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]AvailablePlugin, 0)

	for _, plugin := range m.availablePlugins {
		for _, cap := range plugin.Capabilities {
			if cap == capability {
				results = append(results, plugin)
				break
			}
		}
	}

	return results
}

// ByProvider returns plugins for a specific provider
func (m *Marketplace) ByProvider(provider string) []AvailablePlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]AvailablePlugin, 0)

	for _, plugin := range m.availablePlugins {
		if plugin.Provider == provider {
			results = append(results, plugin)
		}
	}

	return results
}

// AddAvailable adds a new plugin to available list (for admin use)
func (m *Marketplace) AddAvailable(plugin AvailablePlugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if plugin.ID == "" {
		return fmt.Errorf("plugin id is required")
	}

	m.availablePlugins[plugin.ID] = plugin

	logger.Info("plugin added to marketplace", zap.String("plugin_id", plugin.ID))

	return nil
}

// RemoveAvailable removes a plugin from available list
func (m *Marketplace) RemoveAvailable(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.availablePlugins[pluginID]; !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	delete(m.availablePlugins, pluginID)

	logger.Info("plugin removed from marketplace", zap.String("plugin_id", pluginID))

	return nil
}

// SaveInstalled saves installed plugins state to file
func (m *Marketplace) SaveInstalled(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.installedPlugins, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadInstalled loads installed plugins state from file
func (m *Marketplace) LoadInstalled(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No installed plugins file
		}
		return err
	}

	var installed map[string]InstalledPlugin
	if err := json.Unmarshal(data, &installed); err != nil {
		return err
	}

	m.mu.Lock()
	m.installedPlugins = installed
	m.mu.Unlock()

	return nil
}

// GetPluginInfo converts AvailablePlugin to PluginInfo
func (p *AvailablePlugin) ToPluginInfo() PluginInfo {
	return PluginInfo{
		ID:              p.ID,
		Name:            p.Name,
		Version:         p.Version,
		Type:            p.Type,
		Provider:        p.Provider,
		Description:     p.Description,
		Author:          p.Author,
		Capabilities:    p.Capabilities,
		AdapterURL:      p.AdapterURL,
		AdapterProtocol: p.AdapterProtocol,
		ConfigSchema:    p.ConfigSchema,
	}
}

// parseConfigSchema converts schema map to PluginConfigSchema
func parseConfigSchema(schemaMap map[string]interface{}) (PluginConfigSchema, error) {
	var schema PluginConfigSchema
	data, err := json.Marshal(schemaMap)
	if err != nil {
		return schema, err
	}
	err = json.Unmarshal(data, &schema)
	return schema, err
}

// getCurrentDate returns current date string
func getCurrentDate() string {
	return "2024-01-01" // Placeholder - use time.Now().Format("2006-01-02")
}

// NewMarketplaceFromYAML creates marketplace from YAML-style config
func NewMarketplaceFromYAML(pluginsConfig map[string]interface{}) *Marketplace {
	m := &Marketplace{
		availablePlugins: make(map[string]AvailablePlugin),
		installedPlugins: make(map[string]InstalledPlugin),
	}

	// Parse available_plugins from config
	if available, ok := pluginsConfig["available_plugins"].(map[string]interface{}); ok {
		for id, pluginData := range available {
			if dataMap, ok := pluginData.(map[string]interface{}); ok {
				plugin := parseAvailablePlugin(id, dataMap)
				m.availablePlugins[id] = plugin
			}
		}
	}

	return m
}

// parseAvailablePlugin parses plugin data from config
func parseAvailablePlugin(id string, data map[string]interface{}) AvailablePlugin {
	plugin := AvailablePlugin{ID: id}

	if name, ok := data["name"].(string); ok {
		plugin.Name = name
	}
	if version, ok := data["version"].(string); ok {
		plugin.Version = version
	}
	if typ, ok := data["type"].(string); ok {
		plugin.Type = PluginType(typ)
	}
	if provider, ok := data["provider"].(string); ok {
		plugin.Provider = provider
	} else {
		plugin.Provider = id // Default to ID
	}
	if desc, ok := data["description"].(string); ok {
		plugin.Description = desc
	}
	if author, ok := data["author"].(string); ok {
		plugin.Author = author
	}
	if caps, ok := data["capabilities"].([]interface{}); ok {
		plugin.Capabilities = make([]string, 0)
		for _, c := range caps {
			if capStr, ok := c.(string); ok {
				plugin.Capabilities = append(plugin.Capabilities, capStr)
			}
		}
	}
	if url, ok := data["adapter_url"].(string); ok {
		plugin.AdapterURL = url
	}
	if protocol, ok := data["adapter_protocol"].(string); ok {
		plugin.AdapterProtocol = protocol
	}
	if schema, ok := data["config_schema"].(map[string]interface{}); ok {
		plugin.ConfigSchema = schema
	}

	return plugin
}