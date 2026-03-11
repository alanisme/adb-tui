package adb

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type DeviceInfo struct {
	AndroidVersion   string
	SDKVersion       string
	BuildNumber      string
	Brand            string
	Manufacturer     string
	Model            string
	Product          string
	Hardware         string
	Serial           string
	ABIs             []string
	DisplaySize      string
	DisplayDensity   string
	IPAddress        string
	MacAddress       string
	BatteryLevel     string
	BatteryStatus    string
	TotalRAM         string
	AvailableRAM     string
	TotalStorage     string
	AvailableStorage string
	Uptime           string
	CPUInfo          string
}

func (c *Client) GetProp(ctx context.Context, serial, key string) (string, error) {
	result, err := c.ShellArgs(ctx, serial, "getprop", key)
	if err != nil {
		return "", fmt.Errorf("getprop %s: %w", key, err)
	}
	return result.Output, nil
}

func (c *Client) SetProp(ctx context.Context, serial, key, value string) error {
	_, err := c.ShellArgs(ctx, serial, "setprop", key, value)
	if err != nil {
		return fmt.Errorf("setprop %s: %w", key, err)
	}
	return nil
}

func (c *Client) ListProps(ctx context.Context, serial string) (map[string]string, error) {
	result, err := c.Shell(ctx, serial, "getprop")
	if err != nil {
		return nil, fmt.Errorf("list props: %w", err)
	}

	return parsePropsOutput(result.Output), nil
}

func parsePropsOutput(output string) map[string]string {
	props := make(map[string]string)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		key, value, ok := strings.Cut(line, "]: [")
		if !ok {
			continue
		}
		key = strings.TrimPrefix(key, "[")
		value = strings.TrimSuffix(value, "]")
		props[key] = value
	}
	return props
}

func (c *Client) GetDeviceInfo(ctx context.Context, serial string) (*DeviceInfo, error) {
	props, err := c.ListProps(ctx, serial)
	if err != nil {
		return nil, err
	}

	info := &DeviceInfo{
		AndroidVersion: props["ro.build.version.release"],
		SDKVersion:     props["ro.build.version.sdk"],
		BuildNumber:    props["ro.build.display.id"],
		Brand:          props["ro.product.brand"],
		Manufacturer:   props["ro.product.manufacturer"],
		Model:          props["ro.product.model"],
		Product:        props["ro.product.name"],
		Hardware:       props["ro.hardware"],
		Serial:         props["ro.serialno"],
	}

	if abis := props["ro.product.cpu.abilist"]; abis != "" {
		info.ABIs = strings.Split(abis, ",")
	}

	// Fetch all supplementary info concurrently — each shell call is independent.
	var mu sync.Mutex
	var wg sync.WaitGroup

	type shellTask struct {
		cmd    string // Shell string (for commands needing shell features)
		args   []string
		handle func(output string)
	}

	tasks := []shellTask{
		{args: []string{"wm", "size"}, handle: func(out string) {
			if _, size, ok := strings.Cut(out, ": "); ok {
				mu.Lock()
				info.DisplaySize = strings.TrimSpace(size)
				mu.Unlock()
			}
		}},
		{args: []string{"wm", "density"}, handle: func(out string) {
			if _, density, ok := strings.Cut(out, ": "); ok {
				mu.Lock()
				info.DisplayDensity = strings.TrimSpace(density)
				mu.Unlock()
			}
		}},
		{args: []string{"ip", "route"}, handle: func(out string) {
			for line := range strings.SplitSeq(out, "\n") {
				if fields := strings.Fields(line); len(fields) > 0 {
					for i, f := range fields {
						if f == "src" && i+1 < len(fields) {
							mu.Lock()
							info.IPAddress = fields[i+1]
							mu.Unlock()
							return
						}
					}
				}
			}
		}},
		{args: []string{"cat", "/sys/class/net/wlan0/address"}, handle: func(out string) {
			mu.Lock()
			info.MacAddress = strings.TrimSpace(out)
			mu.Unlock()
		}},
		{args: []string{"dumpsys", "battery"}, handle: func(out string) {
			mu.Lock()
			defer mu.Unlock()
			for line := range strings.SplitSeq(out, "\n") {
				line = strings.TrimSpace(line)
				if key, value, ok := strings.Cut(line, ": "); ok {
					switch strings.TrimSpace(key) {
					case "level":
						info.BatteryLevel = strings.TrimSpace(value)
					case "status":
						info.BatteryStatus = strings.TrimSpace(value)
					}
				}
			}
		}},
		{args: []string{"cat", "/proc/meminfo"}, handle: func(out string) {
			mu.Lock()
			defer mu.Unlock()
			for line := range strings.SplitSeq(out, "\n") {
				if key, value, ok := strings.Cut(line, ":"); ok {
					key = strings.TrimSpace(key)
					value = strings.TrimSpace(value)
					switch key {
					case "MemTotal":
						info.TotalRAM = value
					case "MemAvailable":
						info.AvailableRAM = value
					}
				}
			}
		}},
		{args: []string{"df", "/data"}, handle: func(out string) {
			lines := strings.Split(out, "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 4 {
					mu.Lock()
					info.TotalStorage = fields[1]
					info.AvailableStorage = fields[3]
					mu.Unlock()
				}
			}
		}},
		{args: []string{"cat", "/proc/uptime"}, handle: func(out string) {
			if up, _, ok := strings.Cut(out, " "); ok {
				mu.Lock()
				info.Uptime = strings.TrimSpace(up) + "s"
				mu.Unlock()
			}
		}},
		{cmd: "cat /proc/cpuinfo | head -20", handle: func(out string) {
			mu.Lock()
			info.CPUInfo = out
			mu.Unlock()
		}},
	}

	for _, task := range tasks {
		wg.Add(1)
		go func(t shellTask) {
			defer wg.Done()
			var result *ExecResult
			var err error
			if t.cmd != "" {
				result, err = c.Shell(ctx, serial, t.cmd)
			} else {
				result, err = c.ShellArgs(ctx, serial, t.args...)
			}
			if err == nil && result != nil {
				t.handle(result.Output)
			}
		}(task)
	}
	wg.Wait()

	return info, nil
}
