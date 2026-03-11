package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

func testModel() Model {
	client := adb.NewClientWithPath("adb")
	return Model{
		client:      client,
		activeView:  ViewDevices,
		width:       80,
		height:      24,
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
}

func TestViewNames(t *testing.T) {
	if len(viewNames) != 11 {
		t.Fatalf("expected 11 view names, got %d", len(viewNames))
	}
	if viewNames[0] != "Devices" {
		t.Fatalf("expected Devices, got %s", viewNames[0])
	}
	if viewNames[10] != "Perf" {
		t.Fatalf("expected Perf, got %s", viewNames[10])
	}
}

func TestModelInit(t *testing.T) {
	m := testModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected init command")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(Model)
	if model.width != 120 {
		t.Fatalf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Fatalf("expected height 40, got %d", model.height)
	}
}

func TestSwitchView(t *testing.T) {
	m := testModel()

	updated, _ := m.switchView(ViewShell)
	model := updated.(Model)
	if model.activeView != ViewShell {
		t.Fatalf("expected ViewShell, got %d", model.activeView)
	}

	updated, _ = model.switchView(ViewDevices)
	model = updated.(Model)
	if model.activeView != ViewDevices {
		t.Fatalf("expected ViewDevices, got %d", model.activeView)
	}
}

func TestTabCycling(t *testing.T) {
	m := testModel()
	m.activeView = ViewPerformance

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.activeView != ViewDevices {
		t.Fatalf("expected wrap to ViewDevices, got %d", model.activeView)
	}
}

func TestRenderTabBar(t *testing.T) {
	m := testModel()
	bar := m.renderTabBar()
	if bar == "" {
		t.Fatal("expected non-empty tab bar")
	}
}

func TestRenderStatusBar(t *testing.T) {
	m := testModel()
	bar := m.renderStatusBar()
	if bar == "" {
		t.Fatal("expected non-empty status bar")
	}

	m.serial = "ABC123"
	bar = m.renderStatusBar()
	if bar == "" {
		t.Fatal("expected non-empty status bar with serial")
	}
}

func TestRenderHelp(t *testing.T) {
	m := testModel()
	help := m.renderHelp()
	if help == "" {
		t.Fatal("expected non-empty help")
	}
}

func TestHelpToggle(t *testing.T) {
	m := testModel()
	m.showHelp = true

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.showHelp {
		t.Fatal("expected help to be closed")
	}
}

func TestQuit(t *testing.T) {
	// ctrl+c quits immediately
	m := testModel()
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected quit command from ctrl+c")
	}

	// q requires confirmation: first press shows prompt
	m = testModel()
	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	result, cmd := m.Update(qMsg)
	model := result.(Model)
	if cmd != nil {
		t.Fatal("first q should not quit")
	}
	if !model.confirmQuit {
		t.Fatal("first q should set confirmQuit")
	}

	// second q confirms
	_, cmd = model.Update(qMsg)
	if cmd == nil {
		t.Fatal("second q should quit")
	}
}

func TestViewLoading(t *testing.T) {
	m := testModel()
	m.width = 0
	view := m.View()
	if view != "Loading..." {
		t.Fatalf("expected Loading..., got %s", view)
	}
}

func TestIsInputCaptured(t *testing.T) {
	m := testModel()

	m.activeView = ViewDevices
	if m.isInputCaptured() {
		t.Fatal("devices should not capture input by default")
	}

	m.activeView = ViewShell
	if m.isInputCaptured() {
		t.Fatal("shell should not capture input when not editing")
	}
	m.shell.editing = true
	if !m.isInputCaptured() {
		t.Fatal("shell should capture input when editing")
	}
}

func TestRenderActiveView(t *testing.T) {
	m := testModel()
	for v := range viewNames {
		m.activeView = View(v)
		view := m.renderActiveView()
		if view == "" && v != int(ViewShell) {
			t.Fatalf("view %d returned empty", v)
		}
	}
}

func TestKeyMapBindings(t *testing.T) {
	km := DefaultKeyMap

	if !key.Matches(tea.KeyMsg{Type: tea.KeyCtrlC}, km.Quit) {
		t.Fatal("ctrl+c should match quit")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}, km.QuitConfirm) {
		t.Fatal("q should match quit confirm")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}, km.Help) {
		t.Fatal("? should match help")
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}, km.Tab1) {
		t.Fatal("1 should match tab1")
	}
}

func TestDeviceStateStyle(t *testing.T) {
	cases := []string{"device", "offline", "unauthorized", "unknown"}
	for _, c := range cases {
		style := DeviceStateStyle(c)
		if style.GetForeground() == nil {
			t.Fatalf("expected foreground for state %s", c)
		}
	}
}

func TestFitHeight(t *testing.T) {
	tests := []struct {
		name    string
		content string
		n       int
		want    int // expected number of lines
	}{
		{"exact fit", "a\nb\nc\n", 3, 3},
		{"pad short", "a\n", 3, 3},
		{"truncate long", "a\nb\nc\nd\ne\n", 3, 3},
		{"empty content", "", 3, 3},
		{"zero height", "a\nb\n", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fitHeight(tt.content, tt.n)
			// Count lines (result always ends with \n)
			lines := 0
			for _, c := range result {
				if c == '\n' {
					lines++
				}
			}
			if lines != tt.want {
				t.Errorf("fitHeight() produced %d lines, want %d\nresult: %q", lines, tt.want, result)
			}
		})
	}
}

func TestFitHeight_Truncation(t *testing.T) {
	// Verify truncated content preserves first N lines
	content := "line1\nline2\nline3\nline4\nline5\n"
	result := fitHeight(content, 3)
	if result != "line1\nline2\nline3\n" {
		t.Errorf("expected first 3 lines, got %q", result)
	}
}

func TestLogLevelStyle(t *testing.T) {
	levels := []string{"V", "D", "I", "W", "E", "F", "X"}
	for _, l := range levels {
		style := LogLevelStyle(l)
		_ = style.Render("test")
	}
}
