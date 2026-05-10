package service

// Resource handling for MCP service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/pkg/mcp"
)

// ListResources returns available resources
func (s *MCPService) ListResources(session *MCPSession) []mcp.Resource {
	resources := []mcp.Resource{
		{URI: "user://profile", Name: "User Profile", MimeType: "application/json"},
		{URI: "user://balance", Name: "User Balance", MimeType: "application/json"},
		{URI: "user://usage", Name: "User Usage Records", MimeType: "application/json"},
	}
	if session.Role == "manager" {
		resources = append(resources,
			mcp.Resource{URI: "tenant://list", Name: "All Tenants", MimeType: "application/json"},
			mcp.Resource{URI: "apikey://list", Name: "All API Keys", MimeType: "application/json"},
		)
	}
	return resources
}

// ReadResource reads a resource by URI
func (s *MCPService) ReadResource(ctx context.Context, session *MCPSession, uri string) (*mcp.ReadResourceResult, error) {
	parts := strings.Split(uri, "://")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}

	resourceType := parts[0]
	resourceID := parts[1]

	var content string
	var err error

	switch resourceType {
	case "user":
		content, err = s.readUserResource(ctx, session, resourceID)
	case "tenant":
		if session.Role != "manager" {
			return nil, fmt.Errorf("permission denied: tenant resources require manager role")
		}
		content, err = s.readTenantResource(ctx, resourceID)
	case "apikey":
		if session.Role != "manager" {
			return nil, fmt.Errorf("permission denied: apikey resources require manager role")
		}
		content, err = s.readAPIKeyResource(ctx)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	if err != nil {
		return nil, err
	}

	return &mcp.ReadResourceResult{
		Contents: []mcp.ResourceContents{{URI: uri, MimeType: "application/json", Text: content}},
	}, nil
}

func (s *MCPService) readUserResource(ctx context.Context, session *MCPSession, resourceID string) (string, error) {
	switch resourceID {
	case "profile":
		user, err := s.authService.GetUserByID(ctx, session.UserID)
		if err != nil {
			return "", err
		}
		data := map[string]interface{}{
			"id":         user.ID.String(),
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		return string(jsonData), nil

	case "balance":
		balance, err := s.billingService.CheckBalance(ctx, session.TenantID)
		if err != nil {
			return "", err
		}
		data := map[string]interface{}{
			"tenant_id": session.TenantID.String(),
			"balance":   balance.String(),
			"currency":  "USD",
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		return string(jsonData), nil

	case "usage":
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		records, _, err := s.billingService.GetUsage(ctx, session.TenantID, startOfMonth, now, 1, 50)
		if err != nil {
			return "", err
		}
		var usageList []map[string]interface{}
		for _, r := range records {
			usageList = append(usageList, map[string]interface{}{
				"request_id": r.RequestID,
				"provider":   r.Provider,
				"model":      r.ModelID,
				"tokens":     r.TotalTokens,
				"cost":       r.Cost.String(),
				"timestamp":  r.CreatedAt,
			})
		}
		jsonData, _ := json.MarshalIndent(usageList, "", "  ")
		return string(jsonData), nil

	default:
		return "", fmt.Errorf("unknown user resource: %s", resourceID)
	}
}

func (s *MCPService) readTenantResource(ctx context.Context, resourceID string) (string, error) {
	switch resourceID {
	case "list":
		tenants, _, err := s.authService.ListTenants(ctx, 1, 100)
		if err != nil {
			return "", err
		}
		var tenantList []map[string]interface{}
		for _, t := range tenants {
			tenantList = append(tenantList, map[string]interface{}{
				"id":      t.ID.String(),
				"name":    t.Name,
				"slug":    t.Slug,
				"balance": t.Balance.String(),
				"status":  t.Status,
			})
		}
		jsonData, _ := json.MarshalIndent(tenantList, "", "  ")
		return string(jsonData), nil

	default:
		tenantID, err := uuid.Parse(resourceID)
		if err != nil {
			return "", fmt.Errorf("invalid tenant ID: %s", resourceID)
		}
		tenant, err := s.authService.GetTenantByID(ctx, tenantID)
		if err != nil {
			return "", err
		}
		data := map[string]interface{}{
			"id":         tenant.ID.String(),
			"name":       tenant.Name,
			"slug":       tenant.Slug,
			"balance":    tenant.Balance.String(),
			"status":     tenant.Status,
			"created_at": tenant.CreatedAt,
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		return string(jsonData), nil
	}
}

func (s *MCPService) readAPIKeyResource(ctx context.Context) (string, error) {
	keys, err := s.authService.ListAllAPIKeys(ctx)
	if err != nil {
		return "", err
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
		})
	}
	jsonData, _ := json.MarshalIndent(keyList, "", "  ")
	return string(jsonData), nil
}