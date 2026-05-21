package handler

import "strings"

// ResolveProvider detects the upstream provider from the model name.
// Model names pass through unchanged — this only identifies the provider.
//
// Detection rules:
//  1. gpt-*, o1*, o3* → openai
//  2. claude-* → claude
//  3. deepseek-* → deepseek
//  4. glm-* → glm
//  5. Fallback to the default provider from the API type
func ResolveProvider(model string, defaultProvider string) string {
	// OpenAI models
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") {
		return "openai"
	}

	// Claude models self-identify via claude- prefix
	if strings.HasPrefix(model, "claude-") {
		return "claude"
	}

	if strings.HasPrefix(model, "deepseek-") {
		return "deepseek"
	}

	if strings.HasPrefix(model, "glm-") {
		return "glm"
	}

	return defaultProvider
}
