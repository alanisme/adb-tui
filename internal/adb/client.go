package adb

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	adbPath string
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

type ExecResult struct {
	Output   string
	ExitCode int
	Duration time.Duration
}

func NewClient() (*Client, error) {
	path, err := exec.LookPath("adb")
	if err != nil {
		return nil, fmt.Errorf("adb not found in PATH: %w", err)
	}
	return &Client{adbPath: path}, nil
}

func NewClientWithPath(adbPath string) *Client {
	return &Client{adbPath: adbPath}
}

func (c *Client) Exec(ctx context.Context, args ...string) (*ExecResult, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, c.adbPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &ExecResult{
		Output:   strings.TrimSpace(stdout.String()),
		Duration: time.Since(start),
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if err != nil {
		if stderr.Len() > 0 {
			return result, fmt.Errorf("%s: %w", strings.TrimSpace(stderr.String()), err)
		}
		return result, err
	}
	return result, nil
}

func (c *Client) ExecDevice(ctx context.Context, serial string, args ...string) (*ExecResult, error) {
	if serial == "" {
		// Auto-resolve: if exactly one device is connected, use it.
		// Otherwise fall through to bare adb which will error on ambiguity.
		if resolved, err := c.resolveDevice(ctx); err == nil {
			serial = resolved
		}
	}
	if serial == "" {
		return c.Exec(ctx, args...)
	}
	fullArgs := make([]string, 0, len(args)+2)
	fullArgs = append(fullArgs, "-s", serial)
	fullArgs = append(fullArgs, args...)
	return c.Exec(ctx, fullArgs...)
}

// resolveDevice returns the serial of the sole connected device, or an error
// if zero or multiple devices are found.
func (c *Client) resolveDevice(ctx context.Context) (string, error) {
	devices, err := c.ListDevices(ctx)
	if err != nil {
		return "", err
	}
	if len(devices) == 1 {
		return devices[0].Serial, nil
	}
	return "", fmt.Errorf("expected 1 device, found %d", len(devices))
}

func (c *Client) Shell(ctx context.Context, serial string, command string) (*ExecResult, error) {
	return c.ExecDevice(ctx, serial, "shell", command)
}

func (c *Client) ShellArgs(ctx context.Context, serial string, args ...string) (*ExecResult, error) {
	fullArgs := make([]string, 0, len(args)+1)
	fullArgs = append(fullArgs, "shell")
	fullArgs = append(fullArgs, args...)
	return c.ExecDevice(ctx, serial, fullArgs...)
}

func (c *Client) StartServer(ctx context.Context) error {
	_, err := c.Exec(ctx, "start-server")
	return err
}

func (c *Client) KillServer(ctx context.Context) error {
	_, err := c.Exec(ctx, "kill-server")
	return err
}

func (c *Client) Version(ctx context.Context) (string, error) {
	result, err := c.Exec(ctx, "version")
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

func (c *Client) Connect(ctx context.Context, host string) error {
	_, err := c.Exec(ctx, "connect", host)
	return err
}

func (c *Client) Disconnect(ctx context.Context, host string) error {
	if host == "" {
		_, err := c.Exec(ctx, "disconnect")
		return err
	}
	_, err := c.Exec(ctx, "disconnect", host)
	return err
}

func (c *Client) Pair(ctx context.Context, host, code string) error {
	_, err := c.Exec(ctx, "pair", host, code)
	return err
}

func (c *Client) AdbPath() string {
	return c.adbPath
}
