package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

type networkInfoMsg struct {
	interfaces  []adb.NetworkInterface
	connections []adb.NetConnection
	err         error
}

type pingResultMsg struct {
	host   string
	result *adb.PingResult
	err    error
}

type networkOverlay struct {
	interfaces  []adb.NetworkInterface
	connections []adb.NetConnection
	scroll      int
	pingInput   textinput.Model
	pingResult  *adb.PingResult
	pingHost    string
	showPing    bool
}

func newNetworkOverlay() networkOverlay {
	ti := textinput.New()
	ti.Placeholder = "hostname or IP"
	ti.CharLimit = 128
	return networkOverlay{pingInput: ti}
}

func (o networkOverlay) update(msg tea.KeyMsg, client *adb.Client, serial string, viewHeight int) (networkOverlay, tea.Cmd, bool) {
	if msg.Type == tea.KeyEsc {
		if o.showPing {
			o.showPing = false
			o.pingInput.Reset()
			o.pingInput.Blur()
			return o, nil, false
		}
		return o, nil, true
	}

	// Ping input mode
	if o.showPing {
		if msg.Type == tea.KeyEnter {
			host := o.pingInput.Value()
			o.pingInput.Reset()
			o.pingInput.Blur()
			o.showPing = false
			if host != "" {
				return o, cmdPing(client, serial, host), false
			}
			return o, nil, false
		}
		var cmd tea.Cmd
		o.pingInput, cmd = o.pingInput.Update(msg)
		return o, cmd, false
	}

	// Scrollable overview
	lines := o.networkLines()
	maxVisible := viewHeight - 4
	maxScroll := max(len(lines)-maxVisible, 0)

	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		o.scroll = max(o.scroll-1, 0)
	case key.Matches(msg, DefaultKeyMap.Down):
		o.scroll = min(o.scroll+1, maxScroll)
	case key.Matches(msg, DefaultKeyMap.HalfPageUp):
		o.scroll = max(o.scroll-maxVisible/2, 0)
	case key.Matches(msg, DefaultKeyMap.HalfPageDown):
		o.scroll = min(o.scroll+maxVisible/2, maxScroll)
	case key.Matches(msg, DefaultKeyMap.PageUp):
		o.scroll = max(o.scroll-maxVisible, 0)
	case key.Matches(msg, DefaultKeyMap.PageDown):
		o.scroll = min(o.scroll+maxVisible, maxScroll)
	case key.Matches(msg, DefaultKeyMap.Refresh):
		return o, cmdFetchNetworkInfo(client, serial), false
	case msg.String() == "p":
		o.showPing = true
		o.pingInput.Focus()
		return o, textinput.Blink, false
	}
	return o, nil, false
}

func (o networkOverlay) networkLines() []string {
	var lines []string

	lines = append(lines, "  "+AccentStyle.Render("Network Interfaces"))
	if len(o.interfaces) == 0 {
		lines = append(lines, "  "+DimStyle.Render("No interfaces found"))
	} else {
		for _, iface := range o.interfaces {
			ip := iface.IPAddress
			if ip == "" {
				ip = DimStyle.Render("no ip")
			}
			mask := ""
			if iface.Mask != "" {
				mask = "/" + iface.Mask
			}
			lines = append(lines, fmt.Sprintf("  %-16s %s%s",
				SelectedStyle.Render(iface.Name), ip, DimStyle.Render(mask)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, "  "+AccentStyle.Render("Active Connections"))
	if len(o.connections) == 0 {
		lines = append(lines, "  "+DimStyle.Render("No connections found"))
	} else {
		lines = append(lines, fmt.Sprintf("  %-8s %-24s %-24s %s",
			TableHeaderStyle.Render("Proto"),
			TableHeaderStyle.Render("Local"),
			TableHeaderStyle.Render("Remote"),
			TableHeaderStyle.Render("State")))
		for _, conn := range o.connections {
			lines = append(lines, fmt.Sprintf("  %-8s %-24s %-24s %s",
				conn.Protocol, conn.LocalAddr, conn.RemoteAddr,
				DimStyle.Render(conn.State)))
		}
	}

	if o.pingResult != nil {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("Ping: "+o.pingHost))
		pr := o.pingResult
		lines = append(lines, fmt.Sprintf("  Transmitted: %d, Received: %d, Loss: %.0f%%",
			pr.Transmitted, pr.Received, pr.LossPercent))
		if pr.AvgRTT > 0 {
			lines = append(lines, fmt.Sprintf("  Avg RTT: %.1f ms", pr.AvgRTT))
		}
	}

	return lines
}

func (o networkOverlay) view(statusMsg string, viewHeight int) string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render("Network Diagnostics") + "\n\n")

	lines := o.networkLines()
	maxVisible := viewHeight - 4
	scroll := min(o.scroll, max(len(lines)-maxVisible, 0))
	end := min(scroll+maxVisible, len(lines))
	for _, line := range lines[scroll:end] {
		b.WriteString(line + "\n")
	}

	if o.showPing {
		b.WriteString("\n  " + DialogStyle.Render("Ping host: "+o.pingInput.View()) + "\n")
	}

	if statusMsg != "" {
		b.WriteString("\n  " + statusMsg + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("j/k", "scroll"),
		keyHint("p", "ping"),
		keyHint("r", "refresh"),
		keyHint("esc", "back"),
	))
	return b.String()
}

func cmdFetchNetworkInfo(client *adb.Client, serial string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		info, err := client.GetNetworkInfo(ctx, serial)
		if err != nil {
			return networkInfoMsg{err: err}
		}
		conns, _ := client.GetNetstat(ctx, serial)
		return networkInfoMsg{interfaces: info.Interfaces, connections: conns}
	}
}

func cmdPing(client *adb.Client, serial, host string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := client.Ping(ctx, serial, host, 4)
		return pingResultMsg{host: host, result: result, err: err}
	}
}
