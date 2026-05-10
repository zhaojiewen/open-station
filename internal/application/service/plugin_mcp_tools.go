package service

// Plugin Management MCP Tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"github.com/zhaojiewen/open-station/pkg/mcp"
	"go.uber.org/zap"
)

// getPluginUserTools returns plugin tools available to regular users
func getPluginUserTools() []mcp.Tool {
	return []mcp.Tool{
		{Name: "list_plugins", Title: "List Plugins", Description: "List all installed plugins",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page":     map[string]interface{}{"type": "integer", "description": "Page number (default: 1)", "default": 1},
					"pageSize": map[string]interface{}{"type": "integer", "description": "Page size (default: 20)", "default": 20},
				}}},
		{Name: "list_available_plugins", Title: "List Available Plugins", Description: "List available plugins from marketplace",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":     map[string]interface{}{"type": "string", "description": "Search query (optional)"},
					"capability": map[string]interface{}{"type": "string", "description": "Filter by capability (optional)"},
				}}},
		{Name: "get_plugin_status", Title: "Get Plugin Status", Description: "Get status and statistics of a plugin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plugin_id": map[string]interface{}{"type": "string", "description": "Plugin ID (e.g., 'openai', 'anthropic')"},
				},
				"required": []string{"plugin_id"}}},
		{Name: "get_plugin_providers", Title: "Get Plugin Providers", Description: "Get list of all available provider names with plugins",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{}}},
	}
}

// getPluginManagerTools returns plugin management tools available to managers
func getPluginManagerTools() []mcp.Tool {
	return []mcp.Tool{
		{Name: "install_plugin", Title: "Install Plugin", Description: "Install a plugin from marketplace",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plugin_id": map[string]interface{}{"type": "string", "description": "Plugin ID to install"},
					"config":    map[string]interface{}{"type": "object", "description": "Plugin configuration"},
					"activate":  map[string]interface{}{"type": "boolean", "description": "Activate after install (default: false)", "default": false},
				},
				"required": []string{"plugin_id"}}},
		{Name: "configure_plugin", Title: "Configure Plugin", Description: "Configure a plugin settings",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plugin_id": map[string]interface{}{"type": "string", "description": "Plugin ID"},
					"config":    map[string]interface{}{"type": "object", "description": "Configuration to apply"},
				},
				"required": []string{"plugin_id", "config"}}},
		{Name: "activate_plugin", Title: "Activate Plugin", Description: "Activate a plugin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plugin_id": map[string]interface{}{"type": "string", "description": "Plugin ID to activate"},
				},
				"required": []string{"plugin_id"}}},
		{Name: "deactivate_plugin", Title: "Deactivate Plugin", Description: "Deactivate a plugin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plugin_id": map[string]interface{}{"type": "string", "description": "Plugin ID to deactivate"},
				},
				"required": []string{"plugin_id"}}},
		{Name: "uninstall_plugin", Title: "Uninstall Plugin", Description: "Uninstall a plugin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plugin_id": map[string]interface{}{"type": "string", "description": "Plugin ID to uninstall"},
				},
				"required": []string{"plugin_id"}}},
		{Name: "check_plugin_health", Title: "Check Plugin Health", Description: "Check health status of a plugin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plugin_id": map[string]interface{}{"type": "string", "description": "Plugin ID to check"},
				},
				"required": []string{"plugin_id"}}},
		{Name: "get_all_plugin_stats", Title: "Get All Plugin Stats", Description: "Get statistics for all plugins",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{}}},
	}
}

// PluginMCPHandlers provides MCP tool handlers for plugin management
type PluginMCPHandlers struct {
	pluginService *PluginService
}

// NewPluginMCPHandlers creates new plugin MCP handlers
func NewPluginMCPHandlers(pluginService *PluginService) *PluginMCPHandlers {
	return &PluginMCPHandlers{
		pluginService: pluginService,
	}
}

// HandleTool handles plugin-related MCP tool calls
func (h *PluginMCPHandlers) HandleTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	switch toolName {
	case "list_plugins":
		return h.handleListPlugins(ctx, args)
	case "list_available_plugins":
		return h.handleListAvailablePlugins(ctx, args)
	case "get_plugin_status":
		return h.handleGetPluginStatus(ctx, args)
	case "get_plugin_providers":
		return h.handleGetPluginProviders(ctx)
	case "install_plugin":
		return h.handleInstallPlugin(ctx, args)
	case "configure_plugin":
		return h.handleConfigurePlugin(ctx, args)
	case "activate_plugin":
		return h.handleActivatePlugin(ctx, args)
	case "deactivate_plugin":
		return h.handleDeactivatePlugin(ctx, args)
	case "uninstall_plugin":
		return h.handleUninstallPlugin(ctx, args)
	case "check_plugin_health":
		return h.handleCheckPluginHealth(ctx, args)
	case "get_all_plugin_stats":
		return h.handleGetAllPluginStats(ctx)
	default:
		return nil, fmt.Errorf("unknown plugin tool: %s", toolName)
	}
}

// Handler implementations

func (h *PluginMCPHandlers) handleListPlugins(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	page := h.parseInt(args, "page", 1)
	pageSize := h.parseInt(args, "pageSize", 20)

	plugins, total, err := h.pluginService.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: h.formatJSON(map[string]interface{}{
				"plugins":  plugins,
				"total":    total,
				"page":     page,
				"pageSize": pageSize,
			})},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleListAvailablePlugins(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	query := h.parseStr(args, "query", "")
	capability := h.parseStr(args, "capability", "")

	var plugins []interface{}
	var err error

	if capability != "" {
		result, err := h.pluginService.ByCapability(ctx, capability)
		if err == nil {
			plugins = make([]interface{}, len(result))
			for i, p := range result {
				plugins[i] = p
			}
		}
	} else if query != "" {
		result, err := h.pluginService.Search(ctx, query)
		if err == nil {
			plugins = make([]interface{}, len(result))
			for i, p := range result {
				plugins[i] = p
			}
		}
	} else {
		result, err := h.pluginService.ListAvailable(ctx)
		if err == nil {
			plugins = make([]interface{}, len(result))
			for i, p := range result {
				plugins[i] = p
			}
		}
	}

	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: h.formatJSON(map[string]interface{}{
				"plugins": plugins,
				"count":   len(plugins),
			})},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleGetPluginStatus(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pluginID, ok := args["plugin_id"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_id is required")
	}

	status, err := h.pluginService.GetStats(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin status: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: h.formatJSON(status)},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleGetPluginProviders(ctx context.Context) (*mcp.CallToolResult, error) {
	providers, err := h.pluginService.GetProviders(ctx)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: h.formatJSON(map[string]interface{}{
				"providers": providers,
				"count":     len(providers),
			})},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleInstallPlugin(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pluginID, ok := args["plugin_id"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_id is required")
	}

	config, _ := args["config"].(map[string]interface{})
	activate := h.parseBool(args, "activate", false)

	installReq := &entity.PluginInstallRequest{
		PluginID: pluginID,
		Source:   "marketplace",
		Config:   config,
		Activate: activate,
	}

	status, err := h.pluginService.Install(ctx, installReq)
	if err != nil {
		return nil, fmt.Errorf("failed to install plugin: %w", err)
	}

	logger.Info("Plugin installed via MCP", zap.String("plugin_id", pluginID))

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Plugin %s installed successfully\n%s", pluginID, h.formatJSON(status))},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleConfigurePlugin(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pluginID, ok := args["plugin_id"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_id is required")
	}

	config, ok := args["config"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config is required")
	}

	err := h.pluginService.Configure(ctx, pluginID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to configure plugin: %w", err)
	}

	logger.Info("Plugin configured via MCP", zap.String("plugin_id", pluginID))

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Plugin %s configured successfully", pluginID)},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleActivatePlugin(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pluginID, ok := args["plugin_id"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_id is required")
	}

	err := h.pluginService.Activate(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to activate plugin: %w", err)
	}

	logger.Info("Plugin activated via MCP", zap.String("plugin_id", pluginID))

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Plugin %s activated", pluginID)},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleDeactivatePlugin(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pluginID, ok := args["plugin_id"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_id is required")
	}

	err := h.pluginService.Deactivate(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate plugin: %w", err)
	}

	logger.Info("Plugin deactivated via MCP", zap.String("plugin_id", pluginID))

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Plugin %s deactivated", pluginID)},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleUninstallPlugin(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pluginID, ok := args["plugin_id"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_id is required")
	}

	err := h.pluginService.Uninstall(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to uninstall plugin: %w", err)
	}

	logger.Info("Plugin uninstalled via MCP", zap.String("plugin_id", pluginID))

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Plugin %s uninstalled", pluginID)},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleCheckPluginHealth(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	pluginID, ok := args["plugin_id"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_id is required")
	}

	err := h.pluginService.HealthCheck(ctx, pluginID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.ContentBlock{
				{Type: "text", Text: fmt.Sprintf("Plugin %s health check failed: %v", pluginID, err)},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Plugin %s is healthy", pluginID)},
		},
	}, nil
}

func (h *PluginMCPHandlers) handleGetAllPluginStats(ctx context.Context) (*mcp.CallToolResult, error) {
	stats, err := h.pluginService.GetAllStats(ctx)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: h.formatJSON(map[string]interface{}{
				"stats": stats,
				"count": len(stats),
			})},
		},
	}, nil
}

// Helper functions

func (h *PluginMCPHandlers) parseInt(args map[string]interface{}, key string, defaultVal int) int {
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return defaultVal
}

func (h *PluginMCPHandlers) parseStr(args map[string]interface{}, key string, defaultVal string) string {
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultVal
}

func (h *PluginMCPHandlers) parseBool(args map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := args[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultVal
}

func (h *PluginMCPHandlers) formatJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(data)
}