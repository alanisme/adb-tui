package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/key"

	"github.com/alanisme/adb-tui/internal/adb"
)

// deviceInfoMessage is implemented by all messages routed to DeviceInfoModel.
type deviceInfoMessage interface{ deviceInfoMsg() }

type deviceInfoMsg struct {
	props map[string]string
	extra map[string]string
	err   error
}

type wifiToggleMsg struct {
	enabled bool
	err     error
}

func (deviceInfoMsg) deviceInfoMsg()       {}
func (wifiToggleMsg) deviceInfoMsg()       {}
func (batterySimMsg) deviceInfoMsg()       {}
func (displayAdjustMsg) deviceInfoMsg()    {}
func (notificationListMsg) deviceInfoMsg() {}
func (rebootMsg) deviceInfoMsg()           {}
func (systemActionMsg) deviceInfoMsg()     {}
func (backupActionMsg) deviceInfoMsg()     {}
func (networkInfoMsg) deviceInfoMsg()      {}
func (pingResultMsg) deviceInfoMsg()       {}

type InfoOverlay int

const (
	InfoOverlayNone InfoOverlay = iota
	InfoOverlayBattery
	InfoOverlayDisplay
	InfoOverlayNotifications
	InfoOverlayReboot
	InfoOverlaySystem
	InfoOverlayBackup
	InfoOverlayNetwork
)

type DeviceInfoModel struct {
	client    *adb.Client
	serial    string
	props     map[string]string
	extra     map[string]string
	err       error
	loading   bool
	scroll    int
	width     int
	height    int
	statusMsg string

	overlay InfoOverlay
	battery batteryOverlay
	display displayOverlay
	notif   notifOverlay
	reboot  rebootOverlay
	system  systemOverlay
	backup  backupOverlay
	network networkOverlay
}

// HasActiveOverlay returns true if any overlay or nested confirmation is active.
func (m DeviceInfoModel) HasActiveOverlay() bool {
	return m.overlay != InfoOverlayNone
}

func NewDeviceInfoModel(client *adb.Client) DeviceInfoModel {
	return DeviceInfoModel{
		client:  client,
		props:   make(map[string]string),
		extra:   make(map[string]string),
		battery: newBatteryOverlay(),
		display: newDisplayOverlay(),
		backup:  newBackupOverlay(),
		network: newNetworkOverlay(),
	}
}

func (m DeviceInfoModel) Init() tea.Cmd {
	return nil
}

func (m DeviceInfoModel) Update(msg tea.Msg) (DeviceInfoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case deviceInfoMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.props = msg.props
			m.extra = msg.extra
		}
		return m, nil

	case wifiToggleMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("WiFi toggle failed: " + msg.err.Error())
		} else {
			state := "enabled"
			if !msg.enabled {
				state = "disabled"
			}
			m.statusMsg = SuccessStyle.Render("WiFi " + state)
		}
		return m, clearStatusAfter(5 * time.Second)

	case batterySimMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action)
		}
		return m, clearStatusAfter(5 * time.Second)

	case displayAdjustMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action)
		}
		return m, clearStatusAfter(5 * time.Second)

	case rebootMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Reboot failed: " + msg.err.Error())
		} else {
			label := "System"
			if msg.mode != "" {
				label = msg.mode
			}
			m.statusMsg = SuccessStyle.Render("Rebooting (" + label + ")...")
		}
		return m, clearStatusAfter(5 * time.Second)

	case systemActionMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action)
		}
		return m, tea.Batch(m.fetchInfo(), clearStatusAfter(5*time.Second))

	case backupActionMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action)
		}
		return m, clearStatusAfter(8 * time.Second)

	case networkInfoMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Network: " + msg.err.Error())
			return m, clearStatusAfter(5 * time.Second)
		}
		m.network.interfaces = msg.interfaces
		m.network.connections = msg.connections
		m.network.scroll = 0
		m.overlay = InfoOverlayNetwork
		return m, nil

	case pingResultMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Ping failed: " + msg.err.Error())
			return m, clearStatusAfter(5 * time.Second)
		}
		m.network.pingHost = msg.host
		m.network.pingResult = msg.result
		return m, nil

	case notificationListMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render("Notifications: " + msg.err.Error())
			return m, clearStatusAfter(5 * time.Second)
		}
		m.notif.items = msg.items
		m.notif.scroll = 0
		m.overlay = InfoOverlayNotifications
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.overlay != InfoOverlayNone {
			return m.updateOverlay(msg)
		}

		lines := m.renderInfo()
		maxVisible := m.viewHeight()
		maxScroll := max(len(lines)-maxVisible, 0)

		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			m.scroll = max(m.scroll-1, 0)
		case key.Matches(msg, DefaultKeyMap.Down):
			m.scroll = min(m.scroll+1, maxScroll)
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.scroll = 0
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			m.scroll = maxScroll
		case key.Matches(msg, DefaultKeyMap.HalfPageUp):
			m.scroll = max(m.scroll-maxVisible/2, 0)
		case key.Matches(msg, DefaultKeyMap.HalfPageDown):
			m.scroll = min(m.scroll+maxVisible/2, maxScroll)
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.scroll = max(m.scroll-maxVisible, 0)
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.scroll = min(m.scroll+maxVisible, maxScroll)
		case key.Matches(msg, DefaultKeyMap.Refresh):
			if m.serial != "" {
				m.loading = true
				return m, m.fetchInfo()
			}
		case msg.String() == "W":
			return m, m.toggleWifi()
		case msg.String() == "B":
			if m.serial != "" {
				m.overlay = InfoOverlayBattery
				m.battery = newBatteryOverlay()
			}
		case msg.String() == "D":
			if m.serial != "" {
				m.overlay = InfoOverlayDisplay
				m.display = m.display.open()
				return m, nil
			}
		case msg.String() == "N":
			if m.serial != "" {
				return m, cmdFetchNotifications(m.client, m.serial)
			}
		case msg.String() == "P":
			if m.serial != "" {
				m.overlay = InfoOverlayReboot
				m.reboot = rebootOverlay{}
			}
		case msg.String() == "S":
			if m.serial != "" {
				m.overlay = InfoOverlaySystem
				m.system = systemOverlay{}
			}
		case msg.String() == "A":
			if m.serial != "" {
				m.overlay = InfoOverlayBackup
				m.backup.cursor = 0
				m.backup.step = 0
			}
		case msg.String() == "E":
			if m.serial != "" {
				return m, cmdFetchNetworkInfo(m.client, m.serial)
			}
		}
	}
	return m, nil
}

func (m DeviceInfoModel) updateOverlay(msg tea.KeyMsg) (DeviceInfoModel, tea.Cmd) {
	var cmd tea.Cmd
	var closed bool

	switch m.overlay {
	case InfoOverlayBattery:
		m.battery, cmd, closed = m.battery.update(msg, m.client, m.serial)
	case InfoOverlayDisplay:
		m.display, cmd, closed = m.display.update(msg, m.client, m.serial)
	case InfoOverlayNotifications:
		m.notif, cmd, closed = m.notif.update(msg, m.client, m.serial)
	case InfoOverlayReboot:
		m.reboot, cmd, closed = m.reboot.update(msg, m.client, m.serial)
	case InfoOverlaySystem:
		m.system, cmd, closed = m.system.update(msg, m.client, m.serial)
	case InfoOverlayBackup:
		m.backup, cmd, closed = m.backup.update(msg, m.client, m.serial)
	case InfoOverlayNetwork:
		m.network, cmd, closed = m.network.update(msg, m.client, m.serial, m.viewHeight())
	}

	if closed {
		m.overlay = InfoOverlayNone
	}
	return m, cmd
}

func (m DeviceInfoModel) viewHeight() int {
	return safeViewHeight(m.height, 4, 10)
}

func (m DeviceInfoModel) View() string {
	var b strings.Builder

	b.WriteString("\n")

	if m.serial == "" {
		b.WriteString("  " + DimStyle.Render("No device selected. Press Enter on a device in the Devices tab.") + "\n")
		return b.String()
	}

	if m.loading && len(m.props) == 0 {
		b.WriteString("  " + DimStyle.Render("Loading device info...") + "\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString("  " + ErrorStyle.Render("Error: "+m.err.Error()) + "\n")
	}

	// Overlays
	vh := m.viewHeight()
	switch m.overlay {
	case InfoOverlayBattery:
		b.WriteString(m.battery.view(m.statusMsg, vh))
		return b.String()
	case InfoOverlayDisplay:
		b.WriteString(m.display.view(m.statusMsg, vh))
		return b.String()
	case InfoOverlayNotifications:
		b.WriteString(m.notif.view(m.statusMsg, vh))
		return b.String()
	case InfoOverlayReboot:
		b.WriteString(m.reboot.view(m.statusMsg, vh))
		return b.String()
	case InfoOverlaySystem:
		b.WriteString(m.system.view(m.statusMsg, vh))
		return b.String()
	case InfoOverlayBackup:
		b.WriteString(m.backup.view(m.statusMsg, vh))
		return b.String()
	case InfoOverlayNetwork:
		b.WriteString(m.network.view(m.statusMsg, vh))
		return b.String()
	}

	lines := m.renderInfo()
	maxVisible := m.viewHeight()
	scroll := min(m.scroll, max(len(lines)-maxVisible, 0))

	end := min(scroll+maxVisible, len(lines))
	for _, line := range lines[scroll:end] {
		b.WriteString(line + "\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n")
	}

	b.WriteString("\n")
	pos := ""
	if len(lines) > maxVisible {
		pct := 0
		if len(lines)-maxVisible > 0 {
			pct = scroll * 100 / (len(lines) - maxVisible)
		}
		pos = ScrollPosStyle.Render(fmt.Sprintf("  %d%%", pct))
	}
	b.WriteString(helpBar(
		keyHint("j/k", "scroll"),
		keyHint("^d/^u", "½page"),
		keyHint("r", "refresh"),
		keyHint("W", "wifi"),
		keyHint("B", "battery"),
		keyHint("D", "display"),
		keyHint("N", "notifs"),
		keyHint("P", "reboot"),
		keyHint("S", "system"),
		keyHint("A", "backup"),
		keyHint("E", "network"),
	) + pos)

	return b.String()
}

// prop returns a property value or "—" dim placeholder
func (m DeviceInfoModel) prop(key string) string {
	if v := m.props[key]; v != "" {
		return v
	}
	return DimStyle.Render("—")
}

// ext returns an extra (computed) value or "—"
func (m DeviceInfoModel) ext(key string) string {
	if v := m.extra[key]; v != "" {
		return v
	}
	return DimStyle.Render("—")
}

type infoSection struct {
	title  string
	fields [][2]string // [label, value]
}

func (m DeviceInfoModel) renderInfo() []string {
	var lines []string

	sections := []infoSection{
		m.sectionIdentity(),
		m.sectionSystem(),
		m.sectionFingerprint(),
		m.sectionHardware(),
		m.sectionCPU(),
		m.sectionDisplay(),
		m.sectionMemory(),
		m.sectionBattery(),
		m.sectionNetwork(),
		m.sectionSecurity(),
		m.sectionTelephony(),
		m.sectionFeatures(),
	}

	for _, sec := range sections {
		if len(sec.fields) == 0 {
			continue
		}
		lines = append(lines, "")
		lines = append(lines, "  "+TitleStyle.Render("┌ "+sec.title))
		for _, f := range sec.fields {
			lines = append(lines, fmt.Sprintf("  │ %-22s %s", AccentStyle.Render(f[0]), f[1]))
		}
		lines = append(lines, "  "+DimStyle.Render("└"+strings.Repeat("─", 50)))
	}
	return lines
}

func (m DeviceInfoModel) sectionIdentity() infoSection {
	return infoSection{
		title: "Identity",
		fields: [][2]string{
			{"Model", m.prop("ro.product.model")},
			{"Manufacturer", m.prop("ro.product.manufacturer")},
			{"Brand", m.prop("ro.product.brand")},
			{"Product Name", m.prop("ro.product.name")},
			{"Device Name", m.prop("ro.product.device")},
			{"Marketing Name", m.prop("ro.product.marketname")},
			{"Serial", m.prop("ro.serialno")},
		},
	}
}

func (m DeviceInfoModel) sectionSystem() infoSection {
	return infoSection{
		title: "System",
		fields: [][2]string{
			{"Android Version", m.prop("ro.build.version.release")},
			{"SDK / API Level", m.prop("ro.build.version.sdk")},
			{"Build Number", m.prop("ro.build.display.id")},
			{"Build Type", m.prop("ro.build.type")},
			{"Build Tags", m.prop("ro.build.tags")},
			{"Build Date", m.prop("ro.build.date")},
			{"Security Patch", m.prop("ro.build.version.security_patch")},
			{"Bootloader", m.prop("ro.bootloader")},
			{"Baseband", m.prop("gsm.version.baseband")},
			{"Java VM", m.prop("persist.sys.dalvik.vm.lib.2")},
			{"Kernel", m.ext("kernel")},
			{"Uptime", m.ext("uptime")},
		},
	}
}

func (m DeviceInfoModel) sectionFingerprint() infoSection {
	return infoSection{
		title: "Build Fingerprint",
		fields: [][2]string{
			{"Fingerprint", m.prop("ro.build.fingerprint")},
			{"Description", m.prop("ro.build.description")},
			{"Incremental", m.prop("ro.build.version.incremental")},
			{"Codename", m.prop("ro.build.version.codename")},
			{"Base OS", m.prop("ro.build.version.base_os")},
		},
	}
}

func (m DeviceInfoModel) sectionHardware() infoSection {
	abiList := m.prop("ro.product.cpu.abilist")
	if abiList == DimStyle.Render("—") {
		abiList = m.prop("ro.product.cpu.abi")
	}
	return infoSection{
		title: "Hardware",
		fields: [][2]string{
			{"Hardware", m.prop("ro.hardware")},
			{"Chipset", m.firstProp("ro.hardware.chipname", "ro.board.platform", "ro.hardware.soc")},
			{"Board", m.prop("ro.product.board")},
			{"Platform", m.prop("ro.board.platform")},
			{"CPU ABI", abiList},
			{"ABI (32-bit)", m.prop("ro.product.cpu.abilist32")},
			{"ABI (64-bit)", m.prop("ro.product.cpu.abilist64")},
			{"GPU Renderer", m.ext("gpu_renderer")},
			{"GPU Version", m.ext("gpu_version")},
			{"OpenGL ES", m.prop("ro.opengles.version")},
		},
	}
}

func (m DeviceInfoModel) sectionCPU() infoSection {
	return infoSection{
		title: "CPU",
		fields: [][2]string{
			{"Processor", m.ext("cpu_processor")},
			{"Cores", m.ext("cpu_cores")},
			{"Architecture", m.ext("cpu_arch")},
			{"Max Frequency", m.ext("cpu_freq_max")},
			{"Current Frequency", m.ext("cpu_freq_cur")},
			{"Governor", m.ext("cpu_governor")},
		},
	}
}

func (m DeviceInfoModel) sectionDisplay() infoSection {
	return infoSection{
		title: "Display",
		fields: [][2]string{
			{"Resolution", m.ext("display_size")},
			{"Override", m.ext("display_override")},
			{"Density", m.ext("display_density")},
			{"Refresh Rate", m.ext("display_fps")},
		},
	}
}

func (m DeviceInfoModel) sectionMemory() infoSection {
	return infoSection{
		title: "Memory & Storage",
		fields: [][2]string{
			{"Total RAM", m.ext("mem_total")},
			{"Available RAM", m.ext("mem_available")},
			{"Swap Total", m.ext("swap_total")},
			{"Swap Free", m.ext("swap_free")},
			{"Internal Storage", m.ext("storage_internal")},
			{"External Storage", m.ext("storage_external")},
		},
	}
}

func (m DeviceInfoModel) sectionBattery() infoSection {
	return infoSection{
		title: "Battery",
		fields: [][2]string{
			{"Level", m.ext("bat_level")},
			{"Status", m.ext("bat_status")},
			{"Health", m.ext("bat_health")},
			{"Temperature", m.ext("bat_temp")},
			{"Voltage", m.ext("bat_voltage")},
			{"Technology", m.ext("bat_tech")},
			{"Charging Source", m.ext("bat_plugged")},
		},
	}
}

func (m DeviceInfoModel) sectionNetwork() infoSection {
	return infoSection{
		title: "Network",
		fields: [][2]string{
			{"WiFi MAC", m.ext("wifi_mac")},
			{"Bluetooth MAC", m.ext("bt_mac")},
			{"IP Address", m.ext("ip_addr")},
			{"WiFi SSID", m.ext("wifi_ssid")},
			{"WiFi Frequency", m.ext("wifi_freq")},
			{"WiFi Link Speed", m.ext("wifi_speed")},
			{"Operator", m.prop("gsm.operator.alpha")},
			{"Network Type", m.prop("gsm.network.type")},
			{"SIM Operator", m.prop("gsm.sim.operator.alpha")},
			{"SIM Country", m.prop("gsm.sim.operator.iso-country")},
			{"IMEI", m.ext("imei")},
		},
	}
}

func (m DeviceInfoModel) sectionSecurity() infoSection {
	return infoSection{
		title: "Security",
		fields: [][2]string{
			{"SELinux", m.ext("selinux")},
			{"Root Access", m.ext("root_status")},
			{"Encryption", m.prop("ro.crypto.state")},
			{"Verified Boot", m.prop("ro.boot.verifiedbootstate")},
			{"Secure Boot", m.prop("ro.secure")},
			{"Debug Mode", m.prop("ro.debuggable")},
			{"ADB Auth", m.prop("ro.adb.secure")},
		},
	}
}

func (m DeviceInfoModel) sectionTelephony() infoSection {
	return infoSection{
		title: "Telephony",
		fields: [][2]string{
			{"Phone Type", m.prop("gsm.current.phone-type")},
			{"SIM State", m.prop("gsm.sim.state")},
			{"Voice Capable", m.prop("ro.telephony.default_network")},
			{"Data Roaming", m.prop("gsm.nitz.time")},
		},
	}
}

func (m DeviceInfoModel) sectionFeatures() infoSection {
	return infoSection{
		title: "Features",
		fields: [][2]string{
			{"Bluetooth", m.ext("has_bluetooth")},
			{"NFC", m.ext("has_nfc")},
			{"GPS", m.ext("has_gps")},
			{"Camera", m.ext("has_camera")},
			{"Fingerprint", m.ext("has_fingerprint")},
			{"IR Blaster", m.ext("has_ir")},
			{"Locale", m.prop("persist.sys.locale")},
			{"Timezone", m.prop("persist.sys.timezone")},
			{"USB Config", m.prop("sys.usb.config")},
			{"USB State", m.prop("sys.usb.state")},
		},
	}
}

func (m DeviceInfoModel) firstProp(keys ...string) string {
	for _, k := range keys {
		if v := m.props[k]; v != "" {
			return v
		}
	}
	return DimStyle.Render("—")
}

func (m DeviceInfoModel) SetDevice(serial string) (DeviceInfoModel, tea.Cmd) {
	m.serial = serial
	m.scroll = 0
	if serial == "" {
		m.props = make(map[string]string)
		m.extra = make(map[string]string)
		return m, nil
	}
	m.loading = true
	return m, m.fetchInfo()
}

func (m DeviceInfoModel) SetSize(w, h int) DeviceInfoModel {
	m.width = w
	m.height = h
	return m
}

func (m DeviceInfoModel) fetchInfo() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Single getprop call gets ALL properties
		props, err := client.ListProps(ctx, serial)
		if err != nil {
			return deviceInfoMsg{err: err}
		}

		extra := make(map[string]string)

		// Kernel version
		if r, err := client.Shell(ctx, serial, "uname -r"); err == nil && r.Output != "" {
			extra["kernel"] = strings.TrimSpace(r.Output)
		}

		// Uptime
		if r, err := client.Shell(ctx, serial, "uptime -p 2>/dev/null || uptime"); err == nil {
			extra["uptime"] = strings.TrimSpace(r.Output)
		}

		// CPU info
		if r, err := client.Shell(ctx, serial, "cat /proc/cpuinfo"); err == nil {
			for line := range strings.SplitSeq(r.Output, "\n") {
				if k, v, ok := strings.Cut(line, ":"); ok {
					k = strings.TrimSpace(k)
					v = strings.TrimSpace(v)
					switch k {
					case "Processor", "model name":
						if extra["cpu_processor"] == "" {
							extra["cpu_processor"] = v
						}
					case "Hardware":
						if extra["cpu_processor"] == "" {
							extra["cpu_processor"] = v
						}
					case "CPU architecture":
						extra["cpu_arch"] = v
					}
				}
			}
		}

		// CPU cores
		if r, err := client.Shell(ctx, serial, "nproc"); err == nil && r.Output != "" {
			extra["cpu_cores"] = strings.TrimSpace(r.Output)
		}

		// CPU frequency
		if r, err := client.Shell(ctx, serial, "cat /sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq 2>/dev/null"); err == nil && r.Output != "" {
			if freq := strings.TrimSpace(r.Output); freq != "" {
				extra["cpu_freq_max"] = freq + " kHz"
			}
		}
		if r, err := client.Shell(ctx, serial, "cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq 2>/dev/null"); err == nil && r.Output != "" {
			extra["cpu_freq_cur"] = strings.TrimSpace(r.Output) + " kHz"
		}
		if r, err := client.Shell(ctx, serial, "cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null"); err == nil && r.Output != "" {
			extra["cpu_governor"] = strings.TrimSpace(r.Output)
		}

		// GPU info
		if r, err := client.Shell(ctx, serial, "dumpsys SurfaceFlinger 2>/dev/null | grep -i 'GLES'"); err == nil && r.Output != "" {
			line := strings.TrimSpace(r.Output)
			if parts := strings.SplitN(line, ",", 3); len(parts) >= 2 {
				extra["gpu_renderer"] = strings.TrimSpace(parts[1])
				if len(parts) >= 3 {
					extra["gpu_version"] = strings.TrimSpace(parts[2])
				}
			}
		}

		// Display
		if r, err := client.Shell(ctx, serial, "wm size"); err == nil {
			for line := range strings.SplitSeq(r.Output, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "Physical size:") {
					extra["display_size"] = strings.TrimPrefix(line, "Physical size: ")
				} else if strings.HasPrefix(line, "Override size:") {
					extra["display_override"] = strings.TrimPrefix(line, "Override size: ")
				}
			}
		}
		if r, err := client.Shell(ctx, serial, "wm density"); err == nil {
			if _, v, ok := strings.Cut(r.Output, ": "); ok {
				extra["display_density"] = strings.TrimSpace(v) + " dpi"
			}
		}
		if r, err := client.Shell(ctx, serial, "dumpsys display | grep -i 'mDefaultModeId\\|fps\\|refreshRate' | head -3"); err == nil && r.Output != "" {
			for line := range strings.SplitSeq(r.Output, "\n") {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "fps") || strings.Contains(line, "refreshRate") {
					extra["display_fps"] = line
					break
				}
			}
		}

		// Memory
		if r, err := client.Shell(ctx, serial, "cat /proc/meminfo"); err == nil {
			for line := range strings.SplitSeq(r.Output, "\n") {
				if k, v, ok := strings.Cut(line, ":"); ok {
					k = strings.TrimSpace(k)
					v = strings.TrimSpace(v)
					switch k {
					case "MemTotal":
						extra["mem_total"] = v
					case "MemAvailable":
						extra["mem_available"] = v
					case "SwapTotal":
						extra["swap_total"] = v
					case "SwapFree":
						extra["swap_free"] = v
					}
				}
			}
		}

		// Storage
		if r, err := client.Shell(ctx, serial, "df -h /data 2>/dev/null"); err == nil {
			lines := strings.Split(r.Output, "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 4 {
					extra["storage_internal"] = fmt.Sprintf("%s total, %s used, %s free", fields[1], fields[2], fields[3])
				}
			}
		}
		if r, err := client.Shell(ctx, serial, "df -h /sdcard 2>/dev/null"); err == nil {
			lines := strings.Split(r.Output, "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 4 {
					extra["storage_external"] = fmt.Sprintf("%s total, %s used, %s free", fields[1], fields[2], fields[3])
				}
			}
		}

		// Battery (structured)
		if bat, err := client.GetBatteryInfo(ctx, serial); err == nil {
			extra["bat_level"] = fmt.Sprintf("%d%%", bat.Level)
			extra["bat_status"] = bat.Status
			extra["bat_health"] = bat.Health
			extra["bat_temp"] = fmt.Sprintf("%.1f°C", float64(bat.Temperature)/10.0)
			extra["bat_voltage"] = fmt.Sprintf("%.2fV", float64(bat.Voltage)/1000.0)
		}
		// Additional battery fields
		if r, err := client.Shell(ctx, serial, "dumpsys battery"); err == nil {
			for line := range strings.SplitSeq(r.Output, "\n") {
				line = strings.TrimSpace(line)
				if k, v, ok := strings.Cut(line, ": "); ok {
					k = strings.TrimSpace(k)
					v = strings.TrimSpace(v)
					switch k {
					case "technology":
						extra["bat_tech"] = v
					case "plugged":
						switch v {
						case "0":
							extra["bat_plugged"] = "None"
						case "1":
							extra["bat_plugged"] = "AC"
						case "2":
							extra["bat_plugged"] = "USB"
						case "4":
							extra["bat_plugged"] = "Wireless"
						default:
							extra["bat_plugged"] = v
						}
					}
				}
			}
		}

		// Network
		if r, err := client.Shell(ctx, serial, "cat /sys/class/net/wlan0/address 2>/dev/null"); err == nil && r.Output != "" {
			extra["wifi_mac"] = strings.TrimSpace(r.Output)
		}
		if r, err := client.Shell(ctx, serial, "settings get secure bluetooth_address 2>/dev/null"); err == nil && r.Output != "" && r.Output != "null" {
			extra["bt_mac"] = strings.TrimSpace(r.Output)
		}
		if ip, err := client.GetIPAddress(ctx, serial); err == nil {
			extra["ip_addr"] = ip
		}
		if r, err := client.Shell(ctx, serial, "dumpsys wifi | grep 'mWifiInfo'"); err == nil && r.Output != "" {
			line := strings.TrimSpace(r.Output)
			if idx := strings.Index(line, "SSID:"); idx >= 0 {
				rest := line[idx+5:]
				if end := strings.Index(rest, ","); end > 0 {
					extra["wifi_ssid"] = strings.TrimSpace(rest[:end])
				}
			}
			if idx := strings.Index(line, "Frequency:"); idx >= 0 {
				rest := line[idx+10:]
				if end := strings.Index(rest, ","); end > 0 {
					extra["wifi_freq"] = strings.TrimSpace(rest[:end])
				}
			}
			if idx := strings.Index(line, "Link speed:"); idx >= 0 {
				rest := line[idx+11:]
				if end := strings.Index(rest, ","); end > 0 {
					extra["wifi_speed"] = strings.TrimSpace(rest[:end])
				}
			}
		}

		// IMEI (requires phone permission, may fail)
		if r, err := client.Shell(ctx, serial, "service call iphonesubinfo 1 2>/dev/null | grep -oP \"[0-9a-f]{8}\" 2>/dev/null"); err == nil && r.Output != "" {
			extra["imei"] = strings.TrimSpace(r.Output)
		}

		// Security
		if se, err := client.GetSELinuxStatus(ctx, serial); err == nil {
			extra["selinux"] = se
		}
		if client.IsRooted(ctx, serial) {
			extra["root_status"] = WarningStyle.Render("Yes (root)")
		} else {
			extra["root_status"] = SuccessStyle.Render("No")
		}

		// Features detection
		if r, err := client.Shell(ctx, serial, "pm list features"); err == nil {
			features := r.Output
			extra["has_bluetooth"] = featureCheck(features, "bluetooth")
			extra["has_nfc"] = featureCheck(features, "nfc")
			extra["has_gps"] = featureCheck(features, "location.gps")
			extra["has_camera"] = featureCheck(features, "camera")
			extra["has_fingerprint"] = featureCheck(features, "fingerprint")
			extra["has_ir"] = featureCheck(features, "consumerir")
		}

		return deviceInfoMsg{props: props, extra: extra}
	}
}

func (m DeviceInfoModel) toggleWifi() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		enabled, err := client.GetWifiStatus(ctx, serial)
		if err != nil {
			return wifiToggleMsg{err: err}
		}
		if enabled {
			err = client.DisableWifi(ctx, serial)
			return wifiToggleMsg{enabled: false, err: err}
		}
		err = client.EnableWifi(ctx, serial)
		return wifiToggleMsg{enabled: true, err: err}
	}
}

func featureCheck(features, keyword string) string {
	if strings.Contains(strings.ToLower(features), keyword) {
		return SuccessStyle.Render("Yes")
	}
	return DimStyle.Render("No")
}
