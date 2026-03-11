package adb

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type ScreenRecordOptions struct {
	TimeLimit int
	BitRate   int
	Size      string
}

func (c *Client) Screenshot(ctx context.Context, serial, path string) error {
	remotePath := "/sdcard/screenshot.png"
	_, err := c.ShellArgs(ctx, serial, "screencap", "-p", remotePath)
	if err != nil {
		return fmt.Errorf("screencap: %w", err)
	}
	defer func() {
		_, _ = c.ShellArgs(ctx, serial, "rm", remotePath)
	}()

	if err := c.Pull(ctx, serial, remotePath, path); err != nil {
		return fmt.Errorf("pull screenshot: %w", err)
	}

	return nil
}

func (c *Client) ScreenRecord(ctx context.Context, serial, path string, options ScreenRecordOptions) (*exec.Cmd, error) {
	shellCmd := "screenrecord"
	if options.TimeLimit > 0 {
		shellCmd += fmt.Sprintf(" --time-limit %d", options.TimeLimit)
	}
	if options.BitRate > 0 {
		shellCmd += fmt.Sprintf(" --bit-rate %d", options.BitRate)
	}
	if options.Size != "" {
		shellCmd += " --size " + shellQuote(options.Size)
	}
	shellCmd += " " + shellQuote(path)

	var args []string
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "shell", shellCmd)

	cmd := exec.Command(c.adbPath, args...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("screen record: %w", err)
	}
	return cmd, nil
}

func (c *Client) GetScreenSize(ctx context.Context, serial string) (width, height int, err error) {
	result, err := c.Shell(ctx, serial, "wm size")
	if err != nil {
		return 0, 0, fmt.Errorf("get screen size: %w", err)
	}

	line, _, _ := strings.Cut(result.Output, "\n")
	_, size, ok := strings.Cut(line, ": ")
	if !ok {
		return 0, 0, fmt.Errorf("unexpected wm size output: %s", result.Output)
	}

	size = strings.TrimSpace(size)
	w, h, ok := strings.Cut(size, "x")
	if !ok {
		return 0, 0, fmt.Errorf("unexpected size format: %s", size)
	}

	width, err = strconv.Atoi(strings.TrimSpace(w))
	if err != nil {
		return 0, 0, fmt.Errorf("parse width: %w", err)
	}
	height, err = strconv.Atoi(strings.TrimSpace(h))
	if err != nil {
		return 0, 0, fmt.Errorf("parse height: %w", err)
	}
	return width, height, nil
}
