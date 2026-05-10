package handler

import (
	"encoding/json"
)

// JSON-RPC 2.0 structures
type JSONRPCRequest struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Protocol Version
const MCPProtocolVersion = "2025-11-25"

// Initialize types
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ImplementationInfo `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ImplementationInfo `json:"serverInfo"`
}

type ClientCapabilities struct {
	Roots      *RootsCapability      `json:"roots,omitempty"`
	Sampling   *SamplingCapability   `json:"sampling,omitempty"`
	Elicitation *ElicitationCapability `json:"elicitation,omitempty"`
}

type ServerCapabilities struct {
	Tools      *ToolsCapability      `json:"tools,omitempty"`
	Resources  *ResourcesCapability  `json:"resources,omitempty"`
	Prompts    *PromptsCapability    `json:"prompts,omitempty"`
	Logging    *LoggingCapability    `json:"logging,omitempty"`
}

type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type SamplingCapability struct{}

type ElicitationCapability struct{}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type LoggingCapability struct{}

type ImplementationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool types
type Tool struct {
	Name        string                 `json:"name"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools       []Tool `json:"tools"`
	NextCursor  string `json:"nextCursor,omitempty"`
}

type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	MimeType string      `json:"mimeType,omitempty"`
}

// Resource types
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

type ReadResourceRequest struct {
	URI string `json:"uri"`
}

type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

type ResourceContents struct {
	URI      string      `json:"uri"`
	MimeType string      `json:"mimeType,omitempty"`
	Text     string      `json:"text,omitempty"`
	Blob     interface{} `json:"blob,omitempty"`
}

// Prompt types
type Prompt struct {
	Name        string                 `json:"name"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// JSON-RPC error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
	ErrorCodePermissionDenied = -32000
)

// NewJSONRPCError creates a JSON-RPC error
func NewJSONRPCError(code int, message string, data interface{}) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewJSONRPCResponse creates a success response
func NewJSONRPCResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      id,
		Result:  result,
	}
}

// NewJSONRPCErrorResponse creates an error response
func NewJSONRPCErrorResponse(id interface{}, err *RPCError) *JSONRPCResponse {
	return &JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      id,
		Error:   err,
	}
}