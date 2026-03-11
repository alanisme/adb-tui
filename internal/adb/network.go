package adb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type NetworkInfo struct {
	Interfaces []NetworkInterface
}

type NetworkInterface struct {
	Name      string
	IPAddress string
	Mask      string
	Flags     string
}

func (c *Client) GetIPAddress(ctx context.Context, serial string) (string, error) {
	result, err := c.Shell(ctx, serial, "ip route")
	if err != nil {
		return "", fmt.Errorf("get ip address: %w", err)
	}

	for line := range strings.SplitSeq(result.Output, "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "src" && i+1 < len(fields) {
				return fields[i+1], nil
			}
		}
	}
	return "", fmt.Errorf("no ip address found")
}

func (c *Client) EnableWifi(ctx context.Context, serial string) error {
	_, err := c.Shell(ctx, serial, "svc wifi enable")
	if err != nil {
		return fmt.Errorf("enable wifi: %w", err)
	}
	return nil
}

func (c *Client) DisableWifi(ctx context.Context, serial string) error {
	_, err := c.Shell(ctx, serial, "svc wifi disable")
	if err != nil {
		return fmt.Errorf("disable wifi: %w", err)
	}
	return nil
}

func (c *Client) GetWifiStatus(ctx context.Context, serial string) (bool, error) {
	result, err := c.Shell(ctx, serial, "dumpsys wifi | grep 'Wi-Fi is'")
	if err != nil {
		return false, fmt.Errorf("get wifi status: %w", err)
	}
	return strings.Contains(result.Output, "enabled"), nil
}

func (c *Client) GetNetworkInfo(ctx context.Context, serial string) (*NetworkInfo, error) {
	result, err := c.Shell(ctx, serial, "ifconfig")
	if err != nil {
		return nil, fmt.Errorf("get network info: %w", err)
	}

	info := parseIfconfigOutput(result.Output)
	return info, nil
}

func parseIfconfigOutput(output string) *NetworkInfo {
	info := &NetworkInfo{}
	var current *NetworkInterface

	for line := range strings.SplitSeq(output, "\n") {
		if line == "" {
			if current != nil {
				info.Interfaces = append(info.Interfaces, *current)
				current = nil
			}
			continue
		}

		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if current != nil {
				info.Interfaces = append(info.Interfaces, *current)
			}
			fields := strings.Fields(line)
			current = &NetworkInterface{}
			if len(fields) > 0 {
				current.Name = strings.TrimSuffix(fields[0], ":")
			}
			continue
		}

		if current == nil {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "inet ") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				if addr, mask, ok := strings.Cut(fields[1], "/"); ok {
					current.IPAddress = addr
					current.Mask = mask
				} else {
					current.IPAddress = fields[1]
				}
			}
		}
	}

	if current != nil {
		info.Interfaces = append(info.Interfaces, *current)
	}
	return info
}

type NetConnection struct {
	Protocol   string
	LocalAddr  string
	RemoteAddr string
	State      string
	PID        string
	Program    string
}

type PingResult struct {
	Transmitted int
	Received    int
	LossPercent float64
	AvgRTT      float64
}

func (c *Client) GetNetstat(ctx context.Context, serial string) ([]NetConnection, error) {
	result, err := c.Shell(ctx, serial, "ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("get netstat: %w", err)
	}

	return parseNetstatOutput(result.Output), nil
}

func parseNetstatOutput(output string) []NetConnection {
	var conns []NetConnection
	// Detect format: ss starts with "State" header, netstat with "Proto"
	isSS := strings.HasPrefix(strings.TrimSpace(output), "State") ||
		strings.HasPrefix(strings.TrimSpace(output), "Netid")
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "State") || strings.HasPrefix(line, "Proto") ||
			strings.HasPrefix(line, "Netid") || strings.HasPrefix(line, "Active") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if isSS {
			// ss format: State Recv-Q Send-Q Local:Port Peer:Port [Process]
			conn := NetConnection{
				State: fields[0],
			}
			if len(fields) >= 5 {
				conn.LocalAddr = fields[3]
				conn.RemoteAddr = fields[4]
			}
			conn.Protocol = "tcp"
			conns = append(conns, conn)
		} else {
			// netstat format: Proto Recv-Q Send-Q Local Foreign State PID/Program
			conn := NetConnection{
				Protocol: fields[0],
			}
			if len(fields) >= 5 {
				conn.LocalAddr = fields[3]
				conn.RemoteAddr = fields[4]
			}
			if len(fields) >= 6 {
				conn.State = fields[5]
			}
			if len(fields) >= 7 {
				conn.PID = fields[6]
			}
			conns = append(conns, conn)
		}
	}
	return conns
}

func (c *Client) Ping(ctx context.Context, serial, host string, count int) (*PingResult, error) {
	result, err := c.ShellArgs(ctx, serial, "ping", "-c", strconv.Itoa(count), host)
	if err != nil && result == nil {
		return nil, fmt.Errorf("ping %s: %w", host, err)
	}

	pr := &PingResult{}
	output := ""
	if result != nil {
		output = result.Output
	}
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "packets transmitted") {
			parsePingStats(line, pr)
		}
		if strings.Contains(line, "avg") && strings.Contains(line, "=") {
			parsePingRTT(line, pr)
		}
	}
	return pr, nil
}

func parsePingStats(line string, pr *PingResult) {
	// Typical Android ping output:
	//   "3 packets transmitted, 3 received, 0% packet loss, time 2003ms"
	fields := strings.Fields(line)
	for i, f := range fields {
		// Strip trailing punctuation (commas, etc.) for matching
		clean := strings.TrimRight(f, ",%")
		if f == "packets" && i > 0 {
			if strings.Contains(line, "transmitted") {
				pr.Transmitted, _ = strconv.Atoi(strings.TrimRight(fields[i-1], ","))
			}
		}
		if clean == "received" && i > 0 {
			pr.Received, _ = strconv.Atoi(strings.TrimRight(fields[i-1], ","))
		}
		if val, ok := strings.CutSuffix(f, "%"); ok {
			pr.LossPercent, _ = strconv.ParseFloat(val, 64)
		}
	}
	if pr.Transmitted > 0 && pr.LossPercent == 0 && pr.Received < pr.Transmitted {
		pr.LossPercent = float64(pr.Transmitted-pr.Received) / float64(pr.Transmitted) * 100
	}
}

func parsePingRTT(line string, pr *PingResult) {
	_, after, ok := strings.Cut(line, "=")
	if !ok {
		return
	}
	after = strings.TrimSpace(after)
	parts := strings.Split(after, "/")
	if len(parts) >= 2 {
		pr.AvgRTT, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	}
}

func (c *Client) TcpIp(ctx context.Context, serial string, port int) error {
	_, err := c.ExecDevice(ctx, serial, "tcpip", fmt.Sprintf("%d", port))
	if err != nil {
		return fmt.Errorf("tcpip: %w", err)
	}
	return nil
}

func (c *Client) Usb(ctx context.Context, serial string) error {
	_, err := c.ExecDevice(ctx, serial, "usb")
	if err != nil {
		return fmt.Errorf("usb: %w", err)
	}
	return nil
}
