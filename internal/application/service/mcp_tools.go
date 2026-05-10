package service

// Tool definitions for MCP service

import (
	"github.com/zhaojiewen/open-station/pkg/mcp"
)

// getUserTools returns tools available to regular users
func (s *MCPService) getUserTools() []mcp.Tool {
	tools := []mcp.Tool{
		{Name: "check_balance", Title: "Check Token Balance", Description: "Check current token balance for your tenant",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{Name: "get_usage_summary", Title: "Get Usage Summary", Description: "Get usage summary for a time period",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"start_date": map[string]interface{}{"type": "string", "description": "Start date (YYYY-MM-DD)"},
				"end_date":   map[string]interface{}{"type": "string", "description": "End date (YYYY-MM-DD)"},
			}}},
		{Name: "get_usage_details", Title: "Get Usage Details", Description: "Get detailed usage records for a time period",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"start_date": map[string]interface{}{"type": "string", "description": "Start date (YYYY-MM-DD)"},
				"end_date":   map[string]interface{}{"type": "string", "description": "End date (YYYY-MM-DD)"},
				"limit":      map[string]interface{}{"type": "integer", "description": "Maximum records (default: 50)", "default": 50},
			}}},
		{Name: "get_billing_info", Title: "Get Billing Info", Description: "Get billing and payment information",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{Name: "get_recharge_history", Title: "Get Recharge History", Description: "Get recharge/payment history for your tenant",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "integer", "description": "Maximum records (default: 20)", "default": 20},
			}}},
		{Name: "get_my_api_keys", Title: "Get My API Keys", Description: "List all API keys belonging to you",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
	}

	// Add plugin user tools if plugin service is available
	if s.pluginService != nil {
		tools = append(tools, getPluginUserTools()...)
	}

	return tools
}

// getManagerTools returns tools available to managers/admins
func (s *MCPService) getManagerTools() []mcp.Tool {
	tools := []mcp.Tool{
		{Name: "list_all_api_keys", Title: "List All API Keys", Description: "List all API keys in the system",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"tenant_id": map[string]interface{}{"type": "string", "description": "Filter by tenant ID (optional)"},
				"status":    map[string]interface{}{"type": "string", "description": "Filter by status", "enum": []string{"active", "revoked", "expired", "all"}},
			}}},
		{Name: "create_api_key", Title: "Create API Key", Description: "Create API key. Auto-create user from user_email/user_name if user_id not provided",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"user_id":     map[string]interface{}{"type": "string", "description": "Existing user ID (optional)"},
				"user_email":  map[string]interface{}{"type": "string", "description": "User email (required if no user_id)"},
				"user_name":   map[string]interface{}{"type": "string", "description": "User name (optional)"},
				"tenant_id":   map[string]interface{}{"type": "string", "description": "Tenant ID (optional)"},
				"name":        map[string]interface{}{"type": "string", "description": "API key name"},
				"permissions": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Permissions: chat, embeddings, admin", "default": []string{"chat"}},
				"models":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Allowed models (optional)"},
				"providers":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Allowed providers (optional)"},
			}, "required": []string{"name"}}},
		{Name: "revoke_api_key", Title: "Revoke API Key", Description: "Revoke an API key",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"api_key_id": map[string]interface{}{"type": "string", "description": "API key ID to revoke"},
			}, "required": []string{"api_key_id"}}},
		{Name: "update_api_key", Title: "Update API Key", Description: "Update API key permissions and settings",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"api_key_id":  map[string]interface{}{"type": "string", "description": "API key ID to update"},
				"permissions": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "New permissions"},
				"models":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "New allowed models"},
				"providers":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "New allowed providers"},
			}, "required": []string{"api_key_id"}}},
		{Name: "list_users", Title: "List Users", Description: "List all users in a tenant or all tenants",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"tenant_id": map[string]interface{}{"type": "string", "description": "Filter by tenant ID (optional)"},
				"page":      map[string]interface{}{"type": "integer", "description": "Page number (default: 1)", "default": 1},
				"limit":     map[string]interface{}{"type": "integer", "description": "Records per page (default: 20)", "default": 20},
			}}},
		{Name: "get_user_detail", Title: "Get User Detail", Description: "Get detailed user information",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"user_id": map[string]interface{}{"type": "string", "description": "User ID"},
			}, "required": []string{"user_id"}}},
		{Name: "adjust_balance", Title: "Adjust Balance", Description: "Adjust tenant balance (recharge or deduct)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"tenant_id": map[string]interface{}{"type": "string", "description": "Tenant ID"},
				"amount":    map[string]interface{}{"type": "number", "description": "Amount to add (positive) or deduct (negative)"},
				"reason":    map[string]interface{}{"type": "string", "description": "Reason for adjustment"},
			}, "required": []string{"tenant_id", "amount", "reason"}}},
		{Name: "get_tenant_summary", Title: "Get Tenant Summary", Description: "Get tenant summary with balance and usage statistics",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"tenant_id": map[string]interface{}{"type": "string", "description": "Tenant ID"},
			}, "required": []string{"tenant_id"}}},
		{Name: "list_tenants", Title: "List Tenants", Description: "List all tenants in the system",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"page":  map[string]interface{}{"type": "integer", "description": "Page number (default: 1)", "default": 1},
				"limit": map[string]interface{}{"type": "integer", "description": "Records per page (default: 20)", "default": 20},
			}}},
		// Provider Account Management
		{Name: "list_provider_accounts", Title: "List Provider Accounts", Description: "List provider accounts with status",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"provider": map[string]interface{}{"type": "string", "description": "Provider name (optional)"},
				"status":   map[string]interface{}{"type": "string", "description": "Filter by status (optional)"},
			}}},
		{Name: "create_provider_account", Title: "Create Provider Account", Description: "Add provider API account",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"provider":     map[string]interface{}{"type": "string", "description": "Provider: openai, anthropic, gemini, deepseek, glm"},
				"name":         map[string]interface{}{"type": "string", "description": "Account name"},
				"api_key":      map[string]interface{}{"type": "string", "description": "API key"},
				"base_url":     map[string]interface{}{"type": "string", "description": "Base URL (optional)"},
				"priority":     map[string]interface{}{"type": "integer", "description": "Priority (0=highest)"},
				"monthly_limit": map[string]interface{}{"type": "number", "description": "Monthly limit (optional)"},
			}, "required": []string{"provider", "name", "api_key"}}},
		{Name: "update_provider_account", Title: "Update Provider Account", Description: "Update provider account",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"account_id":   map[string]interface{}{"type": "string", "description": "Account ID"},
				"name":         map[string]interface{}{"type": "string", "description": "New name (optional)"},
				"api_key":      map[string]interface{}{"type": "string", "description": "New API key (optional)"},
				"base_url":     map[string]interface{}{"type": "string", "description": "New base URL (optional)"},
				"priority":     map[string]interface{}{"type": "integer", "description": "New priority (optional)"},
				"monthly_limit": map[string]interface{}{"type": "number", "description": "New monthly limit (optional)"},
			}, "required": []string{"account_id"}}},
		{Name: "set_default_provider_account", Title: "Set Default Provider Account", Description: "Set account as default",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"provider":   map[string]interface{}{"type": "string", "description": "Provider name"},
				"account_id": map[string]interface{}{"type": "string", "description": "Account ID"},
			}, "required": []string{"provider", "account_id"}}},
		{Name: "enable_provider_account", Title: "Enable Provider Account", Description: "Enable account",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"account_id": map[string]interface{}{"type": "string", "description": "Account ID"},
			}, "required": []string{"account_id"}}},
		{Name: "disable_provider_account", Title: "Disable Provider Account", Description: "Disable account (switches to next available)",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"account_id": map[string]interface{}{"type": "string", "description": "Account ID"},
			}, "required": []string{"account_id"}}},
		{Name: "delete_provider_account", Title: "Delete Provider Account", Description: "Delete account permanently",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"account_id": map[string]interface{}{"type": "string", "description": "Account ID"},
			}, "required": []string{"account_id"}}},
		{Name: "get_provider_status", Title: "Get Provider Status", Description: "Get provider status summary",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"provider": map[string]interface{}{"type": "string", "description": "Provider name (optional)"},
			}}},
	}

	// Add plugin manager tools if plugin service is available
	if s.pluginService != nil {
		tools = append(tools, getPluginManagerTools()...)
	}

	return tools
}