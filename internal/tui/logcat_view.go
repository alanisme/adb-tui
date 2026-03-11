package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

// logcatMessage is implemented by all messages routed to LogcatModel.
type logcatMessage interface{ logcatMsg() }

type logcatEntryMsg struct {
	entry adb.LogEntry
	gen   uint64
}

type logcatStreamStoppedMsg struct {
	gen uint64
}

type logcatStreamStartedMsg struct {
	err error
}

type logcatClearedMsg struct {
	err error
}

type logcatExportMsg struct {
	path string
	err  error
}

func (logcatEntryMsg) logcatMsg()          {}
func (logcatStreamStoppedMsg) logcatMsg()  {}
func (logcatStreamStartedMsg) logcatMsg()  {}
func (logcatClearedMsg) logcatMsg()        {}
func (logcatExportMsg) logcatMsg()         {}

type LogcatModel struct {
	client           *adb.Client
	serial           string
	entries          []adb.LogEntry
	filtered         []int
	scroll           int
	width            int
	height           int
	paused           bool
	autoScroll       bool
	streaming        bool
	err              error
	cancel           context.CancelFunc
	ch               <-chan adb.LogEntry
	filterTag        string
	filterLevel      string
	filterPID        string
	searchInput      textinput.Model
	showSearch       bool
	searchQuery      string
	searchQueryLower string // cached lowercase of searchQuery
	filterTagLower   string // cached lowercase of filterTag
	maxEntries       int
	statusMsg        string
	streamGen        uint64 // incremented on each new stream to discard stale messages
}

func NewLogcatModel(client *adb.Client) LogcatModel {
	si := textinput.New()
	si.Placeholder = "search..."
	si.CharLimit = 128
	return LogcatModel{
		client:      client,
		autoScroll:  true,
		maxEntries:  10000,
		searchInput: si,
	}
}

func (m LogcatModel) IsInputCaptured() bool {
	return m.showSearch
}

func (m LogcatModel) Init() tea.Cmd {
	return nil
}

func (m LogcatModel) Update(msg tea.Msg) (LogcatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case logcatEntryMsg:
		if msg.gen != m.streamGen {
			return m, nil // stale message from a previous stream
		}
		if !m.paused {
			m.entries = append(m.entries, msg.entry)
			if len(m.entries) > m.maxEntries {
				drop := len(m.entries) - m.maxEntries
				m.entries = m.entries[drop:]
				// Adjust filtered indices after dropping old entries
				j := 0
				for _, idx := range m.filtered {
					adjusted := idx - drop
					if adjusted >= 0 {
						m.filtered[j] = adjusted
						j++
					}
				}
				m.filtered = m.filtered[:j]
			}
			if m.matchesFilter(msg.entry) {
				m.filtered = append(m.filtered, len(m.entries)-1)
			}
			if m.autoScroll {
				m.scrollToBottom()
			}
		}
		return m, m.readNextEntry()

	case logcatStreamStoppedMsg:
		if msg.gen != m.streamGen {
			return m, nil // stale stop from a previous stream
		}
		m.streaming = false
		m.ch = nil
		return m, nil

	case logcatStreamStartedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.streaming = false
		}
		return m, nil

	case logcatClearedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.entries = nil
			m.filtered = nil
			m.scroll = 0
		}
		return m, nil

	case logcatExportMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = SuccessStyle.Render("Exported to " + msg.path)
		}
		return m, clearStatusAfter(5 * time.Second)

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.showSearch {
			return m.updateSearch(msg)
		}
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			m.autoScroll = false
			if m.scroll > 0 {
				m.scroll--
			}
		case key.Matches(msg, DefaultKeyMap.Down):
			m.scroll++
			m.clampScroll()
			if m.scroll >= m.maxScroll() {
				m.autoScroll = true
			}
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.autoScroll = false
			m.scroll -= 20
			if m.scroll < 0 {
				m.scroll = 0
			}
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.scroll += 20
			m.clampScroll()
			if m.scroll >= m.maxScroll() {
				m.autoScroll = true
			}
		case key.Matches(msg, DefaultKeyMap.Search):
			m.showSearch = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.HalfPageUp):
			m.autoScroll = false
			m.scroll = max(m.scroll-m.viewHeight()/2, 0)
		case key.Matches(msg, DefaultKeyMap.HalfPageDown):
			m.scroll = min(m.scroll+m.viewHeight()/2, m.maxScroll())
			if m.scroll >= m.maxScroll() {
				m.autoScroll = true
			}
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.autoScroll = false
			m.scroll = 0
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			m.autoScroll = true
			m.scrollToBottom()
		case key.Matches(msg, DefaultKeyMap.Space):
			m.paused = !m.paused
		case msg.String() == "c":
			return m, m.clearLogcat()
		case msg.String() == "v":
			m.filterLevel = cycleLevel(m.filterLevel)
			m.rebuildFiltered()
		case msg.String() == "o":
			return m, m.exportLogcat()
		case msg.String() == "t":
			m.filterTag = ""
			m.filterTagLower = ""
			m.filterPID = ""
			m.searchQuery = ""
			m.searchQueryLower = ""
			m.rebuildFiltered()
		}
	}
	return m, nil
}

func (m LogcatModel) updateSearch(msg tea.KeyMsg) (LogcatModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.searchQuery = m.searchInput.Value()
		m.showSearch = false
		m.searchInput.Blur()
		m.rebuildFiltered()
		return m, nil
	case tea.KeyEsc:
		m.showSearch = false
		m.searchInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m LogcatModel) View() string {
	var b strings.Builder

	status := ""
	if m.paused {
		status = WarningStyle.Render(" [PAUSED]")
	}
	if m.streaming {
		status += SuccessStyle.Render(" [STREAMING]")
	}

	entryInfo := ScrollPosStyle.Render(fmt.Sprintf("[%d entries]", len(m.filtered)))
	title := HeaderStyle.Render(fmt.Sprintf("Logcat [%s]%s %s", m.serial, status, entryInfo))
	b.WriteString(title)
	b.WriteString("\n")

	if m.serial == "" {
		b.WriteString(DimStyle.Render("  No device selected") + "\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render("  Error: "+m.err.Error()) + "\n")
	}

	filterInfo := m.filterStatus()
	if filterInfo != "" {
		b.WriteString("  " + DimStyle.Render("Filter: "+filterInfo) + "\n")
	}

	viewHeight := safeViewHeight(m.height, 8, 15)

	visible := m.visibleEntries(viewHeight)
	for _, idx := range visible {
		if idx < len(m.entries) {
			b.WriteString("  " + m.renderEntry(m.entries[idx]) + "\n")
		}
	}

	padding := viewHeight - len(visible)
	for range padding {
		b.WriteString("\n")
	}

	if m.showSearch {
		b.WriteString("  " + m.searchInput.View() + "\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n")
	}

	scrollInfo := ""
	if !m.autoScroll {
		scrollInfo = WarningStyle.Render(" (scroll locked)")
	}
	b.WriteString(helpBar(
		keyHint("space", "pause"),
		keyHint("/", "search"),
		keyHint("v", "level"),
		keyHint("c", "clear"),
		keyHint("o", "export"),
		keyHint("t", "reset"),
	) + scrollInfo)

	return b.String()
}

func (m LogcatModel) renderEntry(e adb.LogEntry) string {
	style := LogLevelStyle(string(e.Level))
	ts := e.Timestamp.Format("15:04:05.000")
	tag := AccentStyle.Render(e.Tag)
	msg := e.Message
	if m.searchQuery != "" {
		tag = highlightMatches(e.Tag, m.searchQuery)
		msg = highlightMatches(e.Message, m.searchQuery)
	}
	return fmt.Sprintf("%s %s %s/%s: %s",
		DimStyle.Render(ts),
		style.Render(string(e.Level)),
		DimStyle.Render(e.PID),
		tag,
		msg,
	)
}

func (m LogcatModel) filterStatus() string {
	var parts []string
	if m.filterLevel != "" {
		parts = append(parts, "level>="+m.filterLevel)
	}
	if m.filterTag != "" {
		parts = append(parts, "tag="+m.filterTag)
	}
	if m.filterPID != "" {
		parts = append(parts, "pid="+m.filterPID)
	}
	if m.searchQuery != "" {
		parts = append(parts, "search="+m.searchQuery)
	}
	return strings.Join(parts, " ")
}

func (m LogcatModel) matchesFilter(e adb.LogEntry) bool {
	if m.filterLevel != "" && logLevelPriority(string(e.Level)) < logLevelPriority(m.filterLevel) {
		return false
	}
	if m.filterTag != "" {
		// Pre-lowercase comparison avoids allocation per entry
		tagLower := strings.ToLower(e.Tag)
		if !strings.Contains(tagLower, m.filterTagLower) {
			return false
		}
	}
	if m.filterPID != "" && e.PID != m.filterPID {
		return false
	}
	if m.searchQuery != "" {
		msgLower := strings.ToLower(e.Message)
		tagLower := strings.ToLower(e.Tag)
		if !strings.Contains(msgLower, m.searchQueryLower) && !strings.Contains(tagLower, m.searchQueryLower) {
			return false
		}
	}
	return true
}

func (m *LogcatModel) rebuildFiltered() {
	// Cache lowercased filter values to avoid per-entry allocation
	m.filterTagLower = strings.ToLower(m.filterTag)
	m.searchQueryLower = strings.ToLower(m.searchQuery)
	m.filtered = nil
	for i, e := range m.entries {
		if m.matchesFilter(e) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.scrollToBottom()
}

func (m LogcatModel) viewHeight() int {
	return safeViewHeight(m.height, 8, 15)
}

func (m LogcatModel) visibleEntries(maxLines int) []int {
	if len(m.filtered) == 0 {
		return nil
	}
	start := m.scroll
	end := min(start+maxLines, len(m.filtered))
	if start >= end {
		return nil
	}
	return m.filtered[start:end]
}

func (m *LogcatModel) scrollToBottom() {
	viewHeight := safeViewHeight(m.height, 8, 15)
	max := len(m.filtered) - viewHeight
	if max < 0 {
		max = 0
	}
	m.scroll = max
}

func (m *LogcatModel) clampScroll() {
	max := m.maxScroll()
	if m.scroll > max {
		m.scroll = max
	}
}

func (m LogcatModel) maxScroll() int {
	viewHeight := safeViewHeight(m.height, 8, 15)
	max := len(m.filtered) - viewHeight
	if max < 0 {
		return 0
	}
	return max
}

func (m *LogcatModel) SetDevice(serial string) (LogcatModel, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.serial = serial
	m.entries = nil
	m.filtered = nil
	m.scroll = 0
	m.ch = nil
	m.streaming = false
	m.err = nil
	if serial == "" {
		return *m, nil
	}
	cmd := m.startStream()
	return *m, cmd
}

func (m LogcatModel) SetSize(w, h int) LogcatModel {
	m.width = w
	m.height = h
	return m
}

func (m *LogcatModel) Stop() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.ch = nil
	m.streaming = false
}

func (m *LogcatModel) startStream() tea.Cmd {
	m.streamGen++
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := m.client.LogcatStream(ctx, m.serial, adb.LogcatOptions{
		Format: "threadtime",
		Since:  time.Now().Format("01-02 15:04:05.000"),
	})
	if err != nil {
		cancel()
		m.err = err
		return nil
	}
	m.cancel = cancel
	m.ch = ch
	m.streaming = true
	return m.readNextEntry()
}

func (m *LogcatModel) readNextEntry() tea.Cmd {
	ch := m.ch
	gen := m.streamGen
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		entry, ok := <-ch
		if !ok {
			return logcatStreamStoppedMsg{gen: gen}
		}
		return logcatEntryMsg{entry: entry, gen: gen}
	}
}

func (m LogcatModel) clearLogcat() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.LogcatClear(ctx, serial)
		return logcatClearedMsg{err: err}
	}
}

func (m LogcatModel) exportLogcat() tea.Cmd {
	entries := make([]adb.LogEntry, 0, len(m.filtered))
	for _, idx := range m.filtered {
		if idx < len(m.entries) {
			entries = append(entries, m.entries[idx])
		}
	}
	return func() tea.Msg {
		ts := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("logcat_%s.txt", ts)
		fullPath := filepath.Join(downloadsDir(), filename)
		f, err := os.Create(fullPath)
		if err != nil {
			return logcatExportMsg{err: err}
		}
		defer f.Close()
		for _, e := range entries {
			fmt.Fprintf(f, "%s %s %s/%s: %s\n",
				e.Timestamp.Format("2006-01-02 15:04:05.000"),
				string(e.Level), e.PID, e.Tag, e.Message)
		}
		return logcatExportMsg{path: fullPath}
	}
}

func logLevelPriority(level string) int {
	switch level {
	case "V":
		return 0
	case "D":
		return 1
	case "I":
		return 2
	case "W":
		return 3
	case "E":
		return 4
	case "F":
		return 5
	default:
		return -1
	}
}

func cycleLevel(current string) string {
	switch current {
	case "":
		return "D"
	case "D":
		return "I"
	case "I":
		return "W"
	case "W":
		return "E"
	case "E":
		return "F"
	default:
		return ""
	}
}
