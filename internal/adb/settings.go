package adb

import (
	"context"
	"fmt"
	"strings"
)

type SettingNamespace string

const (
	NamespaceSystem SettingNamespace = "system"
	NamespaceSecure SettingNamespace = "secure"
	NamespaceGlobal SettingNamespace = "global"
)

func (c *Client) GetSetting(ctx context.Context, serial string, namespace SettingNamespace, key string) (string, error) {
	result, err := c.ShellArgs(ctx, serial, "settings", "get", string(namespace), key)
	if err != nil {
		return "", fmt.Errorf("get setting %s/%s: %w", namespace, key, err)
	}
	return strings.TrimSpace(result.Output), nil
}

func (c *Client) PutSetting(ctx context.Context, serial string, namespace SettingNamespace, key, value string) error {
	_, err := c.ShellArgs(ctx, serial, "settings", "put", string(namespace), key, value)
	if err != nil {
		return fmt.Errorf("put setting %s/%s: %w", namespace, key, err)
	}
	return nil
}

func (c *Client) ListSettings(ctx context.Context, serial string, namespace SettingNamespace) (map[string]string, error) {
	result, err := c.ShellArgs(ctx, serial, "settings", "list", string(namespace))
	if err != nil {
		return nil, fmt.Errorf("list settings %s: %w", namespace, err)
	}

	return parseSettingsOutput(result.Output), nil
}

func parseSettingsOutput(output string) map[string]string {
	settings := make(map[string]string)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		settings[key] = value
	}
	return settings
}

func (c *Client) DeleteSetting(ctx context.Context, serial string, namespace SettingNamespace, key string) error {
	_, err := c.ShellArgs(ctx, serial, "settings", "delete", string(namespace), key)
	if err != nil {
		return fmt.Errorf("delete setting %s/%s: %w", namespace, key, err)
	}
	return nil
}
