package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"github.com/zhaojiewen/open-station/pkg/mcp"
	"go.uber.org/zap"
)

// MCPHandler handles MCP protocol requests
type MCPHandler struct {
	mcpService *service.MCPService
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(mcpService *service.MCPService) *MCPHandler {
	return &MCPHandler{
		mcpService: mcpService,
	}
}

// HandleMCP handles MCP JSON-RPC requests (POST)
func (h *MCPHandler) HandleMCP(c *gin.Context) {
	// Get API key from header
	apiKey := h.extractAPIKey(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, NewJSONRPCErrorResponse(nil, NewJSONRPCError(
			ErrorCodeInvalidRequest,
			"authentication required",
			nil,
		)))
		return
	}

	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewJSONRPCErrorResponse(nil, NewJSONRPCError(
			ErrorCodeParseError,
			"invalid JSON",
			err.Error(),
		)))
		return
	}

	// Validate JSON-RPC version
	if req.Jsonrpc != "2.0" {
		c.JSON(http.StatusBadRequest, NewJSONRPCErrorResponse(req.ID, NewJSONRPCError(
			ErrorCodeInvalidRequest,
			"invalid JSON-RPC version",
			nil,
		)))
		return
	}

	// Get or create session
	sessionID := c.GetHeader("MCP-Session-Id")
	var session *service.MCPSession

	if req.Method == "initialize" {
		// Create new session on initialize
		var initReq InitializeRequest
		if err := json.Unmarshal(req.Params, &initReq); err != nil {
			c.JSON(http.StatusBadRequest, NewJSONRPCErrorResponse(req.ID, NewJSONRPCError(
				ErrorCodeInvalidParams,
				"invalid initialize params",
				err.Error(),
			)))
			return
		}

		clientInfo := mcp.ImplementationInfo{
			Name:    initReq.ClientInfo.Name,
			Version: initReq.ClientInfo.Version,
		}
		result, newSession, err := h.mcpService.Initialize(c.Request.Context(), apiKey, clientInfo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, NewJSONRPCErrorResponse(req.ID, NewJSONRPCError(
				ErrorCodeInvalidRequest,
				"authentication failed",
				err.Error(),
			)))
			return
		}

		session = newSession
		c.Header("MCP-Session-Id", session.ID)
		c.JSON(http.StatusOK, NewJSONRPCResponse(req.ID, result))
		return
	}

	// Validate existing session
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, NewJSONRPCErrorResponse(req.ID, NewJSONRPCError(
			ErrorCodeInvalidRequest,
			"session required",
			"initialize first to create a session",
		)))
		return
	}

	session, err := h.mcpService.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewJSONRPCErrorResponse(req.ID, NewJSONRPCError(
			ErrorCodeInvalidRequest,
			"invalid or expired session",
			err.Error(),
		)))
		return
	}

	// Route to method handler
	var result interface{}
	var rpcErr *RPCError

	switch req.Method {
	case "notifications/initialized":
		// No response needed for notifications
		c.Status(http.StatusNoContent)
		return

	case "tools/list":
		result = &mcp.ListToolsResult{
			Tools: h.mcpService.ListTools(session),
		}

	case "tools/call":
		result, rpcErr = h.handleToolsCall(c.Request.Context(), session, req.Params)

	case "resources/list":
		result = &mcp.ListResourcesResult{
			Resources: h.mcpService.ListResources(session),
		}

	case "resources/read":
		result, rpcErr = h.handleResourcesRead(c.Request.Context(), session, req.Params)

	case "prompts/list":
		result = &mcp.ListPromptsResult{
			Prompts: []mcp.Prompt{},
		}

	case "ping":
		result = map[string]interface{}{}

	default:
		rpcErr = NewJSONRPCError(
			ErrorCodeMethodNotFound,
			fmt.Sprintf("method not found: %s", req.Method),
			nil,
		)
	}

	// Send response
	if rpcErr != nil {
		c.JSON(http.StatusOK, NewJSONRPCErrorResponse(req.ID, rpcErr))
	} else {
		c.JSON(http.StatusOK, NewJSONRPCResponse(req.ID, result))
	}
}

// handleToolsCall handles tools/call method
func (h *MCPHandler) handleToolsCall(ctx context.Context, session *service.MCPSession, params json.RawMessage) (*mcp.CallToolResult, *RPCError) {
	var req CallToolRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, NewJSONRPCError(
			ErrorCodeInvalidParams,
			"invalid tool call params",
			err.Error(),
		)
	}

	logger.Info("MCP tool call",
		zap.String("tool", req.Name),
		zap.String("session_id", session.ID),
		zap.String("role", session.Role),
	)

	result, err := h.mcpService.CallTool(ctx, session, req.Name, req.Arguments)
	if err != nil {
		return nil, NewJSONRPCError(
			ErrorCodeInternalError,
			"tool execution failed",
			err.Error(),
		)
	}

	return result, nil
}

// handleResourcesRead handles resources/read method
func (h *MCPHandler) handleResourcesRead(ctx context.Context, session *service.MCPSession, params json.RawMessage) (*mcp.ReadResourceResult, *RPCError) {
	var req ReadResourceRequest
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, NewJSONRPCError(
			ErrorCodeInvalidParams,
			"invalid resource read params",
			err.Error(),
		)
	}

	logger.Info("MCP resource read",
		zap.String("uri", req.URI),
		zap.String("session_id", session.ID),
	)

	result, err := h.mcpService.ReadResource(ctx, session, req.URI)
	if err != nil {
		return nil, NewJSONRPCError(
			ErrorCodeInternalError,
			"resource read failed",
			err.Error(),
		)
	}

	return result, nil
}

// HandleSSE handles SSE streaming (GET) - for future server-initiated messages
func (h *MCPHandler) HandleSSE(c *gin.Context) {
	// Get API key from header
	apiKey := h.extractAPIKey(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Get session ID
	sessionID := c.GetHeader("MCP-Session-Id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session required"})
		return
	}

	session, err := h.mcpService.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Send initial connection message
	c.SSEvent("connected", gin.H{
		"session_id": session.ID,
		"role":       session.Role,
	})

	// Keep connection open for server-initiated messages
	// For now, just keep alive
	c.Stream(func(w io.Writer) bool {
		// Send periodic heartbeat
		c.SSEvent("ping", gin.H{})
		time.Sleep(30 * time.Second)
		return true
	})
}

// extractAPIKey extracts API key from headers
func (h *MCPHandler) extractAPIKey(c *gin.Context) string {
	// Try Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Try X-Api-Key header
	apiKeyHeader := c.GetHeader("X-Api-Key")
	if apiKeyHeader != "" {
		return apiKeyHeader
	}

	// Try custom header from MCP config
	customHeader := c.GetHeader("Api-Key")
	if customHeader != "" {
		return customHeader
	}

	return ""
}