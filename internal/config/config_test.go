package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MCP.Transport != "stdio" {
		t.Fatalf("expected stdio, got %s", cfg.MCP.Transport)
	}
	if cfg.MCP.Addr != ":8080" {
		t.Fatalf("expected :8080, got %s", cfg.MCP.Addr)
	}
}

func TestLoadNonExistent(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MCP.Transport != "stdio" {
		t.Fatalf("expected default transport, got %s", cfg.MCP.Transport)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	cfg.ADBPath = "/usr/bin/adb"
	cfg.Theme = "dark"
	cfg.MCP.Addr = ":9090"

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	loaded := DefaultConfig()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := json.Unmarshal(raw, loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.ADBPath != "/usr/bin/adb" {
		t.Fatalf("expected adb path, got %s", loaded.ADBPath)
	}
	if loaded.Theme != "dark" {
		t.Fatalf("expected dark, got %s", loaded.Theme)
	}
	if loaded.MCP.Addr != ":9090" {
		t.Fatalf("expected :9090, got %s", loaded.MCP.Addr)
	}
}

func TestSaveToFile(t *testing.T) {
	dir := t.TempDir()
	// Override configPath by testing the Save logic directly
	cfg := DefaultConfig()
	cfg.ADBPath = "/test/adb"
	cfg.Theme = "light"

	path := filepath.Join(dir, "subdir", "config.json")
	// Save manually to test the marshaling + directory creation
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Verify the file content
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var loaded Config
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.ADBPath != "/test/adb" {
		t.Fatalf("expected /test/adb, got %s", loaded.ADBPath)
	}
	if loaded.Theme != "light" {
		t.Fatalf("expected light, got %s", loaded.Theme)
	}
}

func TestConfigOmitzero(t *testing.T) {
	// Verify that a config with zero MCP omits the mcp field
	cfg := &Config{ADBPath: "/usr/bin/adb"}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	// With omitzero, zero-value MCP struct should be omitted
	if json.Valid(data) && len(s) > 0 {
		var m map[string]json.RawMessage
		json.Unmarshal(data, &m)
		if _, hasMCP := m["mcp"]; hasMCP {
			t.Fatal("expected mcp to be omitted for zero value")
		}
	}
}

func TestConfigJSON(t *testing.T) {
	input := `{"adb_path":"/opt/adb","mcp":{"transport":"http","addr":":3000"},"shortcuts":{"screenshot":"s"}}`

	cfg := DefaultConfig()
	if err := json.Unmarshal([]byte(input), cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cfg.ADBPath != "/opt/adb" {
		t.Fatalf("expected /opt/adb, got %s", cfg.ADBPath)
	}
	if cfg.MCP.Transport != "http" {
		t.Fatalf("expected http, got %s", cfg.MCP.Transport)
	}
	if cfg.Shortcuts["screenshot"] != "s" {
		t.Fatalf("expected shortcut s, got %s", cfg.Shortcuts["screenshot"])
	}
}
