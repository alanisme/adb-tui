package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/alanisme/adb-tui/pkg/jsonrpc"
)

func newTestID(n int64) *jsonrpc.ID {
	id := jsonrpc.NewNumberID(n)
	return &id
}

func makeRequest(method string, id *jsonrpc.ID, params any) *jsonrpc.Request {
	req := &jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  method,
		ID:      id,
	}
	if params != nil {
		data, _ := json.Marshal(params)
		req.Params = data
	}
	return req
}

func TestNewServer(t *testing.T) {
	s := NewServer("test", "1.0")
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.info.Name != "test" {
		t.Fatalf("expected name test, got %s", s.info.Name)
	}
	if s.info.Version != "1.0" {
		t.Fatalf("expected version 1.0, got %s", s.info.Version)
	}
	if s.tools == nil {
		t.Fatal("expected non-nil tools map")
	}
}

func TestRegisterTool(t *testing.T) {
	s := NewServer("test", "1.0")
	tool := Tool{
		Name:        "my_tool",
		Description: "desc",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}
	s.RegisterTool(tool, func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
		return &ToolCallResult{Content: []Content{TextContent("ok")}}, nil
	})

	if _, ok := s.tools["my_tool"]; !ok {
		t.Fatal("expected tool to be registered")
	}
}

func TestHandleInitialize(t *testing.T) {
	s := NewServer("adb-mcp", "0.1.0")
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		ClientInfo:      ClientInfo{Name: "test-client", Version: "1.0"},
	}
	req := makeRequest(MethodInitialize, newTestID(1), params)
	resp := s.HandleRequest(context.Background(), req)

	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var result InitializeResult
	json.Unmarshal(resp.Result, &result)
	if result.ProtocolVersion != ProtocolVersion {
		t.Fatalf("expected protocol version %s, got %s", ProtocolVersion, result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "adb-mcp" {
		t.Fatalf("expected adb-mcp, got %s", result.ServerInfo.Name)
	}
	if result.Capabilities.Tools == nil {
		t.Fatal("expected tools capability")
	}
}

func TestHandleInitializeWithNilParams(t *testing.T) {
	s := NewServer("test", "1.0")
	req := makeRequest(MethodInitialize, newTestID(1), nil)
	resp := s.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleToolsList(t *testing.T) {
	s := NewServer("test", "1.0")
	s.RegisterTool(
		Tool{Name: "tool_a", InputSchema: json.RawMessage(`{}`)},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			return &ToolCallResult{Content: []Content{TextContent("a")}}, nil
		},
	)
	s.RegisterTool(
		Tool{Name: "tool_b", InputSchema: json.RawMessage(`{}`)},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			return &ToolCallResult{Content: []Content{TextContent("b")}}, nil
		},
	)

	req := makeRequest(MethodToolsList, newTestID(2), nil)
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result ListToolsResult
	json.Unmarshal(resp.Result, &result)
	if len(result.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result.Tools))
	}

	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	if !names["tool_a"] || !names["tool_b"] {
		t.Fatalf("expected tool_a and tool_b, got %v", names)
	}
}

func TestHandleToolsCallValid(t *testing.T) {
	s := NewServer("test", "1.0")
	s.RegisterTool(
		Tool{Name: "echo", InputSchema: json.RawMessage(`{"type":"object","properties":{"msg":{"type":"string"}}}`)},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			var p struct{ Msg string }
			json.Unmarshal(params, &p)
			return &ToolCallResult{Content: []Content{TextContent(p.Msg)}}, nil
		},
	)

	callParams := ToolCallParams{
		Name:      "echo",
		Arguments: json.RawMessage(`{"msg":"hello"}`),
	}
	req := makeRequest(MethodToolsCall, newTestID(3), callParams)
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result ToolCallResult
	json.Unmarshal(resp.Result, &result)
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content, got %d", len(result.Content))
	}
	if result.Content[0].Text != "hello" {
		t.Fatalf("expected hello, got %s", result.Content[0].Text)
	}
}

func TestHandleToolsCallInvalidTool(t *testing.T) {
	s := NewServer("test", "1.0")
	callParams := ToolCallParams{
		Name: "nonexistent",
	}
	req := makeRequest(MethodToolsCall, newTestID(4), callParams)
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != jsonrpc.InvalidParams {
		t.Fatalf("expected InvalidParams, got %d", resp.Error.Code)
	}
}

func TestHandleToolsCallInvalidParams(t *testing.T) {
	s := NewServer("test", "1.0")
	req := &jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  MethodToolsCall,
		ID:      newTestID(5),
		Params:  json.RawMessage(`not json`),
	}
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for invalid params")
	}
}

func TestHandleToolsCallHandlerError(t *testing.T) {
	s := NewServer("test", "1.0")
	s.RegisterTool(
		Tool{Name: "failing", InputSchema: json.RawMessage(`{}`)},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			return nil, context.DeadlineExceeded
		},
	)

	callParams := ToolCallParams{Name: "failing"}
	req := makeRequest(MethodToolsCall, newTestID(6), callParams)
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatal("handler errors should return result with isError, not jsonrpc error")
	}
	var result ToolCallResult
	json.Unmarshal(resp.Result, &result)
	if !result.IsError {
		t.Fatal("expected isError true")
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	s := NewServer("test", "1.0")
	req := makeRequest("unknown/method", newTestID(7), nil)
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != jsonrpc.MethodNotFound {
		t.Fatalf("expected MethodNotFound, got %d", resp.Error.Code)
	}
}

func TestHandlePing(t *testing.T) {
	s := NewServer("test", "1.0")
	req := makeRequest(MethodPing, newTestID(8), nil)
	resp := s.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleInitialized(t *testing.T) {
	s := NewServer("test", "1.0")
	req := makeRequest(MethodInitialized, nil, nil)
	resp := s.HandleRequest(context.Background(), req)
	if resp != nil {
		t.Fatal("expected nil response for initialized notification")
	}
}

func TestHandleInvalidVersion(t *testing.T) {
	s := NewServer("test", "1.0")
	req := &jsonrpc.Request{
		JSONRPC: "1.0",
		Method:  MethodPing,
		ID:      newTestID(9),
	}
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for invalid version")
	}
	if resp.Error.Code != jsonrpc.InvalidRequest {
		t.Fatalf("expected InvalidRequest, got %d", resp.Error.Code)
	}
}

func TestHandleResourceList(t *testing.T) {
	s := NewServer("test", "1.0")
	req := makeRequest(MethodResourceList, newTestID(10), nil)
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var result ListResourcesResult
	json.Unmarshal(resp.Result, &result)
	if len(result.Resources) != 0 {
		t.Fatalf("expected empty resources, got %d", len(result.Resources))
	}
}

func TestHandleResourceRead(t *testing.T) {
	s := NewServer("test", "1.0")
	req := makeRequest(MethodResourceRead, newTestID(11), ReadResourceParams{URI: "test://x"})
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for resource read")
	}
}

func TestHandlePromptsList(t *testing.T) {
	s := NewServer("test", "1.0")
	req := makeRequest(MethodPromptsList, newTestID(12), nil)
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandlePromptsGet(t *testing.T) {
	s := NewServer("test", "1.0")
	req := makeRequest(MethodPromptsGet, newTestID(13), GetPromptParams{Name: "x"})
	resp := s.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error for prompts get")
	}
}

func TestConcurrentToolCalls(t *testing.T) {
	s := NewServer("test", "1.0")
	s.RegisterTool(
		Tool{Name: "counter", InputSchema: json.RawMessage(`{}`)},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			return &ToolCallResult{Content: []Content{TextContent("done")}}, nil
		},
	)

	var wg sync.WaitGroup
	errs := make(chan error, 50)

	for i := range 50 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			callParams := ToolCallParams{Name: "counter"}
			req := makeRequest(MethodToolsCall, newTestID(int64(100+idx)), callParams)
			resp := s.HandleRequest(context.Background(), req)
			if resp.Error != nil {
				errs <- resp.Error
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent call error: %v", err)
	}
}

func TestRegisterToolOverwrite(t *testing.T) {
	s := NewServer("test", "1.0")
	s.RegisterTool(
		Tool{Name: "t", InputSchema: json.RawMessage(`{}`)},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			return &ToolCallResult{Content: []Content{TextContent("v1")}}, nil
		},
	)
	s.RegisterTool(
		Tool{Name: "t", InputSchema: json.RawMessage(`{}`)},
		func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error) {
			return &ToolCallResult{Content: []Content{TextContent("v2")}}, nil
		},
	)

	callParams := ToolCallParams{Name: "t"}
	req := makeRequest(MethodToolsCall, newTestID(1), callParams)
	resp := s.HandleRequest(context.Background(), req)
	var result ToolCallResult
	json.Unmarshal(resp.Result, &result)
	if result.Content[0].Text != "v2" {
		t.Fatalf("expected v2 after overwrite, got %s", result.Content[0].Text)
	}
}
