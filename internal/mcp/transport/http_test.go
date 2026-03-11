package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alanisme/adb-tui/internal/mcp"
	"github.com/alanisme/adb-tui/pkg/jsonrpc"
)

func setupHTTPTest() (*mcp.Server, http.Handler) {
	server := mcp.NewServer("test", "1.0")
	server.RegisterTool(
		mcp.Tool{
			Name:        "echo",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"msg":{"type":"string"}}}`),
		},
		func(ctx context.Context, params json.RawMessage) (*mcp.ToolCallResult, error) {
			var p struct{ Msg string }
			json.Unmarshal(params, &p)
			return &mcp.ToolCallResult{Content: []mcp.Content{mcp.TextContent(p.Msg)}}, nil
		},
	)

	transport := NewHTTPTransport(":0")
	transport.server = server

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", transport.handleMCP)
	mux.HandleFunc("/mcp/sse", transport.handleSSE)
	mux.HandleFunc("/message", transport.handleMessage)
	return server, mux
}

func TestHTTPTransport_MCPEndpoint_Ping(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	id := jsonrpc.NewNumberID(1)
	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "ping",
		ID:      &id,
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var rpcResp jsonrpc.Response
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error != nil {
		t.Fatalf("unexpected error: %v", rpcResp.Error)
	}
}

func TestHTTPTransport_MCPEndpoint_Initialize(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	id := jsonrpc.NewNumberID(1)
	params := mcp.InitializeParams{
		ProtocolVersion: mcp.ProtocolVersion,
		ClientInfo:      mcp.ClientInfo{Name: "test", Version: "1.0"},
	}
	paramsData, _ := json.Marshal(params)
	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "initialize",
		Params:  paramsData,
		ID:      &id,
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var rpcResp jsonrpc.Response
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error != nil {
		t.Fatalf("unexpected error: %v", rpcResp.Error)
	}

	var result mcp.InitializeResult
	json.Unmarshal(rpcResp.Result, &result)
	if result.ServerInfo.Name != "test" {
		t.Fatalf("expected test, got %s", result.ServerInfo.Name)
	}
}

func TestHTTPTransport_MCPEndpoint_ToolsCall(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	id := jsonrpc.NewNumberID(1)
	callParams := mcp.ToolCallParams{
		Name:      "echo",
		Arguments: json.RawMessage(`{"msg":"hello"}`),
	}
	paramsData, _ := json.Marshal(callParams)
	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "tools/call",
		Params:  paramsData,
		ID:      &id,
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var rpcResp jsonrpc.Response
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error != nil {
		t.Fatalf("unexpected error: %v", rpcResp.Error)
	}

	var result mcp.ToolCallResult
	json.Unmarshal(rpcResp.Result, &result)
	if len(result.Content) != 1 || result.Content[0].Text != "hello" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHTTPTransport_MCPEndpoint_InvalidJSON(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader([]byte("not json")))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var rpcResp jsonrpc.Response
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil {
		t.Fatal("expected error for invalid json")
	}
	if rpcResp.Error.Code != jsonrpc.ParseError {
		t.Fatalf("expected ParseError, got %d", rpcResp.Error.Code)
	}
}

func TestHTTPTransport_MCPEndpoint_MethodNotAllowed(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

func TestHTTPTransport_MCPEndpoint_Notification(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "notifications/initialized",
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for notification, got %d", resp.StatusCode)
	}
}

func TestHTTPTransport_MessageEndpoint_NoSession(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/message", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing session, got %d", resp.StatusCode)
	}
}

func TestHTTPTransport_MessageEndpoint_InvalidSession(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/message?sessionId=fake", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid session, got %d", resp.StatusCode)
	}
}

func TestHTTPTransport_MessageEndpoint_MethodNotAllowed(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/message?sessionId=test")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

func TestHTTPTransport_MessageEndpoint_BufferFull(t *testing.T) {
	transport := NewHTTPTransport(":0")
	server := mcp.NewServer("test", "1.0")
	transport.server = server

	// Register a session with a tiny buffer (size 1)
	ch := make(chan []byte, 1)
	transport.mu.Lock()
	transport.clients["full-session"] = ch
	transport.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/message", transport.handleMessage)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Fill the buffer
	ch <- []byte(`{"filled":"true"}`)

	// Now send a request — should get 503 since buffer is full
	id := jsonrpc.NewNumberID(1)
	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "ping",
		ID:      &id,
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(ts.URL+"/message?sessionId=full-session", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for full buffer, got %d", resp.StatusCode)
	}
}

func TestHTTPTransport_SendToClient(t *testing.T) {
	transport := NewHTTPTransport(":0")

	ch := make(chan []byte, 2)
	if !transport.sendToClient(ch, []byte("first")) {
		t.Fatal("expected send to succeed")
	}
	if !transport.sendToClient(ch, []byte("second")) {
		t.Fatal("expected send to succeed")
	}
	// Buffer full now
	if transport.sendToClient(ch, []byte("third")) {
		t.Fatal("expected send to fail on full buffer")
	}
}

func TestHTTPTransport_MCPEndpoint_UnknownMethod(t *testing.T) {
	_, handler := setupHTTPTest()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	id := jsonrpc.NewNumberID(1)
	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "unknown/method",
		ID:      &id,
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var rpcResp jsonrpc.Response
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if rpcResp.Error.Code != jsonrpc.MethodNotFound {
		t.Fatalf("expected MethodNotFound, got %d", rpcResp.Error.Code)
	}
}
