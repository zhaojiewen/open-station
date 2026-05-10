package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Plugin represents a registered plugin in the system
type Plugin struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	// Plugin identification
	PluginID string `gorm:"uniqueIndex;not null" json:"plugin_id"` // e.g., "openai", "anthropic"
	Name     string `gorm:"not null" json:"name"`
	Version  string `gorm:"not null" json:"version"`

	// Plugin type and provider
	Type     string `gorm:"not null" json:"type"`        // "go" or "adapter"
	Provider string `gorm:"index;not null" json:"provider"` // Provider this plugin handles

	// Status
	Status string `gorm:"default:inactive" json:"status"` // active, inactive, error, loading

	// Metadata
	Description string `json:"description"`
	Author      string `json:"author"`
	Repository  string `json:"repository"`

	// Configuration
	Capabilities string `gorm:"type:jsonb" json:"capabilities"`    // JSON array of capabilities
	ConfigSchema string `gorm:"type:jsonb" json:"config_schema"`   // JSON object for config schema
	Config       string `gorm:"type:jsonb" json:"config"`         // Current configuration
	Dependencies string `gorm:"type:jsonb" json:"dependencies"`   // JSON array of dependencies

	// Adapter-specific fields
	AdapterURL      string `json:"adapter_url"`      // For external adapters (HTTP endpoint)
	AdapterProtocol string `json:"adapter_protocol"` // "http" or "grpc"

	// Plugin file path (for Go plugins)
	PluginPath string `json:"plugin_path"` // Path to .so file

	// Marketplace metadata
	DownloadURL   string     `json:"download_url"`
	InstalledFrom string     `json:"installed_from"` // "marketplace", "local", "url"
	InstallDate   *time.Time `json:"install_date"`
	SHA256        string     `json:"sha256"` // Checksum for validation

	// Statistics
	RequestCount int64           `gorm:"default:0" json:"request_count"`
	SuccessCount int64           `gorm:"default:0" json:"success_count"`
	ErrorCount   int64           `gorm:"default:0" json:"error_count"`
	TotalCost    decimal.Decimal `json:"total_cost"`
	LastError    *string         `json:"last_error"`
	LastErrorAt  *time.Time      `json:"last_error_at"`

	// Timestamps
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DisabledAt *time.Time `json:"disabled_at"`

	// Health and performance
	HealthScore int     `gorm:"default:100" json:"health_score"` // 0-100
	AvgLatency  float64 `gorm:"default:0" json:"avg_latency"`    // Average latency in ms
}

// TableName returns the table name for Plugin entity
func (Plugin) TableName() string {
	return "plugins"
}

// BeforeCreate sets UUID before creating
func (p *Plugin) BeforeCreate() error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// IsActive checks if plugin is active
func (p *Plugin) IsActive() bool {
	return p.Status == "active"
}

// SetStatus changes the plugin status
func (p *Plugin) SetStatus(status string) {
	p.Status = status
	p.UpdatedAt = time.Now()

	if status == "inactive" && p.DisabledAt == nil {
		p.DisabledAt = &p.UpdatedAt
	}
}

// RecordRequest increments request count
func (p *Plugin) RecordRequest() {
	p.RequestCount++
	p.UpdatedAt = time.Now()
}

// RecordSuccess increments success count
func (p *Plugin) RecordSuccess(latencyMs int64) {
	p.SuccessCount++
	p.UpdatedAt = time.Now()

	// Update average latency
	if p.RequestCount > 1 {
		p.AvgLatency = (p.AvgLatency * float64(p.RequestCount-1) + float64(latencyMs)) / float64(p.RequestCount)
	} else {
		p.AvgLatency = float64(latencyMs)
	}

	// Improve health score on success
	if p.HealthScore < 100 {
		p.HealthScore++
	}
}

// RecordError increments error count
func (p *Plugin) RecordError(errMsg string) {
	p.ErrorCount++
	p.LastError = &errMsg
	p.LastErrorAt = &p.UpdatedAt
	p.UpdatedAt = time.Now()

	// Reduce health score on error
	if p.HealthScore > 0 {
		p.HealthScore -= 5
		if p.HealthScore < 0 {
			p.HealthScore = 0
		}
	}

	// Mark as error if health score too low
	if p.HealthScore < 20 {
		p.Status = "error"
	}
}

// GetCapabilitiesList parses capabilities JSON
func (p *Plugin) GetCapabilitiesList() []string {
	// Would parse JSON from Capabilities field
	return []string{} // Placeholder
}

// HasCapability checks if plugin has a specific capability
func (p *Plugin) HasCapability(capability string) bool {
	caps := p.GetCapabilitiesList()
	for _, c := range caps {
		if c == capability {
			return true
		}
	}
	return false
}

// PluginInstallRequest represents a request to install a plugin
type PluginInstallRequest struct {
	PluginID   string                 `json:"plugin_id"`
	Source     string                 `json:"source"`     // "marketplace", "local", "url"
	URL        string                 `json:"url"`        // For URL source
	Config     map[string]interface{} `json:"config"`     // Initial configuration
	Activate   bool                   `json:"activate"`   // Activate after install
	SkipCheck  bool                   `json:"skip_check"` // Skip checksum validation
}

// PluginConfigRequest represents a configuration update request
type PluginConfigRequest struct {
	PluginID string                 `json:"plugin_id"`
	Config   map[string]interface{} `json:"config"`
	Restart  bool                   `json:"restart"` // Restart plugin after config change
}

// PluginStatus represents the full status of a plugin
type PluginStatus struct {
	PluginID     string    `json:"plugin_id"`
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Status       string    `json:"status"`
	HealthScore  int       `json:"health_score"`
	Provider     string    `json:"provider"`
	Type         string    `json:"type"`
	Capabilities []string  `json:"capabilities"`
	RequestCount int64     `json:"request_count"`
	SuccessCount int64     `json:"success_count"`
	ErrorCount   int64     `json:"error_count"`
	LastError    string    `json:"last_error"`
	LastErrorAt  *time.Time `json:"last_error_at"`
	AvgLatency   float64   `json:"avg_latency"`
	InstallDate  *time.Time `json:"install_date"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToPluginStatus converts Plugin entity to PluginStatus
func (p *Plugin) ToPluginStatus() PluginStatus {
	var lastErr string
	if p.LastError != nil {
		lastErr = *p.LastError
	}

	return PluginStatus{
		PluginID:     p.PluginID,
		Name:         p.Name,
		Version:      p.Version,
		Status:       p.Status,
		HealthScore:  p.HealthScore,
		Provider:     p.Provider,
		Type:         p.Type,
		Capabilities: p.GetCapabilitiesList(),
		RequestCount: p.RequestCount,
		SuccessCount: p.SuccessCount,
		ErrorCount:   p.ErrorCount,
		LastError:    lastErr,
		LastErrorAt:  p.LastErrorAt,
		AvgLatency:   p.AvgLatency,
		InstallDate:  p.InstallDate,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}