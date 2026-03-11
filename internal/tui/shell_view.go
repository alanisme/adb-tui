package tui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
	"github.com/alanisme/adb-tui/internal/config"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[^[\]()]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// shellMessage is implemented by all messages routed to ShellModel.
type shellMessage interface{ shellMsg() }

type shellOutputMsg struct {
	command string
	output  string
	err     error
}

func (shellOutputMsg) shellMsg() {}

type ShellModel struct {
	client      *adb.Client
	serial      string
	input       textinput.Model
	history     []string
	historyIdx  int
	output      []string
	scroll      int
	width       int
	height      int
	running     bool
	editing     bool // true when textinput has focus
	quickCmds   bool
	quickCmdIdx int
}

var defaultQuickCommands = []struct {
	label string
	cmd   string
}{
	{"Device info", "getprop ro.product.model && getprop ro.build.version.release"},
	{"IP address", "ip addr show wlan0 | grep 'inet '"},
	{"Disk usage", "df -h"},
	{"Top processes", "top -n 1 -b | head -20"},
	{"Logcat errors", "logcat -d -s '*:E' | tail -30"},
	{"Current activity", "dumpsys activity activities | grep mResumedActivity"},
	{"Screen density", "wm density && wm size"},
	{"Battery info", "dumpsys battery"},
	{"Network stats", "dumpsys connectivity | head -30"},
	{"Memory info", "cat /proc/meminfo | head -10"},
}

const maxShellHistory = 500

func NewShellModel(client *adb.Client) ShellModel {
	ti := textinput.New()
	ti.Placeholder = "Enter shell command..."
	ti.CharLimit = 512
	ti.Prompt = "$ "
	m := ShellModel{
		client:     client,
		input:      ti,
		historyIdx: -1,
	}
	m.loadHistory()
	return m
}

func (m ShellModel) IsInputCaptured() bool {
	return m.editing || m.quickCmds
}

func (m ShellModel) Init() tea.Cmd {
	return nil
}

func (m ShellModel) Update(msg tea.Msg) (ShellModel, tea.Cmd) {
	switch msg := msg.(type) {
	case shellOutputMsg:
		m.running = false
		prefix := AccentStyle.Render("$ " + msg.command)
		m.output = append(m.output, prefix)
		if msg.err != nil {
			m.output = append(m.output, ErrorStyle.Render(msg.err.Error()))
		} else if msg.output != "" {
			cleaned := stripANSI(msg.output)
			for line := range strings.SplitSeq(cleaned, "\n") {
				m.output = append(m.output, line)
			}
		}
		m.output = append(m.output, "")
		m.scrollToBottom()
		return m, nil

	case tea.KeyMsg:
		if m.quickCmds {
			return m.updateQuickCmds(msg)
		}
		if m.editing {
			return m.updateEditing(msg)
		}
		// Normal mode: vim-like navigation of output
		switch {
		case key.Matches(msg, DefaultKeyMap.Enter):
			m.editing = true
			m.input.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.Up):
			m.scroll = max(m.scroll-1, 0)
		case key.Matches(msg, DefaultKeyMap.Down):
			m.scroll++
			m.clampScroll()
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.scroll = 0
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			m.scrollToBottom()
		case key.Matches(msg, DefaultKeyMap.HalfPageUp):
			m.scroll = max(m.scroll-10, 0)
		case key.Matches(msg, DefaultKeyMap.HalfPageDown):
			m.scroll += 10
			m.clampScroll()
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.scroll = max(m.scroll-20, 0)
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.scroll += 20
			m.clampScroll()
		case key.Matches(msg, DefaultKeyMap.Refresh):
			m.output = nil
			m.scroll = 0
		case msg.String() == "f":
			m.quickCmds = true
			m.quickCmdIdx = 0
		}
		return m, nil
	}

	return m, nil
}

func (m ShellModel) View() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render(fmt.Sprintf("  Shell  %s", AccentStyle.Render(m.serial))))
	b.WriteString("\n")

	if m.serial == "" {
		b.WriteString(DimStyle.Render("  No device selected") + "\n")
		return b.String()
	}

	outputHeight := m.height - 7
	if outputHeight < 1 {
		outputHeight = 15
	}

	visibleLines := m.visibleOutput(outputHeight)
	for _, line := range visibleLines {
		b.WriteString("  " + line + "\n")
	}

	padding := outputHeight - len(visibleLines)
	for range padding {
		b.WriteString("\n")
	}

	separator := strings.Repeat("─", m.width-4)
	b.WriteString(DimStyle.Render("  "+separator) + "\n")

	if m.running {
		b.WriteString("  " + DimStyle.Render("running...") + "\n")
	} else if m.editing {
		b.WriteString("  " + m.input.View() + "\n")
	} else {
		b.WriteString("  " + DimStyle.Render("$ ") + DimStyle.Render(m.input.Value()) + "\n")
	}

	if m.quickCmds {
		var qb strings.Builder
		qb.WriteString("Quick Commands:\n")
		for i, qc := range defaultQuickCommands {
			prefix := "  "
			if i == m.quickCmdIdx {
				prefix = "▸ "
			}
			fmt.Fprintf(&qb, "%s%-18s %s\n", prefix, qc.label, DimStyle.Render(qc.cmd))
		}
		b.WriteString("\n" + DialogStyle.Render(qb.String()) + "\n")
	}

	if m.editing {
		b.WriteString(helpBar(
			keyHint("⏎", "run"),
			keyHint("↑/↓", "history"),
			keyHint("esc", "normal mode"),
		))
	} else {
		b.WriteString(helpBar(
			keyHint("⏎", "edit command"),
			keyHint("j/k", "scroll"),
			keyHint("g/G", "top/bottom"),
			keyHint("r", "clear"),
			keyHint("f", "quick cmds"),
		))
	}

	return b.String()
}

func (m ShellModel) visibleOutput(maxLines int) []string {
	if len(m.output) == 0 {
		return nil
	}
	start := m.scroll
	end := min(start+maxLines, len(m.output))
	if start >= end {
		return nil
	}
	return m.output[start:end]
}

func (m *ShellModel) scrollToBottom() {
	outputHeight := m.height - 7
	if outputHeight < 1 {
		outputHeight = 15
	}
	if len(m.output) > outputHeight {
		m.scroll = len(m.output) - outputHeight
	} else {
		m.scroll = 0
	}
}

func (m *ShellModel) clampScroll() {
	outputHeight := m.height - 7
	if outputHeight < 1 {
		outputHeight = 15
	}
	max := len(m.output) - outputHeight
	if max < 0 {
		max = 0
	}
	if m.scroll > max {
		m.scroll = max
	}
}

func (m ShellModel) updateEditing(msg tea.KeyMsg) (ShellModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.editing = false
		m.input.Blur()
		return m, nil
	case tea.KeyEnter:
		if m.running {
			return m, nil
		}
		cmd := m.input.Value()
		if cmd == "" {
			return m, nil
		}
		m.history = append(m.history, cmd)
		if len(m.history) > maxShellHistory {
			m.history = m.history[len(m.history)-maxShellHistory:]
		}
		m.historyIdx = len(m.history)
		m.saveHistory()
		m.input.Reset()
		// Handle local commands that would break TUI
		trimmed := strings.TrimSpace(cmd)
		if trimmed == "clear" || trimmed == "reset" {
			m.output = nil
			m.scroll = 0
			return m, nil
		}
		m.running = true
		return m, m.execCommand(cmd)
	case tea.KeyUp:
		if len(m.history) > 0 {
			if m.historyIdx > 0 {
				m.historyIdx--
			}
			if m.historyIdx < len(m.history) {
				m.input.SetValue(m.history[m.historyIdx])
				m.input.CursorEnd()
			}
		}
		return m, nil
	case tea.KeyDown:
		if m.historyIdx < len(m.history)-1 {
			m.historyIdx++
			m.input.SetValue(m.history[m.historyIdx])
			m.input.CursorEnd()
		} else {
			m.historyIdx = len(m.history)
			m.input.Reset()
		}
		return m, nil
	case tea.KeyPgUp:
		m.scroll -= 10
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m, nil
	case tea.KeyPgDown:
		m.scroll += 10
		m.clampScroll()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ShellModel) Focus() ShellModel {
	// Don't auto-enter editing mode; user presses Enter to start typing.
	// This allows number keys and other shortcuts to work in normal mode.
	return m
}

func (m ShellModel) Blur() ShellModel {
	m.editing = false
	m.input.Blur()
	return m
}

func (m ShellModel) SetDevice(serial string) ShellModel {
	m.serial = serial
	return m
}

func (m ShellModel) SetSize(w, h int) ShellModel {
	m.width = w
	m.height = h
	return m
}

func (m ShellModel) execCommand(command string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// TERM=dumb suppresses color/cursor output from most programs at the source;
		// stripANSI in Update handles anything that leaks through.
		wrapped := "export TERM=dumb && " + command
		output, err := client.RunCommand(ctx, serial, wrapped)
		return shellOutputMsg{command: command, output: output, err: err}
	}
}

func (m ShellModel) updateQuickCmds(msg tea.KeyMsg) (ShellModel, tea.Cmd) {
	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.quickCmdIdx > 0 {
			m.quickCmdIdx--
		}
	case key.Matches(msg, DefaultKeyMap.Down):
		if m.quickCmdIdx < len(defaultQuickCommands)-1 {
			m.quickCmdIdx++
		}
	case key.Matches(msg, DefaultKeyMap.Enter):
		m.quickCmds = false
		cmd := defaultQuickCommands[m.quickCmdIdx].cmd
		m.history = append(m.history, cmd)
		m.historyIdx = len(m.history)
		m.saveHistory()
		m.running = true
		return m, m.execCommand(cmd)
	case msg.Type == tea.KeyEsc:
		m.quickCmds = false
	}
	return m, nil
}

func shellHistoryPath() string {
	dir, err := config.ConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "shell_history")
}

func (m *ShellModel) loadHistory() {
	path := shellHistoryPath()
	if path == "" {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			m.history = append(m.history, line)
		}
	}
	if len(m.history) > maxShellHistory {
		m.history = m.history[len(m.history)-maxShellHistory:]
	}
	m.historyIdx = len(m.history)
}

func (m ShellModel) saveHistory() {
	path := shellHistoryPath()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	// Save last N entries
	start := 0
	if len(m.history) > maxShellHistory {
		start = len(m.history) - maxShellHistory
	}
	w := bufio.NewWriter(f)
	for _, h := range m.history[start:] {
		fmt.Fprintln(w, h)
	}
	w.Flush()
}
