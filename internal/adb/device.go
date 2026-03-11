package adb

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type DeviceState string

const (
	StateDevice       DeviceState = "device"
	StateOffline      DeviceState = "offline"
	StateUnauthorized DeviceState = "unauthorized"
	StateNoDevice     DeviceState = "no device"
)

type Device struct {
	Serial      string
	State       DeviceState
	Product     string
	Model       string
	Device      string
	TransportID string
}

func (c *Client) ListDevices(ctx context.Context) ([]*Device, error) {
	result, err := c.Exec(ctx, "devices", "-l")
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}

	var devices []*Device
	for line := range strings.SplitSeq(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of") {
			continue
		}
		dev := parseDeviceLine(line)
		if dev != nil {
			devices = append(devices, dev)
		}
	}
	return devices, nil
}

func parseDeviceLine(line string) *Device {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	dev := &Device{
		Serial: parts[0],
		State:  DeviceState(parts[1]),
	}

	for _, part := range parts[2:] {
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		switch key {
		case "product":
			dev.Product = value
		case "model":
			dev.Model = value
		case "device":
			dev.Device = value
		case "transport_id":
			dev.TransportID = value
		}
	}
	return dev
}

func (c *Client) GetDevice(ctx context.Context, serial string) (*Device, error) {
	devices, err := c.ListDevices(ctx)
	if err != nil {
		return nil, err
	}
	for _, dev := range devices {
		if dev.Serial == serial {
			return dev, nil
		}
	}
	return nil, fmt.Errorf("device %s not found", serial)
}

func (c *Client) WaitForDevice(ctx context.Context, serial string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			dev, err := c.GetDevice(ctx, serial)
			if err == nil && dev.State == StateDevice {
				return nil
			}
		}
	}
}
