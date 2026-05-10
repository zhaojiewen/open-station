package service

// Manager tool implementations for MCP service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/pkg/mcp"
)

func (s *MCPService) toolListAllAPIKeys(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	var tenantID uuid.UUID
	if tid, ok := args["tenant_id"].(string); ok && tid != "" {
		tenantID, _ = uuid.Parse(tid)
	}
	status := "active"
	if s, ok := args["status"].(string); ok {
		status = s
	}
	keys, err := s.authService.ListAPIKeysByTenant(ctx, tenantID, status)
	if err != nil {
		return nil, err
	}
	var keyList []map[string]interface{}
	for _, k := range keys {
		keyList = append(keyList, map[string]interface{}{
			"id":          k.ID.String(),
			"key_prefix":  k.KeyPrefix,
			"name":        k.Name,
			"user_id":     k.UserID.String(),
			"tenant_id":   k.TenantID.String(),
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

func (s *MCPService) toolCreateAPIKey(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get or create user
	var userID uuid.UUID
	var tenantID uuid.UUID
	var isNewUser bool

	// Check if user_id provided
	if userIDStr, ok := args["user_id"].(string); ok && userIDStr != "" {
		parsedID, parseErr := uuid.Parse(userIDStr)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid user_id: %w", parseErr)
		}
		existingUser, findErr := s.authService.GetUserByID(ctx, parsedID)
		if findErr != nil {
			return nil, fmt.Errorf("user not found: %w", findErr)
		}
		userID = existingUser.ID
		tenantID = existingUser.TenantID
	} else {
		// Auto-create user from email/name
		userEmail, ok := args["user_email"].(string)
		if !ok || userEmail == "" {
			return nil, fmt.Errorf("user_id or user_email is required")
		}

		userName, _ := args["user_name"].(string)
		if userName == "" {
			userName = userEmail // Use email as name if not provided
		}

		// Check if user already exists by email
		existingUser, err := s.authService.GetUserByEmail(ctx, userEmail)
		if err == nil {
			// User exists, use existing
			userID = existingUser.ID
			tenantID = existingUser.TenantID
		} else {
			// Create new user
			isNewUser = true

			// Get tenant
			if tenantIDStr, ok := args["tenant_id"].(string); ok && tenantIDStr != "" {
				tenantID, err = uuid.Parse(tenantIDStr)
				if err != nil {
					return nil, fmt.Errorf("invalid tenant_id: %w", err)
				}
			} else {
				// Use default tenant (first tenant or create one)
				tenants, _, err := s.authService.ListTenants(ctx, 1, 1)
				if err != nil || len(tenants) == 0 {
					return nil, fmt.Errorf("no tenant found, please create a tenant first")
				}
				tenantID = tenants[0].ID
			}

			newUser := &entity.User{
				ID:        uuid.New(),
				TenantID:  tenantID,
				Email:     userEmail,
				Name:      userName,
				Role:      "member",
				Status:    "active",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := s.authService.CreateUser(ctx, newUser); err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
			userID = newUser.ID
		}
	}

	// Get API key name
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Parse permissions
	permissions := s.parseStringArray(args["permissions"])
	if len(permissions) == 0 {
		permissions = []string{"chat"}
	}
	models := s.parseStringArray(args["models"])
	providers := s.parseStringArray(args["providers"])

	key, rawKey, err := s.authService.CreateAPIKey(ctx, userID, tenantID, name, permissions, models, providers, nil, nil)
	if err != nil {
		return nil, err
	}

	// Get user info for response
	user, _ := s.authService.GetUserByID(ctx, userID)

	result := map[string]interface{}{
		"id":          key.ID.String(),
		"key_prefix":  key.KeyPrefix,
		"raw_key":     rawKey,
		"name":        key.Name,
		"permissions": key.Permissions,
		"created_at":  key.CreatedAt,
		"user_id":     userID.String(),
		"user_email":  user.Email,
		"user_name":   user.Name,
		"is_new_user": isNewUser,
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolRevokeAPIKey(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	keyIDStr, ok := args["api_key_id"].(string)
	if !ok {
		return nil, fmt.Errorf("api_key_id is required")
	}
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid api_key_id: %w", err)
	}

	err = s.authService.RevokeAPIKey(ctx, keyID)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: fmt.Sprintf("API key %s has been revoked", keyIDStr)}}}, nil
}

func (s *MCPService) toolUpdateAPIKey(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	keyIDStr, ok := args["api_key_id"].(string)
	if !ok {
		return nil, fmt.Errorf("api_key_id is required")
	}
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid api_key_id: %w", err)
	}

	key, err := s.authService.GetAPIKeyByID(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("api key not found: %w", err)
	}

	if perms, ok := args["permissions"]; ok {
		permsArray := s.parseStringArray(perms)
		permsJSON, _ := json.Marshal(permsArray)
		key.Permissions = string(permsJSON)
	}
	if models, ok := args["models"]; ok {
		modelsArray := s.parseStringArray(models)
		modelsJSON, _ := json.Marshal(modelsArray)
		key.AllowedModels = string(modelsJSON)
	}
	if providers, ok := args["providers"]; ok {
		providersArray := s.parseStringArray(providers)
		providersJSON, _ := json.Marshal(providersArray)
		key.AllowedProviders = string(providersJSON)
	}

	err = s.authService.UpdateAPIKey(ctx, key)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":          key.ID.String(),
		"key_prefix":  key.KeyPrefix,
		"name":        key.Name,
		"permissions": key.Permissions,
		"models":      key.AllowedModels,
		"providers":   key.AllowedProviders,
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolListUsers(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	var tenantID uuid.UUID
	if tid, ok := args["tenant_id"].(string); ok && tid != "" {
		tenantID, _ = uuid.Parse(tid)
	}
	page := s.parseInt(args["page"], 1)
	limit := s.parseLimit(args, 20)

	users, total, err := s.authService.ListUsers(ctx, tenantID, page, limit)
	if err != nil {
		return nil, err
	}
	var userList []map[string]interface{}
	for _, u := range users {
		userList = append(userList, map[string]interface{}{
			"id":         u.ID.String(),
			"tenant_id":  u.TenantID.String(),
			"email":      u.Email,
			"name":       u.Name,
			"role":       u.Role,
			"status":     u.Status,
			"created_at": u.CreatedAt,
			"last_login": u.LastLoginAt,
		})
	}
	result := map[string]interface{}{"total_count": total, "page": page, "limit": limit, "users": userList}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolGetUserDetail(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	userIDStr, ok := args["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user_id is required")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	user, err := s.authService.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	keys, err := s.authService.ListAPIKeys(ctx, userID)
	if err != nil {
		keys = []entity.APIKey{}
	}

	balance, err := s.billingService.CheckBalance(ctx, user.TenantID)
	if err != nil {
		balance = decimal.Zero
	}

	result := map[string]interface{}{
		"user": map[string]interface{}{
			"id":         user.ID.String(),
			"tenant_id":  user.TenantID.String(),
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"status":     user.Status,
			"created_at": user.CreatedAt,
			"last_login": user.LastLoginAt,
		},
		"tenant_balance": balance.String(),
		"api_keys_count": len(keys),
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolAdjustBalance(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	tenantIDStr, ok := args["tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("tenant_id is required")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	amountFloat, ok := args["amount"].(float64)
	if !ok {
		amountInt, ok := args["amount"].(int)
		if !ok {
			return nil, fmt.Errorf("amount is required and must be a number")
		}
		amountFloat = float64(amountInt)
	}
	amount := decimal.NewFromFloat(amountFloat)

	reason, ok := args["reason"].(string)
	if !ok {
		reason = "Manual adjustment via MCP"
	}

	err = s.authService.UpdateTenantBalance(ctx, tenantID, amount)
	if err != nil {
		return nil, err
	}

	newBalance, err := s.billingService.CheckBalance(ctx, tenantID)
	if err != nil {
		newBalance = decimal.Zero
	}

	result := map[string]interface{}{
		"tenant_id":   tenantIDStr,
		"adjustment":  amount.String(),
		"reason":      reason,
		"new_balance": newBalance.String(),
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolGetTenantSummary(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	tenantIDStr, ok := args["tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("tenant_id is required")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	tenant, err := s.authService.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	totalCost, totalTokens, err := s.billingService.GetTotalCost(ctx, tenantID, startOfMonth, now)
	if err != nil {
		totalCost = decimal.Zero
		totalTokens = 0
	}

	_, userCount, err := s.authService.ListUsers(ctx, tenantID, 1, 1000)
	if err != nil {
		userCount = 0
	}

	keys, err := s.authService.ListAPIKeysByTenant(ctx, tenantID, "active")
	keyCount := 0
	if err == nil {
		keyCount = len(keys)
	}

	result := map[string]interface{}{
		"tenant": map[string]interface{}{
			"id":     tenant.ID.String(),
			"name":   tenant.Name,
			"slug":   tenant.Slug,
			"status": tenant.Status,
		},
		"balance": tenant.Balance.String(),
		"monthly_usage": map[string]interface{}{
			"tokens": totalTokens,
			"cost":   totalCost.String(),
		},
		"users_count":    userCount,
		"api_keys_count": keyCount,
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

func (s *MCPService) toolListTenants(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	page := s.parseInt(args["page"], 1)
	limit := s.parseLimit(args, 20)

	tenants, total, err := s.authService.ListTenants(ctx, page, limit)
	if err != nil {
		return nil, err
	}
	var tenantList []map[string]interface{}
	for _, t := range tenants {
		tenantList = append(tenantList, map[string]interface{}{
			"id":         t.ID.String(),
			"name":       t.Name,
			"slug":       t.Slug,
			"status":     t.Status,
			"balance":    t.Balance.String(),
			"created_at": t.CreatedAt,
		})
	}
	result := map[string]interface{}{"total_count": total, "page": page, "limit": limit, "tenants": tenantList}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.ContentBlock{{Type: "text", Text: string(jsonData)}}}, nil
}

