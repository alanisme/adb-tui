package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alanisme/adb-tui/internal/mcp"
	"github.com/alanisme/adb-tui/pkg/jsonrpc"
)

type HTTPTransport struct {
	addr    string
	server  *mcp.Server
	clients map[string]chan []byte
	mu      sync.RWMutex
}

func NewHTTPTransport(addr string) *HTTPTransport {
	return &HTTPTransport{
		addr:    addr,
		clients: make(map[string]chan []byte),
	}
}

func (t *HTTPTransport) Serve(ctx context.Context, server *mcp.Server) error {
	t.server = server

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", t.handleMCP)
	mux.HandleFunc("/mcp/sse", t.handleSSE)
	mux.HandleFunc("/message", t.handleMessage)

	srv := &http.Server{
		Addr:              t.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (t *HTTPTransport) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req jsonrpc.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := jsonrpc.NewError(nil, jsonrpc.ParseError, "parse error")
		t.writeJSON(w, resp)
		return
	}

	resp := t.server.HandleRequest(r.Context(), &req)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	t.writeJSON(w, resp)
}

func (t *HTTPTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientID := fmt.Sprintf("%d", time.Now().UnixNano())
	ch := make(chan []byte, 64)

	t.mu.Lock()
	t.clients[clientID] = ch
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.clients, clientID)
		t.mu.Unlock()
		close(ch)
	}()

	fmt.Fprintf(w, "event: endpoint\ndata: /message?sessionId=%s\n\n", clientID)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (t *HTTPTransport) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "missing sessionId", http.StatusBadRequest)
		return
	}

	t.mu.RLock()
	ch, ok := t.clients[sessionID]
	t.mu.RUnlock()

	if !ok {
		http.Error(w, "invalid session", http.StatusNotFound)
		return
	}

	var req jsonrpc.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := jsonrpc.NewError(nil, jsonrpc.ParseError, "parse error")
		data, _ := json.Marshal(resp)
		if !t.sendToClient(ch, data) {
			http.Error(w, "session buffer full", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}

	resp := t.server.HandleRequest(r.Context(), &req)
	if resp != nil {
		data, _ := json.Marshal(resp)
		if !t.sendToClient(ch, data) {
			http.Error(w, "session buffer full", http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

// sendToClient attempts to send data to a client's SSE channel.
// Returns false if the channel buffer is full or the channel was closed
// (SSE connection disconnected between map lookup and send).
func (t *HTTPTransport) sendToClient(ch chan []byte, data []byte) (sent bool) {
	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()
	select {
	case ch <- data:
		return true
	default:
		return false
	}
}

func (t *HTTPTransport) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
