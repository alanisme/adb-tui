package adb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GetBrightness returns the current screen brightness (0-255).
func (c *Client) GetBrightness(ctx context.Context, serial string) (int, error) {
	result, err := c.ShellArgs(ctx, serial, "settings", "get", "system", "screen_brightness")
	if err != nil {
		return 0, fmt.Errorf("get brightness: %w", err)
	}
	val, err := strconv.Atoi(strings.TrimSpace(result.Output))
	if err != nil {
		return 0, fmt.Errorf("parse brightness: %w", err)
	}
	return val, nil
}

// SetBrightness sets the screen brightness (0-255).
func (c *Client) SetBrightness(ctx context.Context, serial string, level int) error {
	if level < 0 || level > 255 {
		return fmt.Errorf("brightness must be 0-255, got %d", level)
	}
	// Disable auto-brightness first
	_, _ = c.ShellArgs(ctx, serial, "settings", "put", "system", "screen_brightness_mode", "0")
	_, err := c.ShellArgs(ctx, serial, "settings", "put", "system", "screen_brightness", strconv.Itoa(level))
	if err != nil {
		return fmt.Errorf("set brightness: %w", err)
	}
	return nil
}

// GetAutoRotation returns whether auto-rotation is enabled.
func (c *Client) GetAutoRotation(ctx context.Context, serial string) (bool, error) {
	result, err := c.ShellArgs(ctx, serial, "settings", "get", "system", "accelerometer_rotation")
	if err != nil {
		return false, fmt.Errorf("get auto rotation: %w", err)
	}
	return strings.TrimSpace(result.Output) == "1", nil
}

// SetRotation sets the screen rotation. Values: 0=natural, 1=90°, 2=180°, 3=270°.
func (c *Client) SetRotation(ctx context.Context, serial string, rotation int) error {
	// Disable auto-rotation
	_, err := c.ShellArgs(ctx, serial, "settings", "put", "system", "accelerometer_rotation", "0")
	if err != nil {
		return fmt.Errorf("disable auto rotation: %w", err)
	}
	_, err = c.ShellArgs(ctx, serial, "settings", "put", "system", "user_rotation", strconv.Itoa(rotation))
	if err != nil {
		return fmt.Errorf("set rotation: %w", err)
	}
	return nil
}

// SetAutoRotation enables or disables auto-rotation.
func (c *Client) SetAutoRotation(ctx context.Context, serial string, enabled bool) error {
	val := "0"
	if enabled {
		val = "1"
	}
	_, err := c.ShellArgs(ctx, serial, "settings", "put", "system", "accelerometer_rotation", val)
	if err != nil {
		return fmt.Errorf("set auto rotation: %w", err)
	}
	return nil
}

// GetAirplaneMode returns whether airplane mode is enabled.
func (c *Client) GetAirplaneMode(ctx context.Context, serial string) (bool, error) {
	result, err := c.ShellArgs(ctx, serial, "settings", "get", "global", "airplane_mode_on")
	if err != nil {
		return false, fmt.Errorf("get airplane mode: %w", err)
	}
	return strings.TrimSpace(result.Output) == "1", nil
}

// SetAirplaneMode enables or disables airplane mode.
func (c *Client) SetAirplaneMode(ctx context.Context, serial string, enabled bool) error {
	val := "0"
	if enabled {
		val = "1"
	}
	_, err := c.ShellArgs(ctx, serial, "settings", "put", "global", "airplane_mode_on", val)
	if err != nil {
		return fmt.Errorf("set airplane mode: %w", err)
	}
	// Broadcast the change so the system picks it up immediately
	_, _ = c.ShellArgs(ctx, serial, "am", "broadcast",
		"-a", "android.intent.action.AIRPLANE_MODE",
		"--ez", "state", fmt.Sprintf("%v", enabled))
	return nil
}

// GetVolume returns the current media volume via dumpsys audio.
func (c *Client) GetVolume(ctx context.Context, serial string) (int, error) {
	result, err := c.Shell(ctx, serial, "dumpsys audio | grep -A1 'STREAM_MUSIC'")
	if err != nil {
		return 0, fmt.Errorf("get volume: %w", err)
	}
	for line := range strings.SplitSeq(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "- STREAM_MUSIC:"); ok {
			// Parse "- STREAM_MUSIC:\n   Mute count: 0\n   Max: 15\n   Headset: 10\n   Speaker: 11"
			_ = after
			continue
		}
		if strings.HasPrefix(line, "Speaker:") || strings.HasPrefix(line, "Headset:") {
			if _, val, ok := strings.Cut(line, ":"); ok {
				v, err := strconv.Atoi(strings.TrimSpace(val))
				if err == nil {
					return v, nil
				}
			}
		}
	}
	return 0, nil
}

// VolumeUp increases the media volume by one step.
func (c *Client) VolumeUp(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeyVolumeUp)
}

// VolumeDown decreases the media volume by one step.
func (c *Client) VolumeDown(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeyVolumeDown)
}

// VolumeMute toggles the mute state.
func (c *Client) VolumeMute(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeyMute)
}

// MediaPlay sends the media play key event.
func (c *Client) MediaPlay(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeyMediaPlay)
}

// MediaPause sends the media pause key event.
func (c *Client) MediaPause(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeyMediaPause)
}

// MediaNext sends the media next track key event.
func (c *Client) MediaNext(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeyMediaNext)
}

// MediaPrev sends the media previous track key event.
func (c *Client) MediaPrev(ctx context.Context, serial string) error {
	return c.KeyEvent(ctx, serial, KeyMediaPrev)
}

// ExpandNotifications pulls down the notification shade.
func (c *Client) ExpandNotifications(ctx context.Context, serial string) error {
	_, err := c.ShellArgs(ctx, serial, "cmd", "statusbar", "expand-notifications")
	if err != nil {
		return fmt.Errorf("expand notifications: %w", err)
	}
	return nil
}

// CollapseNotifications closes the notification shade.
func (c *Client) CollapseNotifications(ctx context.Context, serial string) error {
	_, err := c.ShellArgs(ctx, serial, "cmd", "statusbar", "collapse")
	if err != nil {
		return fmt.Errorf("collapse notifications: %w", err)
	}
	return nil
}

// ExpandQuickSettings pulls down the quick settings panel.
func (c *Client) ExpandQuickSettings(ctx context.Context, serial string) error {
	_, err := c.ShellArgs(ctx, serial, "cmd", "statusbar", "expand-settings")
	if err != nil {
		return fmt.Errorf("expand quick settings: %w", err)
	}
	return nil
}
