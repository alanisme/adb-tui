package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ADBPath   string            `json:"adb_path,omitempty"`
	Theme     string            `json:"theme,omitempty"`
	LogLevel  string            `json:"log_level,omitempty"`
	MCP       MCP               `json:"mcp,omitzero"`
	Shortcuts map[string]string `json:"shortcuts,omitempty"`
}

type MCP struct {
	Transport string `json:"transport,omitempty"`
	Addr      string `json:"addr,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		MCP: MCP{
			Transport: "stdio",
			Addr:      ":8080",
		},
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "adb-tui", "config.json"), nil
}

func ConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "adb-tui"), nil
}
