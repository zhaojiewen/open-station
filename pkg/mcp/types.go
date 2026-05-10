package mcp

// MCP Protocol Types - shared between service and handler

// ImplementationInfo represents client/server implementation details
type ImplementationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities represents client capabilities
type ClientCapabilities struct {
	Roots       *RootsCapability       `json:"roots,omitempty"`
	Sampling    *SamplingCapability    `json:"sampling,omitempty"`
	Elicitation *ElicitationCapability `json:"elicitation,omitempty"`
}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
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

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// CallToolResult represents result of tool execution
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents content in tool result
type ContentBlock struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	MimeType string      `json:"mimeType,omitempty"`
}

// Resource represents an MCP resource definition
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ReadResourceResult represents result of resource read
type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

// ResourceContents represents resource content
type ResourceContents struct {
	URI      string      `json:"uri"`
	MimeType string      `json:"mimeType,omitempty"`
	Text     string      `json:"text,omitempty"`
	Blob     interface{} `json:"blob,omitempty"`
}

// Prompt represents an MCP prompt definition
type Prompt struct {
	Name        string           `json:"name"`
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
}

// InitializeResult represents result of initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ImplementationInfo `json:"serverInfo"`
}

// ListToolsResult represents result of tools/list
type ListToolsResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ListResourcesResult represents result of resources/list
type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// ListPromptsResult represents result of prompts/list
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}