package adb

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) RunCommand(ctx context.Context, serial, cmd string) (string, error) {
	result, err := c.Shell(ctx, serial, cmd)
	if err != nil {
		return "", fmt.Errorf("run command: %w", err)
	}
	return result.Output, nil
}

func (c *Client) RunCommandAsRoot(ctx context.Context, serial, cmd string) (string, error) {
	result, err := c.Shell(ctx, serial, "su -c "+shellQuote(cmd))
	if err != nil {
		return "", fmt.Errorf("run command as root: %w", err)
	}
	return result.Output, nil
}

func (c *Client) IsRooted(ctx context.Context, serial string) bool {
	result, err := c.Shell(ctx, serial, "id")
	if err != nil {
		return false
	}
	return strings.Contains(result.Output, "uid=0")
}

func (c *Client) Remount(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "remount")
	if err != nil {
		return fmt.Errorf("remount: %w", err)
	}
	return nil
}

func (c *Client) Reboot(ctx context.Context, serial, mode string) error {
	args := []string{"reboot"}
	if mode != "" {
		args = append(args, mode)
	}
	_, err := c.ExecDevice(ctx, serial, args...)
	if err != nil {
		return fmt.Errorf("reboot: %w", err)
	}
	return nil
}

func (c *Client) Root(ctx context.Context) error {
	_, err := c.Exec(ctx, "root")
	if err != nil {
		return fmt.Errorf("root: %w", err)
	}
	return nil
}

func (c *Client) Unroot(ctx context.Context) error {
	_, err := c.Exec(ctx, "unroot")
	if err != nil {
		return fmt.Errorf("unroot: %w", err)
	}
	return nil
}

func (c *Client) RootDevice(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "root")
	if err != nil {
		return fmt.Errorf("root: %w", err)
	}
	return nil
}

func (c *Client) UnrootDevice(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "unroot")
	if err != nil {
		return fmt.Errorf("unroot: %w", err)
	}
	return nil
}

func (c *Client) DisableVerity(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "disable-verity")
	if err != nil {
		return fmt.Errorf("disable-verity: %w", err)
	}
	return nil
}

func (c *Client) EnableVerity(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "enable-verity")
	if err != nil {
		return fmt.Errorf("enable-verity: %w", err)
	}
	return nil
}

func (c *Client) GetState(ctx context.Context, serial string) (DeviceState, error) {
	result, err := c.ExecDevice(ctx, serial, "get-state")
	if err != nil {
		return "", fmt.Errorf("get-state: %w", err)
	}
	return DeviceState(strings.TrimSpace(result.Output)), nil
}

func (c *Client) GetSerialNo(ctx context.Context, serial string) (string, error) {
	result, err := c.ExecDevice(ctx, serial, "get-serialno")
	if err != nil {
		return "", fmt.Errorf("get-serialno: %w", err)
	}
	return strings.TrimSpace(result.Output), nil
}
