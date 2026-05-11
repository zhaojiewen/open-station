package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/pkg/mcp"
)

// Budget and cost limit tool implementations

func (s *MCPService) toolSetUserBudget(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.costLimitService == nil {
		return nil, fmt.Errorf("cost limit service not available")
	}

	userIDStr, ok := args["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user_id is required")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	var monthlyBudget, dailyBudget decimal.Decimal
	if mb, ok := args["monthly_budget"].(float64); ok {
		monthlyBudget = decimal.NewFromFloat(mb)
	}
	if db, ok := args["daily_budget"].(float64); ok {
		dailyBudget = decimal.NewFromFloat(db)
	}

	var tokenQuota int64
	if tq, ok := args["token_quota"].(float64); ok {
		tokenQuota = int64(tq)
	} else if tq, ok := args["token_quota"].(int); ok {
		tokenQuota = int64(tq)
	}

	if err := s.costLimitService.SetUserBudget(ctx, userID, monthlyBudget, dailyBudget, tokenQuota); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("User budget updated successfully for user %s", userID)},
		},
	}, nil
}

func (s *MCPService) toolGetUserBudgetUsage(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.costLimitService == nil {
		return nil, fmt.Errorf("cost limit service not available")
	}

	userIDStr, ok := args["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user_id is required")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id")
	}

	usage, err := s.costLimitService.GetUserBudgetUsage(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("User budget usage: Monthly Used: %s, Daily Used: %s, Tokens Used: %d", usage.MonthlyUsed.StringFixed(2), usage.DailyUsed.StringFixed(2), usage.TokensUsed)},
		},
	}, nil
}

func (s *MCPService) toolSetAPIKeyCostLimit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.costLimitService == nil {
		return nil, fmt.Errorf("cost limit service not available")
	}

	apiKeyIDStr, ok := args["api_key_id"].(string)
	if !ok {
		return nil, fmt.Errorf("api_key_id is required")
	}
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid api_key_id")
	}

	var monthlyLimit, dailyLimit, perRequestLimit decimal.Decimal
	if ml, ok := args["monthly_cost_limit"].(float64); ok {
		monthlyLimit = decimal.NewFromFloat(ml)
	}
	if dl, ok := args["daily_cost_limit"].(float64); ok {
		dailyLimit = decimal.NewFromFloat(dl)
	}
	if prl, ok := args["per_request_limit"].(float64); ok {
		perRequestLimit = decimal.NewFromFloat(prl)
	}

	var monthlyTokenLimit, dailyTokenLimit int64
	if mtl, ok := args["monthly_token_limit"].(float64); ok {
		monthlyTokenLimit = int64(mtl)
	} else if mtl, ok := args["monthly_token_limit"].(int); ok {
		monthlyTokenLimit = int64(mtl)
	}
	if dtl, ok := args["daily_token_limit"].(float64); ok {
		dailyTokenLimit = int64(dtl)
	} else if dtl, ok := args["daily_token_limit"].(int); ok {
		dailyTokenLimit = int64(dtl)
	}

	alertThreshold1 := s.parseInt(args["alert_threshold_1"], 80)
	alertThreshold2 := s.parseInt(args["alert_threshold_2"], 90)
	alertThreshold3 := s.parseInt(args["alert_threshold_3"], 100)

	if err := s.costLimitService.SetAPIKeyCostLimit(ctx, apiKeyID, monthlyLimit, dailyLimit, perRequestLimit, monthlyTokenLimit, dailyTokenLimit, alertThreshold1, alertThreshold2, alertThreshold3); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("API key cost limit updated successfully for key %s", apiKeyID)},
		},
	}, nil
}

func (s *MCPService) toolGetAPIKeyCostUsage(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.costLimitService == nil {
		return nil, fmt.Errorf("cost limit service not available")
	}

	apiKeyIDStr, ok := args["api_key_id"].(string)
	if !ok {
		return nil, fmt.Errorf("api_key_id is required")
	}
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid api_key_id")
	}

	usage, err := s.costLimitService.GetAPIKeyCostUsage(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("API key cost usage: Monthly Cost Used: %s, Daily Cost Used: %s, Monthly Tokens Used: %d, Daily Tokens Used: %d", usage.MonthlyCostUsed.StringFixed(2), usage.DailyCostUsed.StringFixed(2), usage.MonthlyTokensUsed, usage.DailyTokensUsed)},
		},
	}, nil
}

func (s *MCPService) toolSetTenantBudget(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.costLimitService == nil {
		return nil, fmt.Errorf("cost limit service not available")
	}

	tenantIDStr, ok := args["tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("tenant_id is required")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id")
	}

	var monthlyBudget decimal.Decimal
	if mb, ok := args["monthly_budget_limit"].(float64); ok {
		monthlyBudget = decimal.NewFromFloat(mb)
	}

	var tokenLimit int64
	if tl, ok := args["token_limit"].(float64); ok {
		tokenLimit = int64(tl)
	} else if tl, ok := args["token_limit"].(int); ok {
		tokenLimit = int64(tl)
	}

	if err := s.costLimitService.SetTenantBudget(ctx, tenantID, monthlyBudget, tokenLimit); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Tenant budget updated successfully for tenant %s", tenantID)},
		},
	}, nil
}

func (s *MCPService) toolGetCostSummary(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.costLimitService == nil {
		return nil, fmt.Errorf("cost limit service not available")
	}

	apiKeyIDStr, ok := args["api_key_id"].(string)
	if !ok {
		return nil, fmt.Errorf("api_key_id is required")
	}
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid api_key_id")
	}

	var userID, tenantID uuid.UUID
	if uid, ok := args["user_id"].(string); ok {
		userID, _ = uuid.Parse(uid)
	}
	if tid, ok := args["tenant_id"].(string); ok {
		tenantID, _ = uuid.Parse(tid)
	}

	summary, err := s.costLimitService.GetCostSummary(ctx, apiKeyID, userID, tenantID)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Cost summary: Tenant Monthly Used: %s, Tenant Balance: %s, User Monthly Used: %s, API Key Monthly Used: %s", summary.TenantMonthlyUsed.StringFixed(2), summary.TenantBalance.StringFixed(2), summary.UserMonthlyUsed.StringFixed(2), summary.APIKeyMonthlyCostUsed.StringFixed(2))},
		},
	}, nil
}

// Budget alert tool implementations

func (s *MCPService) toolCreateBudgetAlert(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.budgetAlertService == nil {
		return nil, fmt.Errorf("budget alert service not available")
	}

	scope, ok := args["scope"].(string)
	if !ok {
		return nil, fmt.Errorf("scope is required")
	}
	scopeIDStr, ok := args["scope_id"].(string)
	if !ok {
		return nil, fmt.Errorf("scope_id is required")
	}
	scopeID, err := uuid.Parse(scopeIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid scope_id")
	}

	alertType, _ := args["alert_type"].(string)
	thresholdPercent := s.parseInt(args["threshold_percent"], 80)
	notifyEmails := s.parseStringArray(args["notify_emails"])
	notifySlack, _ := args["notify_slack"].(string)
	notifyWebhook, _ := args["notify_webhook"].(string)

	alert, err := s.budgetAlertService.Create(ctx, scope, scopeID, alertType, thresholdPercent, notifyEmails, notifySlack, notifyWebhook)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Budget alert created successfully with ID: %s", alert.ID)},
		},
	}, nil
}

func (s *MCPService) toolListBudgetAlerts(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.budgetAlertService == nil {
		return nil, fmt.Errorf("budget alert service not available")
	}

	scope, _ := args["scope"].(string)
	scopeIDStr, _ := args["scope_id"].(string)
	page := s.parseInt(args["page"], 1)
	limit := s.parseInt(args["limit"], 20)

	if scope != "" && scopeIDStr != "" {
		scopeID, err := uuid.Parse(scopeIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid scope_id")
		}
		alerts, err := s.budgetAlertService.GetByScope(ctx, scope, scopeID)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.ContentBlock{
				{Type: "text", Text: fmt.Sprintf("Found %d alerts for scope %s", len(alerts), scope)},
			},
		}, nil
	}

	alerts, total, err := s.budgetAlertService.List(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Found %d alerts (total: %d, page: %d)", len(alerts), total, page)},
		},
	}, nil
}

func (s *MCPService) toolUpdateBudgetAlert(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.budgetAlertService == nil {
		return nil, fmt.Errorf("budget alert service not available")
	}

	alertIDStr, ok := args["alert_id"].(string)
	if !ok {
		return nil, fmt.Errorf("alert_id is required")
	}
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid alert_id")
	}

	thresholdPercent := s.parseInt(args["threshold_percent"], 80)
	notifyEmails := s.parseStringArray(args["notify_emails"])
	notifySlack, _ := args["notify_slack"].(string)
	notifyWebhook, _ := args["notify_webhook"].(string)

	if err := s.budgetAlertService.Update(ctx, alertID, thresholdPercent, notifyEmails, notifySlack, notifyWebhook); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Budget alert %s updated successfully", alertID)},
		},
	}, nil
}

func (s *MCPService) toolDeleteBudgetAlert(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.budgetAlertService == nil {
		return nil, fmt.Errorf("budget alert service not available")
	}

	alertIDStr, ok := args["alert_id"].(string)
	if !ok {
		return nil, fmt.Errorf("alert_id is required")
	}
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid alert_id")
	}

	if err := s.budgetAlertService.Delete(ctx, alertID); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Budget alert %s deleted successfully", alertID)},
		},
	}, nil
}

func (s *MCPService) toolEnableBudgetAlert(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.budgetAlertService == nil {
		return nil, fmt.Errorf("budget alert service not available")
	}

	alertIDStr, ok := args["alert_id"].(string)
	if !ok {
		return nil, fmt.Errorf("alert_id is required")
	}
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid alert_id")
	}

	if err := s.budgetAlertService.Enable(ctx, alertID); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Budget alert %s enabled successfully", alertID)},
		},
	}, nil
}

func (s *MCPService) toolDisableBudgetAlert(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.budgetAlertService == nil {
		return nil, fmt.Errorf("budget alert service not available")
	}

	alertIDStr, ok := args["alert_id"].(string)
	if !ok {
		return nil, fmt.Errorf("alert_id is required")
	}
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid alert_id")
	}

	if err := s.budgetAlertService.Disable(ctx, alertID); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Budget alert %s disabled successfully", alertID)},
		},
	}, nil
}