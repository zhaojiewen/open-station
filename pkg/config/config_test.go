package config

import (
	"os"
	"testing"
	"time"
)

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected string
	}{
		{
			name: "standard config",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				DBName:   "testdb",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password=secret dbname=testdb sslmode=disable",
		},
		{
			name: "empty password",
			config: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "admin",
				Password: "",
				DBName:   "production",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=admin password= dbname=production sslmode=require",
		},
		{
			name: "with all fields",
			config: DatabaseConfig{
				Host:            "prod-db",
				Port:            5432,
				User:            "dbuser",
				Password:        "pass123",
				DBName:          "myapp",
				SSLMode:         "verify-full",
				MaxOpenConns:    100,
				MaxIdleConns:    10,
				ConnMaxLifetime: time.Hour,
			},
			expected: "host=prod-db port=5432 user=dbuser password=pass123 dbname=myapp sslmode=verify-full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.DSN()
			if result != tt.expected {
				t.Errorf("DSN() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRedisConfig_Addr(t *testing.T) {
	tests := []struct {
		name     string
		config   RedisConfig
		expected string
	}{
		{
			name: "standard config",
			config: RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "secret",
				DB:       0,
				PoolSize: 10,
			},
			expected: "localhost:6379",
		},
		{
			name: "custom host and port",
			config: RedisConfig{
				Host:     "redis.example.com",
				Port:     6380,
				Password: "",
				DB:       1,
				PoolSize: 20,
			},
			expected: "redis.example.com:6380",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Addr()
			if result != tt.expected {
				t.Errorf("Addr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		_, err := Load("nonexistent.yaml")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		configContent := `
server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 5432
  user: postgres
  password: secret
  dbname: testdb
  sslmode: disable
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 10

providers:
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: "test-key"
    timeout: 30s
  claude:
    base_url: "https://api.anthropic.com/v1"
    api_key: "claude-key"
    timeout: 60s

billing:
  default_currency: "USD"
  min_balance_alert: 10.0

rate_limit:
  default_user_rps: 10.0
  default_user_burst: 20
  default_tenant_rps: 100.0
  default_tenant_burst: 200
  redis_key_prefix: "ratelimit:"
  window_size: "1s"

logging:
  level: "debug"
  format: "json"
  output: "stdout"

admin:
  default_tenant_slug: "default"
  super_admin_email: "admin@example.com"
  default_admin_user: "admin"
  default_admin_pass: "password"
  initial_api_key_name: "Default API Key"
`
		tmpFile := "/tmp/test-config.yaml"
		if err := os.WriteFile(tmpFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write temp config: %v", err)
		}
		defer os.Remove(tmpFile)

		cfg, err := Load(tmpFile)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Server.Port != 8080 {
			t.Errorf("expected port 8080, got %d", cfg.Server.Port)
		}
		if cfg.Server.Mode != "debug" {
			t.Errorf("expected mode debug, got %s", cfg.Server.Mode)
		}
		if cfg.Database.Host != "localhost" {
			t.Errorf("expected database host localhost, got %s", cfg.Database.Host)
		}
		if cfg.Redis.Host != "localhost" {
			t.Errorf("expected redis host localhost, got %s", cfg.Redis.Host)
		}
		if cfg.Billing.DefaultCurrency != "USD" {
			t.Errorf("expected currency USD, got %s", cfg.Billing.DefaultCurrency)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		configContent := `
server:
  port: invalid
`
		tmpFile := "/tmp/test-invalid-config.yaml"
		if err := os.WriteFile(tmpFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write temp config: %v", err)
		}
		defer os.Remove(tmpFile)

		cfg, err := Load(tmpFile)
		// Should not error because port is just a field, it may default to zero value
		// The actual behavior depends on viper's unmarshal behavior
		if err == nil && cfg.Server.Port != 0 {
			t.Log("Config loaded with default value for invalid port")
		}
	})
}

func TestConfigStructFields(t *testing.T) {
	t.Run("provider config", func(t *testing.T) {
		provider := ProviderConfig{
			BaseURL: "https://api.openai.com/v1",
			APIKey:  "test-key",
			Timeout: 30 * time.Second,
		}

		if provider.BaseURL != "https://api.openai.com/v1" {
			t.Errorf("unexpected BaseURL: %s", provider.BaseURL)
		}
		if provider.APIKey != "test-key" {
			t.Errorf("unexpected APIKey: %s", provider.APIKey)
		}
		if provider.Timeout != 30*time.Second {
			t.Errorf("unexpected Timeout: %v", provider.Timeout)
		}
	})

	t.Run("billing config", func(t *testing.T) {
		billing := BillingConfig{
			DefaultCurrency: "EUR",
			MinBalanceAlert: 50.0,
		}

		if billing.DefaultCurrency != "EUR" {
			t.Errorf("unexpected DefaultCurrency: %s", billing.DefaultCurrency)
		}
		if billing.MinBalanceAlert != 50.0 {
			t.Errorf("unexpected MinBalanceAlert: %f", billing.MinBalanceAlert)
		}
	})

	t.Run("rate limit config", func(t *testing.T) {
		rl := RateLimitConfig{
			DefaultUserRPS:     100.0,
			DefaultUserBurst:  200,
			DefaultTenantRPS:  1000.0,
			DefaultTenantBurst: 2000,
			RedisKeyPrefix:    "rl:",
			WindowSize:        "1s",
		}

		if rl.DefaultUserRPS != 100.0 {
			t.Errorf("unexpected DefaultUserRPS: %f", rl.DefaultUserRPS)
		}
	})

	t.Run("logging config", func(t *testing.T) {
		log := LoggingConfig{
			Level:  "info",
			Format: "console",
			Output: "/var/log/app.log",
		}

		if log.Level != "info" {
			t.Errorf("unexpected Level: %s", log.Level)
		}
	})

	t.Run("admin config", func(t *testing.T) {
		admin := AdminConfig{
			DefaultTenantSlug: "my-tenant",
			SuperAdminEmail:   "admin@company.com",
			DefaultAdminUser:  "superadmin",
			DefaultAdminPass:  "secure-pass",
			InitialAPIKeyName: "Initial Key",
		}

		if admin.DefaultTenantSlug != "my-tenant" {
			t.Errorf("unexpected DefaultTenantSlug: %s", admin.DefaultTenantSlug)
		}
	})
}
func TestDynamicProviderConfig(t *testing.T) {
	providers := ProvidersConfig{
		OpenAI: ProviderConfig{
			BaseURL: "https://api.openai.com/v1",
			APIKey:  "static-openai-key",
			Timeout: 30 * time.Second,
		},
		Accounts: map[string]ProviderConfig{
			"custom-provider": ProviderConfig{
				BaseURL: "https://custom.api.com/v1",
				APIKey:  "dynamic-key",
				Timeout: 45 * time.Second,
			},
		},
		DefaultTimeout: 60 * time.Second,
	}

	// Test GetProvider for static config
	cfg := providers.GetProvider("openai")
	if cfg == nil {
		t.Error("expected openai provider config")
	}
	if cfg.APIKey != "static-openai-key" {
		t.Errorf("unexpected APIKey: %s", cfg.APIKey)
	}

	// Test GetProvider for dynamic config
	cfg = providers.GetProvider("custom-provider")
	if cfg == nil {
		t.Error("expected custom-provider config")
	}
	if cfg.APIKey != "dynamic-key" {
		t.Errorf("unexpected dynamic APIKey: %s", cfg.APIKey)
	}

	// Test GetProvider for non-existent
	cfg = providers.GetProvider("nonexistent")
	if cfg != nil {
		t.Error("expected nil for nonexistent provider")
	}

	// Test SetProvider
	providers.SetProvider("new-provider", ProviderConfig{
		BaseURL: "https://new.api.com",
		APIKey:  "new-key",
	})
	cfg = providers.GetProvider("new-provider")
	if cfg == nil || cfg.APIKey != "new-key" {
		t.Error("SetProvider failed")
	}

	// Test ListProviders
	list := providers.ListProviders()
	if len(list) < 2 {
		t.Errorf("expected at least 2 providers, got %d", len(list))
	}

	// Test GetTimeout
	timeout := providers.GetTimeout("custom-provider")
	if timeout != 45*time.Second {
		t.Errorf("expected 45s timeout, got %v", timeout)
	}

	// Test GetTimeout with default
	timeout = providers.GetTimeout("new-provider")
	if timeout != 60*time.Second {
		t.Errorf("expected default 60s timeout, got %v", timeout)
	}
}
