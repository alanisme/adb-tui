package tui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alanisme/adb-tui/internal/adb"
)

// perfMessage is implemented by all messages routed to PerformanceModel.
type perfMessage interface{ perfMsg() }

func (perfDataMsg) perfMsg() {}
func (perfTickMsg) perfMsg() {}

type perfDataMsg struct {
	cpuInfo   *adb.CPUInfo
	memInfo   *adb.MemInfo
	battery   *adb.BatteryInfo
	display   *adb.DisplayInfo
	topProcs  []adb.ProcessInfo
	diskUsage []adb.DiskUsage
	thermals  []thermalZone
	netStats  netIO
	cpuFreqs  []cpuFreqInfo
	gpuInfo   string
	loadAvg   string
	vmstat    string
	err       error
}

type perfTickMsg struct{}

type thermalZone struct {
	name string
	temp float64
}

type netIO struct {
	rxBytes string
	txBytes string
	iface   string
}

type cpuFreqInfo struct {
	cpu     string
	curFreq string
	maxFreq string
}

type PerformanceModel struct {
	client    *adb.Client
	serial    string
	cpuInfo   *adb.CPUInfo
	memInfo   *adb.MemInfo
	battery   *adb.BatteryInfo
	display   *adb.DisplayInfo
	topProcs  []adb.ProcessInfo
	diskUsage []adb.DiskUsage
	thermals  []thermalZone
	netStats  netIO
	cpuFreqs  []cpuFreqInfo
	gpuInfo   string
	loadAvg   string
	vmstat    string
	width     int
	height    int
	scroll    int
	err       error
	loading   bool
}

func NewPerformanceModel(client *adb.Client) PerformanceModel {
	return PerformanceModel{
		client: client,
	}
}

func (m PerformanceModel) Init() tea.Cmd {
	return nil
}

func (m PerformanceModel) Update(msg tea.Msg) (PerformanceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case perfDataMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.cpuInfo = msg.cpuInfo
			m.memInfo = msg.memInfo
			m.battery = msg.battery
			m.display = msg.display
			m.topProcs = msg.topProcs
			m.diskUsage = msg.diskUsage
			m.thermals = msg.thermals
			m.netStats = msg.netStats
			m.cpuFreqs = msg.cpuFreqs
			m.gpuInfo = msg.gpuInfo
			m.loadAvg = msg.loadAvg
			m.vmstat = msg.vmstat
		}
		return m, m.scheduleRefresh()

	case perfTickMsg:
		if m.serial != "" {
			return m, m.fetchData()
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			m.scroll = max(m.scroll-1, 0)
		case key.Matches(msg, DefaultKeyMap.Down):
			m.scroll++
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.scroll = 0
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			m.scroll = 99999 // clamped in View
		case key.Matches(msg, DefaultKeyMap.HalfPageUp):
			m.scroll = max(m.scroll-10, 0)
		case key.Matches(msg, DefaultKeyMap.HalfPageDown):
			m.scroll += 10
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.scroll = max(m.scroll-20, 0)
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.scroll += 20
		case key.Matches(msg, DefaultKeyMap.Refresh):
			m.loading = true
			return m, m.fetchData()
		}
	}
	return m, nil
}

func (m PerformanceModel) View() string {
	var lines []string

	lines = append(lines, HeaderStyle.Render("Performance Monitor"))

	if m.serial == "" {
		lines = append(lines, DimStyle.Render("  No device selected"))
		return strings.Join(lines, "\n")
	}

	if m.loading && m.cpuInfo == nil {
		lines = append(lines, DimStyle.Render("  Loading..."))
		return strings.Join(lines, "\n")
	}

	if m.err != nil {
		lines = append(lines, ErrorStyle.Render("  Error: "+m.err.Error()))
	}

	barWidth := m.width - 20
	if barWidth < 20 {
		barWidth = 20
	}
	if barWidth > 60 {
		barWidth = 60
	}

	// Load Average
	if m.loadAvg != "" {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("Load Average"))
		lines = append(lines, "  "+m.loadAvg)
	}

	// CPU Usage
	if m.cpuInfo != nil {
		cpuUsed := m.cpuInfo.User + m.cpuInfo.System
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("CPU Usage"))
		lines = append(lines, fmt.Sprintf("  User: %5.1f%%  System: %5.1f%%  Idle: %5.1f%%  IO: %5.1f%%",
			m.cpuInfo.User, m.cpuInfo.System, m.cpuInfo.Idle, m.cpuInfo.IOWait))
		lines = append(lines, "  "+renderBar(cpuUsed, 100, barWidth))
	}

	// CPU Frequencies
	if len(m.cpuFreqs) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("CPU Frequencies"))
		for _, f := range m.cpuFreqs {
			lines = append(lines, fmt.Sprintf("  %-6s  cur: %-12s  max: %s",
				DimStyle.Render(f.cpu), f.curFreq, f.maxFreq))
		}
	}

	// Memory
	if m.memInfo != nil {
		usedKB := m.memInfo.Total - m.memInfo.Available
		usedPct := 0.0
		if m.memInfo.Total > 0 {
			usedPct = float64(usedKB) / float64(m.memInfo.Total) * 100
		}
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("Memory"))
		lines = append(lines, fmt.Sprintf("  Total: %dMB  Used: %dMB  Free: %dMB  Buffers: %dMB  Cached: %dMB",
			m.memInfo.Total/1024, usedKB/1024, m.memInfo.Free/1024,
			m.memInfo.Buffers/1024, m.memInfo.Cached/1024))
		lines = append(lines, fmt.Sprintf("  Swap Total: %dMB  Swap Free: %dMB  (%.1f%% RAM used)",
			m.memInfo.SwapTotal/1024, m.memInfo.SwapFree/1024, usedPct))
		lines = append(lines, "  "+renderBar(usedPct, 100, barWidth))
	}

	// VM stat
	if m.vmstat != "" {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("VM Stats"))
		lines = append(lines, "  "+DimStyle.Render(m.vmstat))
	}

	// Thermal — show top zones sorted by temperature
	if len(m.thermals) > 0 {
		zones := make([]thermalZone, len(m.thermals))
		copy(zones, m.thermals)
		sort.Slice(zones, func(i, j int) bool { return zones[i].temp > zones[j].temp })

		lines = append(lines, "")
		shown := min(8, len(zones))
		extra := ""
		if len(zones) > shown {
			extra = DimStyle.Render(fmt.Sprintf(" (%d more)", len(zones)-shown))
		}
		lines = append(lines, "  "+AccentStyle.Render("Thermal Zones")+extra)

		// Compact: 2 columns
		for i := 0; i < shown; i += 2 {
			left := zones[i]
			lStyle := thermalStyle(left.temp)
			cell := fmt.Sprintf("%-24s %s", truncate(left.name, 24), lStyle.Render(fmt.Sprintf("%5.1f°C", left.temp)))
			if i+1 < shown {
				right := zones[i+1]
				rStyle := thermalStyle(right.temp)
				cell += fmt.Sprintf("   %-24s %s", truncate(right.name, 24), rStyle.Render(fmt.Sprintf("%5.1f°C", right.temp)))
			}
			lines = append(lines, "  "+cell)
		}
	}

	// GPU
	if m.gpuInfo != "" {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("GPU"))
		lines = append(lines, "  "+m.gpuInfo)
	}

	// Network I/O
	if m.netStats.iface != "" {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("Network I/O")+" "+DimStyle.Render("("+m.netStats.iface+")"))
		lines = append(lines, fmt.Sprintf("  RX: %s   TX: %s", m.netStats.rxBytes, m.netStats.txBytes))
	}

	// Top Processes
	if len(m.topProcs) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("Top Processes by CPU"))
		lines = append(lines, fmt.Sprintf("  %-8s %7s %7s  %s",
			TableHeaderStyle.Render("PID"),
			TableHeaderStyle.Render("CPU%"),
			TableHeaderStyle.Render("MEM%"),
			TableHeaderStyle.Render("NAME")))
		count := min(10, len(m.topProcs))
		for i := range count {
			p := m.topProcs[i]
			style := NormalStyle
			if p.CPU > 50 {
				style = ErrorStyle
			} else if p.CPU > 20 {
				style = WarningStyle
			}
			lines = append(lines, style.Render(fmt.Sprintf("  %-8d %6.1f%% %6.1f%%  %s",
				p.PID, p.CPU, p.MEM, p.Name)))
		}
	}

	// Battery
	if m.battery != nil {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("Battery"))
		tempC := float64(m.battery.Temperature) / 10.0
		voltV := float64(m.battery.Voltage) / 1000.0
		lines = append(lines, fmt.Sprintf("  Level: %d%%  Temp: %.1f°C  Voltage: %.2fV  Status: %s  Health: %s",
			m.battery.Level, tempC, voltV, m.battery.Status, m.battery.Health))
		lines = append(lines, "  "+renderBar(float64(m.battery.Level), 100, barWidth))
	}

	// Display
	if m.display != nil {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("Display"))
		fpsStr := "N/A"
		if m.display.FPS > 0 {
			fpsStr = fmt.Sprintf("%.0f", m.display.FPS)
		}
		lines = append(lines, fmt.Sprintf("  Resolution: %dx%d  Density: %ddpi  FPS: %s",
			m.display.Width, m.display.Height, m.display.Density, fpsStr))
	}

	// Disk
	if len(m.diskUsage) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+AccentStyle.Render("Disk Usage"))
		lines = append(lines, fmt.Sprintf("  %-20s %8s %8s %8s %6s  %s",
			TableHeaderStyle.Render("FILESYSTEM"),
			TableHeaderStyle.Render("SIZE"),
			TableHeaderStyle.Render("USED"),
			TableHeaderStyle.Render("AVAIL"),
			TableHeaderStyle.Render("USE%"),
			TableHeaderStyle.Render("MOUNT")))
		for _, d := range m.diskUsage {
			fs := d.Filesystem
			if len(fs) > 20 {
				fs = fs[:17] + "..."
			}
			mount := d.MountPoint
			if len(mount) > 20 {
				mount = mount[:17] + "..."
			}
			lines = append(lines, DimStyle.Render(fmt.Sprintf("  %-20s %8s %8s %8s %6s  %s",
				fs, d.Size, d.Used, d.Available, d.UsePercent, mount)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, helpBar(
		keyHint("j/k", "scroll"),
		keyHint("g/G", "top/bottom"),
		keyHint("r", "refresh"),
		DimStyle.Render("auto:5s"),
	))

	// Scrolling — use local variable since View() is a value receiver
	viewHeight := safeViewHeight(m.height, 4, 30)

	scroll := max(0, min(m.scroll, max(len(lines)-viewHeight, 0)))
	end := min(scroll+viewHeight, len(lines))

	return strings.Join(lines[scroll:end], "\n")
}

func (m PerformanceModel) SetDevice(serial string) (PerformanceModel, tea.Cmd) {
	m.serial = serial
	m.cpuInfo = nil
	m.memInfo = nil
	m.battery = nil
	m.display = nil
	m.topProcs = nil
	m.diskUsage = nil
	m.thermals = nil
	m.cpuFreqs = nil
	m.gpuInfo = ""
	m.loadAvg = ""
	m.vmstat = ""
	m.scroll = 0
	m.err = nil
	if serial == "" {
		return m, nil
	}
	m.loading = true
	return m, m.fetchData()
}

func (m PerformanceModel) SetSize(w, h int) PerformanceModel {
	m.width = w
	m.height = h
	return m
}

func (m PerformanceModel) fetchData() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		result := perfDataMsg{}

		cpuInfo, err := client.GetCPUUsage(ctx, serial)
		if err != nil {
			result.err = err
			return result
		}
		result.cpuInfo = cpuInfo

		memInfo, _ := client.GetMemInfo(ctx, serial)
		result.memInfo = memInfo

		battery, _ := client.GetBatteryInfo(ctx, serial)
		result.battery = battery

		display, _ := client.GetDisplayInfo(ctx, serial)
		result.display = display

		procs, _ := client.ListProcesses(ctx, serial)
		if len(procs) > 0 {
			sort.Slice(procs, func(i, j int) bool {
				return procs[i].CPU > procs[j].CPU
			})
			if len(procs) > 10 {
				procs = procs[:10]
			}
		}
		result.topProcs = procs

		disks, _ := client.GetDiskUsage(ctx, serial)
		result.diskUsage = disks

		// Load average
		if r, err := client.Shell(ctx, serial, "cat /proc/loadavg"); err == nil && r.Output != "" {
			result.loadAvg = strings.TrimSpace(r.Output)
		}

		// Thermal zones
		if r, err := client.Shell(ctx, serial, "for f in /sys/class/thermal/thermal_zone*/temp; do echo \"$(dirname $f | xargs -I{} cat {}/type):$(cat $f)\"; done 2>/dev/null"); err == nil {
			var zones []thermalZone
			for line := range strings.SplitSeq(r.Output, "\n") {
				line = strings.TrimSpace(line)
				if name, val, ok := strings.Cut(line, ":"); ok {
					if t, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
						if t > 1000 {
							t /= 1000 // millidegree to degree
						}
						if t > 0 && t < 150 {
							zones = append(zones, thermalZone{name: strings.TrimSpace(name), temp: t})
						}
					}
				}
			}
			result.thermals = zones
		}

		// CPU frequencies per core
		if r, err := client.Shell(ctx, serial, "for i in /sys/devices/system/cpu/cpu[0-9]*; do echo \"$(basename $i):$(cat $i/cpufreq/scaling_cur_freq 2>/dev/null):$(cat $i/cpufreq/cpuinfo_max_freq 2>/dev/null)\"; done 2>/dev/null"); err == nil {
			var freqs []cpuFreqInfo
			for line := range strings.SplitSeq(r.Output, "\n") {
				parts := strings.SplitN(strings.TrimSpace(line), ":", 3)
				if len(parts) == 3 && parts[1] != "" {
					freqs = append(freqs, cpuFreqInfo{
						cpu:     parts[0],
						curFreq: formatFreq(parts[1]),
						maxFreq: formatFreq(parts[2]),
					})
				}
			}
			result.cpuFreqs = freqs
		}

		// Network I/O
		if r, err := client.Shell(ctx, serial, "cat /proc/net/dev 2>/dev/null | grep wlan0"); err == nil && r.Output != "" {
			result.netStats = parseNetDev(r.Output, "wlan0")
		} else if r, err := client.Shell(ctx, serial, "cat /proc/net/dev 2>/dev/null | grep rmnet"); err == nil && r.Output != "" {
			result.netStats = parseNetDev(r.Output, "rmnet")
		}

		// GPU info
		if r, err := client.Shell(ctx, serial, "dumpsys gpu 2>/dev/null | head -5"); err == nil && r.Output != "" && !strings.Contains(r.Output, "not found") {
			result.gpuInfo = strings.TrimSpace(r.Output)
		}

		// VM stats
		if r, err := client.Shell(ctx, serial, "cat /proc/vmstat 2>/dev/null | grep -E 'pgfault|pgmajfault|pswpin|pswpout' | head -4"); err == nil && r.Output != "" {
			var parts []string
			for line := range strings.SplitSeq(r.Output, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					parts = append(parts, line)
				}
			}
			result.vmstat = strings.Join(parts, "  ")
		}

		return result
	}
}

func (m PerformanceModel) scheduleRefresh() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return perfTickMsg{}
	})
}

func thermalStyle(temp float64) lipgloss.Style {
	if temp > 60 {
		return ErrorStyle
	}
	if temp > 45 {
		return WarningStyle
	}
	return NormalStyle
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func formatFreq(kHz string) string {
	kHz = strings.TrimSpace(kHz)
	if kHz == "" {
		return "—"
	}
	v, err := strconv.Atoi(kHz)
	if err != nil {
		return kHz
	}
	if v >= 1000000 {
		return fmt.Sprintf("%.2f GHz", float64(v)/1000000)
	}
	return fmt.Sprintf("%d MHz", v/1000)
}

func parseNetDev(line, iface string) netIO {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 10 {
		return netIO{}
	}
	return netIO{
		iface:   iface,
		rxBytes: formatBytes(fields[1]),
		txBytes: formatBytes(fields[9]),
	}
}

func formatBytes(s string) string {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return s
	}
	switch {
	case v >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(v)/float64(1<<30))
	case v >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(v)/float64(1<<20))
	case v >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(v)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", v)
	}
}
