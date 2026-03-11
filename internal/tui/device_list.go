package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alanisme/adb-tui/internal/adb"
)

// deviceListMessage is implemented by all messages routed to DeviceListModel.
type deviceListMessage interface{ deviceListMsg() }

type devicesRefreshMsg struct {
	devices []*adb.Device
	err     error
}

type wirelessSwitchMsg struct {
	ip  string // device IP after tcpip switch
	err error
}

type serverActionMsg struct {
	action  string
	isStart bool
	err     error
}

type adbVersionMsg struct {
	version string
}

func (devicesRefreshMsg) deviceListMsg()  {}
func (wirelessSwitchMsg) deviceListMsg()  {}
func (serverActionMsg) deviceListMsg()    {}
func (adbVersionMsg) deviceListMsg()      {}
func (tickDevicesMsg) deviceListMsg()     {}

type DeviceListModel struct {
	client            *adb.Client
	devices           []*adb.Device
	cursor            int
	width             int
	height            int
	err               error
	loading           bool
	connectInput      textinput.Model
	showConnect       bool
	showPair          bool
	pairStep          int // 0=host, 1=code
	pairHost          string
	pairInput         textinput.Model
	statusMsg         string
	pauseRefreshUntil time.Time // suppress auto-refresh after kill-server
	adbVersion        string
}

func NewDeviceListModel(client *adb.Client) DeviceListModel {
	ti := textinput.New()
	ti.Placeholder = "host:port (e.g. 192.168.1.100:5555)"
	ti.CharLimit = 64

	pi := textinput.New()
	pi.Placeholder = "host:port (e.g. 192.168.1.100:37847)"
	pi.CharLimit = 64

	return DeviceListModel{
		client:       client,
		connectInput: ti,
		pairInput:    pi,
	}
}

func (m DeviceListModel) Init() tea.Cmd {
	return tea.Batch(m.refreshDevices(), m.fetchADBVersion())
}

func (m DeviceListModel) IsInputCaptured() bool {
	return m.showConnect || m.showPair
}

func (m DeviceListModel) fetchADBVersion() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		ver, err := client.Version(ctx)
		if err != nil {
			return adbVersionMsg{}
		}
		// Extract version number from multiline output (e.g. "Android Debug Bridge version 1.0.41\nVersion 35.0.2-12147458")
		for _, line := range strings.Split(ver, "\n") {
			if strings.HasPrefix(line, "Version ") {
				return adbVersionMsg{version: strings.TrimPrefix(line, "Version ")}
			}
		}
		// Fallback: try first line
		if idx := strings.Index(ver, "version "); idx >= 0 {
			return adbVersionMsg{version: strings.TrimSpace(ver[idx+8:])}
		}
		return adbVersionMsg{version: strings.TrimSpace(strings.Split(ver, "\n")[0])}
	}
}

func (m DeviceListModel) Update(msg tea.Msg) (DeviceListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case devicesRefreshMsg:
		m.loading = false
		// During refresh pause (after kill-server), discard stale error results
		if time.Now().Before(m.pauseRefreshUntil) {
			return m, tickRefreshDevices()
		}
		if msg.err != nil {
			m.err = simplifyADBError(msg.err)
		} else {
			m.err = nil
			m.devices = msg.devices
			if m.cursor >= len(m.devices) {
				m.cursor = max(len(m.devices)-1, 0)
			}
		}
		return m, tickRefreshDevices()

	case wirelessSwitchMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = SuccessStyle.Render(fmt.Sprintf("Wireless ready: %s:5555 — you can unplug USB now", msg.ip))
		}
		return m, tea.Batch(m.refreshDevices(), clearStatusAfter(8*time.Second))

	case serverActionMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + ": " + msg.err.Error())
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action)
		}
		if msg.isStart {
			// Only refresh after start; after kill the server is gone
			return m, tea.Batch(m.refreshDevices(), clearStatusAfter(5*time.Second))
		}
		// After kill-server, pause auto-refresh for 5s to let the port release
		m.devices = nil
		m.cursor = 0
		m.err = nil
		m.pauseRefreshUntil = time.Now().Add(5 * time.Second)
		return m, clearStatusAfter(5 * time.Second)

	case adbVersionMsg:
		m.adbVersion = msg.version
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.showConnect {
			return m.updateConnectInput(msg)
		}
		if m.showPair {
			return m.updatePairInput(msg)
		}
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, DefaultKeyMap.Down):
			if m.cursor < len(m.devices)-1 {
				m.cursor++
			}
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.cursor = 0
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			if len(m.devices) > 0 {
				m.cursor = len(m.devices) - 1
			}
		case key.Matches(msg, DefaultKeyMap.Refresh):
			m.loading = true
			return m, m.refreshDevices()
		case msg.String() == "c":
			m.showConnect = true
			m.connectInput.Focus()
			return m, textinput.Blink
		case msg.String() == "p":
			m.showPair = true
			m.pairStep = 0
			m.pairInput.Placeholder = "host:port (e.g. 192.168.1.100:37847)"
			m.pairInput.Focus()
			return m, textinput.Blink
		case msg.String() == "x":
			if sel := m.SelectedDevice(); sel != nil {
				if !isTCPDevice(sel.Serial) {
					m.err = fmt.Errorf("disconnect only works for TCP/IP devices")
					return m, nil
				}
				return m, m.disconnectDevice(sel.Serial)
			}
		case msg.String() == "t":
			if sel := m.SelectedDevice(); sel != nil {
				if isTCPDevice(sel.Serial) {
					m.err = fmt.Errorf("device is already connected via TCP/IP")
					return m, nil
				}
				return m, m.switchToWireless(sel.Serial)
			}
		case msg.String() == "u":
			if sel := m.SelectedDevice(); sel != nil {
				if !isTCPDevice(sel.Serial) {
					m.err = fmt.Errorf("device is already in USB mode")
					return m, nil
				}
				return m, m.switchToUSB(sel.Serial)
			}
		case msg.String() == "S":
			return m, m.startServer()
		case msg.String() == "K":
			return m, m.killServer()
		}
	}
	return m, nil
}

func (m DeviceListModel) updateConnectInput(msg tea.KeyMsg) (DeviceListModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		host := m.connectInput.Value()
		m.showConnect = false
		m.connectInput.Reset()
		if host != "" {
			return m, m.connectDevice(host)
		}
		return m, nil
	case tea.KeyEsc:
		m.showConnect = false
		m.connectInput.Reset()
		return m, nil
	}
	var cmd tea.Cmd
	m.connectInput, cmd = m.connectInput.Update(msg)
	return m, cmd
}

func (m DeviceListModel) updatePairInput(msg tea.KeyMsg) (DeviceListModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := m.pairInput.Value()
		if m.pairStep == 0 {
			if val != "" {
				m.pairHost = val
				m.pairStep = 1
				m.pairInput.Reset()
				m.pairInput.Placeholder = "pairing code (6 digits)"
			}
			return m, nil
		}
		// step 1: execute pairing
		code := val
		m.showPair = false
		m.pairInput.Reset()
		if code != "" {
			return m, m.pairDevice(m.pairHost, code)
		}
		return m, nil
	case tea.KeyEsc:
		m.showPair = false
		m.pairInput.Reset()
		return m, nil
	}
	var cmd tea.Cmd
	m.pairInput, cmd = m.pairInput.Update(msg)
	return m, cmd
}

func (m DeviceListModel) pairDevice(host, code string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := m.client.Pair(ctx, host, code)
		if err != nil {
			return devicesRefreshMsg{err: err}
		}
		devices, err := m.client.ListDevices(ctx)
		return devicesRefreshMsg{devices: devices, err: err}
	}
}

func (m DeviceListModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("Devices (%d)", len(m.devices))
	if m.adbVersion != "" {
		title += DimStyle.Render("  adb " + m.adbVersion)
	}
	if m.loading {
		title += DimStyle.Render("  refreshing...")
	}
	b.WriteString(HeaderStyle.Render(title) + "\n")

	if m.err != nil {
		b.WriteString("  " + ErrorStyle.Render("Error: "+m.err.Error()) + "\n\n")
	}

	if len(m.devices) == 0 {
		b.WriteString("  " + DimStyle.Render("No devices connected.") + "\n")
		b.WriteString("  " + DimStyle.Render("Press c to connect via TCP/IP, or plug in a USB device.") + "\n")
	} else {
		// table header
		header := fmt.Sprintf("  %-24s %-14s %-20s %s", "SERIAL", "STATE", "MODEL", "PRODUCT")
		b.WriteString(TableHeaderStyle.Render(header) + "\n")
		b.WriteString("  " + DimStyle.Render(strings.Repeat("─", min(m.width-4, 80))) + "\n")

		for i, dev := range m.devices {
			stateStr := DeviceStateStyle(string(dev.State)).Render(string(dev.State))
			prefix := "  "
			style := NormalStyle
			if i == m.cursor {
				prefix = CursorStyle.Render("▸ ")
				style = SelectedStyle
			}
			line := fmt.Sprintf("%-24s %s %-20s %s",
				style.Render(dev.Serial),
				lipgloss.NewStyle().Width(14).Render(stateStr),
				style.Render(dev.Model),
				style.Render(dev.Product),
			)
			b.WriteString(prefix + line + "\n")
		}
	}

	if m.statusMsg != "" {
		b.WriteString("  " + m.statusMsg + "\n")
	}

	b.WriteString("\n")
	if m.showConnect {
		b.WriteString("  " + DialogStyle.Render("Connect: "+m.connectInput.View()) + "\n")
	}
	if m.showPair {
		label := "Pair (host:port): "
		if m.pairStep == 1 {
			label = "Pair (code): "
		}
		b.WriteString("  " + DialogStyle.Render(label+m.pairInput.View()) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpBar(
		keyHint("⏎", "select"),
		keyHint("c", "connect"),
		keyHint("p", "pair"),
		keyHint("t", "wireless"),
		keyHint("u", "usb"),
		keyHint("x", "disconnect"),
		keyHint("r", "refresh"),
		keyHint("S", "start srv"),
		keyHint("K", "kill srv"),
	))

	return b.String()
}

func (m DeviceListModel) SelectedDevice() *adb.Device {
	if m.cursor >= 0 && m.cursor < len(m.devices) {
		return m.devices[m.cursor]
	}
	return nil
}

func (m DeviceListModel) SetSize(w, h int) DeviceListModel {
	m.width = w
	m.height = h
	return m
}

func (m DeviceListModel) refreshDevices() tea.Cmd {
	if time.Now().Before(m.pauseRefreshUntil) {
		// Server was recently killed; skip this cycle but keep the tick chain alive
		return tickRefreshDevices()
	}
	client := m.client
	return func() tea.Msg {
		// Fast path: short timeout for normal case
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		devices, err := client.ListDevices(ctx)
		cancel()
		if err == nil {
			return devicesRefreshMsg{devices: devices}
		}

		// Not a daemon problem — return error directly
		if !isDaemonError(err) {
			return devicesRefreshMsg{err: err}
		}

		// Auto-recovery: forcibly kill zombie adb processes on port 5037,
		// because `adb kill-server` will also hang on a zombie server.
		forceKillADBServer()
		time.Sleep(2 * time.Second)

		startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = client.StartServer(startCtx)
		startCancel()

		time.Sleep(1 * time.Second)

		retryCtx, retryCancel := context.WithTimeout(context.Background(), 5*time.Second)
		devices, err = client.ListDevices(retryCtx)
		retryCancel()

		return devicesRefreshMsg{devices: devices, err: err}
	}
}

func isDaemonError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "daemon not running") ||
		strings.Contains(msg, "failed to start daemon") ||
		strings.Contains(msg, "cannot connect to daemon") ||
		strings.Contains(msg, "Address already in use") ||
		strings.Contains(msg, "signal: killed") ||
		strings.Contains(msg, "context deadline exceeded")
}

func (m DeviceListModel) connectDevice(host string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := m.client.Connect(ctx, host)
		if err != nil {
			return devicesRefreshMsg{err: err}
		}
		devices, err := m.client.ListDevices(ctx)
		return devicesRefreshMsg{devices: devices, err: err}
	}
}

func (m DeviceListModel) disconnectDevice(serial string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.client.Disconnect(ctx, serial)
		if err != nil {
			return devicesRefreshMsg{err: err}
		}
		devices, err := m.client.ListDevices(ctx)
		return devicesRefreshMsg{devices: devices, err: err}
	}
}

func (m DeviceListModel) switchToWireless(serial string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Get device IP before switching (needed for connect after tcpip)
		ip, err := client.GetIPAddress(ctx, serial)
		if err != nil || ip == "" {
			return wirelessSwitchMsg{err: fmt.Errorf("cannot detect device IP: %v", err)}
		}

		// Switch device to TCP/IP mode on port 5555
		if err := client.TcpIp(ctx, serial, 5555); err != nil {
			return wirelessSwitchMsg{err: err}
		}

		// Wait briefly for the device to restart in TCP/IP mode
		time.Sleep(2 * time.Second)

		// Auto-connect to the device
		if err := client.Connect(ctx, ip+":5555"); err != nil {
			// tcpip succeeded but connect failed — still report IP so user can connect manually
			return wirelessSwitchMsg{ip: ip, err: nil}
		}
		return wirelessSwitchMsg{ip: ip}
	}
}

func (m DeviceListModel) switchToUSB(serial string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Usb(ctx, serial); err != nil {
			return wirelessSwitchMsg{err: fmt.Errorf("switch to USB: %w", err)}
		}
		// Disconnect TCP device after switching to USB
		_ = client.Disconnect(ctx, serial)
		time.Sleep(2 * time.Second)
		devices, err := client.ListDevices(ctx)
		if err != nil {
			return devicesRefreshMsg{err: err}
		}
		return devicesRefreshMsg{devices: devices}
	}
}

func (m DeviceListModel) startServer() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		// Kill any zombie/stuck server first to free the port
		forceKillADBServer()
		time.Sleep(2 * time.Second)

		startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := client.StartServer(startCtx)
		startCancel()

		return serverActionMsg{action: "ADB server started", isStart: true, err: err}
	}
}

func (m DeviceListModel) killServer() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		// Try graceful shutdown first (short timeout — don't hang on zombie)
		gracefulCtx, gracefulCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = client.KillServer(gracefulCtx)
		gracefulCancel()

		// Always force-kill to ensure no zombie survives
		forceKillADBServer()

		return serverActionMsg{action: "ADB server killed", isStart: false, err: nil}
	}
}

func tickRefreshDevices() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickDevicesMsg{}
	})
}

type tickDevicesMsg struct{}

// isTCPDevice returns true if the serial looks like a TCP/IP address (contains ':').
// USB devices have serials like "ABCD1234" while TCP/IP has "192.168.1.100:5555".
func isTCPDevice(serial string) bool {
	return strings.Contains(serial, ":")
}

// forceKillADBServer kills any process holding port 5037 (the ADB server port).
// This bypasses `adb kill-server` which hangs on zombie servers.
// Uses two strategies: lsof (macOS/Linux) and pkill fallback.
func forceKillADBServer() {
	// Strategy 1: find PIDs on port 5037 via lsof and kill -9
	if out, err := exec.Command("lsof", "-ti", ":5037").Output(); err == nil && len(out) > 0 {
		for _, pid := range strings.Fields(strings.TrimSpace(string(out))) {
			if pid != "" {
				_ = exec.Command("kill", "-9", pid).Run()
			}
		}
		return
	}
	// Strategy 2: fallback — kill all adb server processes by name
	// (only targets "adb" with "fork-server", won't kill adb clients)
	_ = exec.Command("pkill", "-9", "-f", "adb.*fork-server").Run()
}

// simplifyADBError condenses verbose ADB daemon errors into a single-line message.
func simplifyADBError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "signal: killed"),
		strings.Contains(msg, "context deadline exceeded"):
		return fmt.Errorf("ADB server not responding — auto-recovering...")
	case strings.Contains(msg, "Address already in use"):
		return fmt.Errorf("ADB server port in use — auto-recovering...")
	case strings.Contains(msg, "failed to start daemon"),
		strings.Contains(msg, "daemon not running"):
		return fmt.Errorf("ADB server not available — auto-recovering...")
	case strings.Contains(msg, "cannot connect to daemon"):
		return fmt.Errorf("Cannot connect to ADB server — auto-recovering...")
	}
	return err
}
