package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PluginValidator validates plugins for security and compatibility
type PluginValidator interface {
	Validate(info PluginInfo) error
	ValidateConfig(config map[string]interface{}, schema map[string]interface{}) error
	ValidateChecksum(filePath, expectedSHA256 string) error
}

// DefaultValidator provides basic validation
type DefaultValidator struct {
	minVersion    string
	allowedTypes  []PluginType
	allowedProviders []string
	blocklist     []string
}

// NewDefaultValidator creates a new default validator
func NewDefaultValidator(minVersion string) *DefaultValidator {
	return &DefaultValidator{
		minVersion:    minVersion,
		allowedTypes:  []PluginType{PluginTypeGo, PluginTypeAdapter},
		allowedProviders: []string{}, // Empty means all allowed
		blocklist:     []string{},
	}
}

// Validate validates plugin info
func (v *DefaultValidator) Validate(info PluginInfo) error {
	// Check blocklist
	for _, blocked := range v.blocklist {
		if info.ID == blocked || info.Provider == blocked {
			return fmt.Errorf("plugin %s is blocked", info.ID)
		}
	}

	// Validate type
	validType := false
	for _, t := range v.allowedTypes {
		if info.Type == t {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("plugin type %s is not allowed", info.Type)
	}

	// Validate provider
	if len(v.allowedProviders) > 0 {
		validProvider := false
		for _, p := range v.allowedProviders {
			if info.Provider == p {
				validProvider = true
				break
			}
		}
		if !validProvider {
			return fmt.Errorf("provider %s is not allowed", info.Provider)
		}
	}

	// Validate required fields
	if info.ID == "" {
		return fmt.Errorf("plugin id is required")
	}
	if info.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if info.Provider == "" {
		return fmt.Errorf("plugin provider is required")
	}
	if info.Version == "" {
		return fmt.Errorf("plugin version is required")
	}

	// Validate ID format (alphanumeric, dash, underscore)
	if !isValidID(info.ID) {
		return fmt.Errorf("plugin id must be alphanumeric with dashes/underscores")
	}

	// Validate capabilities
	validCapabilities := []string{"chat", "stream", "embedding", "models", "tools", "vision", "audio"}
	for _, cap := range info.Capabilities {
		valid := false
		for _, vc := range validCapabilities {
			if cap == vc {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid capability: %s", cap)
		}
	}

	return nil
}

// ValidateConfig validates configuration against schema
func (v *DefaultValidator) ValidateConfig(config map[string]interface{}, schema map[string]interface{}) error {
	if schema == nil {
		return nil // No schema to validate against
	}

	// Check required fields
	if required, ok := schema["required"].([]interface{}); ok {
		for _, r := range required {
			field, ok := r.(string)
			if !ok {
				continue
			}
			if _, exists := config[field]; !exists {
				return fmt.Errorf("required config field %s is missing", field)
			}
		}
	}

	// Validate field types
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for field, fieldSchema := range properties {
			value, exists := config[field]
			if !exists {
				continue // Not provided, skip validation
			}

			fieldSchemaMap, ok := fieldSchema.(map[string]interface{})
			if !ok {
				continue
			}

			fieldType, ok := fieldSchemaMap["type"].(string)
			if !ok {
				continue
			}

			if err := validateFieldType(field, value, fieldType); err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateChecksum validates file checksum
func (v *DefaultValidator) ValidateChecksum(filePath, expectedSHA256 string) error {
	if expectedSHA256 == "" {
		return nil // No checksum to validate
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file for checksum: %w", err)
	}

	// Calculate SHA256
	actualSHA256 := calculateSHA256(data)
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
	}

	return nil
}

// SetAllowedProviders sets the allowed providers list
func (v *DefaultValidator) SetAllowedProviders(providers []string) {
	v.allowedProviders = providers
}

// SetBlocklist sets the plugin blocklist
func (v *DefaultValidator) SetBlocklist(blocklist []string) {
	v.blocklist = blocklist
}

// isValidID checks if an ID is valid format
func isValidID(id string) bool {
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// validateFieldType validates a value against expected type
func validateFieldType(field string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field %s must be string", field)
		}
	case "integer", "number":
		switch value.(type) {
		case int, int64, float64:
			// OK
		default:
			return fmt.Errorf("field %s must be number", field)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %s must be boolean", field)
		}
	case "array":
		switch value.(type) {
		case []interface{}, []string, []int, []float64:
			// OK
		default:
			return fmt.Errorf("field %s must be array", field)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("field %s must be object", field)
		}
	}
	return nil
}

// calculateSHA256 calculates SHA256 hash (placeholder - use crypto/sha256)
func calculateSHA256(data []byte) string {
	// TODO: Implement with crypto/sha256
	return fmt.Sprintf("%x", len(data)) // Placeholder
}

// ConfigSchemaBuilder helps build config schemas
type ConfigSchemaBuilder struct {
	properties map[string]interface{}
	required   []string
}

// NewConfigSchemaBuilder creates a new schema builder
func NewConfigSchemaBuilder() *ConfigSchemaBuilder {
	return &ConfigSchemaBuilder{
		properties: make(map[string]interface{}),
		required:   make([]string, 0),
	}
}

// AddString adds a string field
func (b *ConfigSchemaBuilder) AddString(name string, required bool, description string) *ConfigSchemaBuilder {
	b.properties[name] = map[string]interface{}{
		"type":        "string",
		"description": description,
	}
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// AddNumber adds a number field
func (b *ConfigSchemaBuilder) AddNumber(name string, required bool, description string, defaultValue interface{}) *ConfigSchemaBuilder {
	prop := map[string]interface{}{
		"type":        "number",
		"description": description,
	}
	if defaultValue != nil {
		prop["default"] = defaultValue
	}
	b.properties[name] = prop
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// AddBoolean adds a boolean field
func (b *ConfigSchemaBuilder) AddBoolean(name string, required bool, description string, defaultValue bool) *ConfigSchemaBuilder {
	prop := map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
	prop["default"] = defaultValue
	b.properties[name] = prop
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// AddEnum adds an enum field
func (b *ConfigSchemaBuilder) AddEnum(name string, required bool, description string, values []string, defaultValue string) *ConfigSchemaBuilder {
	prop := map[string]interface{}{
		"type":        "string",
		"description": description,
		"enum":        values,
	}
	if defaultValue != "" {
		prop["default"] = defaultValue
	}
	b.properties[name] = prop
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// Build returns the schema
func (b *ConfigSchemaBuilder) Build() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": b.properties,
		"required":   b.required,
	}
}

// BuildJSON returns the schema as JSON string
func (b *ConfigSchemaBuilder) BuildJSON() (string, error) {
	schema := b.Build()
	data, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Common config schema for provider plugins
func StandardProviderConfigSchema() map[string]interface{} {
	return NewConfigSchemaBuilder().
		AddString("api_key", true, "API key for the provider").
		AddString("base_url", false, "Custom API base URL").
		AddNumber("timeout", false, "Request timeout in seconds", 120).
		AddNumber("max_retries", false, "Maximum retry count", 3).
		AddBoolean("verify_ssl", false, "Verify SSL certificates", true).
		Build()
}

// SecurityValidator provides enhanced security validation
type SecurityValidator struct {
	DefaultValidator
	signatureVerifier SignatureVerifier
	sandboxChecker    SandboxChecker
}

// SignatureVerifier verifies plugin signatures
type SignatureVerifier interface {
	Verify(filePath string, signature string) error
}

// SandboxChecker checks sandbox compatibility
type SandboxChecker interface {
	Check(info PluginInfo) error
}

// NewSecurityValidator creates a security validator
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		DefaultValidator: *NewDefaultValidator("1.0.0"),
	}
}

// ValidateWithSignature validates plugin with signature check
func (v *SecurityValidator) ValidateWithSignature(info PluginInfo, filePath, signature string) error {
	if err := v.Validate(info); err != nil {
		return err
	}

	if v.signatureVerifier != nil && signature != "" {
		if err := v.signatureVerifier.Verify(filePath, signature); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	return nil
}

// ValidatePath validates plugin file path for security
func (v *SecurityValidator) ValidatePath(path string) error {
	// Clean path to prevent traversal
	cleanPath := filepath.Clean(path)

	// Check for path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: potential path traversal")
	}

	// Ensure path is within allowed directory
	// (This would be configured based on plugin_dir setting)
	return nil
}