package adb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type BatteryInfo struct {
	Level       int
	Temperature int
	Voltage     int
	Status      string
	Health      string
}

type DisplayInfo struct {
	Width   int
	Height  int
	Density int
	FPS     float64
}

type DiskUsage struct {
	Filesystem string
	Size       string
	Used       string
	Available  string
	UsePercent string
	MountPoint string
}

type CPUInfo struct {
	User   float64
	System float64
	Idle   float64
	IOWait float64
}

func (c *Client) Dumpsys(ctx context.Context, serial, service string) (string, error) {
	result, err := c.ShellArgs(ctx, serial, "dumpsys", service)
	if err != nil {
		return "", fmt.Errorf("dumpsys %s: %w", service, err)
	}
	return result.Output, nil
}

func (c *Client) DumpsysList(ctx context.Context, serial string) ([]string, error) {
	result, err := c.Shell(ctx, serial, "dumpsys -l")
	if err != nil {
		return nil, fmt.Errorf("dumpsys list: %w", err)
	}

	return parseDumpsysList(result.Output), nil
}

func (c *Client) GetBatteryInfo(ctx context.Context, serial string) (*BatteryInfo, error) {
	result, err := c.Shell(ctx, serial, "dumpsys battery")
	if err != nil {
		return nil, fmt.Errorf("get battery info: %w", err)
	}

	return parseBatteryOutput(result.Output), nil
}

func parseBatteryStatus(val string) string {
	switch val {
	case "1":
		return "Unknown"
	case "2":
		return "Charging"
	case "3":
		return "Discharging"
	case "4":
		return "Not charging"
	case "5":
		return "Full"
	default:
		return val
	}
}

func parseBatteryHealth(val string) string {
	switch val {
	case "1":
		return "Unknown"
	case "2":
		return "Good"
	case "3":
		return "Overheat"
	case "4":
		return "Dead"
	case "5":
		return "Over voltage"
	case "6":
		return "Failure"
	case "7":
		return "Cold"
	default:
		return val
	}
}

func (c *Client) GetDisplayInfo(ctx context.Context, serial string) (*DisplayInfo, error) {
	info := &DisplayInfo{}

	w, h, err := c.GetScreenSize(ctx, serial)
	if err == nil {
		info.Width = w
		info.Height = h
	}

	densityResult, err := c.Shell(ctx, serial, "wm density")
	if err == nil {
		_, val, ok := strings.Cut(densityResult.Output, ": ")
		if ok {
			info.Density, _ = strconv.Atoi(strings.TrimSpace(val))
		}
	}

	fpsResult, err := c.Shell(ctx, serial, "dumpsys SurfaceFlinger --latency 2>/dev/null | head -1")
	if err == nil {
		val := strings.TrimSpace(fpsResult.Output)
		if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
			info.FPS = 1000000000.0 / f
		}
	}

	return info, nil
}

func (c *Client) GetWindowInfo(ctx context.Context, serial string) (string, error) {
	result, err := c.Shell(ctx, serial, "dumpsys window windows")
	if err != nil {
		return "", fmt.Errorf("get window info: %w", err)
	}
	return result.Output, nil
}

func (c *Client) GetUsageStats(ctx context.Context, serial string) (string, error) {
	result, err := c.Shell(ctx, serial, "dumpsys usagestats")
	if err != nil {
		return "", fmt.Errorf("get usage stats: %w", err)
	}
	return result.Output, nil
}

func (c *Client) GetDiskUsage(ctx context.Context, serial string) ([]DiskUsage, error) {
	result, err := c.Shell(ctx, serial, "df -h 2>/dev/null || df")
	if err != nil {
		return nil, fmt.Errorf("get disk usage: %w", err)
	}

	return parseDfOutput(result.Output), nil
}

func parseDumpsysList(output string) []string {
	var services []string
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			services = append(services, line)
		}
	}
	return services
}

func parseBatteryOutput(output string) *BatteryInfo {
	info := &BatteryInfo{}
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "level":
			info.Level, _ = strconv.Atoi(val)
		case "temperature":
			info.Temperature, _ = strconv.Atoi(val)
		case "voltage":
			info.Voltage, _ = strconv.Atoi(val)
		case "status":
			info.Status = parseBatteryStatus(val)
		case "health":
			info.Health = parseBatteryHealth(val)
		}
	}
	return info
}

func parseDfOutput(output string) []DiskUsage {
	var disks []DiskUsage
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Filesystem") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		disks = append(disks, DiskUsage{
			Filesystem: fields[0],
			Size:       fields[1],
			Used:       fields[2],
			Available:  fields[3],
			UsePercent: fields[4],
			MountPoint: fields[5],
		})
	}
	return disks
}

func (c *Client) GetCPUUsage(ctx context.Context, serial string) (*CPUInfo, error) {
	result, err := c.Shell(ctx, serial, "top -bn1 -m5 2>/dev/null | head -5 || dumpsys cpuinfo | head -1")
	if err != nil {
		return nil, fmt.Errorf("get cpu usage: %w", err)
	}

	info := &CPUInfo{}
	for line := range strings.SplitSeq(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "cpu") || strings.Contains(line, "CPU") || strings.Contains(line, "User") {
			parts := strings.Fields(line)
			for i, p := range parts {
				val := strings.TrimSuffix(p, "%")
				val = strings.TrimSuffix(val, ",")
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					continue
				}
				if i+1 < len(parts) {
					label := strings.ToLower(parts[i+1])
					switch {
					case strings.HasPrefix(label, "user"):
						info.User = f
					case strings.HasPrefix(label, "sys") || strings.HasPrefix(label, "kernel"):
						info.System = f
					case strings.HasPrefix(label, "idle"):
						info.Idle = f
					case strings.HasPrefix(label, "io"):
						info.IOWait = f
					}
				}
			}
			break
		}
	}
	return info, nil
}
