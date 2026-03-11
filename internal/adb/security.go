package adb

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) GetSELinuxStatus(ctx context.Context, serial string) (string, error) {
	result, err := c.Shell(ctx, serial, "getenforce")
	if err != nil {
		return "", fmt.Errorf("get selinux status: %w", err)
	}
	return strings.TrimSpace(result.Output), nil
}

func (c *Client) SetSELinux(ctx context.Context, serial, mode string) error {
	val := "1"
	if mode == "permissive" {
		val = "0"
	}
	_, err := c.ShellArgs(ctx, serial, "setenforce", val)
	if err != nil {
		return fmt.Errorf("set selinux %s: %w", mode, err)
	}
	return nil
}

func (c *Client) ListPermissions(ctx context.Context, serial, group string) ([]string, error) {
	args := []string{"pm", "list", "permissions", "-g"}
	if group != "" {
		args = append(args, group)
	}
	result, err := c.ShellArgs(ctx, serial, args...)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}

	return parsePermissionsOutput(result.Output), nil
}

func parsePermissionsOutput(output string) []string {
	var perms []string
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "permission:"); ok {
			perms = append(perms, rest)
		}
	}
	return perms
}

func parseAPKPathOutput(output string) string {
	output = strings.TrimSpace(output)
	return strings.TrimPrefix(output, "package:")
}

func (c *Client) GetAPKPath(ctx context.Context, serial, pkg string) (string, error) {
	result, err := c.ShellArgs(ctx, serial, "pm", "path", pkg)
	if err != nil {
		return "", fmt.Errorf("get apk path %s: %w", pkg, err)
	}
	return parseAPKPathOutput(result.Output), nil
}
