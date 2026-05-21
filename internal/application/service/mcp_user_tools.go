package service

// User tool implementations for MCP service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zhaojiewen/open-station/pkg/mcp"
)

func (s *MCPService) toolCheckBalance(ctx context.Context, session *MCPSession) (*mcp.CallToolResult, error) {
	balance, err := s.billingService.CheckBalance(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	text := fmt.Sprintf("Current Balance: $%s", balance.StringFixed(2))
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: text}}}, nil
}

func (s *MCPService) toolGetUsageSummary(ctx context.Context, session *MCPSession, args map[string]interface{}) (*mcp.CallToolResult, error) {
	startDate, endDate := s.parseDateRange(args)
	totalCost, totalTokens, err := s.billingService.GetTotalCost(ctx, session.TenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	text := fmt.Sprintf("Usage Summary (%s to %s):\n- Total Tokens: %d\n- Total Cost: $%s",
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), totalTokens, totalCost.StringFixed(2))
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: text}}}, nil
}

func (s *MCPService) toolGetUsageDetails(ctx context.Context, session *MCPSession, args map[string]interface{}) (*mcp.CallToolResult, error) {
	startDate, endDate := s.parseDateRange(args)
	limit := s.parseLimit(args, 50)
	records, total, err := s.billingService.GetUsage(ctx, session.TenantID, startDate, endDate, 1, limit)
	if err != nil {
		return nil, err
	}
	var details []map[string]interface{}
	for _, r := range records {
		details = append(details, map[string]interface{}{
			"request_id":        r.RequestID,
			"provider":          r.Provider,
			"model":             r.ModelID,
			"prompt_tokens":     r.PromptTokens,
			"completion_tokens": r.CompletionTokens,
			"total_tokens":      r.TotalTokens,
			"cost":              r.Cost.String(),
			"timestamp":         r.CreatedAt,
		})
	}
	result := map[string]interface{}{"total_count": total, "records": details}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolGetBillingInfo(ctx context.Context, session *MCPSession) (*mcp.CallToolResult, error) {
	tenant, err := s.authService.GetTenantByID(ctx, session.TenantID)
	if err != nil {
		return nil, err
	}
	info := map[string]interface{}{
		"tenant_id":   tenant.ID.String(),
		"tenant_name": tenant.Name,
		"balance":     tenant.Balance.String(),
		"currency":    "USD",
	}
	jsonData, _ := json.MarshalIndent(info, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolGetRechargeHistory(ctx context.Context, session *MCPSession, args map[string]interface{}) (*mcp.CallToolResult, error) {
	limit := s.parseLimit(args, 20)
	records, total, err := s.billingService.GetRechargeRecords(ctx, session.TenantID, 1, limit)
	if err != nil {
		return nil, err
	}
	var history []map[string]interface{}
	for _, r := range records {
		history = append(history, map[string]interface{}{
			"id":             r.ID.String(),
			"amount":         r.Amount.String(),
			"payment_method": r.PaymentMethod,
			"status":         r.Status,
			"created_at":     r.CreatedAt,
			"completed_at":   r.CompletedAt,
			"notes":          r.Notes,
		})
	}
	result := map[string]interface{}{"total_count": total, "records": history}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolGetMyAPIKeys(ctx context.Context, session *MCPSession) (*mcp.CallToolResult, error) {
	keys, err := s.authService.ListAPIKeys(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	var keyList []map[string]interface{}
	for _, k := range keys {
		keyList = append(keyList, map[string]interface{}{
			"id":          k.ID.String(),
			"key_prefix":  k.KeyPrefix,
			"name":        k.Name,
			"status":      k.Status,
			"permissions": k.Permissions,
			"created_at":  k.CreatedAt,
			"last_used":   k.LastUsedAt,
		})
	}
	result := map[string]interface{}{"count": len(keyList), "keys": keyList}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}