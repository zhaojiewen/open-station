package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// PluginRepository defines the interface for plugin storage
type PluginRepository interface {
	// Create creates a new plugin
	Create(ctx context.Context, plugin *entity.Plugin) error

	// GetByID retrieves a plugin by its UUID
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Plugin, error)

	// GetByPluginID retrieves a plugin by its string ID (e.g., "openai")
	GetByPluginID(ctx context.Context, pluginID string) (*entity.Plugin, error)

	// GetByProvider retrieves a plugin by provider name
	GetByProvider(ctx context.Context, provider string) (*entity.Plugin, error)

	// Update updates a plugin
	Update(ctx context.Context, plugin *entity.Plugin) error

	// Delete removes a plugin
	Delete(ctx context.Context, id uuid.UUID) error

	// List returns all plugins
	List(ctx context.Context, page, pageSize int) ([]entity.Plugin, int64, error)

	// ListByStatus returns plugins with a specific status
	ListByStatus(ctx context.Context, status string) ([]entity.Plugin, error)

	// ListActive returns only active plugins
	ListActive(ctx context.Context) ([]entity.Plugin, error)

	// SetStatus updates plugin status
	SetStatus(ctx context.Context, pluginID string, status string) error

	// SetConfig updates plugin configuration
	SetConfig(ctx context.Context, pluginID string, config string) error

	// RecordRequest increments request count
	RecordRequest(ctx context.Context, pluginID string) error

	// RecordSuccess increments success count and updates latency
	RecordSuccess(ctx context.Context, pluginID string, latencyMs int64) error

	// RecordError increments error count
	RecordError(ctx context.Context, pluginID string, errMsg string) error

	// IncrementCost adds to total cost
	IncrementCost(ctx context.Context, pluginID string, cost decimal.Decimal) error

	// GetStats returns usage statistics for a plugin
	GetStats(ctx context.Context, pluginID string) (*entity.PluginStatus, error)

	// GetAllStats returns statistics for all plugins
	GetAllStats(ctx context.Context) ([]entity.PluginStatus, error)

	// HealthCheckAll returns health status for all active plugins
	HealthCheckAll(ctx context.Context) (map[string]int, error)

	// Exists checks if a plugin exists
	Exists(ctx context.Context, pluginID string) bool

	// ProviderExists checks if a provider has a plugin
	ProviderExists(ctx context.Context, provider string) bool

	// GetProviders returns list of all provider names with plugins
	GetProviders(ctx context.Context) ([]string, error)

	// ResetStats resets statistics for a plugin
	ResetStats(ctx context.Context, pluginID string) error

	// UpdateHealthScore updates health score
	UpdateHealthScore(ctx context.Context, pluginID string, score int) error
}

// PluginInstallRepository extends PluginRepository for install tracking
type PluginInstallRepository interface {
	PluginRepository

	// RecordInstall records a plugin installation
	RecordInstall(ctx context.Context, pluginID string, source string, installDate time.Time) error

	// GetInstallHistory returns installation history for a plugin
	GetInstallHistory(ctx context.Context, pluginID string) ([]InstallRecord, error)

	// ListRecentlyInstalled returns recently installed plugins
	ListRecentlyInstalled(ctx context.Context, since time.Time) ([]entity.Plugin, error)
}

// InstallRecord represents an installation event
type InstallRecord struct {
	PluginID    string    `json:"plugin_id"`
	Source      string    `json:"source"`
	InstallDate time.Time `json:"install_date"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
}