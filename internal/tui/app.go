package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alanisme/adb-tui/internal/adb"
	"github.com/alanisme/adb-tui/internal/config"
)

type View int

const (
	ViewDevices View = iota
	ViewDeviceInfo
	ViewShell
	ViewLogcat
	ViewFiles
	ViewPackages
	ViewForward
	ViewInput
	ViewProcess
	ViewSettings
	ViewPerformance
)

var viewNames = []string{
	"Devices",
	"Info",
	"Shell",
	"Logcat",
	"Files",
	"Packages",
	"Forward",
	"Input",
	"Process",
	"Settings",
	"Perf",
}

var viewKeys = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "-"}

type Model struct {
	client           *adb.Client
	activeView       View
	width            int
	height           int
	serial           string
	showHelp         bool
	showThemePicker  bool
	themeCursor      int
	confirmQuit      bool
	deviceList       DeviceListModel
	deviceInfo       DeviceInfoModel
	shell            ShellModel
	logcat           LogcatModel
	files            FileModel
	packages         PackageModel
	forward          ForwardModel
	input            InputModel
	process          ProcessModel
	settings         SettingsModel
	performance      PerformanceModel
}

func NewApp(client *adb.Client) *App {
	m := Model{
		client:      client,
		activeView:  ViewDevices,
		deviceList:  NewDeviceListModel(client),
		deviceInfo:  NewDeviceInfoModel(client),
		shell:       NewShellModel(client),
		logcat:      NewLogcatModel(client),
		files:       NewFileModel(client),
		packages:    NewPackageModel(client),
		forward:     NewForwardModel(client),
		input:       NewInputModel(client),
		process:     NewProcessModel(client),
		settings:    NewSettingsModel(client),
		performance: NewPerformanceModel(client),
	}
	return &App{model: m}
}

type App struct {
	model Model
}


func (a *App) Run() error {
	p := tea.NewProgram(a.model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return m.deviceList.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentHeight := m.contentHeight()
		m.deviceList = m.deviceList.SetSize(m.width, contentHeight)
		m.deviceInfo = m.deviceInfo.SetSize(m.width, contentHeight)
		m.shell = m.shell.SetSize(m.width, contentHeight)
		m.logcat = m.logcat.SetSize(m.width, contentHeight)
		m.files = m.files.SetSize(m.width, contentHeight)
		m.packages = m.packages.SetSize(m.width, contentHeight)
		m.forward = m.forward.SetSize(m.width, contentHeight)
		m.input = m.input.SetSize(m.width, contentHeight)
		m.process = m.process.SetSize(m.width, contentHeight)
		m.settings = m.settings.SetSize(m.width, contentHeight)
		m.performance = m.performance.SetSize(m.width, contentHeight)
		return m, nil

	case tea.KeyMsg:
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		if m.showThemePicker {
			return m.updateThemePicker(msg)
		}

		if m.isInputCaptured() && !m.isGlobalKey(msg) {
			return m.updateActiveView(msg)
		}

		// Quit confirmation: q once shows prompt, q again confirms
		if m.confirmQuit {
			if key.Matches(msg, DefaultKeyMap.QuitConfirm) || key.Matches(msg, DefaultKeyMap.Enter) {
				m.logcat.Stop()
				return m, tea.Quit
			}
			m.confirmQuit = false
			return m, nil
		}

		switch {
		case key.Matches(msg, DefaultKeyMap.Quit):
			m.logcat.Stop()
			return m, tea.Quit
		case key.Matches(msg, DefaultKeyMap.QuitConfirm):
			if !m.isInputCaptured() {
				m.confirmQuit = true
				return m, nil
			}
		case key.Matches(msg, DefaultKeyMap.Help):
			m.showHelp = true
			return m, nil
		case msg.String() == "T":
			m.showThemePicker = true
			names := ThemeNames()
			for i, n := range names {
				if n == activeTheme.Name {
					m.themeCursor = i
					break
				}
			}
			return m, nil
		case key.Matches(msg, DefaultKeyMap.Tab1):
			return m.switchView(ViewDevices)
		case key.Matches(msg, DefaultKeyMap.Tab2):
			return m.switchView(ViewDeviceInfo)
		case key.Matches(msg, DefaultKeyMap.Tab3):
			return m.switchView(ViewShell)
		case key.Matches(msg, DefaultKeyMap.Tab4):
			return m.switchView(ViewLogcat)
		case key.Matches(msg, DefaultKeyMap.Tab5):
			return m.switchView(ViewFiles)
		case key.Matches(msg, DefaultKeyMap.Tab6):
			return m.switchView(ViewPackages)
		case key.Matches(msg, DefaultKeyMap.Tab7):
			return m.switchView(ViewForward)
		case key.Matches(msg, DefaultKeyMap.Tab8):
			return m.switchView(ViewInput)
		case key.Matches(msg, DefaultKeyMap.Tab9):
			return m.switchView(ViewProcess)
		case key.Matches(msg, DefaultKeyMap.Tab10):
			return m.switchView(ViewSettings)
		case key.Matches(msg, DefaultKeyMap.Tab11):
			return m.switchView(ViewPerformance)
		case key.Matches(msg, DefaultKeyMap.Tab):
			next := View((int(m.activeView) + 1) % len(viewNames))
			return m.switchView(next)
		case key.Matches(msg, DefaultKeyMap.ShiftTab):
			prev := View((int(m.activeView) - 1 + len(viewNames)) % len(viewNames))
			return m.switchView(prev)
		}

		return m.updateActiveView(msg)

	// --- Marker-interface routing ---
	// Each view's messages implement a marker interface, so adding a new
	// message type only requires implementing the interface on it — no
	// changes to this file are needed.

	case deviceListMessage:
		// Special: devicesRefreshMsg needs extra logic for device disappearance
		if refresh, ok := msg.(devicesRefreshMsg); ok {
			return m.handleDevicesRefresh(refresh)
		}
		// Special: tick only when active
		if _, ok := msg.(tickDevicesMsg); ok {
			if m.activeView != ViewDevices {
				return m, nil
			}
			return m.updateDeviceList(m.deviceList.refreshDevices())
		}
		var cmd tea.Cmd
		m.deviceList, cmd = m.deviceList.Update(msg)
		return m, cmd

	case deviceInfoMessage:
		var cmd tea.Cmd
		m.deviceInfo, cmd = m.deviceInfo.Update(msg)
		return m, cmd

	case shellMessage:
		var cmd tea.Cmd
		m.shell, cmd = m.shell.Update(msg)
		return m, cmd

	case logcatMessage:
		var cmd tea.Cmd
		m.logcat, cmd = m.logcat.Update(msg)
		return m, cmd

	case fileMessage:
		var cmd tea.Cmd
		m.files, cmd = m.files.Update(msg)
		return m, cmd

	case packageMessage:
		var cmd tea.Cmd
		m.packages, cmd = m.packages.Update(msg)
		return m, cmd

	case forwardMessage:
		var cmd tea.Cmd
		m.forward, cmd = m.forward.Update(msg)
		return m, cmd

	case inputMessage:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case processMessage:
		// Special: tick only when active
		if _, ok := msg.(processTickMsg); ok {
			if m.activeView != ViewProcess {
				return m, nil
			}
			m.process, _ = m.process.Update(msg)
			return m, m.process.fetchProcesses()
		}
		var cmd tea.Cmd
		m.process, cmd = m.process.Update(msg)
		return m, cmd

	case settingsMessage:
		var cmd tea.Cmd
		m.settings, cmd = m.settings.Update(msg)
		return m, cmd

	case perfMessage:
		// Special: tick only when active
		if _, ok := msg.(perfTickMsg); ok {
			if m.activeView != ViewPerformance {
				return m, nil
			}
			m.performance, _ = m.performance.Update(msg)
			return m, m.performance.fetchData()
		}
		var cmd tea.Cmd
		m.performance, cmd = m.performance.Update(msg)
		return m, cmd

	// Clear status messages — broadcast to ALL views so no list to maintain.
	case clearStatusMsg:
		m.broadcastClearStatus(msg)
		return m, nil
	}

	return m.updateActiveView(msg)
}

// handleDevicesRefresh processes device list updates and detects device disappearance.
func (m Model) handleDevicesRefresh(msg devicesRefreshMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.deviceList, cmd = m.deviceList.Update(msg)
	if m.serial != "" && msg.err == nil {
		found := false
		for _, d := range msg.devices {
			if d.Serial == m.serial {
				found = true
				break
			}
		}
		if !found {
			m.serial = ""
			cmd = tea.Batch(cmd, m.selectDevice(""))
		}
	}
	return m, cmd
}

// broadcastClearStatus sends clearStatusMsg to every view that may hold a
// statusMsg. Since views without statusMsg simply ignore it, this is safe
// to call unconditionally — no hand-maintained list needed.
func (m *Model) broadcastClearStatus(msg clearStatusMsg) {
	m.deviceList, _ = m.deviceList.Update(msg)
	m.deviceInfo, _ = m.deviceInfo.Update(msg)
	m.shell, _ = m.shell.Update(msg)
	m.logcat, _ = m.logcat.Update(msg)
	m.files, _ = m.files.Update(msg)
	m.packages, _ = m.packages.Update(msg)
	m.forward, _ = m.forward.Update(msg)
	m.input, _ = m.input.Update(msg)
	m.process, _ = m.process.Update(msg)
	m.settings, _ = m.settings.Update(msg)
	m.performance, _ = m.performance.Update(msg)
}

func (m Model) updateThemePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	names := ThemeNames()
	switch {
	case msg.Type == tea.KeyEsc:
		m.showThemePicker = false
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.themeCursor > 0 {
			m.themeCursor--
		}
	case key.Matches(msg, DefaultKeyMap.Down):
		if m.themeCursor < len(names)-1 {
			m.themeCursor++
		}
	case key.Matches(msg, DefaultKeyMap.Enter):
		name := names[m.themeCursor]
		if t, ok := BuiltinThemes[name]; ok {
			ApplyTheme(t)
			// Persist to config
			if cfg, err := config.Load(); err == nil {
				cfg.Theme = name
				_ = cfg.Save()
			}
		}
		m.showThemePicker = false
	}
	return m, nil
}

func (m Model) renderThemePicker() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(2).Render(
		TitleStyle.Render("Select Theme"),
	))
	b.WriteString("\n\n")

	names := ThemeNames()
	for i, name := range names {
		t, ok := BuiltinThemes[name]
		if !ok {
			continue
		}
		prefix := "  "
		style := NormalStyle
		if i == m.themeCursor {
			prefix = CursorStyle.Render("▸ ")
			style = SelectedStyle
		}
		label := style.Render(name)
		if name == activeTheme.Name {
			label += DimStyle.Render(" (current)")
		}
		// Color preview: show primary, accent, success, warning, error
		preview := " " +
			lipgloss.NewStyle().Foreground(t.Primary).Render("●") +
			lipgloss.NewStyle().Foreground(t.Accent).Render("●") +
			lipgloss.NewStyle().Foreground(t.Success).Render("●") +
			lipgloss.NewStyle().Foreground(t.Warning).Render("●") +
			lipgloss.NewStyle().Foreground(t.Error).Render("●") +
			lipgloss.NewStyle().Foreground(t.Highlight).Render("●")
		b.WriteString(prefix + label + preview + "\n")
	}

	b.WriteString("\n")
	b.WriteString(DimStyle.Render("  ⏎ apply  esc close"))
	return b.String()
}

func (m Model) contentHeight() int {
	// tab bar (1) + separator (1) + status bar (1) = 3 lines reserved
	h := m.height - 3
	if h < 10 {
		h = 10
	}
	return h
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showHelp {
		return fitHeight(m.renderHelp(), m.height)
	}

	if m.showThemePicker {
		return fitHeight(m.renderThemePicker(), m.height)
	}

	var b strings.Builder

	b.WriteString(m.renderTabBar())
	b.WriteString("\n")

	content := m.renderActiveView()
	target := m.contentHeight()

	// Truncate content to exactly target lines to prevent terminal scrolling,
	// then pad if content is shorter than target.
	content = fitHeight(content, target)

	b.WriteString(content)
	b.WriteString(m.renderStatusBar())

	return b.String()
}

func (m Model) renderTabBar() string {
	// Build all tab labels with their plain-text widths.
	type tabLabel struct {
		rendered string
		width    int
	}
	labels := make([]tabLabel, len(viewNames))
	for i, name := range viewNames {
		key := viewKeys[i]
		if View(i) == m.activeView {
			r := ActiveTabStyle.Render(key + ":" + name)
			labels[i] = tabLabel{r, len(key) + 1 + len(name)}
		} else {
			r := InactiveTabStyle.Render(key + ":" + name)
			labels[i] = tabLabel{r, len(key) + 1 + len(name)}
		}
	}

	sep := DimStyle.Render(" │ ")
	sepWidth := 3 // " │ "
	leading := 1  // leading space

	// Try showing all tabs
	totalWidth := leading
	for i, l := range labels {
		if i > 0 {
			totalWidth += sepWidth
		}
		totalWidth += l.width
	}

	if totalWidth <= m.width {
		// Everything fits — show all
		var parts []string
		for _, l := range labels {
			parts = append(parts, l.rendered)
		}
		tabs := " " + strings.Join(parts, sep)
		line := DimStyle.Render(strings.Repeat("─", m.width))
		return tabs + "\n" + line
	}

	// Narrow terminal: show a sliding window centered on active tab.
	// Always include the active tab, expand outward until width exhausted.
	active := int(m.activeView)
	lo, hi := active, active
	used := leading + labels[active].width

	for {
		expanded := false
		// Try expanding left
		if lo > 0 {
			need := sepWidth + labels[lo-1].width
			if used+need <= m.width-2 { // reserve 2 for "‹" / "›"
				lo--
				used += need
				expanded = true
			}
		}
		// Try expanding right
		if hi < len(labels)-1 {
			need := sepWidth + labels[hi+1].width
			if used+need <= m.width-2 {
				hi++
				used += need
				expanded = true
			}
		}
		if !expanded {
			break
		}
	}

	var parts []string
	if lo > 0 {
		parts = append(parts, DimStyle.Render("‹"))
	} else {
		parts = append(parts, " ")
	}
	for i := lo; i <= hi; i++ {
		if i > lo {
			parts = append(parts, sep)
		}
		parts = append(parts, labels[i].rendered)
	}
	if hi < len(labels)-1 {
		parts = append(parts, DimStyle.Render(" ›"))
	}

	tabs := strings.Join(parts, "")
	line := DimStyle.Render(strings.Repeat("─", m.width))
	return tabs + "\n" + line
}

func (m Model) renderActiveView() string {
	switch m.activeView {
	case ViewDevices:
		return m.deviceList.View()
	case ViewDeviceInfo:
		return m.deviceInfo.View()
	case ViewShell:
		return m.shell.View()
	case ViewLogcat:
		return m.logcat.View()
	case ViewFiles:
		return m.files.View()
	case ViewPackages:
		return m.packages.View()
	case ViewForward:
		return m.forward.View()
	case ViewInput:
		return m.input.View()
	case ViewProcess:
		return m.process.View()
	case ViewSettings:
		return m.settings.View()
	case ViewPerformance:
		return m.performance.View()
	default:
		return ""
	}
}

func (m Model) renderStatusBar() string {
	if m.confirmQuit {
		msg := WarningStyle.Render(" Press q or Enter to quit, any other key to cancel ")
		pad := max(m.width-lipgloss.Width(msg), 0)
		return StatusBarStyle.Width(m.width).Render(msg + strings.Repeat(" ", pad))
	}

	// Left: device info
	devicePart := DimStyle.Render("no device")
	if m.serial != "" {
		devicePart = SuccessStyle.Render(m.serial)
	}
	left := fmt.Sprintf(" %s %s", CursorStyle.Render("▍"), devicePart)

	// Right: key hints
	right := " " +
		keyHint("?", "help") + "  " +
		keyHint("tab", "next") + "  " +
		keyHint("q", "quit") + " "

	gap := max(m.width-lipgloss.Width(left)-lipgloss.Width(right), 0)

	return StatusBarStyle.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right,
	)
}

func (m Model) renderHelp() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().PaddingLeft(2).Render(
		TitleStyle.Render("Keyboard Reference"),
	))
	b.WriteString("\n\n")

	sections := []struct {
		title string
		keys  [][2]string
	}{
		{
			"Navigation",
			[][2]string{
				{"j / k / ↑ / ↓", "Move cursor up / down"},
				{"g / Home", "Go to first item"},
				{"G / End", "Go to last item"},
				{"^d / ^u", "Half-page down / up"},
				{"^f / ^b", "Full page down / up"},
				{"PgDn / PgUp", "Full page down / up"},
				{"Enter", "Select / confirm"},
				{"Esc", "Back / cancel"},
			},
		},
		{
			"Views",
			[][2]string{
				{"Tab / Shift+Tab", "Next / previous view"},
				{"1-9, 0, -", "Jump to view (when not typing)"},
			},
		},
		{
			"Actions",
			[][2]string{
				{"/", "Search / filter"},
				{"r", "Refresh data"},
				{"d", "Delete / uninstall"},
				{"i", "Install APK (packages)"},
				{"I", "Batch install from directory"},
				{"f", "Cycle filter mode"},
				{"s", "Sort (process) / screenshot (input)"},
				{"x/X", "Kill by PID / name (process)"},
				{"L", "Long press (input)"},
				{"p", "Show APK path (packages)"},
				{"t/u", "Switch wireless / USB (devices)"},
				{"S/K", "Start / kill ADB server (devices)"},
				{"P", "Reboot device (info)"},
				{"S", "System ops: root/remount (info)"},
				{"A", "Backup / restore (info)"},
				{"E", "Network diagnostics (info)"},
				{"m", "Create directory (files)"},
				{"F", "Find files by pattern (files)"},
				{"c/o", "Chmod / chown (files)"},
			},
		},
		{
			"Global",
			[][2]string{
				{"q", "Quit (with confirmation)"},
				{"^c", "Quit immediately"},
				{"?", "Toggle this help"},
				{"T", "Switch theme"},
			},
		},
	}

	for _, sec := range sections {
		line := fmt.Sprintf("  %s", AccentStyle.Render(sec.title))
		b.WriteString(line + "\n")
		for _, k := range sec.keys {
			kw := 24
			desc := k[1]
			if lipgloss.Width(k[0]) > kw {
				kw = lipgloss.Width(k[0]) + 2
			}
			fmt.Fprintf(&b, "    %-*s %s\n", kw, SelectedStyle.Render(k[0]), DimStyle.Render(desc))
		}
		b.WriteString("\n")
	}

	b.WriteString(DimStyle.Render("  Press any key to close"))
	return b.String()
}

func (m Model) switchView(v View) (tea.Model, tea.Cmd) {
	prev := m.activeView
	m.activeView = v

	if prev == ViewShell {
		m.shell = m.shell.Blur()
	}

	var cmd tea.Cmd
	switch v {
	case ViewShell:
		m.shell = m.shell.Focus()
	}

	return m, cmd
}

func (m Model) updateActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.activeView {
	case ViewDevices:
		wasDialog := m.deviceList.IsInputCaptured()
		m.deviceList, cmd = m.deviceList.Update(msg)
		if !wasDialog {
			if kmsg, ok := msg.(tea.KeyMsg); ok && key.Matches(kmsg, DefaultKeyMap.Enter) {
				if dev := m.deviceList.SelectedDevice(); dev != nil {
					m.serial = dev.Serial
					cmd = tea.Batch(cmd, m.selectDevice(dev.Serial))
				}
			}
		}
	case ViewDeviceInfo:
		m.deviceInfo, cmd = m.deviceInfo.Update(msg)
	case ViewShell:
		m.shell, cmd = m.shell.Update(msg)
	case ViewLogcat:
		m.logcat, cmd = m.logcat.Update(msg)
	case ViewFiles:
		m.files, cmd = m.files.Update(msg)
	case ViewPackages:
		m.packages, cmd = m.packages.Update(msg)
	case ViewForward:
		m.forward, cmd = m.forward.Update(msg)
	case ViewInput:
		m.input, cmd = m.input.Update(msg)
	case ViewProcess:
		m.process, cmd = m.process.Update(msg)
	case ViewSettings:
		m.settings, cmd = m.settings.Update(msg)
	case ViewPerformance:
		m.performance, cmd = m.performance.Update(msg)
	}
	return m, cmd
}

func (m Model) updateDeviceList(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	return m, cmd
}

func (m *Model) selectDevice(serial string) tea.Cmd {
	var cmds []tea.Cmd

	var cmd tea.Cmd
	m.deviceInfo, cmd = m.deviceInfo.SetDevice(serial)
	cmds = append(cmds, cmd)

	m.shell = m.shell.SetDevice(serial)

	m.logcat, cmd = m.logcat.SetDevice(serial)
	cmds = append(cmds, cmd)

	m.files, cmd = m.files.SetDevice(serial)
	cmds = append(cmds, cmd)

	m.packages, cmd = m.packages.SetDevice(serial)
	cmds = append(cmds, cmd)

	m.forward, cmd = m.forward.SetDevice(serial)
	cmds = append(cmds, cmd)

	m.input = m.input.SetDevice(serial)

	m.process, cmd = m.process.SetDevice(serial)
	cmds = append(cmds, cmd)

	m.settings, cmd = m.settings.SetDevice(serial)
	cmds = append(cmds, cmd)

	m.performance, cmd = m.performance.SetDevice(serial)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

// isInputCaptured delegates to the active view's IsInputCaptured method.
func (m Model) isInputCaptured() bool {
	switch m.activeView {
	case ViewShell:
		return m.shell.IsInputCaptured()
	case ViewDevices:
		return m.deviceList.IsInputCaptured()
	case ViewDeviceInfo:
		return m.deviceInfo.HasActiveOverlay()
	case ViewLogcat:
		return m.logcat.IsInputCaptured()
	case ViewFiles:
		return m.files.IsInputCaptured()
	case ViewPackages:
		return m.packages.IsInputCaptured()
	case ViewForward:
		return m.forward.IsInputCaptured()
	case ViewInput:
		return m.input.IsInputCaptured()
	case ViewProcess:
		return m.process.IsInputCaptured()
	case ViewSettings:
		return m.settings.IsInputCaptured()
	}
	return false
}

// isGlobalKey returns true for keys that should always be handled globally,
// even when a view is capturing input.
// Ctrl+C always passes through. Tab/Shift+Tab only pass through for views
// that don't use Tab internally (e.g. Shell editing mode). Views like Input
// tap/swipe and Forward dialog use Tab to switch between fields.
func (m Model) isGlobalKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyCtrlC {
		return true
	}
	if key.Matches(msg, DefaultKeyMap.Tab) || key.Matches(msg, DefaultKeyMap.ShiftTab) {
		// Only let Tab through for views that don't use it internally
		switch m.activeView {
		case ViewShell:
			return true
		case ViewLogcat, ViewProcess, ViewPackages, ViewFiles:
			// These use search inputs where Tab has no meaning
			return true
		}
		// Input (tap/swipe), Forward (dialog), Settings (dialog) use Tab for field switching
		return false
	}
	return false
}
