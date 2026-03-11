package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

// --- helpers ---

func newTestClient() *adb.Client {
	return adb.NewClientWithPath("adb")
}

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func specialKey(kt tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: kt}
}

func containsPlainText(s, sub string) bool {
	// Strip ANSI to check if plain text is present
	return strings.Contains(stripANSI(s), sub)
}

// --- DeviceListModel ---

func TestDeviceListModel_DevicesRefreshMsg(t *testing.T) {
	m := NewDeviceListModel(newTestClient())
	m.cursor = 5 // out of range after refresh

	devices := []*adb.Device{
		{Serial: "abc123", State: "device", Model: "Pixel"},
		{Serial: "192.168.1.100:5555", State: "device", Model: "Galaxy"},
	}
	updated, _ := m.Update(devicesRefreshMsg{devices: devices})
	if updated.cursor != 1 {
		t.Fatalf("cursor should clamp to %d, got %d", 1, updated.cursor)
	}
	if len(updated.devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(updated.devices))
	}
}

func TestDeviceListModel_CursorNavigation(t *testing.T) {
	m := NewDeviceListModel(newTestClient())
	m.devices = []*adb.Device{
		{Serial: "a"}, {Serial: "b"}, {Serial: "c"},
	}

	// Down
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", m.cursor)
	}
	// Down at bottom stays
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2 (clamped), got %d", m.cursor)
	}
	// Up
	m, _ = m.Update(keyMsg("k"))
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.cursor)
	}
	// GotoTop
	m, _ = m.Update(keyMsg("g"))
	if m.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.cursor)
	}
	// GotoBottom
	m, _ = m.Update(keyMsg("G"))
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", m.cursor)
	}
}

func TestDeviceListModel_ConnectDialog(t *testing.T) {
	m := NewDeviceListModel(newTestClient())

	m, _ = m.Update(keyMsg("c"))
	if !m.showConnect {
		t.Fatal("expected connect dialog open")
	}
	m, _ = m.Update(specialKey(tea.KeyEsc))
	if m.showConnect {
		t.Fatal("expected connect dialog closed")
	}
}

func TestDeviceListModel_PairDialog(t *testing.T) {
	m := NewDeviceListModel(newTestClient())

	m, _ = m.Update(keyMsg("p"))
	if !m.showPair {
		t.Fatal("expected pair dialog open")
	}
	if m.pairStep != 0 {
		t.Fatalf("expected pairStep 0, got %d", m.pairStep)
	}
	m, _ = m.Update(specialKey(tea.KeyEsc))
	if m.showPair {
		t.Fatal("expected pair dialog closed")
	}
}

func TestDeviceListModel_DisconnectUSBBlocked(t *testing.T) {
	m := NewDeviceListModel(newTestClient())
	m.devices = []*adb.Device{{Serial: "USB123", State: "device"}}
	m.cursor = 0

	m, _ = m.Update(keyMsg("x"))
	if m.err == nil {
		t.Fatal("expected error for USB disconnect")
	}
}

func TestDeviceListModel_SelectedDevice(t *testing.T) {
	m := NewDeviceListModel(newTestClient())
	if m.SelectedDevice() != nil {
		t.Fatal("expected nil when no devices")
	}
	m.devices = []*adb.Device{{Serial: "test"}}
	m.cursor = 0
	if m.SelectedDevice() == nil {
		t.Fatal("expected device")
	}
}

func TestIsTCPDevice(t *testing.T) {
	tests := []struct {
		serial string
		want   bool
	}{
		{"192.168.1.100:5555", true},
		{"[::1]:5555", true},
		{"ABCD1234", false},
		{"emulator-5554", false},
	}
	for _, tt := range tests {
		if got := isTCPDevice(tt.serial); got != tt.want {
			t.Errorf("isTCPDevice(%q) = %v, want %v", tt.serial, got, tt.want)
		}
	}
}

// --- ProcessModel ---

func TestProcessModel_ProcessListMsg(t *testing.T) {
	m := NewProcessModel(newTestClient())
	m.serial = "test"
	m.loading = true

	procs := []adb.ProcessInfo{
		{PID: 1, Name: "init", User: "root", CPU: 0.5, MEM: 1.0},
		{PID: 100, Name: "zygote", User: "root", CPU: 10.0, MEM: 5.0},
		{PID: 200, Name: "chrome", User: "u0_a100", CPU: 50.0, MEM: 20.0},
	}
	mem := &adb.MemInfo{Total: 4096000, Free: 1024000, Available: 2048000}

	m, _ = m.Update(processListMsg{procs: procs, memInfo: mem})
	if m.loading {
		t.Fatal("expected loading false")
	}
	if len(m.processes) != 3 {
		t.Fatalf("expected 3 processes, got %d", len(m.processes))
	}
	if m.memInfo == nil {
		t.Fatal("expected memInfo")
	}
}

func TestProcessModel_SortCycling(t *testing.T) {
	m := NewProcessModel(newTestClient())
	m.serial = "test"
	m.processes = []adb.ProcessInfo{
		{PID: 1, Name: "a", CPU: 10, MEM: 5},
		{PID: 2, Name: "b", CPU: 5, MEM: 10},
	}
	m.rebuildVisible()

	if m.sortField != SortByCPU {
		t.Fatalf("expected SortByCPU, got %d", m.sortField)
	}
	m, _ = m.Update(keyMsg("s"))
	if m.sortField != SortByMEM {
		t.Fatalf("expected SortByMEM, got %d", m.sortField)
	}
	m, _ = m.Update(keyMsg("s"))
	if m.sortField != SortByPID {
		t.Fatalf("expected SortByPID, got %d", m.sortField)
	}
	m, _ = m.Update(keyMsg("s"))
	if m.sortField != SortByName {
		t.Fatalf("expected SortByName, got %d", m.sortField)
	}
	m, _ = m.Update(keyMsg("s"))
	if m.sortField != SortByCPU {
		t.Fatalf("expected SortByCPU (wrapped), got %d", m.sortField)
	}
}

func TestProcessModel_KillConfirmation(t *testing.T) {
	m := NewProcessModel(newTestClient())
	m.serial = "test"
	m.processes = []adb.ProcessInfo{{PID: 42, Name: "target"}}
	m.rebuildVisible()

	m, _ = m.Update(keyMsg("x"))
	if !m.confirmKill {
		t.Fatal("expected confirmKill true")
	}
	// Cancel
	m, _ = m.Update(keyMsg("n"))
	if m.confirmKill {
		t.Fatal("expected confirmKill false after cancel")
	}
}

func TestProcessModel_CursorNavigation(t *testing.T) {
	m := NewProcessModel(newTestClient())
	m.height = 40
	m.processes = []adb.ProcessInfo{
		{PID: 1, Name: "a"}, {PID: 2, Name: "b"}, {PID: 3, Name: "c"},
	}
	m.rebuildVisible()

	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = m.Update(keyMsg("G"))
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", m.cursor)
	}
	m, _ = m.Update(keyMsg("g"))
	if m.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.cursor)
	}
}

func TestProcessModel_Search(t *testing.T) {
	m := NewProcessModel(newTestClient())
	m.processes = []adb.ProcessInfo{
		{PID: 1, Name: "chrome", User: "u0_a1"},
		{PID: 2, Name: "system_server", User: "system"},
		{PID: 3, Name: "chromium", User: "u0_a2"},
	}
	m.rebuildVisible()
	if len(m.visible) != 3 {
		t.Fatalf("expected 3 visible, got %d", len(m.visible))
	}

	// Apply filter
	m.searchQuery = "chrome"
	m.rebuildVisible()
	if len(m.visible) != 1 {
		t.Fatalf("expected 1 visible for 'chrome', got %d", len(m.visible))
	}
}

func TestProcessModel_ClearStatus(t *testing.T) {
	m := NewProcessModel(newTestClient())
	m.statusMsg = "some status"
	m, _ = m.Update(clearStatusMsg{})
	if m.statusMsg != "" {
		t.Fatal("expected empty status after clear")
	}
}

// --- LogcatModel ---

func TestLogcatModel_EntryAppend(t *testing.T) {
	m := NewLogcatModel(newTestClient())
	m.serial = "test"
	m.height = 30

	entry := adb.LogEntry{
		Level:     "I",
		Tag:       "MyApp",
		Message:   "Hello",
		PID:       "1234",
		Timestamp: time.Now(),
	}
	m, _ = m.Update(logcatEntryMsg{entry: entry})
	if len(m.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.entries))
	}
	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered, got %d", len(m.filtered))
	}
}

func TestLogcatModel_Pause(t *testing.T) {
	m := NewLogcatModel(newTestClient())
	m.serial = "test"
	m.height = 30

	// Pause
	m, _ = m.Update(specialKey(tea.KeySpace))
	if !m.paused {
		t.Fatal("expected paused")
	}

	// Entries ignored while paused
	entry := adb.LogEntry{Level: "I", Tag: "Test", Message: "ignored"}
	m, _ = m.Update(logcatEntryMsg{entry: entry})
	if len(m.entries) != 0 {
		t.Fatal("expected no entries while paused")
	}

	// Unpause
	m, _ = m.Update(specialKey(tea.KeySpace))
	if m.paused {
		t.Fatal("expected unpaused")
	}
}

func TestLogcatModel_LevelCycling(t *testing.T) {
	m := NewLogcatModel(newTestClient())
	m.serial = "test"

	m, _ = m.Update(keyMsg("v"))
	if m.filterLevel != "D" {
		t.Fatalf("expected D, got %q", m.filterLevel)
	}
	m, _ = m.Update(keyMsg("v"))
	if m.filterLevel != "I" {
		t.Fatalf("expected I, got %q", m.filterLevel)
	}
	m, _ = m.Update(keyMsg("v"))
	if m.filterLevel != "W" {
		t.Fatalf("expected W, got %q", m.filterLevel)
	}
	m, _ = m.Update(keyMsg("v"))
	if m.filterLevel != "E" {
		t.Fatalf("expected E, got %q", m.filterLevel)
	}
	m, _ = m.Update(keyMsg("v"))
	if m.filterLevel != "F" {
		t.Fatalf("expected F, got %q", m.filterLevel)
	}
	m, _ = m.Update(keyMsg("v"))
	if m.filterLevel != "" {
		t.Fatalf("expected empty (reset), got %q", m.filterLevel)
	}
}

func TestLogcatModel_FilterMatching(t *testing.T) {
	m := NewLogcatModel(newTestClient())
	m.height = 30
	m.serial = "test"

	entries := []adb.LogEntry{
		{Level: "V", Tag: "Verbose", Message: "verbose msg", PID: "1"},
		{Level: "I", Tag: "Info", Message: "info msg", PID: "2"},
		{Level: "E", Tag: "Error", Message: "error msg", PID: "3"},
	}
	for _, e := range entries {
		m, _ = m.Update(logcatEntryMsg{entry: e})
	}
	if len(m.filtered) != 3 {
		t.Fatalf("expected 3 filtered, got %d", len(m.filtered))
	}

	// Set level filter to W
	m.filterLevel = "W"
	m.rebuildFiltered()
	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered (E only), got %d", len(m.filtered))
	}

	// Reset tag/pid/search filters (level is NOT reset by "t")
	m.filterTag = "sometag"
	m.searchQuery = "query"
	m, _ = m.Update(keyMsg("t"))
	if m.filterTag != "" || m.searchQuery != "" {
		t.Fatal("expected tag/search filters reset")
	}
	// filterLevel is preserved by "t" key
	if m.filterLevel != "W" {
		t.Fatalf("expected filterLevel preserved as W, got %q", m.filterLevel)
	}
}

func TestLogcatModel_MaxEntries(t *testing.T) {
	m := NewLogcatModel(newTestClient())
	m.serial = "test"
	m.height = 30
	m.maxEntries = 5

	for i := range 10 {
		entry := adb.LogEntry{
			Level:   "I",
			Tag:     "Test",
			Message: string(rune('0' + i)),
		}
		m, _ = m.Update(logcatEntryMsg{entry: entry})
	}
	if len(m.entries) != 5 {
		t.Fatalf("expected 5 entries (max), got %d", len(m.entries))
	}
}

func TestLogcatModel_ScrollNavigation(t *testing.T) {
	m := NewLogcatModel(newTestClient())
	m.serial = "test"
	m.height = 30

	// Add enough entries to scroll
	for i := range 50 {
		entry := adb.LogEntry{Level: "I", Tag: "Test", Message: string(rune('a' + (i % 26)))}
		m, _ = m.Update(logcatEntryMsg{entry: entry})
	}

	// GotoTop
	m, _ = m.Update(keyMsg("g"))
	if m.scroll != 0 {
		t.Fatalf("expected scroll 0, got %d", m.scroll)
	}
	if m.autoScroll {
		t.Fatal("expected autoScroll off after goto top")
	}

	// GotoBottom
	m, _ = m.Update(keyMsg("G"))
	if !m.autoScroll {
		t.Fatal("expected autoScroll on after goto bottom")
	}
}

func TestLogcatModel_StreamStopped(t *testing.T) {
	m := NewLogcatModel(newTestClient())
	m.streaming = true

	m, _ = m.Update(logcatStreamStoppedMsg{})
	if m.streaming {
		t.Fatal("expected streaming false")
	}
}

func TestLogcatModel_ClearStatus(t *testing.T) {
	m := NewLogcatModel(newTestClient())
	m.statusMsg = "exported"
	m, _ = m.Update(clearStatusMsg{})
	if m.statusMsg != "" {
		t.Fatal("expected empty status")
	}
}

// --- ShellModel ---

func TestShellModel_OutputMsg(t *testing.T) {
	m := NewShellModel(newTestClient())
	m.serial = "test"
	m.running = true

	m, _ = m.Update(shellOutputMsg{command: "ls", output: "file1\nfile2"})
	if m.running {
		t.Fatal("expected running false")
	}
	// output: prompt line + file1 + file2 + empty separator
	if len(m.output) < 3 {
		t.Fatalf("expected at least 3 output lines, got %d", len(m.output))
	}
}

func TestShellModel_NormalModeNavigation(t *testing.T) {
	m := NewShellModel(newTestClient())
	m.serial = "test"
	m.height = 30

	// Add output
	for range 30 {
		m.output = append(m.output, "line")
	}
	m.scrollToBottom()

	m, _ = m.Update(keyMsg("g"))
	if m.scroll != 0 {
		t.Fatalf("expected scroll 0, got %d", m.scroll)
	}

	m, _ = m.Update(keyMsg("G"))
	if m.scroll == 0 {
		t.Fatal("expected scroll > 0 after goto bottom")
	}
}

func TestShellModel_EnterEditMode(t *testing.T) {
	m := NewShellModel(newTestClient())
	m.serial = "test"

	m, _ = m.Update(specialKey(tea.KeyEnter))
	if !m.editing {
		t.Fatal("expected editing true")
	}
}

func TestShellModel_QuickCommandsToggle(t *testing.T) {
	m := NewShellModel(newTestClient())
	m.serial = "test"

	m, _ = m.Update(keyMsg("f"))
	if !m.quickCmds {
		t.Fatal("expected quickCmds true")
	}
	if m.quickCmdIdx != 0 {
		t.Fatalf("expected quickCmdIdx 0, got %d", m.quickCmdIdx)
	}

	// Navigate quick commands
	m, _ = m.Update(keyMsg("j"))
	if m.quickCmdIdx != 1 {
		t.Fatalf("expected quickCmdIdx 1, got %d", m.quickCmdIdx)
	}

	// Escape
	m, _ = m.Update(specialKey(tea.KeyEsc))
	if m.quickCmds {
		t.Fatal("expected quickCmds false")
	}
}

func TestShellModel_ClearCommand(t *testing.T) {
	m := NewShellModel(newTestClient())
	m.serial = "test"
	m.output = []string{"line1", "line2"}
	m.scroll = 1

	m, _ = m.Update(keyMsg("r"))
	if len(m.output) != 0 {
		t.Fatal("expected output cleared")
	}
	if m.scroll != 0 {
		t.Fatal("expected scroll reset")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"no escape", "no escape"},
	}
	for _, tt := range tests {
		got := stripANSI(tt.input)
		if got != tt.want {
			t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- ForwardModel ---

func TestForwardModel_ListMsg(t *testing.T) {
	m := NewForwardModel(newTestClient())
	m.serial = "test"

	fwd := []ForwardRule{{Serial: "test", Local: "tcp:8080", Remote: "tcp:8080"}}
	rev := []ForwardRule{{Serial: "test", Local: "tcp:9090", Remote: "tcp:9090"}}

	m, _ = m.Update(forwardListMsg{forwards: fwd, reverses: rev})
	if len(m.forwards) != 1 {
		t.Fatalf("expected 1 forward, got %d", len(m.forwards))
	}
	if len(m.reverses) != 1 {
		t.Fatalf("expected 1 reverse, got %d", len(m.reverses))
	}
}

func TestForwardModel_SwitchView(t *testing.T) {
	m := NewForwardModel(newTestClient())
	m.serial = "test"
	m.forwards = []ForwardRule{{Local: "tcp:8080"}}
	m.reverses = []ForwardRule{{Local: "tcp:9090"}, {Local: "tcp:9091"}}

	if m.showReverse {
		t.Fatal("expected forward view by default")
	}

	m, _ = m.Update(keyMsg("s"))
	if !m.showReverse {
		t.Fatal("expected reverse view")
	}
	if m.cursor != 0 {
		t.Fatal("expected cursor reset on view switch")
	}

	m, _ = m.Update(keyMsg("s"))
	if m.showReverse {
		t.Fatal("expected forward view again")
	}
}

func TestForwardModel_AddDialog(t *testing.T) {
	m := NewForwardModel(newTestClient())
	m.serial = "test"

	m, _ = m.Update(keyMsg("a"))
	if m.dialog != ForwardDialogAdd {
		t.Fatalf("expected ForwardDialogAdd, got %d", m.dialog)
	}
	if !m.focusLocal {
		t.Fatal("expected focus on local input")
	}

	// Tab switches focus
	m, _ = m.Update(specialKey(tea.KeyTab))
	if m.focusLocal {
		t.Fatal("expected focus on remote input after tab")
	}

	// Escape closes dialog
	m, _ = m.Update(specialKey(tea.KeyEsc))
	if m.dialog != ForwardDialogNone {
		t.Fatal("expected dialog closed")
	}
}

func TestForwardModel_AddReverseDialog(t *testing.T) {
	m := NewForwardModel(newTestClient())
	m.serial = "test"

	m, _ = m.Update(keyMsg("A"))
	if m.dialog != ForwardDialogAddReverse {
		t.Fatalf("expected ForwardDialogAddReverse, got %d", m.dialog)
	}
}

func TestForwardModel_CursorNavigation(t *testing.T) {
	m := NewForwardModel(newTestClient())
	m.serial = "test"
	m.forwards = []ForwardRule{
		{Local: "tcp:8080"}, {Local: "tcp:8081"}, {Local: "tcp:8082"},
	}

	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", m.cursor)
	}
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 2 {
		t.Fatal("expected cursor clamped at 2")
	}
	m, _ = m.Update(keyMsg("k"))
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.cursor)
	}
}

func TestForwardModel_ClearStatus(t *testing.T) {
	m := NewForwardModel(newTestClient())
	m.statusMsg = "completed"
	m, _ = m.Update(clearStatusMsg{})
	if m.statusMsg != "" {
		t.Fatal("expected empty status")
	}
}

// --- SettingsModel ---

func TestSettingsModel_ListMsg(t *testing.T) {
	m := NewSettingsModel(newTestClient())
	m.serial = "test"
	m.loading = true

	settings := []settingEntry{
		{Key: "brightness", Value: "128"},
		{Key: "volume", Value: "50"},
		{Key: "wifi_on", Value: "1"},
	}
	m, _ = m.Update(settingsListMsg{settings: settings})
	if m.loading {
		t.Fatal("expected loading false")
	}
	if len(m.settings) != 3 {
		t.Fatalf("expected 3 settings, got %d", len(m.settings))
	}
	if len(m.visible) != 3 {
		t.Fatalf("expected 3 visible, got %d", len(m.visible))
	}
}

func TestSettingsModel_NamespaceCycling(t *testing.T) {
	m := NewSettingsModel(newTestClient())
	m.serial = "test"

	if m.namespace != adb.NamespaceSystem {
		t.Fatalf("expected system namespace, got %s", m.namespace)
	}
	m, _ = m.Update(keyMsg("n"))
	if m.namespace != adb.NamespaceSecure {
		t.Fatalf("expected secure namespace, got %s", m.namespace)
	}
	m, _ = m.Update(keyMsg("n"))
	if m.namespace != adb.NamespaceGlobal {
		t.Fatalf("expected global namespace, got %s", m.namespace)
	}
	m, _ = m.Update(keyMsg("n"))
	if m.namespace != adb.NamespaceSystem {
		t.Fatalf("expected system namespace (wrapped), got %s", m.namespace)
	}
}

func TestSettingsModel_SearchFilter(t *testing.T) {
	m := NewSettingsModel(newTestClient())
	m.serial = "test"
	m.settings = []settingEntry{
		{Key: "brightness", Value: "128"},
		{Key: "volume", Value: "50"},
		{Key: "wifi_on", Value: "1"},
	}
	m.rebuildVisible()
	if len(m.visible) != 3 {
		t.Fatalf("expected 3 visible, got %d", len(m.visible))
	}

	m.searchQuery = "bright"
	m.rebuildVisible()
	if len(m.visible) != 1 {
		t.Fatalf("expected 1 visible for 'bright', got %d", len(m.visible))
	}

	// Search by value
	m.searchQuery = "50"
	m.rebuildVisible()
	if len(m.visible) != 1 {
		t.Fatalf("expected 1 visible for '50', got %d", len(m.visible))
	}
}

func TestSettingsModel_DeleteConfirmation(t *testing.T) {
	m := NewSettingsModel(newTestClient())
	m.serial = "test"
	m.settings = []settingEntry{{Key: "test_key", Value: "val"}}
	m.rebuildVisible()

	// Open delete dialog
	m, _ = m.Update(keyMsg("d"))
	if m.dialog != SettingsDialogDelete {
		t.Fatalf("expected delete dialog, got %d", m.dialog)
	}

	// Cancel
	m, _ = m.Update(keyMsg("n"))
	if m.dialog != SettingsDialogNone {
		t.Fatal("expected dialog closed")
	}
}

func TestSettingsModel_AddDialog(t *testing.T) {
	m := NewSettingsModel(newTestClient())
	m.serial = "test"

	m, _ = m.Update(keyMsg("a"))
	if m.dialog != SettingsDialogAdd {
		t.Fatalf("expected add dialog, got %d", m.dialog)
	}

	// Escape closes
	m, _ = m.Update(specialKey(tea.KeyEsc))
	if m.dialog != SettingsDialogNone {
		t.Fatal("expected dialog closed")
	}
}

func TestSettingsModel_CursorNavigation(t *testing.T) {
	m := NewSettingsModel(newTestClient())
	m.serial = "test"
	m.height = 40
	m.settings = []settingEntry{
		{Key: "a"}, {Key: "b"}, {Key: "c"},
	}
	m.rebuildVisible()

	m, _ = m.Update(keyMsg("j"))
	if m.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.cursor)
	}
	m, _ = m.Update(keyMsg("G"))
	if m.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", m.cursor)
	}
	m, _ = m.Update(keyMsg("g"))
	if m.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.cursor)
	}
}

func TestSettingsModel_ClearStatus(t *testing.T) {
	m := NewSettingsModel(newTestClient())
	m.statusMsg = "done"
	m, _ = m.Update(clearStatusMsg{})
	if m.statusMsg != "" {
		t.Fatal("expected empty status")
	}
}

// --- DeviceInfoModel reboot overlay ---

func TestDeviceInfoModel_RebootOverlay(t *testing.T) {
	m := NewDeviceInfoModel(newTestClient())
	m.serial = "test"

	// Open reboot overlay
	m, _ = m.Update(keyMsg("P"))
	if m.overlay != InfoOverlayReboot {
		t.Fatalf("expected InfoOverlayReboot, got %d", m.overlay)
	}
	if m.reboot.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.reboot.cursor)
	}

	// Navigate down
	m, _ = m.Update(keyMsg("j"))
	if m.reboot.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.reboot.cursor)
	}

	// Press Enter to trigger confirmation
	m, _ = m.Update(specialKey(tea.KeyEnter))
	if !m.reboot.confirm {
		t.Fatal("expected confirm true")
	}

	// Cancel with 'n'
	m, _ = m.Update(keyMsg("n"))
	if m.reboot.confirm {
		t.Fatal("expected confirm false after cancel")
	}
}

func TestDeviceInfoModel_RebootOverlayEsc(t *testing.T) {
	m := NewDeviceInfoModel(newTestClient())
	m.serial = "test"

	m, _ = m.Update(keyMsg("P"))
	if m.overlay != InfoOverlayReboot {
		t.Fatal("expected reboot overlay")
	}
	m, _ = m.Update(specialKey(tea.KeyEsc))
	if m.overlay != InfoOverlayNone {
		t.Fatal("expected overlay closed")
	}
}

func TestDeviceInfoModel_RebootMsg(t *testing.T) {
	m := NewDeviceInfoModel(newTestClient())
	m.serial = "test"

	// Success
	m, _ = m.Update(rebootMsg{mode: "recovery"})
	if m.statusMsg == "" {
		t.Fatal("expected status message for reboot")
	}

	// Error
	m, _ = m.Update(rebootMsg{mode: "", err: fmt.Errorf("failed")})
	if !strings.Contains(stripANSI(m.statusMsg), "failed") {
		t.Fatal("expected error in status")
	}
}

// --- PackageModel batch install ---

func TestPackageModel_BatchInstallDialog(t *testing.T) {
	m := NewPackageModel(newTestClient())
	m.serial = "test"

	m, _ = m.Update(keyMsg("I"))
	if !m.showBatchInstall {
		t.Fatal("expected batch install dialog")
	}

	// Escape closes
	m, _ = m.Update(specialKey(tea.KeyEsc))
	if m.showBatchInstall {
		t.Fatal("expected batch install dialog closed")
	}
}

func TestPackageModel_BatchInstallMsg(t *testing.T) {
	m := NewPackageModel(newTestClient())
	m.serial = "test"

	// No APKs found
	m, _ = m.Update(batchInstallMsg{total: 0})
	if !strings.Contains(stripANSI(m.statusMsg), "No .apk") {
		t.Fatalf("expected no-apk warning, got %q", stripANSI(m.statusMsg))
	}

	// All succeeded
	m, _ = m.Update(batchInstallMsg{total: 3, succeeded: 3})
	if !strings.Contains(stripANSI(m.statusMsg), "3/3") {
		t.Fatalf("expected 3/3 success, got %q", stripANSI(m.statusMsg))
	}

	// Some failed
	m, _ = m.Update(batchInstallMsg{total: 3, succeeded: 1, failed: 2, errors: []string{"a.apk: err"}})
	if !strings.Contains(stripANSI(m.statusMsg), "failed") {
		t.Fatalf("expected failure info, got %q", stripANSI(m.statusMsg))
	}
}

// --- DeviceListModel wireless switch ---

func TestDeviceListModel_WirelessSwitch_USBOnly(t *testing.T) {
	m := NewDeviceListModel(newTestClient())
	m.devices = []*adb.Device{{Serial: "192.168.1.100:5555", State: "device"}}
	m.cursor = 0

	// TCP device should be rejected
	m, _ = m.Update(keyMsg("t"))
	if m.err == nil {
		t.Fatal("expected error for TCP device wireless switch")
	}
}

func TestDeviceListModel_WirelessSwitchMsg(t *testing.T) {
	m := NewDeviceListModel(newTestClient())

	// Success
	m, _ = m.Update(wirelessSwitchMsg{ip: "192.168.1.100"})
	if !strings.Contains(stripANSI(m.statusMsg), "192.168.1.100") {
		t.Fatalf("expected IP in status, got %q", stripANSI(m.statusMsg))
	}

	// Error
	m, _ = m.Update(wirelessSwitchMsg{err: fmt.Errorf("no ip")})
	if m.err == nil {
		t.Fatal("expected error set")
	}
}

func TestDeviceListModel_ClearStatus(t *testing.T) {
	m := NewDeviceListModel(newTestClient())
	m.statusMsg = "something"
	m, _ = m.Update(clearStatusMsg{})
	if m.statusMsg != "" {
		t.Fatal("expected status cleared")
	}
}

// --- App-level device disappearance ---

func TestApp_DeviceDisappearsOnRefresh(t *testing.T) {
	m := testModel()
	m.serial = "abc123"

	// Simulate refresh with device gone
	msg := devicesRefreshMsg{
		devices: []*adb.Device{{Serial: "other_device", State: "device"}},
	}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.serial != "" {
		t.Fatalf("expected serial cleared, got %q", model.serial)
	}
}

func TestApp_DeviceStillPresent(t *testing.T) {
	m := testModel()
	m.serial = "abc123"

	msg := devicesRefreshMsg{
		devices: []*adb.Device{
			{Serial: "abc123", State: "device"},
			{Serial: "other", State: "device"},
		},
	}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.serial != "abc123" {
		t.Fatalf("expected serial preserved, got %q", model.serial)
	}
}

// --- Pure helper functions ---

func TestCycleLevel(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", "D"},
		{"D", "I"},
		{"I", "W"},
		{"W", "E"},
		{"E", "F"},
		{"F", ""},
		{"X", ""},
	}
	for _, tt := range tests {
		got := cycleLevel(tt.in)
		if got != tt.want {
			t.Errorf("cycleLevel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLogLevelPriority(t *testing.T) {
	tests := []struct {
		level string
		want  int
	}{
		{"V", 0}, {"D", 1}, {"I", 2}, {"W", 3}, {"E", 4}, {"F", 5}, {"?", -1},
	}
	for _, tt := range tests {
		got := logLevelPriority(tt.level)
		if got != tt.want {
			t.Errorf("logLevelPriority(%q) = %d, want %d", tt.level, got, tt.want)
		}
	}
}

func TestHighlightMatches(t *testing.T) {
	// Empty query returns original
	if got := highlightMatches("hello", ""); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	// No match returns original
	if got := highlightMatches("hello", "xyz"); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	// Match preserves text content
	result := highlightMatches("hello world", "world")
	if !containsPlainText(result, "hello") {
		t.Error("expected result to contain 'hello'")
	}
	if !containsPlainText(result, "world") {
		t.Error("expected result to contain 'world'")
	}
	// Case insensitive
	result = highlightMatches("Hello WORLD", "world")
	if !containsPlainText(result, "WORLD") {
		t.Error("expected case-insensitive match to preserve original case")
	}
}

func TestScrollInfo(t *testing.T) {
	got := scrollInfo(0, 0)
	if got == "" {
		t.Fatal("expected non-empty for empty list")
	}
	got = scrollInfo(0, 5)
	if got == "" {
		t.Fatal("expected non-empty")
	}
}

func TestRenderBar(t *testing.T) {
	// Normal range
	bar := renderBar(50, 100, 20)
	if bar == "" {
		t.Fatal("expected non-empty bar")
	}
	// Over 100%
	bar = renderBar(150, 100, 20)
	if bar == "" {
		t.Fatal("expected non-empty bar for overflow")
	}
	// Zero max
	bar = renderBar(50, 0, 20)
	if bar == "" {
		t.Fatal("expected non-empty bar for zero max")
	}
	// Negative value
	bar = renderBar(-10, 100, 20)
	if bar == "" {
		t.Fatal("expected non-empty bar for negative")
	}
}

func TestIsDaemonError(t *testing.T) {
	tests := []struct {
		name   string
		err    string
		expect bool
	}{
		{"daemon not running", "* daemon not running; starting now at tcp:5037", true},
		{"failed to start daemon", "failed to start daemon", true},
		{"cannot connect to daemon", "cannot connect to daemon at tcp:5037", true},
		{"address in use", "Address already in use", true},
		{"signal killed", "signal: killed", true},
		{"context deadline", "context deadline exceeded", true},
		{"wrapped daemon error", "list devices: daemon not running; starting now", true},
		{"normal error", "device not found", false},
		{"empty error", "", false},
		{"permission denied", "permission denied", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDaemonError(fmt.Errorf("%s", tt.err))
			if got != tt.expect {
				t.Errorf("isDaemonError(%q) = %v, want %v", tt.err, got, tt.expect)
			}
		})
	}
}

func TestSimplifyADBError(t *testing.T) {
	tests := []struct {
		name     string
		err      string
		contains string
	}{
		{"signal killed", "signal: killed", "not responding"},
		{"context deadline", "context deadline exceeded", "not responding"},
		{"address in use", "Address already in use", "port in use"},
		{"failed to start", "failed to start daemon", "not available"},
		{"daemon not running", "daemon not running; starting now at tcp:5037", "not available"},
		{"cannot connect", "cannot connect to daemon", "cannot connect"},
		{"passthrough", "device offline", "device offline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := simplifyADBError(fmt.Errorf("%s", tt.err))
			if !strings.Contains(got.Error(), tt.contains) {
				t.Errorf("simplifyADBError(%q) = %q, want containing %q", tt.err, got, tt.contains)
			}
		})
	}
}

func TestSafeViewHeight(t *testing.T) {
	// Normal case
	if h := safeViewHeight(40, 10, 15); h != 30 {
		t.Fatalf("expected 30, got %d", h)
	}
	// Small terminal — min clamped to terminal height
	if h := safeViewHeight(8, 10, 15); h != 8 {
		t.Fatalf("expected clamped to 8, got %d", h)
	}
	// Larger terminal but still small — uses minRows
	if h := safeViewHeight(20, 10, 15); h != 15 {
		t.Fatalf("expected min 15, got %d", h)
	}
	// Zero height
	if h := safeViewHeight(0, 10, 15); h != 15 {
		t.Fatalf("expected min 15 for zero height, got %d", h)
	}
}

func TestApplyTheme(t *testing.T) {
	// Save original
	origPrimary := ColorPrimary

	ApplyTheme(ThemeNord)
	if ColorPrimary != ThemeNord.Primary {
		t.Fatal("expected nord primary color after ApplyTheme")
	}

	ApplyTheme(ThemeTokyoNight)
	if ColorPrimary != ThemeTokyoNight.Primary {
		t.Fatal("expected tokyonight primary color after ApplyTheme")
	}

	ApplyTheme(ThemeCatppuccin)
	if ColorPrimary != ThemeCatppuccin.Primary {
		t.Fatal("expected catppuccin primary color after ApplyTheme")
	}

	// Restore default
	ApplyTheme(ThemeDefault)
	if ColorPrimary != origPrimary {
		t.Fatal("expected original primary color after restoring default")
	}
}

func TestThemeNames(t *testing.T) {
	names := ThemeNames()
	if len(names) != 4 {
		t.Fatalf("expected 4 themes, got %d", len(names))
	}
	for _, name := range names {
		if _, ok := BuiltinThemes[name]; !ok {
			t.Fatalf("theme %q listed but not in BuiltinThemes", name)
		}
	}
}
