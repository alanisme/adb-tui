package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alanisme/adb-tui/internal/mcp"
	"github.com/alanisme/adb-tui/pkg/jsonrpc"
)

type StdioTransport struct {
	reader io.Reader
	writer io.Writer
}

func NewStdioTransport() *StdioTransport {
	return &StdioTransport{
		reader: os.Stdin,
		writer: os.Stdout,
	}
}

func NewStdioTransportWithIO(r io.Reader, w io.Writer) *StdioTransport {
	return &StdioTransport{reader: r, writer: w}
}

func (t *StdioTransport) Serve(ctx context.Context, server *mcp.Server) error {
	scanner := bufio.NewScanner(t.reader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonrpc.Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := jsonrpc.NewError(nil, jsonrpc.ParseError, "parse error")
			_ = t.writeResponse(resp)
			continue
		}

		resp := server.HandleRequest(ctx, &req)
		if resp == nil {
			continue
		}

		if err := t.writeResponse(resp); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	return nil
}

func (t *StdioTransport) writeResponse(resp *jsonrpc.Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = t.writer.Write(data)
	return err
}
