package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/pkg/mcp"
)

// User application tool implementations

func (s *MCPService) toolSendUserInvitation(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.userAppService == nil {
		return nil, fmt.Errorf("user application service not available")
	}

	email, ok := args["email"].(string)
	if !ok {
		return nil, fmt.Errorf("email is required")
	}

	name, _ := args["name"].(string)
	requestedRole, _ := args["requested_role"].(string)
	if requestedRole == "" {
		requestedRole = "member"
	}

	var expiresIn int64
	if ei, ok := args["expires_in"].(float64); ok {
		expiresIn = int64(ei)
	} else if ei, ok := args["expires_in"].(int); ok {
		expiresIn = int64(ei)
	}

	invitation, err := s.userAppService.SendInvitationSimple(ctx, email, name, requestedRole, expiresIn)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Invitation sent successfully to %s. Invitation ID: %s, Token: %s", email, invitation.ID, invitation.InviteToken)},
		},
	}, nil
}

func (s *MCPService) toolListUserApplications(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.userAppService == nil {
		return nil, fmt.Errorf("user application service not available")
	}

	status, _ := args["status"].(string)
	if status == "" {
		status = "all"
	}
	page := s.parseInt(args["page"], 1)
	limit := s.parseInt(args["limit"], 20)

	apps, total, err := s.userAppService.ListSimple(ctx, status, page, limit)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Found %d user applications (total: %d, status: %s, page: %d)", len(apps), total, status, page)},
		},
	}, nil
}

func (s *MCPService) toolApproveUserApplication(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.userAppService == nil {
		return nil, fmt.Errorf("user application service not available")
	}

	applicationIDStr, ok := args["application_id"].(string)
	if !ok {
		return nil, fmt.Errorf("application_id is required")
	}
	applicationID, err := uuid.Parse(applicationIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid application_id")
	}

	password, ok := args["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password is required")
	}

	user, err := s.userAppService.ApproveRequestSimple(ctx, applicationID, password)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("User application approved successfully. User created with ID: %s, Email: %s", user.ID, user.Email)},
		},
	}, nil
}

func (s *MCPService) toolRejectUserApplication(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.userAppService == nil {
		return nil, fmt.Errorf("user application service not available")
	}

	applicationIDStr, ok := args["application_id"].(string)
	if !ok {
		return nil, fmt.Errorf("application_id is required")
	}
	applicationID, err := uuid.Parse(applicationIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid application_id")
	}

	if err := s.userAppService.RejectRequestSimple(ctx, applicationID); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("User application %s rejected successfully", applicationID)},
		},
	}, nil
}

func (s *MCPService) toolCancelUserInvitation(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.userAppService == nil {
		return nil, fmt.Errorf("user application service not available")
	}

	invitationIDStr, ok := args["invitation_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invitation_id is required")
	}
	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid invitation_id")
	}

	if err := s.userAppService.CancelInvitation(ctx, invitationID); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Invitation %s cancelled successfully", invitationID)},
		},
	}, nil
}

func (s *MCPService) toolCreateUserDirect(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.userAppService == nil {
		return nil, fmt.Errorf("user application service not available")
	}

	email, ok := args["email"].(string)
	if !ok {
		return nil, fmt.Errorf("email is required")
	}

	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required")
	}

	password, ok := args["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password is required")
	}

	role, _ := args["role"].(string)
	if role == "" {
		role = "member"
	}

	user, err := s.userAppService.CreateDirectSimple(ctx, email, name, password, role)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("User created directly with ID: %s, Email: %s, Role: %s", user.ID, user.Email, user.Role)},
		},
	}, nil
}

// Tenant application tool implementations (platform admin)

func (s *MCPService) toolListTenantApplications(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.tenantAppService == nil {
		return nil, fmt.Errorf("tenant application service not available")
	}

	status, _ := args["status"].(string)
	if status == "" {
		status = "pending"
	}
	page := s.parseInt(args["page"], 1)
	limit := s.parseInt(args["limit"], 20)

	apps, total, err := s.tenantAppService.List(ctx, status, page, limit)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Found %d tenant applications (total: %d, status: %s, page: %d)", len(apps), total, status, page)},
		},
	}, nil
}

func (s *MCPService) toolApproveTenantApplication(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.tenantAppService == nil {
		return nil, fmt.Errorf("tenant application service not available")
	}

	applicationIDStr, ok := args["application_id"].(string)
	if !ok {
		return nil, fmt.Errorf("application_id is required")
	}
	applicationID, err := uuid.Parse(applicationIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid application_id")
	}

	notes, _ := args["notes"].(string)

	tenant, err := s.tenantAppService.ApproveSimple(ctx, applicationID, notes)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Tenant application approved successfully. Tenant created with ID: %s, Name: %s", tenant.ID, tenant.Name)},
		},
	}, nil
}

func (s *MCPService) toolRejectTenantApplication(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.tenantAppService == nil {
		return nil, fmt.Errorf("tenant application service not available")
	}

	applicationIDStr, ok := args["application_id"].(string)
	if !ok {
		return nil, fmt.Errorf("application_id is required")
	}
	applicationID, err := uuid.Parse(applicationIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid application_id")
	}

	reason, ok := args["reason"].(string)
	if !ok {
		return nil, fmt.Errorf("reason is required")
	}

	if err := s.tenantAppService.RejectSimple(ctx, applicationID, reason); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Tenant application %s rejected successfully", applicationID)},
		},
	}, nil
}

func (s *MCPService) toolSuspendTenant(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.tenantAppService == nil {
		return nil, fmt.Errorf("tenant application service not available")
	}

	tenantIDStr, ok := args["tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("tenant_id is required")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id")
	}

	if err := s.tenantAppService.SuspendTenant(ctx, tenantID); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Tenant %s suspended successfully", tenantID)},
		},
	}, nil
}

func (s *MCPService) toolActivateTenant(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if s.tenantAppService == nil {
		return nil, fmt.Errorf("tenant application service not available")
	}

	tenantIDStr, ok := args["tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("tenant_id is required")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id")
	}

	if err := s.tenantAppService.ActivateTenant(ctx, tenantID); err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Tenant %s activated successfully", tenantID)},
		},
	}, nil
}