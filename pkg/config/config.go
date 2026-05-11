package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Providers    ProvidersConfig    `mapstructure:"providers"`
	Billing      BillingConfig      `mapstructure:"billing"`
	RateLimit    RateLimitConfig    `mapstructure:"rate_limit"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	Admin        AdminConfig        `mapstructure:"admin"`
	Safe         SafeConfig         `mapstructure:"safe"`
	LoadBalancer LoadBalancerConfig `mapstructure:"load_balancer"`
	Plugins      PluginsConfig      `mapstructure:"plugins"`
	Notification NotificationConfig `mapstructure:"notification"`
	Auth         AuthConfig         `mapstructure:"auth"`
}

type ServerConfig struct {
	Port          int    `mapstructure:"port"`
	Mode          string `mapstructure:"mode"`
	EncryptionKey string `mapstructure:"encryption_key"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	MaxRetries   int           `mapstructure:"max_retries"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type ProvidersConfig struct {
	// Legacy static provider configs (deprecated, use Accounts map instead)
	OpenAI   ProviderConfig `mapstructure:"openai"`
	Claude   ProviderConfig `mapstructure:"claude"`
	Gemini   ProviderConfig `mapstructure:"gemini"`
	DeepSeek ProviderConfig `mapstructure:"deepseek"`
	GLM      ProviderConfig `mapstructure:"glm"`

	// Dynamic provider accounts loaded from database
	// This allows runtime configuration without restart
	Accounts map[string]ProviderConfig `mapstructure:"accounts"`

	// Default timeout for all providers (can be overridden per account)
	DefaultTimeout time.Duration `mapstructure:"default_timeout"`
}

type ProviderConfig struct {
	BaseURL string        `mapstructure:"base_url"`
	APIKey  string        `mapstructure:"api_key"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// GetProvider retrieves provider config by name, checking both legacy and dynamic accounts
func (p *ProvidersConfig) GetProvider(name string) *ProviderConfig {
	// First check legacy static configs (backward compatibility)
	switch name {
	case "openai":
		if p.OpenAI.APIKey != "" {
			return &p.OpenAI
		}
	case "claude", "anthropic":
		if p.Claude.APIKey != "" {
			return &p.Claude
		}
	case "gemini":
		if p.Gemini.APIKey != "" {
			return &p.Gemini
		}
	case "deepseek":
		if p.DeepSeek.APIKey != "" {
			return &p.DeepSeek
		}
	case "glm":
		if p.GLM.APIKey != "" {
			return &p.GLM
		}
	}

	// Then check dynamic accounts map
	if p.Accounts != nil {
		if cfg, ok := p.Accounts[name]; ok && cfg.APIKey != "" {
			return &cfg
		}
	}

	return nil
}

// SetProvider sets a provider config in the dynamic accounts map
func (p *ProvidersConfig) SetProvider(name string, cfg ProviderConfig) {
	if p.Accounts == nil {
		p.Accounts = make(map[string]ProviderConfig)
	}
	p.Accounts[name] = cfg
}

// ListProviders returns all configured provider names
func (p *ProvidersConfig) ListProviders() []string {
	providers := make([]string, 0)

	// Check legacy static configs
	if p.OpenAI.APIKey != "" {
		providers = append(providers, "openai")
	}
	if p.Claude.APIKey != "" {
		providers = append(providers, "claude")
	}
	if p.Gemini.APIKey != "" {
		providers = append(providers, "gemini")
	}
	if p.DeepSeek.APIKey != "" {
		providers = append(providers, "deepseek")
	}
	if p.GLM.APIKey != "" {
		providers = append(providers, "glm")
	}

	// Add dynamic accounts
	if p.Accounts != nil {
		for name := range p.Accounts {
			found := false
			for _, existing := range providers {
				if existing == name {
					found = true
					break
				}
			}
			if !found {
				providers = append(providers, name)
			}
		}
	}

	return providers
}

// GetTimeout returns the timeout for a provider, using default if not set
func (p *ProvidersConfig) GetTimeout(name string) time.Duration {
	cfg := p.GetProvider(name)
	if cfg != nil && cfg.Timeout > 0 {
		return cfg.Timeout
	}
	if p.DefaultTimeout > 0 {
		return p.DefaultTimeout
	}
	return 30 * time.Second // fallback default
}

type BillingConfig struct {
	DefaultCurrency   string  `mapstructure:"default_currency"`
	MinBalanceAlert   float64 `mapstructure:"min_balance_alert"`
}

type RateLimitConfig struct {
	DefaultUserRPS    float64 `mapstructure:"default_user_rps"`
	DefaultUserBurst  int     `mapstructure:"default_user_burst"`
	DefaultTenantRPS  float64 `mapstructure:"default_tenant_rps"`
	DefaultTenantBurst int    `mapstructure:"default_tenant_burst"`
	RedisKeyPrefix    string  `mapstructure:"redis_key_prefix"`
	WindowSize        string  `mapstructure:"window_size"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type AdminConfig struct {
	DefaultTenantSlug  string `mapstructure:"default_tenant_slug"`
	SuperAdminEmail    string `mapstructure:"super_admin_email"`
	DefaultAdminUser   string `mapstructure:"default_admin_user"`
	DefaultAdminPass   string `mapstructure:"default_admin_pass"`
	InitialAPIKeyName  string `mapstructure:"initial_api_key_name"`
}

type SafeConfig struct {
	Enabled                 bool                 `mapstructure:"enabled"`
	IPRateLimit             IPRateLimitConfig    `mapstructure:"ip_rate_limit"`
	IPBlacklist             []string             `mapstructure:"ip_blacklist"`
	IPWhitelist             []string             `mapstructure:"ip_whitelist"`
	BodySizeLimitMB         int                  `mapstructure:"body_size_limit_mb"`
	FailedAuth              FailedAuthConfig     `mapstructure:"failed_auth"`
	RedisKeyPrefix          string               `mapstructure:"redis_key_prefix"`
	AllowedMethods          []string             `mapstructure:"allowed_methods"`
	MaxHeaderSizeKB         int                  `mapstructure:"max_header_size_kb"`
	BlockEmptyUserAgent     bool                 `mapstructure:"block_empty_user_agent"`
	BlockedUserAgents       []string             `mapstructure:"blocked_user_agents"`
	MaxConcurrentConns      int                  `mapstructure:"max_concurrent_conns"`
	PathTraversalCheck      bool                 `mapstructure:"path_traversal_check"`
	BurstAutoBlock          BurstAutoBlockConfig `mapstructure:"burst_auto_block"`
	EnforceContentType      bool                 `mapstructure:"enforce_content_type"`
	MaxURLLength            int                  `mapstructure:"max_url_length"`
	MaxQueryLength          int                  `mapstructure:"max_query_length"`
	BlockSuspiciousHeaders  bool                 `mapstructure:"block_suspicious_headers"`
	MaxSingleHeaderKB       int                  `mapstructure:"max_single_header_kb"`
	RateViolationBlock      RateViolationConfig  `mapstructure:"rate_violation_block"`
}

type IPRateLimitConfig struct {
	RPS   int `mapstructure:"rps"`
	Burst int `mapstructure:"burst"`
}

type FailedAuthConfig struct {
	MaxAttempts    int `mapstructure:"max_attempts"`
	WindowS        int `mapstructure:"window_s"`
	BlockDurationS int `mapstructure:"block_duration_s"`
}

type BurstAutoBlockConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	BurstFactor     int  `mapstructure:"burst_factor"`
	BlockDurationS  int  `mapstructure:"block_duration_s"`
}

type RateViolationConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	MaxViolations   int  `mapstructure:"max_violations"`
	WindowS         int  `mapstructure:"window_s"`
	BlockDurationS  int  `mapstructure:"block_duration_s"`
}

type LoadBalancerConfig struct {
	Strategy           string        `mapstructure:"strategy"`            // priority, round_robin, weighted_round_robin, least_connections, least_response_time, health_score, random, adaptive
	CooldownDuration   time.Duration `mapstructure:"cooldown_duration"`   // Time to wait after error before using account again
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"` // Interval for health score recalculation
	DefaultWeight      int           `mapstructure:"default_weight"`      // Default weight for weighted strategies
	MaxConnectionsPerAccount int      `mapstructure:"max_connections_per_account"` // Max connections per account for least_connections
	AdaptiveWeights    AdaptiveWeightsConfig `mapstructure:"adaptive_weights"` // Weights for adaptive strategy
}

type AdaptiveWeightsConfig struct {
	HealthScore    float64 `mapstructure:"health_score"`     // Weight for health score factor
	LatencyScore   float64 `mapstructure:"latency_score"`    // Weight for latency factor
	SuccessRate    float64 `mapstructure:"success_rate"`     // Weight for success rate factor
	ConnectionScore float64 `mapstructure:"connection_score"` // Weight for connection factor
	LoadScore      float64 `mapstructure:"load_score"`       // Weight for usage/load factor
}

// PluginsConfig configures the plugin system
type PluginsConfig struct {
	Enabled            bool                      `mapstructure:"enabled"`             // Enable plugin system
	PluginDir          string                    `mapstructure:"plugin_dir"`          // Plugin storage directory
	AllowNativePlugins bool                      `mapstructure:"allow_native_plugins"` // Allow Go plugins (.so files)
	AvailablePlugins   map[string]PluginDefConfig `mapstructure:"available_plugins"`   // Available plugins from marketplace
	Sandbox            PluginSandboxConfig       `mapstructure:"sandbox"`             // Plugin sandbox settings
}

// PluginDefConfig defines an available plugin
type PluginDefConfig struct {
	Name         string   `mapstructure:"name"`
	Version      string   `mapstructure:"version"`
	Type         string   `mapstructure:"type"`          // "go" or "adapter"
	Provider     string   `mapstructure:"provider"`
	Description  string   `mapstructure:"description"`
	Author       string   `mapstructure:"author"`
	AdapterURL   string   `mapstructure:"adapter_url"`   // For external adapters
	Capabilities []string `mapstructure:"capabilities"`
	ConfigSchema map[string]interface{} `mapstructure:"config_schema"`
}

// PluginSandboxConfig configures plugin security sandbox
type PluginSandboxConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowedNetworks  []string `mapstructure:"allowed_networks"`
	MaxMemoryMB      int      `mapstructure:"max_memory_mb"`
	TimeoutSeconds   int      `mapstructure:"timeout_seconds"`
}

// NotificationConfig configures notification services for budget alerts
type NotificationConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	SMTPFrom     string `mapstructure:"smtp_from"`
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// ==================== Auth配置 ====================

// AuthConfig 认证相关配置
type AuthConfig struct {
	JWT              JWTConfig              `mapstructure:"jwt"`
	Encryption       EncryptionConfig       `mapstructure:"encryption"`
	LoginSecurity    LoginSecurityConfig    `mapstructure:"login_security"`
	Password         PasswordConfig         `mapstructure:"password"`
	EmailEncryption  EmailEncryptionConfig  `mapstructure:"email_encryption"`
	EmailVerification EmailVerificationConfig `mapstructure:"email_verification"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey         string        `mapstructure:"secret_key"`          // JWT签名密钥
	AccessTokenExpiry time.Duration `mapstructure:"access_token_expire"` // Access token有效期 (默认15m)
	RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expire"` // Refresh token有效期 (默认168h)
}

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	DataKey    string `mapstructure:"data_key"`     // AES-256数据加密密钥 (32字节)
	JWTKey     string `mapstructure:"jwt_key"`      // JWT签名密钥 (复用或独立)
	KeyVersion int    `mapstructure:"key_version"`  // 密钥版本号
}

// LoginSecurityConfig 登录安全配置
type LoginSecurityConfig struct {
	MaxFailedAttempts int           `mapstructure:"max_failed_attempts"` // 最大失败次数
	FailedWindow      time.Duration `mapstructure:"failed_window"`       // 失败计数窗口
	BlockDuration     time.Duration `mapstructure:"block_duration"`      // 封禁时长
	EnableAuditLog    bool          `mapstructure:"enable_audit_log"`    // 记录审计日志
	EncryptAuditData  bool          `mapstructure:"encrypt_audit_data"`  // 加密审计日志
	AnomalyDetection  bool          `mapstructure:"anomaly_detection"`   // 异常登录检测
	NewDeviceAlert    bool          `mapstructure:"new_device_alert"`    // 新设备提醒
}

// PasswordConfig 密码配置
type PasswordConfig struct {
	MinLength       int  `mapstructure:"min_length"`       // 最小长度
	MaxLength       int  `mapstructure:"max_length"`       // 最大长度
	RequireUpper    bool `mapstructure:"require_upper"`    // 需要大写字母
	RequireLower    bool `mapstructure:"require_lower"`    // 需要小写字母
	RequireDigit    bool `mapstructure:"require_digit"`    // 需要数字
	RequireSpecial  bool `mapstructure:"require_special"`  // 需要特殊字符
	HistoryCount    int  `mapstructure:"history_count"`    // 检查历史密码数
	BcryptCost      int  `mapstructure:"bcrypt_cost"`      // bcrypt cost参数 (默认12)
}

// EmailEncryptionConfig 邮箱加密配置
type EmailEncryptionConfig struct {
	Enabled           bool `mapstructure:"enabled"`             // 是否加密邮箱
	StoreHashForQuery bool `mapstructure:"store_hash_for_query"` // 存hash用于查询
}

// EmailVerificationConfig 邮箱验证配置
type EmailVerificationConfig struct {
	Enabled            bool `mapstructure:"enabled"`              // 是否启用邮箱验证（默认true）
	TokenExpiryHours   int  `mapstructure:"token_expiry_hours"`   // 验证token过期时间（小时，默认24）
}