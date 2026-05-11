package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/infrastructure/auth"
	"github.com/zhaojiewen/open-station/pkg/mcp"
)

// MCPService provides MCP protocol functionality
type MCPService struct {
	authService            *auth.AuthService
	billingService         *BillingService
	providerAccountService *ProviderAccountService
	pluginService          *PluginService
	pluginMCPHandlers      *PluginMCPHandlers
	costLimitService       *CostLimitService
	userAppService         *UserApplicationService
	tenantAppService       *TenantApplicationService
	budgetAlertService     *BudgetAlertService
	sessions               map[string]*MCPSession
	sessionMutex           sync.RWMutex
	sessionTimeout         time.Duration
}

// MCPSession represents an MCP session
type MCPSession struct {
	ID           string
	TenantID     uuid.UUID
	UserID       uuid.UUID
	APIKeyID     uuid.UUID
	Role         string // "user" or "manager"
	Capabilities ClientCapabilities
	CreatedAt    time.Time
	LastActive   time.Time
}

// ClientCapabilities from MCP protocol
type ClientCapabilities struct{}

// NewMCPService creates a new MCP service
func NewMCPService(
	authService *auth.AuthService,
	billingService *BillingService,
	providerAccountService *ProviderAccountService,
	pluginService *PluginService,
	costLimitService *CostLimitService,
	userAppService *UserApplicationService,
	tenantAppService *TenantApplicationService,
	budgetAlertService *BudgetAlertService,
) *MCPService {
	s := &MCPService{
		authService:            authService,
		billingService:         billingService,
		providerAccountService: providerAccountService,
		pluginService:          pluginService,
		costLimitService:       costLimitService,
		userAppService:         userAppService,
		tenantAppService:       tenantAppService,
		budgetAlertService:     budgetAlertService,
		sessions:               make(map[string]*MCPSession),
		sessionTimeout:         30 * time.Minute,
	}

	// Initialize plugin MCP handlers if plugin service is available
	if pluginService != nil {
		s.pluginMCPHandlers = NewPluginMCPHandlers(pluginService)
	}

	return s
}

// Initialize creates a new MCP session
func (s *MCPService) Initialize(ctx context.Context, apiKey string, clientInfo mcp.ImplementationInfo) (*mcp.InitializeResult, *MCPSession, error) {
	key, user, tenant, err := s.authService.ValidateAPIKey(ctx, apiKey)
	if err != nil {
		return nil, nil, fmt.Errorf("authentication failed: %w", err)
	}

	role := "user"
	var permissions []string
	if key.Permissions != "" {
		json.Unmarshal([]byte(key.Permissions), &permissions)
	}
	for _, perm := range permissions {
		if perm == "admin" || perm == "manage" {
			role = "manager"
			break
		}
	}

	sessionID := uuid.New().String()
	session := &MCPSession{
		ID:         sessionID,
		TenantID:   tenant.ID,
		UserID:     user.ID,
		APIKeyID:   key.ID,
		Role:       role,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	s.sessionMutex.Lock()
	s.sessions[sessionID] = session
	s.sessionMutex.Unlock()

	result := &mcp.InitializeResult{
		ProtocolVersion: "2025-11-25",
		Capabilities: mcp.ServerCapabilities{
			Tools:     &mcp.ToolsCapability{ListChanged: true},
			Resources: &mcp.ResourcesCapability{ListChanged: true},
			Prompts:   &mcp.PromptsCapability{},
		},
		ServerInfo: mcp.ImplementationInfo{
			Name:    "open-station",
			Version: "1.0.0",
		},
	}

	return result, session, nil
}

// GetSession retrieves a session by ID
func (s *MCPService) GetSession(sessionID string) (*MCPSession, error) {
	s.sessionMutex.RLock()
	session, ok := s.sessions[sessionID]
	s.sessionMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	if time.Since(session.LastActive) > s.sessionTimeout {
		s.sessionMutex.Lock()
		delete(s.sessions, sessionID)
		s.sessionMutex.Unlock()
		return nil, fmt.Errorf("session expired")
	}

	session.LastActive = time.Now()
	return session, nil
}

// ListTools returns all available tools based on session role
func (s *MCPService) ListTools(session *MCPSession) []mcp.Tool {
	tools := s.getUserTools()
	if session.Role == "manager" {
		tools = append(tools, s.getManagerTools()...)
	}
	return tools
}

// CallTool executes a tool call - public method for handler
func (s *MCPService) CallTool(ctx context.Context, session *MCPSession, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return s.executeTool(ctx, session, toolName, args)
}

// executeTool executes the actual tool logic
func (s *MCPService) executeTool(ctx context.Context, session *MCPSession, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// First try plugin tools if plugin handlers are available
	if s.pluginMCPHandlers != nil {
		if result, err := s.pluginMCPHandlers.HandleTool(ctx, toolName, args); err == nil {
			return result, nil
		} else if err.Error() != "unknown plugin tool: "+toolName {
			// Only return error if it's a known plugin tool that failed
			return result, err
		}
	}

	// Fall through to standard tools
	switch toolName {
	case "check_balance":
		return s.toolCheckBalance(ctx, session)
	case "get_usage_summary":
		return s.toolGetUsageSummary(ctx, session, args)
	case "get_usage_details":
		return s.toolGetUsageDetails(ctx, session, args)
	case "get_billing_info":
		return s.toolGetBillingInfo(ctx, session)
	case "get_recharge_history":
		return s.toolGetRechargeHistory(ctx, session, args)
	case "get_my_api_keys":
		return s.toolGetMyAPIKeys(ctx, session)
	case "list_all_api_keys":
		return s.toolListAllAPIKeys(ctx, args)
	case "create_api_key":
		return s.toolCreateAPIKey(ctx, args)
	case "revoke_api_key":
		return s.toolRevokeAPIKey(ctx, args)
	case "update_api_key":
		return s.toolUpdateAPIKey(ctx, args)
	case "list_users":
		return s.toolListUsers(ctx, args)
	case "get_user_detail":
		return s.toolGetUserDetail(ctx, args)
	case "adjust_balance":
		return s.toolAdjustBalance(ctx, args)
	case "get_tenant_summary":
		return s.toolGetTenantSummary(ctx, args)
	case "list_tenants":
		return s.toolListTenants(ctx, args)
	case "list_provider_accounts":
		return s.toolListProviderAccounts(ctx, args)
	case "create_provider_account":
		return s.toolCreateProviderAccount(ctx, args)
	case "update_provider_account":
		return s.toolUpdateProviderAccount(ctx, args)
	case "set_default_provider_account":
		return s.toolSetDefaultProviderAccount(ctx, args)
	case "enable_provider_account":
		return s.toolEnableProviderAccount(ctx, args)
	case "disable_provider_account":
		return s.toolDisableProviderAccount(ctx, args)
	case "delete_provider_account":
		return s.toolDeleteProviderAccount(ctx, args)
	case "get_provider_status":
		return s.toolGetProviderStatus(ctx, args)
	// Budget and cost limit tools
	case "set_user_budget":
			return s.toolSetUserBudget(ctx, args)
		case "get_user_budget_usage":
			return s.toolGetUserBudgetUsage(ctx, args)
		case "set_api_key_cost_limit":
			return s.toolSetAPIKeyCostLimit(ctx, args)
		case "get_api_key_cost_usage":
			return s.toolGetAPIKeyCostUsage(ctx, args)
		case "set_tenant_budget":
			return s.toolSetTenantBudget(ctx, args)
		case "get_cost_summary":
			return s.toolGetCostSummary(ctx, args)
		// Budget alert tools
		case "create_budget_alert":
			return s.toolCreateBudgetAlert(ctx, args)
		case "list_budget_alerts":
			return s.toolListBudgetAlerts(ctx, args)
		case "update_budget_alert":
			return s.toolUpdateBudgetAlert(ctx, args)
		case "delete_budget_alert":
			return s.toolDeleteBudgetAlert(ctx, args)
		case "enable_budget_alert":
			return s.toolEnableBudgetAlert(ctx, args)
		case "disable_budget_alert":
			return s.toolDisableBudgetAlert(ctx, args)
		// User application tools
		case "send_user_invitation":
			return s.toolSendUserInvitation(ctx, args)
		case "list_user_applications":
			return s.toolListUserApplications(ctx, args)
		case "approve_user_application":
			return s.toolApproveUserApplication(ctx, args)
		case "reject_user_application":
			return s.toolRejectUserApplication(ctx, args)
		case "cancel_user_invitation":
			return s.toolCancelUserInvitation(ctx, args)
		case "create_user_direct":
			return s.toolCreateUserDirect(ctx, args)
		// Tenant application tools (platform admin)
		case "list_tenant_applications":
			return s.toolListTenantApplications(ctx, args)
		case "approve_tenant_application":
			return s.toolApproveTenantApplication(ctx, args)
		case "reject_tenant_application":
			return s.toolRejectTenantApplication(ctx, args)
		case "suspend_tenant":
			return s.toolSuspendTenant(ctx, args)
		case "activate_tenant":
			return s.toolActivateTenant(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// Helper methods

func (s *MCPService) parseDateRange(args map[string]interface{}) (time.Time, time.Time) {
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endDate := now

	if start, ok := args["start_date"].(string); ok {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = t
		}
	}
	if end, ok := args["end_date"].(string); ok {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			endDate = t
		}
	}
	return startDate, endDate
}

func (s *MCPService) parseLimit(args map[string]interface{}, defaultVal int) int {
	if limit, ok := args["limit"]; ok {
		switch v := limit.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
	}
	return defaultVal
}

func (s *MCPService) parseInt(val interface{}, defaultVal int) int {
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func (s *MCPService) parseStringArray(val interface{}) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		var result []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case string:
		if v != "" {
			return strings.Split(v, ",")
		}
	}
	return nil
}

// CleanupSessions removes expired sessions
func (s *MCPService) CleanupSessions() {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.Sub(session.LastActive) > s.sessionTimeout {
			delete(s.sessions, id)
		}
	}
}