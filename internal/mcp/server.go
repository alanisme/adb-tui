package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/alanisme/adb-tui/pkg/jsonrpc"
)

type ToolHandler func(ctx context.Context, params json.RawMessage) (*ToolCallResult, error)

type toolEntry struct {
	tool    Tool
	handler ToolHandler
}

type Server struct {
	info  ServerInfo
	tools map[string]toolEntry
	mu    sync.RWMutex
}

func NewServer(name, version string) *Server {
	return &Server{
		info: ServerInfo{
			Name:    name,
			Version: version,
		},
		tools: make(map[string]toolEntry),
	}
}

func (s *Server) RegisterTool(tool Tool, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = toolEntry{tool: tool, handler: handler}
}

func (s *Server) HandleRequest(ctx context.Context, req *jsonrpc.Request) *jsonrpc.Response {
	if req.JSONRPC != jsonrpc.Version {
		return jsonrpc.NewError(req.ID, jsonrpc.InvalidRequest, "invalid JSON-RPC version")
	}

	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodInitialized:
		return nil
	case MethodPing:
		return s.handlePing(req)
	case MethodToolsList:
		return s.handleToolsList(req)
	case MethodToolsCall:
		return s.handleToolsCall(ctx, req)
	case MethodResourceList:
		return jsonrpc.NewResponse(req.ID, &ListResourcesResult{Resources: []Resource{}})
	case MethodResourceRead:
		return jsonrpc.NewError(req.ID, jsonrpc.InvalidParams, "resource not found")
	case MethodPromptsList:
		return jsonrpc.NewResponse(req.ID, &ListPromptsResult{Prompts: []Prompt{}})
	case MethodPromptsGet:
		return jsonrpc.NewError(req.ID, jsonrpc.InvalidParams, "prompt not found")
	default:
		return jsonrpc.NewError(req.ID, jsonrpc.MethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req *jsonrpc.Request) *jsonrpc.Response {
	var params InitializeParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return jsonrpc.NewError(req.ID, jsonrpc.InvalidParams, "invalid initialize params")
		}
	}

	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: s.info,
	}
	return jsonrpc.NewResponse(req.ID, &result)
}

func (s *Server) handlePing(req *jsonrpc.Request) *jsonrpc.Response {
	return jsonrpc.NewResponse(req.ID, map[string]any{})
}

func (s *Server) handleToolsList(req *jsonrpc.Request) *jsonrpc.Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, entry := range s.tools {
		tools = append(tools, entry.tool)
	}
	return jsonrpc.NewResponse(req.ID, &ListToolsResult{Tools: tools})
}

func (s *Server) handleToolsCall(ctx context.Context, req *jsonrpc.Request) *jsonrpc.Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return jsonrpc.NewError(req.ID, jsonrpc.InvalidParams, "invalid tool call params")
	}

	s.mu.RLock()
	entry, ok := s.tools[params.Name]
	s.mu.RUnlock()

	if !ok {
		return jsonrpc.NewError(req.ID, jsonrpc.InvalidParams, fmt.Sprintf("unknown tool: %s", params.Name))
	}

	result, err := entry.handler(ctx, params.Arguments)
	if err != nil {
		errResult := &ToolCallResult{
			Content: []Content{TextContent(err.Error())},
			IsError: true,
		}
		return jsonrpc.NewResponse(req.ID, errResult)
	}
	return jsonrpc.NewResponse(req.ID, result)
}
