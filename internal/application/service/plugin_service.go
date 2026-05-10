package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"github.com/zhaojiewen/open-station/pkg/plugin"
	"go.uber.org/zap"
)

// PluginService manages plugin lifecycle and operations
type PluginService struct {
	repo       repository.PluginRepository
	registry   *plugin.PluginRegistry
	marketplace *plugin.Marketplace
}

// NewPluginService creates a new plugin service
func NewPluginService(repo repository.PluginRepository, registry *plugin.PluginRegistry, marketplace *plugin.Marketplace) *PluginService {
	return &PluginService{
		repo:       repo,
		registry:   registry,
		marketplace: marketplace,
	}
}

// List returns all installed plugins
func (s *PluginService) List(ctx context.Context, page, pageSize int) ([]entity.PluginStatus, int64, error) {
	plugins, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list plugins: %w", err)
	}

	statuses := make([]entity.PluginStatus, len(plugins))
	for i, p := range plugins {
		statuses[i] = p.ToPluginStatus()
	}

	return statuses, total, nil
}

// ListAvailable returns available plugins from marketplace
func (s *PluginService) ListAvailable(ctx context.Context) ([]plugin.AvailablePlugin, error) {
	return s.marketplace.ListAvailable(), nil
}

// Get returns a plugin by ID
func (s *PluginService) Get(ctx context.Context, pluginID string) (*entity.PluginStatus, error) {
	return s.repo.GetStats(ctx, pluginID)
}

// Install installs a plugin from marketplace
func (s *PluginService) Install(ctx context.Context, req *entity.PluginInstallRequest) (*entity.PluginStatus, error) {
	// Get available plugin from marketplace
	available, err := s.marketplace.GetAvailable(req.PluginID)
	if err != nil {
		return nil, fmt.Errorf("plugin not found in marketplace: %w", err)
	}

	// Check if already installed
	if s.marketplace.IsInstalled(req.PluginID) {
		return nil, fmt.Errorf("plugin %s is already installed", req.PluginID)
	}

	// Install from marketplace
	_, err = s.marketplace.Install(req.PluginID, req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to install from marketplace: %w", err)
	}

	// Create database record
	pluginEntity := &entity.Plugin{
		ID:            uuid.New(),
		PluginID:      available.ID,
		Name:          available.Name,
		Version:       available.Version,
		Type:          string(available.Type),
		Provider:      available.Provider,
		Status:        "inactive",
		Description:   available.Description,
		Author:        available.Author,
		AdapterURL:    available.AdapterURL,
		Capabilities:  mustMarshalJSON(available.Capabilities),
		ConfigSchema:  mustMarshalJSON(available.ConfigSchema),
		InstalledFrom: req.Source,
		InstallDate:   &time.Time{},
	}

	if err := s.repo.Create(ctx, pluginEntity); err != nil {
		return nil, fmt.Errorf("failed to create plugin record: %w", err)
	}

	// Load and register with registry
	if err := s.loadAndRegister(ctx, pluginEntity, req.Config); err != nil {
		logger.Warn("failed to load plugin", zap.Error(err))
	}

	// Activate if requested
	if req.Activate {
		if err := s.Activate(ctx, req.PluginID); err != nil {
			logger.Warn("failed to activate plugin after install", zap.Error(err))
		}
	}

	status, _ := s.repo.GetStats(ctx, req.PluginID)
	return status, nil
}

// Configure updates plugin configuration
func (s *PluginService) Configure(ctx context.Context, pluginID string, config map[string]interface{}) error {
	// Update marketplace config
	if err := s.marketplace.Configure(pluginID, config); err != nil {
		return fmt.Errorf("failed to update marketplace config: %w", err)
	}

	// Update database config
	configJSON := mustMarshalJSON(config)
	if err := s.repo.SetConfig(ctx, pluginID, configJSON); err != nil {
		return fmt.Errorf("failed to update database config: %w", err)
	}

	// Reinitialize plugin in registry
	if s.registry.HasProvider(pluginID) {
		p, err := s.registry.Get(pluginID)
		if err == nil {
			p.Initialize(config)
		}
	}

	logger.Info("plugin configured", zap.String("plugin_id", pluginID))

	return nil
}

// Activate activates a plugin
func (s *PluginService) Activate(ctx context.Context, pluginID string) error {
	// Check if installed
	installed, err := s.marketplace.GetInstalled(pluginID)
	if err != nil {
		return fmt.Errorf("plugin not installed: %w", err)
	}

	// Get config
	config := installed.Config

	// Load if not already in registry
	if !s.registry.HasProvider(installed.Provider) {
		pluginEntity, err := s.repo.GetByPluginID(ctx, pluginID)
		if err != nil {
			return fmt.Errorf("failed to get plugin entity: %w", err)
		}

		if err := s.loadAndRegister(ctx, pluginEntity, config); err != nil {
			return fmt.Errorf("failed to load plugin: %w", err)
		}
	}

	// Update status
	if err := s.repo.SetStatus(ctx, pluginID, "active"); err != nil {
		return fmt.Errorf("failed to set status: %w", err)
	}

	if err := s.marketplace.SetStatus(pluginID, plugin.PluginStatusActive); err != nil {
		logger.Warn("failed to update marketplace status", zap.Error(err))
	}

	logger.Info("plugin activated", zap.String("plugin_id", pluginID))

	return nil
}

// Deactivate deactivates a plugin
func (s *PluginService) Deactivate(ctx context.Context, pluginID string) error {
	// Unregister from registry
	installed, _ := s.marketplace.GetInstalled(pluginID)
	if installed != nil && s.registry.HasProvider(installed.Provider) {
		s.registry.Unregister(pluginID)
	}

	// Update status
	if err := s.repo.SetStatus(ctx, pluginID, "inactive"); err != nil {
		return fmt.Errorf("failed to set status: %w", err)
	}

	if err := s.marketplace.SetStatus(pluginID, plugin.PluginStatusInactive); err != nil {
		logger.Warn("failed to update marketplace status", zap.Error(err))
	}

	logger.Info("plugin deactivated", zap.String("plugin_id", pluginID))

	return nil
}

// Uninstall removes a plugin
func (s *PluginService) Uninstall(ctx context.Context, pluginID string) error {
	// Deactivate first
	s.Deactivate(ctx, pluginID)

	// Remove from marketplace
	if err := s.marketplace.Uninstall(pluginID); err != nil {
		return fmt.Errorf("failed to uninstall from marketplace: %w", err)
	}

	// Remove from database
	pluginEntity, err := s.repo.GetByPluginID(ctx, pluginID)
	if err == nil {
		s.repo.Delete(ctx, pluginEntity.ID)
	}

	logger.Info("plugin uninstalled", zap.String("plugin_id", pluginID))

	return nil
}

// HealthCheck checks plugin health
func (s *PluginService) HealthCheck(ctx context.Context, pluginID string) error {
	p, err := s.registry.Get(pluginID)
	if err != nil {
		return fmt.Errorf("plugin not in registry: %w", err)
	}

	err = p.HealthCheck(ctx)
	if err != nil {
		s.repo.RecordError(ctx, pluginID, err.Error())
		s.repo.SetStatus(ctx, pluginID, "error")
		return err
	}

	s.repo.SetStatus(ctx, pluginID, "active")
	s.repo.UpdateHealthScore(ctx, pluginID, 100)

	return nil
}

// GetStats returns plugin statistics
func (s *PluginService) GetStats(ctx context.Context, pluginID string) (*entity.PluginStatus, error) {
	return s.repo.GetStats(ctx, pluginID)
}

// GetAllStats returns all plugin statistics
func (s *PluginService) GetAllStats(ctx context.Context) ([]entity.PluginStatus, error) {
	return s.repo.GetAllStats(ctx)
}

// GetProviders returns list of available providers
func (s *PluginService) GetProviders(ctx context.Context) ([]string, error) {
	return s.repo.GetProviders(ctx)
}

// Search searches for plugins in marketplace
func (s *PluginService) Search(ctx context.Context, query string) ([]plugin.AvailablePlugin, error) {
	return s.marketplace.Search(query), nil
}

// ByCapability returns plugins with a specific capability
func (s *PluginService) ByCapability(ctx context.Context, capability string) ([]plugin.AvailablePlugin, error) {
	return s.marketplace.ByCapability(capability), nil
}

// loadAndRegister loads a plugin and registers it with the registry
func (s *PluginService) loadAndRegister(ctx context.Context, pluginEntity *entity.Plugin, config map[string]interface{}) error {
	var providerPlugin plugin.ProviderPlugin

	// Load based on type
	if pluginEntity.Type == "adapter" && pluginEntity.AdapterURL != "" {
		// Load as adapter
		loader := plugin.NewAdapterLoader()
		pp, _, err := loader.Load(pluginEntity.AdapterURL)
		if err != nil {
			return fmt.Errorf("failed to load adapter: %w", err)
		}
		providerPlugin = pp
	} else {
		// Would need to load Go plugin from .so file
		// For now, return error for unsupported type
		return fmt.Errorf("unsupported plugin type: %s", pluginEntity.Type)
	}

	// Initialize with config
	if err := providerPlugin.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// Register with registry
	if err := s.registry.Register(providerPlugin); err != nil {
		return fmt.Errorf("failed to register plugin: %w", err)
	}

	return nil
}

// SyncInstalledPlugins synchronizes installed plugins between marketplace and database
func (s *PluginService) SyncInstalledPlugins(ctx context.Context) error {
	installed := s.marketplace.ListInstalled()

	for _, p := range installed {
		// Check if exists in database
		if !s.repo.Exists(ctx, p.ID) {
			// Create new record
			pluginEntity := &entity.Plugin{
				ID:           uuid.New(),
				PluginID:     p.ID,
				Name:         p.Name,
				Version:      p.Version,
				Type:         string(p.Type),
				Provider:     p.Provider,
				Status:       string(p.Status),
				AdapterURL:   p.AdapterURL,
				InstalledFrom: p.InstallFrom,
			}

			if err := s.repo.Create(ctx, pluginEntity); err != nil {
				logger.Warn("failed to create plugin record", zap.String("plugin_id", p.ID), zap.Error(err))
			}
		}
	}

	return nil
}

// GetRegistry returns the plugin registry
func (s *PluginService) GetRegistry() *plugin.PluginRegistry {
	return s.registry
}

// GetMarketplace returns the marketplace
func (s *PluginService) GetMarketplace() *plugin.Marketplace {
	return s.marketplace
}

// mustMarshalJSON marshals to JSON, returning empty string on error
func mustMarshalJSON(v interface{}) string {
	if v == nil {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}