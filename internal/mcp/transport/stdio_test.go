package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alanisme/adb-tui/internal/mcp"
	"github.com/alanisme/adb-tui/pkg/jsonrpc"
)

func TestStdioTransport_PingRequest(t *testing.T) {
	id := jsonrpc.NewNumberID(1)
	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "ping",
		ID:      &id,
	}
	reqData, _ := json.Marshal(req)

	input := bytes.NewBuffer(append(reqData, '\n'))
	output := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(input, output)
	server := mcp.NewServer("test", "1.0")

	err := transport.Serve(context.Background(), server)
	if err != nil {
		t.Fatal(err)
	}

	var resp jsonrpc.Response
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &resp); err != nil {
		t.Fatalf("failed to parse response: %v (raw: %s)", err, output.String())
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.ID == nil || resp.ID.Value() != int64(1) {
		t.Fatalf("unexpected id: %v", resp.ID)
	}
}

func TestStdioTransport_InitializeRequest(t *testing.T) {
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
	reqData, _ := json.Marshal(req)

	input := bytes.NewBuffer(append(reqData, '\n'))
	output := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(input, output)
	server := mcp.NewServer("adb-mcp", "0.1.0")

	transport.Serve(context.Background(), server)

	var resp jsonrpc.Response
	json.Unmarshal(bytes.TrimSpace(output.Bytes()), &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result mcp.InitializeResult
	json.Unmarshal(resp.Result, &result)
	if result.ServerInfo.Name != "adb-mcp" {
		t.Fatalf("expected adb-mcp, got %s", result.ServerInfo.Name)
	}
}

func TestStdioTransport_InvalidJSON(t *testing.T) {
	input := bytes.NewBufferString("this is not json\n")
	output := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(input, output)
	server := mcp.NewServer("test", "1.0")

	transport.Serve(context.Background(), server)

	var resp jsonrpc.Response
	json.Unmarshal(bytes.TrimSpace(output.Bytes()), &resp)
	if resp.Error == nil {
		t.Fatal("expected error for invalid json")
	}
	if resp.Error.Code != jsonrpc.ParseError {
		t.Fatalf("expected ParseError, got %d", resp.Error.Code)
	}
}

func TestStdioTransport_EmptyLines(t *testing.T) {
	id := jsonrpc.NewNumberID(1)
	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "ping",
		ID:      &id,
	}
	reqData, _ := json.Marshal(req)

	input := bytes.NewBufferString("\n\n" + string(reqData) + "\n\n")
	output := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(input, output)
	server := mcp.NewServer("test", "1.0")

	transport.Serve(context.Background(), server)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 response line, got %d", len(lines))
	}
}

func TestStdioTransport_MultipleRequests(t *testing.T) {
	var inputBuf bytes.Buffer
	for i := range 3 {
		id := jsonrpc.NewNumberID(int64(i + 1))
		req := jsonrpc.Request{
			JSONRPC: jsonrpc.Version,
			Method:  "ping",
			ID:      &id,
		}
		data, _ := json.Marshal(req)
		inputBuf.Write(data)
		inputBuf.WriteByte('\n')
	}

	output := &bytes.Buffer{}
	transport := NewStdioTransportWithIO(&inputBuf, output)
	server := mcp.NewServer("test", "1.0")

	transport.Serve(context.Background(), server)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 response lines, got %d", len(lines))
	}
}

func TestStdioTransport_NotificationNoResponse(t *testing.T) {
	req := jsonrpc.Request{
		JSONRPC: jsonrpc.Version,
		Method:  "notifications/initialized",
	}
	reqData, _ := json.Marshal(req)

	input := bytes.NewBuffer(append(reqData, '\n'))
	output := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(input, output)
	server := mcp.NewServer("test", "1.0")

	transport.Serve(context.Background(), server)

	if output.Len() != 0 {
		t.Fatalf("expected no output for notification, got: %s", output.String())
	}
}

func TestStdioTransport_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	pr := newBlockingReader()
	output := &bytes.Buffer{}
	transport := NewStdioTransportWithIO(pr, output)
	server := mcp.NewServer("test", "1.0")

	done := make(chan error, 1)
	go func() {
		done <- transport.Serve(ctx, server)
	}()

	cancel()
	pr.Close()

	<-done
}

type blockingReader struct {
	closed chan struct{}
}

func newBlockingReader() *blockingReader {
	return &blockingReader{closed: make(chan struct{})}
}

func (r *blockingReader) Read(p []byte) (int, error) {
	<-r.closed
	return 0, &readClosedError{}
}

func (r *blockingReader) Close() {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
}

type readClosedError struct{}

func (e *readClosedError) Error() string { return "closed" }
