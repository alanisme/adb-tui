package adb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func (c *Client) SetDisplaySize(ctx context.Context, serial string, w, h int) error {
	_, err := c.ShellArgs(ctx, serial, "wm", "size", fmt.Sprintf("%dx%d", w, h))
	if err != nil {
		return fmt.Errorf("set display size: %w", err)
	}
	return nil
}

func (c *Client) ResetDisplaySize(ctx context.Context, serial string) error {
	_, err := c.ShellArgs(ctx, serial, "wm", "size", "reset")
	if err != nil {
		return fmt.Errorf("reset display size: %w", err)
	}
	return nil
}

func (c *Client) SetDensity(ctx context.Context, serial string, dpi int) error {
	_, err := c.ShellArgs(ctx, serial, "wm", "density", strconv.Itoa(dpi))
	if err != nil {
		return fmt.Errorf("set density: %w", err)
	}
	return nil
}

func (c *Client) ResetDensity(ctx context.Context, serial string) error {
	_, err := c.ShellArgs(ctx, serial, "wm", "density", "reset")
	if err != nil {
		return fmt.Errorf("reset density: %w", err)
	}
	return nil
}

func (c *Client) GetFontScale(ctx context.Context, serial string) (float64, error) {
	result, err := c.ShellArgs(ctx, serial, "settings", "get", "system", "font_scale")
	if err != nil {
		return 1.0, fmt.Errorf("get font scale: %w", err)
	}
	val := strings.TrimSpace(result.Output)
	if val == "null" || val == "" {
		return 1.0, nil
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 1.0, nil
	}
	return f, nil
}

func (c *Client) SetFontScale(ctx context.Context, serial string, scale float64) error {
	_, err := c.ShellArgs(ctx, serial, "settings", "put", "system", "font_scale", fmt.Sprintf("%.2f", scale))
	if err != nil {
		return fmt.Errorf("set font scale: %w", err)
	}
	return nil
}
