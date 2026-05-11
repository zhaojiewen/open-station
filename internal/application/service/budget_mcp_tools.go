package service

// Budget and cost limit MCP tools

import (
	"github.com/zhaojiewen/open-station/pkg/mcp"
)

// getBudgetAlertTools returns tools for budget alert management
func getBudgetAlertTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "set_user_budget",
			Title:       "Set User Budget",
			Description: "Set budget limits for a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id":       map[string]interface{}{"type": "string", "description": "User ID"},
					"monthly_budget": map[string]interface{}{"type": "number", "description": "Monthly budget limit (optional)"},
					"daily_budget":   map[string]interface{}{"type": "number", "description": "Daily budget limit (optional)"},
					"token_quota":    map[string]interface{}{"type": "integer", "description": "Monthly token quota (optional)"},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "get_user_budget_usage",
			Title:       "Get User Budget Usage",
			Description: "Get budget usage information for a user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{"type": "string", "description": "User ID"},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "set_api_key_cost_limit",
			Title:       "Set API Key Cost Limit",
			Description: "Set cost limits for an API key",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"api_key_id":          map[string]interface{}{"type": "string", "description": "API Key ID"},
					"monthly_cost_limit":  map[string]interface{}{"type": "number", "description": "Monthly cost limit (optional)"},
					"daily_cost_limit":    map[string]interface{}{"type": "number", "description": "Daily cost limit (optional)"},
					"per_request_limit":   map[string]interface{}{"type": "number", "description": "Per-request cost limit (optional)"},
					"monthly_token_limit": map[string]interface{}{"type": "integer", "description": "Monthly token limit (optional)"},
					"daily_token_limit":   map[string]interface{}{"type": "integer", "description": "Daily token limit (optional)"},
					"alert_threshold_1":    map[string]interface{}{"type": "integer", "description": "First alert threshold percentage (default: 80)"},
					"alert_threshold_2":    map[string]interface{}{"type": "integer", "description": "Second alert threshold percentage (default: 90)"},
					"alert_threshold_3":    map[string]interface{}{"type": "integer", "description": "Third alert threshold percentage (default: 100)"},
				},
				"required": []string{"api_key_id"},
			},
		},
		{
			Name:        "get_api_key_cost_usage",
			Title:       "Get API Key Cost Usage",
			Description: "Get cost usage information for an API key",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"api_key_id": map[string]interface{}{"type": "string", "description": "API Key ID"},
				},
				"required": []string{"api_key_id"},
			},
		},
		{
			Name:        "set_tenant_budget",
			Title:       "Set Tenant Budget",
			Description: "Set budget limits for a tenant",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tenant_id":          map[string]interface{}{"type": "string", "description": "Tenant ID"},
					"monthly_budget_limit": map[string]interface{}{"type": "number", "description": "Monthly budget limit (optional)"},
					"token_limit":        map[string]interface{}{"type": "integer", "description": "Monthly token limit (optional)"},
				},
				"required": []string{"tenant_id"},
			},
		},
		{
			Name:        "get_cost_summary",
			Title:       "Get Cost Summary",
			Description: "Get comprehensive cost usage summary for tenant, user, and API key",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"api_key_id": map[string]interface{}{"type": "string", "description": "API Key ID"},
					"user_id":    map[string]interface{}{"type": "string", "description": "User ID (optional)"},
					"tenant_id":  map[string]interface{}{"type": "string", "description": "Tenant ID (optional)"},
				},
				"required": []string{"api_key_id"},
			},
		},
		{
			Name:        "create_budget_alert",
			Title:       "Create Budget Alert",
			Description: "Create a budget alert for a resource",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"scope":             map[string]interface{}{"type": "string", "description": "Scope: tenant, user, or api_key"},
					"scope_id":          map[string]interface{}{"type": "string", "description": "Scope resource ID"},
					"alert_type":        map[string]interface{}{"type": "string", "description": "Alert type: budget_80, budget_90, budget_100"},
					"threshold_percent": map[string]interface{}{"type": "integer", "description": "Threshold percentage (1-100)"},
					"notify_emails":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Email addresses to notify"},
					"notify_slack":      map[string]interface{}{"type": "string", "description": "Slack webhook URL"},
					"notify_webhook":    map[string]interface{}{"type": "string", "description": "Custom webhook URL"},
				},
				"required": []string{"scope", "scope_id", "threshold_percent"},
			},
		},
		{
			Name:        "list_budget_alerts",
			Title:       "List Budget Alerts",
			Description: "List budget alerts for a scope",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"scope":    map[string]interface{}{"type": "string", "description": "Scope: tenant, user, or api_key (optional)"},
					"scope_id": map[string]interface{}{"type": "string", "description": "Scope resource ID (optional)"},
					"page":     map[string]interface{}{"type": "integer", "description": "Page number (default: 1)", "default": 1},
					"limit":    map[string]interface{}{"type": "integer", "description": "Records per page (default: 20)", "default": 20},
				},
			},
		},
		{
			Name:        "update_budget_alert",
			Title:       "Update Budget Alert",
			Description: "Update a budget alert",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"alert_id":          map[string]interface{}{"type": "string", "description": "Alert ID"},
					"threshold_percent": map[string]interface{}{"type": "integer", "description": "New threshold percentage"},
					"notify_emails":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "New email addresses"},
					"notify_slack":      map[string]interface{}{"type": "string", "description": "New Slack webhook URL"},
					"notify_webhook":    map[string]interface{}{"type": "string", "description": "New custom webhook URL"},
				},
				"required": []string{"alert_id"},
			},
		},
		{
			Name:        "delete_budget_alert",
			Title:       "Delete Budget Alert",
			Description: "Delete a budget alert",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"alert_id": map[string]interface{}{"type": "string", "description": "Alert ID"},
				},
				"required": []string{"alert_id"},
			},
		},
		{
			Name:        "enable_budget_alert",
			Title:       "Enable Budget Alert",
			Description: "Enable a budget alert",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"alert_id": map[string]interface{}{"type": "string", "description": "Alert ID"},
				},
				"required": []string{"alert_id"},
			},
		},
		{
			Name:        "disable_budget_alert",
			Title:       "Disable Budget Alert",
			Description: "Disable a budget alert",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"alert_id": map[string]interface{}{"type": "string", "description": "Alert ID"},
				},
				"required": []string{"alert_id"},
			},
		},
	}
}

// getUserApplicationTools returns tools for user application/invitation management
func getUserApplicationTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "send_user_invitation",
			Title:       "Send User Invitation",
			Description: "Send an invitation to a user to join the tenant",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"email":          map[string]interface{}{"type": "string", "description": "User email"},
					"name":           map[string]interface{}{"type": "string", "description": "User name (optional)"},
					"requested_role": map[string]interface{}{"type": "string", "description": "Role: member, viewer", "default": "member"},
					"expires_in":     map[string]interface{}{"type": "integer", "description": "Invitation expiry in seconds (default: 7 days)"},
				},
				"required": []string{"email"},
			},
		},
		{
			Name:        "list_user_applications",
			Title:       "List User Applications",
			Description: "List user applications and invitations for the tenant",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "description": "Filter by status: pending, approved, rejected, all", "default": "all"},
					"page":   map[string]interface{}{"type": "integer", "description": "Page number (default: 1)", "default": 1},
					"limit":  map[string]interface{}{"type": "integer", "description": "Records per page (default: 20)", "default": 20},
				},
			},
		},
		{
			Name:        "approve_user_application",
			Title:       "Approve User Application",
			Description: "Approve a user application and create the user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"application_id": map[string]interface{}{"type": "string", "description": "Application ID"},
					"password":       map[string]interface{}{"type": "string", "description": "Initial password for user (min 8 characters)"},
				},
				"required": []string{"application_id", "password"},
			},
		},
		{
			Name:        "reject_user_application",
			Title:       "Reject User Application",
			Description: "Reject a user application",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"application_id": map[string]interface{}{"type": "string", "description": "Application ID"},
				},
				"required": []string{"application_id"},
			},
		},
		{
			Name:        "cancel_user_invitation",
			Title:       "Cancel User Invitation",
			Description: "Cancel a pending user invitation",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"invitation_id": map[string]interface{}{"type": "string", "description": "Invitation ID"},
				},
				"required": []string{"invitation_id"},
			},
		},
		{
			Name:        "create_user_direct",
			Title:       "Create User Directly",
			Description: "Directly create a user without approval process",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"email":    map[string]interface{}{"type": "string", "description": "User email"},
					"name":     map[string]interface{}{"type": "string", "description": "User name"},
					"role":     map[string]interface{}{"type": "string", "description": "Role: admin, member, viewer", "default": "member"},
					"password": map[string]interface{}{"type": "string", "description": "Password (min 8 characters)"},
				},
				"required": []string{"email", "name", "password"},
			},
		},
	}
}

// getTenantApplicationTools returns tools for tenant application management (platform admin)
func getTenantApplicationTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "list_tenant_applications",
			Title:       "List Tenant Applications",
			Description: "List tenant applications pending approval",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "description": "Filter by status: pending, reviewing, approved, rejected, all", "default": "pending"},
					"page":   map[string]interface{}{"type": "integer", "description": "Page number (default: 1)", "default": 1},
					"limit":  map[string]interface{}{"type": "integer", "description": "Records per page (default: 20)", "default": 20},
				},
			},
		},
		{
			Name:        "approve_tenant_application",
			Title:       "Approve Tenant Application",
			Description: "Approve a tenant application and create the tenant",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"application_id": map[string]interface{}{"type": "string", "description": "Application ID"},
					"notes":          map[string]interface{}{"type": "string", "description": "Approval notes (optional)"},
				},
				"required": []string{"application_id"},
			},
		},
		{
			Name:        "reject_tenant_application",
			Title:       "Reject Tenant Application",
			Description: "Reject a tenant application",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"application_id": map[string]interface{}{"type": "string", "description": "Application ID"},
					"reason":         map[string]interface{}{"type": "string", "description": "Rejection reason"},
				},
				"required": []string{"application_id", "reason"},
			},
		},
		{
			Name:        "suspend_tenant",
			Title:       "Suspend Tenant",
			Description: "Suspend a tenant account",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tenant_id": map[string]interface{}{"type": "string", "description": "Tenant ID"},
				},
				"required": []string{"tenant_id"},
			},
		},
		{
			Name:        "activate_tenant",
			Title:       "Activate Tenant",
			Description: "Activate a suspended tenant account",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tenant_id": map[string]interface{}{"type": "string", "description": "Tenant ID"},
				},
				"required": []string{"tenant_id"},
			},
		},
	}
}