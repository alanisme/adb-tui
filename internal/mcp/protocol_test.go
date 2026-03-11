package mcp

import (
	"encoding/json"
	"testing"
)

func TestToolSerialization(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Tool
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Name != "test_tool" {
		t.Fatalf("expected test_tool, got %s", parsed.Name)
	}
	if parsed.Description != "A test tool" {
		t.Fatalf("expected description, got %s", parsed.Description)
	}
}

func TestToolOmitsEmptyDescription(t *testing.T) {
	tool := Tool{
		Name:        "no_desc",
		InputSchema: json.RawMessage(`{}`),
	}
	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	json.Unmarshal(data, &parsed)
	if _, ok := parsed["description"]; ok {
		t.Fatal("expected description to be omitted")
	}
}

func TestInitializeResultSerialization(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	var parsed InitializeResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.ProtocolVersion != ProtocolVersion {
		t.Fatalf("expected %s, got %s", ProtocolVersion, parsed.ProtocolVersion)
	}
	if parsed.ServerInfo.Name != "test-server" {
		t.Fatalf("expected test-server, got %s", parsed.ServerInfo.Name)
	}
	if parsed.Capabilities.Tools == nil {
		t.Fatal("expected tools capability")
	}
}

func TestInitializeResultOmitsEmptyCapabilities(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    ServerCapabilities{},
		ServerInfo:      ServerInfo{Name: "s", Version: "1"},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]json.RawMessage
	json.Unmarshal(data, &parsed)
	var caps map[string]json.RawMessage
	json.Unmarshal(parsed["capabilities"], &caps)
	if _, ok := caps["resources"]; ok {
		t.Fatal("expected resources to be omitted")
	}
	if _, ok := caps["prompts"]; ok {
		t.Fatal("expected prompts to be omitted")
	}
}

func TestToolCallResultWithTextContent(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{TextContent("hello world")},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var parsed ToolCallResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Content) != 1 {
		t.Fatalf("expected 1 content, got %d", len(parsed.Content))
	}
	if parsed.Content[0].Type != "text" {
		t.Fatalf("expected text type, got %s", parsed.Content[0].Type)
	}
	if parsed.Content[0].Text != "hello world" {
		t.Fatalf("expected hello world, got %s", parsed.Content[0].Text)
	}
}

func TestToolCallResultWithImageContent(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{ImageContent("base64data==", "image/png")},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var parsed ToolCallResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Content) != 1 {
		t.Fatalf("expected 1 content, got %d", len(parsed.Content))
	}
	c := parsed.Content[0]
	if c.Type != "image" {
		t.Fatalf("expected image type, got %s", c.Type)
	}
	if c.Data != "base64data==" {
		t.Fatalf("expected base64data==, got %s", c.Data)
	}
	if c.MimeType != "image/png" {
		t.Fatalf("expected image/png, got %s", c.MimeType)
	}
}

func TestToolCallResultWithError(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{TextContent("something failed")},
		IsError: true,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var parsed ToolCallResult
	json.Unmarshal(data, &parsed)
	if !parsed.IsError {
		t.Fatal("expected isError to be true")
	}
}

func TestToolCallResultIsErrorOmittedWhenFalse(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{TextContent("ok")},
		IsError: false,
	}
	data, _ := json.Marshal(result)
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, ok := raw["isError"]; ok {
		t.Fatal("expected isError to be omitted when false")
	}
}

func TestTextContentHelper(t *testing.T) {
	c := TextContent("test")
	if c.Type != "text" {
		t.Fatalf("expected text, got %s", c.Type)
	}
	if c.Text != "test" {
		t.Fatalf("expected test, got %s", c.Text)
	}
}

func TestImageContentHelper(t *testing.T) {
	c := ImageContent("data123", "image/jpeg")
	if c.Type != "image" {
		t.Fatalf("expected image, got %s", c.Type)
	}
	if c.Data != "data123" {
		t.Fatalf("expected data123, got %s", c.Data)
	}
	if c.MimeType != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %s", c.MimeType)
	}
}

func TestToolCallParamsSerialization(t *testing.T) {
	p := ToolCallParams{
		Name:      "my_tool",
		Arguments: json.RawMessage(`{"key":"val"}`),
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var parsed ToolCallParams
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Name != "my_tool" {
		t.Fatalf("expected my_tool, got %s", parsed.Name)
	}
}

func TestMixedContent(t *testing.T) {
	result := ToolCallResult{
		Content: []Content{
			TextContent("description"),
			ImageContent("imgdata", "image/png"),
			TextContent("footer"),
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var parsed ToolCallResult
	json.Unmarshal(data, &parsed)
	if len(parsed.Content) != 3 {
		t.Fatalf("expected 3 content items, got %d", len(parsed.Content))
	}
	if parsed.Content[0].Type != "text" {
		t.Fatal("expected first content to be text")
	}
	if parsed.Content[1].Type != "image" {
		t.Fatal("expected second content to be image")
	}
}

func TestProtocolVersionValue(t *testing.T) {
	if ProtocolVersion != "2024-11-05" {
		t.Fatalf("expected 2024-11-05, got %s", ProtocolVersion)
	}
}

func TestMethodConstants(t *testing.T) {
	if MethodInitialize != "initialize" {
		t.Fatal("unexpected initialize method")
	}
	if MethodToolsList != "tools/list" {
		t.Fatal("unexpected tools/list method")
	}
	if MethodToolsCall != "tools/call" {
		t.Fatal("unexpected tools/call method")
	}
	if MethodPing != "ping" {
		t.Fatal("unexpected ping method")
	}
}

func TestListToolsResultSerialization(t *testing.T) {
	result := ListToolsResult{
		Tools: []Tool{
			{
				Name:        "tool1",
				Description: "First tool",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
			{
				Name:        "tool2",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var parsed ListToolsResult
	json.Unmarshal(data, &parsed)
	if len(parsed.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(parsed.Tools))
	}
}

func TestResourceContentsSerialization(t *testing.T) {
	rc := ResourceContents{
		URI:      "file:///test.txt",
		MimeType: "text/plain",
		Text:     "hello",
	}
	data, _ := json.Marshal(rc)
	var parsed ResourceContents
	json.Unmarshal(data, &parsed)
	if parsed.URI != "file:///test.txt" {
		t.Fatalf("expected uri, got %s", parsed.URI)
	}
	if parsed.Text != "hello" {
		t.Fatalf("expected hello, got %s", parsed.Text)
	}
}
