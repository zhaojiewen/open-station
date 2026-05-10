package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/pkg/mcp"
)

// Provider Account MCP Tool Implementations

func (s *MCPService) toolListProviderAccounts(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.providerAccountService == nil {
		return nil, fmt.Errorf("provider account service not initialized")
	}

	provider, _ := args["provider"].(string)

	if provider != "" {
		accounts, _, err := s.providerAccountService.ListAccounts(ctx, provider, 1, 100)
		if err != nil {
			return nil, err
		}

		result := make([]map[string]interface{}, len(accounts))
		for i, acc := range accounts {
			result[i] = map[string]interface{}{
				"id":             acc.ID.String(),
				"provider":       acc.Provider,
				"name":           acc.Name,
				"status":         acc.Status,
				"is_default":     acc.IsDefault,
				"priority":       acc.Priority,
				"used_this_month": acc.UsedThisMonth.StringFixed(2),
				"error_count":    acc.ErrorCount,
				"last_used":      acc.LastUsedAt,
			}
		}

		jsonData, _ := json.MarshalIndent(map[string]interface{}{
			"provider": provider,
			"accounts": result,
			"count":    len(result),
		}, "", "  ")
		return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
	}

	// List all providers
	statusResult, err := s.providerAccountService.GetAllProvidersStatus(ctx)
	if err != nil {
		return nil, err
	}

	jsonData, _ := json.MarshalIndent(statusResult, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolCreateProviderAccount(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.providerAccountService == nil {
		return nil, fmt.Errorf("provider account service not initialized")
	}

	provider, ok := args["provider"].(string)
	if !ok || provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	apiKey, ok := args["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	baseURL, _ := args["base_url"].(string)
	priority := s.parseInt(args["priority"], 0)

	var monthlyLimit *decimal.Decimal
	if limit, ok := args["monthly_limit"].(float64); ok {
		d := decimal.NewFromFloat(limit)
		monthlyLimit = &d
	}

	account, err := s.providerAccountService.CreateAccount(ctx, provider, name, apiKey, baseURL, priority, monthlyLimit)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":           account.ID.String(),
		"provider":     account.Provider,
		"name":         account.Name,
		"status":       account.Status,
		"is_default":   account.IsDefault,
		"priority":     account.Priority,
		"created_at":   account.CreatedAt,
		"message":      fmt.Sprintf("Provider account created successfully. %s is now available for %s.", name, provider),
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolUpdateProviderAccount(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.providerAccountService == nil {
		return nil, fmt.Errorf("provider account service not initialized")
	}

	accountIDStr, ok := args["account_id"].(string)
	if !ok || accountIDStr == "" {
		return nil, fmt.Errorf("account_id is required")
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	name, _ := args["name"].(string)
	apiKey, _ := args["api_key"].(string)
	baseURL, _ := args["base_url"].(string)
	priority := -1
	if p, ok := args["priority"]; ok {
		priority = s.parseInt(p, -1)
	}

	var monthlyLimit *decimal.Decimal
	if limit, ok := args["monthly_limit"].(float64); ok {
		d := decimal.NewFromFloat(limit)
		monthlyLimit = &d
	}

	account, err := s.providerAccountService.UpdateAccount(ctx, accountID, name, apiKey, baseURL, priority, monthlyLimit)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":           account.ID.String(),
		"provider":     account.Provider,
		"name":         account.Name,
		"status":       account.Status,
		"updated_at":   account.UpdatedAt,
		"message":      "Provider account updated successfully",
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolSetDefaultProviderAccount(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.providerAccountService == nil {
		return nil, fmt.Errorf("provider account service not initialized")
	}

	provider, ok := args["provider"].(string)
	if !ok || provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	accountIDStr, ok := args["account_id"].(string)
	if !ok || accountIDStr == "" {
		return nil, fmt.Errorf("account_id is required")
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	err = s.providerAccountService.SetDefaultAccount(ctx, provider, accountID)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: fmt.Sprintf("Account %s set as default for provider %s", accountIDStr, provider)}}}, nil
}

func (s *MCPService) toolEnableProviderAccount(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.providerAccountService == nil {
		return nil, fmt.Errorf("provider account service not initialized")
	}

	accountIDStr, ok := args["account_id"].(string)
	if !ok || accountIDStr == "" {
		return nil, fmt.Errorf("account_id is required")
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	err = s.providerAccountService.EnableAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: fmt.Sprintf("Provider account %s enabled", accountIDStr)}}}, nil
}

func (s *MCPService) toolDisableProviderAccount(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.providerAccountService == nil {
		return nil, fmt.Errorf("provider account service not initialized")
	}

	accountIDStr, ok := args["account_id"].(string)
	if !ok || accountIDStr == "" {
		return nil, fmt.Errorf("account_id is required")
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	err = s.providerAccountService.DisableAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: fmt.Sprintf("Provider account %s disabled. System will switch to next available account.", accountIDStr)}}}, nil
}

func (s *MCPService) toolDeleteProviderAccount(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.providerAccountService == nil {
		return nil, fmt.Errorf("provider account service not initialized")
	}

	accountIDStr, ok := args["account_id"].(string)
	if !ok || accountIDStr == "" {
		return nil, fmt.Errorf("account_id is required")
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	err = s.providerAccountService.DeleteAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: fmt.Sprintf("Provider account %s deleted permanently", accountIDStr)}}}, nil
}

func (s *MCPService) toolGetProviderStatus(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.providerAccountService == nil {
		return nil, fmt.Errorf("provider account service not initialized")
	}

	provider, _ := args["provider"].(string)

	if provider != "" {
		status, err := s.providerAccountService.GetProviderStatus(ctx, provider)
		if err != nil {
			return nil, err
		}

		jsonData, _ := json.MarshalIndent(status, "", "  ")
		return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
	}

	// All providers status
	status, err := s.providerAccountService.GetAllProvidersStatus(ctx)
	if err != nil {
		return nil, err
	}

	jsonData, _ := json.MarshalIndent(status, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}