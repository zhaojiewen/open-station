package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PluginConfigSchema defines the configuration schema for a plugin
type PluginConfigSchema struct {
	Type       string                            `json:"type"`
	Properties map[string]PluginConfigProperty   `json:"properties"`
	Required   []string                          `json:"required"`
}

// PluginConfigProperty defines a single configuration property
type PluginConfigProperty struct {
	Type         string        `json:"type"`
	Description  string        `json:"description"`
	Default      interface{}   `json:"default,omitempty"`
	Enum         []string      `json:"enum,omitempty"`
	Minimum      *float64      `json:"minimum,omitempty"`
	Maximum      *float64      `json:"maximum,omitempty"`
	Pattern      string        `json:"pattern,omitempty"`     // Regex for string validation
	Sensitive    bool          `json:"sensitive,omitempty"`   // Marks as sensitive (API keys)
	Environment  string        `json:"environment,omitempty"` // Environment variable for value
}

// PluginConfig represents runtime configuration for a plugin
type PluginConfig struct {
	PluginID string                 `json:"plugin_id"`
	Provider string                 `json:"provider"`
	Settings map[string]interface{} `json:"settings"`
}

// ParsePluginConfigFile parses a plugin config file
func ParsePluginConfigFile(path string) (*PluginConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config PluginConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand environment variables
	config.Settings = expandEnvVars(config.Settings)

	return &config, nil
}

// expandEnvVars expands environment variables in config values
func expandEnvVars(settings map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range settings {
		switch v := value.(type) {
		case string:
			// Check for ${VAR} pattern
			if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
				envVar := v[2 : len(v)-1]
				envValue := os.Getenv(envVar)
				if envValue != "" {
					result[key] = envValue
				} else {
					result[key] = v // Keep original if env not set
				}
			} else {
				result[key] = v
			}
		case map[string]interface{}:
			result[key] = expandEnvVars(v)
		default:
			result[key] = v
		}
	}

	return result
}

// ValidateConfig validates config against schema
func ValidateConfig(config map[string]interface{}, schema PluginConfigSchema) error {
	// Check required fields
	for _, req := range schema.Required {
		if _, exists := config[req]; !exists {
			return fmt.Errorf("required field '%s' is missing", req)
		}
	}

	// Validate each property
	for key, value := range config {
		prop, exists := schema.Properties[key]
		if !exists {
			// Unknown field - could allow or reject based on policy
			continue
		}

		if err := validatePropertyValue(key, value, prop); err != nil {
			return err
		}
	}

	return nil
}

// validatePropertyValue validates a single property value
func validatePropertyValue(key string, value interface{}, prop PluginConfigProperty) error {
	switch prop.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be string", key)
		}
		// Check pattern if specified
		if prop.Pattern != "" {
			// TODO: regex validation
		}
		// Check enum if specified
		if len(prop.Enum) > 0 {
			valid := false
			for _, e := range prop.Enum {
				if str == e {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("field '%s' must be one of: %s", key, strings.Join(prop.Enum, ", "))
			}
		}

	case "integer":
		switch value.(type) {
		case int, int64, float64:
			// OK - validate min/max
			if prop.Minimum != nil {
				if floatValue, ok := value.(float64); ok && floatValue < *prop.Minimum {
					return fmt.Errorf("field '%s' must be >= %v", key, *prop.Minimum)
				}
			}
			if prop.Maximum != nil {
				if floatValue, ok := value.(float64); ok && floatValue > *prop.Maximum {
					return fmt.Errorf("field '%s' must be <= %v", key, *prop.Maximum)
				}
			}
		default:
			return fmt.Errorf("field '%s' must be integer", key)
		}

	case "number":
		switch value.(type) {
		case int, int64, float64:
			// OK
		default:
			return fmt.Errorf("field '%s' must be number", key)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be boolean", key)
		}

	case "array":
		switch value.(type) {
		case []interface{}, []string, []int:
			// OK
		default:
			return fmt.Errorf("field '%s' must be array", key)
		}

	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("field '%s' must be object", key)
		}
	}

	return nil
}

// ConfigBuilder helps build plugin configurations
type ConfigBuilder struct {
	config map[string]interface{}
}

// NewConfigBuilder creates a new config builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: make(map[string]interface{}),
	}
}

// Set sets a configuration value
func (b *ConfigBuilder) Set(key string, value interface{}) *ConfigBuilder {
	b.config[key] = value
	return b
}

// SetAPIKey sets the API key (with environment variable support)
func (b *ConfigBuilder) SetAPIKey(apiKey string) *ConfigBuilder {
	b.config["api_key"] = apiKey
	return b
}

// SetAPIKeyFromEnv sets API key from environment variable
func (b *ConfigBuilder) SetAPIKeyFromEnv(envVar string) *ConfigBuilder {
	b.config["api_key"] = "${" + envVar + "}"
	return b
}

// SetBaseURL sets the base URL
func (b *ConfigBuilder) SetBaseURL(url string) *ConfigBuilder {
	b.config["base_url"] = url
	return b
}

// SetTimeout sets the timeout in seconds
func (b *ConfigBuilder) SetTimeout(seconds int) *ConfigBuilder {
	b.config["timeout"] = seconds
	return b
}

// SetMaxRetries sets the max retry count
func (b *ConfigBuilder) SetMaxRetries(count int) *ConfigBuilder {
	b.config["max_retries"] = count
	return b
}

// Build returns the built configuration
func (b *ConfigBuilder) Build() map[string]interface{} {
	return expandEnvVars(b.config)
}

// BuildJSON returns the configuration as JSON
func (b *ConfigBuilder) BuildJSON() (string, error) {
	config := b.Build()
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveToFile saves configuration to a file
func (b *ConfigBuilder) SaveToFile(path string) error {
	config := b.Build()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// StandardProviderConfig creates standard provider configuration
func StandardProviderConfig(apiKeyEnvVar, baseURL string) map[string]interface{} {
	return NewConfigBuilder().
		SetAPIKeyFromEnv(apiKeyEnvVar).
		SetBaseURL(baseURL).
		SetTimeout(120).
		SetMaxRetries(3).
		Build()
}

// MergeConfigs merges multiple configurations (later configs override earlier)
func MergeConfigs(configs ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for _, config := range configs {
		for key, value := range config {
			// Deep merge for nested objects
			if existing, ok := result[key]; ok {
				if existingMap, ok1 := existing.(map[string]interface{}); ok1 {
					if newMap, ok2 := value.(map[string]interface{}); ok2 {
						result[key] = MergeConfigs(existingMap, newMap)
						continue
					}
				}
			}
			result[key] = value
		}
	}

	return result
}

// GetSensitiveFields returns list of sensitive field names from schema
func GetSensitiveFields(schema PluginConfigSchema) []string {
	sensitive := make([]string, 0)
	for key, prop := range schema.Properties {
		if prop.Sensitive {
			sensitive = append(sensitive, key)
		}
	}
	return sensitive
}

// MaskSensitiveConfig masks sensitive values in config for logging/display
func MaskSensitiveConfig(config map[string]interface{}, sensitiveFields []string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range config {
		if isSensitive(key, sensitiveFields) {
			if str, ok := value.(string); ok {
				if len(str) > 8 {
					result[key] = str[:4] + "****" + str[len(str)-4:]
				} else {
					result[key] = "****"
				}
			} else {
				result[key] = "****"
			}
		} else {
			result[key] = value
		}
	}

	return result
}

func isSensitive(key string, sensitiveFields []string) bool {
	keyLower := strings.ToLower(key)
	for _, field := range sensitiveFields {
		if strings.ToLower(field) == keyLower {
			return true
		}
	}
	// Also check common sensitive patterns
	sensitivePatterns := []string{"api_key", "apikey", "secret", "password", "token", "credential"}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(keyLower, pattern) {
			return true
		}
	}
	return false
}