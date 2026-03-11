package adb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type ProcessInfo struct {
	PID  int
	User string
	CPU  float64
	MEM  float64
	Name string
}

type MemInfo struct {
	Total     int64
	Free      int64
	Available int64
	Buffers   int64
	Cached    int64
	SwapTotal int64
	SwapFree  int64
}

func (c *Client) ListProcesses(ctx context.Context, serial string) ([]ProcessInfo, error) {
	result, err := c.Shell(ctx, serial, "ps -eo PID,USER,%CPU,%MEM,NAME 2>/dev/null || ps -A -o PID,USER,%CPU,%MEM,NAME")
	if err != nil {
		result, err = c.Shell(ctx, serial, "ps -A")
		if err != nil {
			return nil, fmt.Errorf("list processes: %w", err)
		}
	}

	var procs []ProcessInfo
	for line := range strings.SplitSeq(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "PID") || strings.HasPrefix(line, "USER") {
			continue
		}
		p := parseProcessLine(line)
		if p != nil {
			procs = append(procs, *p)
		}
	}
	return procs, nil
}

func parseProcessLine(line string) *ProcessInfo {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		if len(fields) >= 2 {
			pid, err := strconv.Atoi(fields[0])
			if err != nil {
				return nil
			}
			return &ProcessInfo{
				PID:  pid,
				User: fields[1],
				Name: fields[len(fields)-1],
			}
		}
		return nil
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil
	}
	cpu, _ := strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
	mem, _ := strconv.ParseFloat(strings.TrimSuffix(fields[3], "%"), 64)
	return &ProcessInfo{
		PID:  pid,
		User: fields[1],
		CPU:  cpu,
		MEM:  mem,
		Name: fields[4],
	}
}

func (c *Client) KillProcess(ctx context.Context, serial string, pid int) error {
	_, err := c.ShellArgs(ctx, serial, "kill", strconv.Itoa(pid))
	if err != nil {
		return fmt.Errorf("kill process %d: %w", pid, err)
	}
	return nil
}

func (c *Client) KillProcessByName(ctx context.Context, serial, name string) error {
	_, err := c.ShellArgs(ctx, serial, "pkill", name)
	if err != nil {
		return fmt.Errorf("kill process %s: %w", name, err)
	}
	return nil
}

func (c *Client) GetMemInfo(ctx context.Context, serial string) (*MemInfo, error) {
	result, err := c.Shell(ctx, serial, "cat /proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("get meminfo: %w", err)
	}

	return parseMemInfoOutput(result.Output), nil
}

func parseMemInfoOutput(output string) *MemInfo {
	info := &MemInfo{}
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		val = strings.TrimSuffix(val, " kB")
		val = strings.TrimSpace(val)
		n, _ := strconv.ParseInt(val, 10, 64)
		switch key {
		case "MemTotal":
			info.Total = n
		case "MemFree":
			info.Free = n
		case "MemAvailable":
			info.Available = n
		case "Buffers":
			info.Buffers = n
		case "Cached":
			info.Cached = n
		case "SwapTotal":
			info.SwapTotal = n
		case "SwapFree":
			info.SwapFree = n
		}
	}
	return info
}

func parseTopOutput(output string, count int) []ProcessInfo {
	var procs []ProcessInfo
	headerFound := false
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "PID") || strings.HasPrefix(line, "  PID") {
			headerFound = true
			continue
		}
		if !headerFound {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		cpu, _ := strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
		mem, _ := strconv.ParseFloat(strings.TrimSuffix(fields[3], "%"), 64)
		procs = append(procs, ProcessInfo{
			PID:  pid,
			User: fields[1],
			CPU:  cpu,
			MEM:  mem,
			Name: fields[len(fields)-1],
		})
		if count > 0 && len(procs) >= count {
			break
		}
	}
	return procs
}

func (c *Client) GetAppMemInfo(ctx context.Context, serial, pkg string) (string, error) {
	result, err := c.ShellArgs(ctx, serial, "dumpsys", "meminfo", pkg)
	if err != nil {
		return "", fmt.Errorf("get app meminfo %s: %w", pkg, err)
	}
	return result.Output, nil
}

func (c *Client) GetTopProcesses(ctx context.Context, serial string, count int) ([]ProcessInfo, error) {
	result, err := c.Shell(ctx, serial, "top -n 1 -b")
	if err != nil {
		return nil, fmt.Errorf("get top processes: %w", err)
	}

	return parseTopOutput(result.Output, count), nil
}
