package mcp

import (
	"encoding/json"
	"testing"
)

func TestImplementationInfo(t *testing.T) {
	info := ImplementationInfo{
		Name:    "test-client",
		Version: "1.0.0",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ImplementationInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != info.Name {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, info.Name)
	}
	if unmarshaled.Version != info.Version {
		t.Errorf("Version = %v, want %v", unmarshaled.Version, info.Version)
	}
}

func TestClientCapabilities(t *testing.T) {
	tests := []struct {
		name string
		cap  ClientCapabilities
	}{
		{
			name: "empty capabilities",
			cap:  ClientCapabilities{},
		},
		{
			name: "with roots",
			cap: ClientCapabilities{
				Roots: &RootsCapability{ListChanged: true},
			},
		},
		{
			name: "with sampling",
			cap: ClientCapabilities{
				Sampling: &SamplingCapability{},
			},
		},
		{
			name: "with all capabilities",
			cap: ClientCapabilities{
				Roots:       &RootsCapability{ListChanged: true},
				Sampling:    &SamplingCapability{},
				Elicitation: &ElicitationCapability{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.cap)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var unmarshaled ClientCapabilities
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Verify marshaling and unmarshaling works
			data2, _ := json.Marshal(unmarshaled)
			if string(data) != string(data2) {
				t.Errorf("round-trip mismatch")
			}
		})
	}
}

func TestServerCapabilities(t *testing.T) {
	cap := ServerCapabilities{
		Tools: &ToolsCapability{ListChanged: true},
		Resources: &ResourcesCapability{
			Subscribe:   true,
			ListChanged: true,
		},
		Prompts: &PromptsCapability{ListChanged: true},
		Logging: &LoggingCapability{},
	}

	data, err := json.Marshal(cap)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ServerCapabilities
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Tools == nil || !unmarshaled.Tools.ListChanged {
		t.Error("Tools capability not preserved")
	}
	if unmarshaled.Resources == nil || !unmarshaled.Resources.Subscribe {
		t.Error("Resources capability not preserved")
	}
}

func TestTool(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Title:       "Test Tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled Tool
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != tool.Name {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, tool.Name)
	}
	if unmarshaled.Description != tool.Description {
		t.Errorf("Description = %v, want %v", unmarshaled.Description, tool.Description)
	}
}

func TestCallToolResult(t *testing.T) {
	tests := []struct {
		name   string
		result CallToolResult
	}{
		{
			name: "text content",
			result: CallToolResult{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Hello, world!",
					},
				},
				IsError: false,
			},
		},
		{
			name: "error result",
			result: CallToolResult{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Error occurred",
					},
				},
				IsError: true,
			},
		},
		{
			name: "multiple content blocks",
			result: CallToolResult{
				Content: []ContentBlock{
					{Type: "text", Text: "First block"},
					{Type: "image", Data: "base64data", MimeType: "image/png"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var unmarshaled CallToolResult
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if len(unmarshaled.Content) != len(tt.result.Content) {
				t.Errorf("Content length = %d, want %d", len(unmarshaled.Content), len(tt.result.Content))
			}
		})
	}
}

func TestResource(t *testing.T) {
	resource := Resource{
		URI:         "file:///test.txt",
		Name:        "Test Resource",
		Description: "A test resource",
		MimeType:    "text/plain",
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled Resource
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.URI != resource.URI {
		t.Errorf("URI = %v, want %v", unmarshaled.URI, resource.URI)
	}
}

func TestReadResourceResult(t *testing.T) {
	result := ReadResourceResult{
		Contents: []ResourceContents{
			{
				URI:      "file:///test.txt",
				MimeType: "text/plain",
				Text:     "Hello, world!",
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ReadResourceResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(unmarshaled.Contents) != 1 {
		t.Errorf("Contents length = %d, want 1", len(unmarshaled.Contents))
	}
}

func TestPrompt(t *testing.T) {
	prompt := Prompt{
		Name:        "test_prompt",
		Title:       "Test Prompt",
		Description: "A test prompt",
		Arguments: []PromptArgument{
			{
				Name:        "arg1",
				Title:       "Argument 1",
				Description: "First argument",
				Required:    true,
			},
			{
				Name:        "arg2",
				Title:       "Argument 2",
				Description: "Second argument",
				Required:    false,
			},
		},
	}

	data, err := json.Marshal(prompt)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled Prompt
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != prompt.Name {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, prompt.Name)
	}
	if len(unmarshaled.Arguments) != len(prompt.Arguments) {
		t.Errorf("Arguments length = %d, want %d", len(unmarshaled.Arguments), len(prompt.Arguments))
	}
}

func TestInitializeResult(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{ListChanged: true},
		},
		ServerInfo: ImplementationInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled InitializeResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.ProtocolVersion != result.ProtocolVersion {
		t.Errorf("ProtocolVersion = %v, want %v", unmarshaled.ProtocolVersion, result.ProtocolVersion)
	}
	if unmarshaled.ServerInfo.Name != result.ServerInfo.Name {
		t.Errorf("ServerInfo.Name = %v, want %v", unmarshaled.ServerInfo.Name, result.ServerInfo.Name)
	}
}

func TestListToolsResult(t *testing.T) {
	result := ListToolsResult{
		Tools: []Tool{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
		},
		NextCursor: "next_page_token",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ListToolsResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(unmarshaled.Tools) != 2 {
		t.Errorf("Tools length = %d, want 2", len(unmarshaled.Tools))
	}
	if unmarshaled.NextCursor != result.NextCursor {
		t.Errorf("NextCursor = %v, want %v", unmarshaled.NextCursor, result.NextCursor)
	}
}

func TestListResourcesResult(t *testing.T) {
	result := ListResourcesResult{
		Resources: []Resource{
			{URI: "file:///test1.txt", Name: "Resource 1"},
			{URI: "file:///test2.txt", Name: "Resource 2"},
		},
		NextCursor: "",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ListResourcesResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(unmarshaled.Resources) != 2 {
		t.Errorf("Resources length = %d, want 2", len(unmarshaled.Resources))
	}
}

func TestListPromptsResult(t *testing.T) {
	result := ListPromptsResult{
		Prompts: []Prompt{
			{Name: "prompt1", Description: "First prompt"},
			{Name: "prompt2", Description: "Second prompt"},
		},
		NextCursor: "cursor123",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ListPromptsResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(unmarshaled.Prompts) != 2 {
		t.Errorf("Prompts length = %d, want 2", len(unmarshaled.Prompts))
	}
}

func TestContentBlockWithBlob(t *testing.T) {
	content := ContentBlock{
		Type:     "image",
		Data:     "base64encodeddata",
		MimeType: "image/png",
	}

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ContentBlock
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != content.Type {
		t.Errorf("Type = %v, want %v", unmarshaled.Type, content.Type)
	}
}

func TestResourceContentsWithBlob(t *testing.T) {
	content := ResourceContents{
		URI:      "file:///image.png",
		MimeType: "image/png",
		Blob:     "base64encodeddata",
	}

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ResourceContents
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.URI != content.URI {
		t.Errorf("URI = %v, want %v", unmarshaled.URI, content.URI)
	}
}