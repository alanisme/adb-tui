package adb

import (
	"context"
	"fmt"
	"strconv"
)

func (c *Client) SetBatteryLevel(ctx context.Context, serial string, level int) error {
	_, err := c.ShellArgs(ctx, serial, "dumpsys", "battery", "set", "level", strconv.Itoa(level))
	if err != nil {
		return fmt.Errorf("set battery level: %w", err)
	}
	return nil
}

func (c *Client) SetBatteryStatus(ctx context.Context, serial string, status int) error {
	// 1=unknown, 2=charging, 3=discharging, 4=not charging, 5=full
	_, err := c.ShellArgs(ctx, serial, "dumpsys", "battery", "set", "status", strconv.Itoa(status))
	if err != nil {
		return fmt.Errorf("set battery status: %w", err)
	}
	return nil
}

func (c *Client) SetBatteryPlugged(ctx context.Context, serial string, plugType int) error {
	// 0=none, 1=AC, 2=USB, 4=wireless
	_, err := c.ShellArgs(ctx, serial, "dumpsys", "battery", "set", "plugged", strconv.Itoa(plugType))
	if err != nil {
		return fmt.Errorf("set battery plugged: %w", err)
	}
	return nil
}

func (c *Client) SimulateBatteryUnplug(ctx context.Context, serial string) error {
	_, err := c.ShellArgs(ctx, serial, "dumpsys", "battery", "unplug")
	if err != nil {
		return fmt.Errorf("battery unplug: %w", err)
	}
	return nil
}

func (c *Client) ResetBattery(ctx context.Context, serial string) error {
	_, err := c.ShellArgs(ctx, serial, "dumpsys", "battery", "reset")
	if err != nil {
		return fmt.Errorf("reset battery: %w", err)
	}
	return nil
}

func BatteryStatusName(status int) string {
	switch status {
	case 1:
		return "Unknown"
	case 2:
		return "Charging"
	case 3:
		return "Discharging"
	case 4:
		return "Not charging"
	case 5:
		return "Full"
	default:
		return fmt.Sprintf("Status %d", status)
	}
}
