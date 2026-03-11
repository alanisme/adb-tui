package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

// processMessage is implemented by all messages routed to ProcessModel.
type processMessage interface{ processMsg() }

type processListMsg struct {
	procs   []adb.ProcessInfo
	memInfo *adb.MemInfo
	err     error
}

type processTickMsg struct{}

type processKillMsg struct {
	pid int
	err error
}

func (processListMsg) processMsg() {}
func (processTickMsg) processMsg() {}
func (processKillMsg) processMsg() {}

type ProcessSortField int

const (
	SortByCPU ProcessSortField = iota
	SortByMEM
	SortByPID
	SortByName
)

type ProcessModel struct {
	client        *adb.Client
	serial        string
	processes     []adb.ProcessInfo
	memInfo       *adb.MemInfo
	visible       []int
	cursor        int
	scroll        int
	width         int
	height        int
	err           error
	loading       bool
	sortField     ProcessSortField
	searchInput   textinput.Model
	showSearch    bool
	searchQuery   string
	confirmKill   bool
	statusMsg     string
	refreshSecs   int
	showKillName  bool
	killNameInput textinput.Model
}

func NewProcessModel(client *adb.Client) ProcessModel {
	si := textinput.New()
	si.Placeholder = "filter processes..."
	si.CharLimit = 128
	ki := textinput.New()
	ki.Placeholder = "process name to kill..."
	ki.CharLimit = 128
	return ProcessModel{
		client:        client,
		sortField:     SortByCPU,
		searchInput:   si,
		killNameInput: ki,
		refreshSecs:   3,
	}
}

func (m ProcessModel) IsInputCaptured() bool {
	return m.showSearch || m.confirmKill || m.showKillName
}

func (m ProcessModel) Init() tea.Cmd {
	return nil
}

func (m ProcessModel) Update(msg tea.Msg) (ProcessModel, tea.Cmd) {
	switch msg := msg.(type) {
	case processListMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.processes = msg.procs
			m.memInfo = msg.memInfo
			m.sortProcesses()
			m.rebuildVisible()
		}
		return m, m.scheduleRefresh()

	case processTickMsg:
		if m.serial != "" {
			m.loading = true
			return m, m.fetchProcesses()
		}
		return m, nil

	case processKillMsg:
		if msg.err != nil {
			if msg.pid > 0 {
				m.statusMsg = ErrorStyle.Render(fmt.Sprintf("Kill PID %d failed: %s", msg.pid, msg.err))
			} else {
				m.statusMsg = ErrorStyle.Render(fmt.Sprintf("Kill failed: %s", msg.err))
			}
		} else {
			if msg.pid > 0 {
				m.statusMsg = SuccessStyle.Render(fmt.Sprintf("Killed PID %d", msg.pid))
			} else {
				m.statusMsg = SuccessStyle.Render("Process killed")
			}
		}
		return m, tea.Batch(m.fetchProcesses(), clearStatusAfter(5*time.Second))

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.showSearch {
			return m.updateSearch(msg)
		}
		if m.showKillName {
			return m.updateKillName(msg)
		}
		if m.confirmKill {
			return m.updateConfirmKill(msg)
		}
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case key.Matches(msg, DefaultKeyMap.Down):
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case key.Matches(msg, DefaultKeyMap.Search):
			m.showSearch = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.Refresh):
			m.loading = true
			return m, m.fetchProcesses()
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.cursor = 0
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			m.cursor = max(len(m.visible)-1, 0)
			m.ensureVisible()
		case msg.String() == "s":
			m.sortField = (m.sortField + 1) % 4
			m.sortProcesses()
			m.rebuildVisible()
		case msg.String() == "x":
			if p := m.selectedProcess(); p != nil {
				m.confirmKill = true
			}
		case msg.String() == "X":
			m.showKillName = true
			m.killNameInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.HalfPageUp):
			m.cursor = max(m.cursor-m.pageSize()/2, 0)
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.HalfPageDown):
			m.cursor = min(m.cursor+m.pageSize()/2, max(len(m.visible)-1, 0))
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.PageUp):
			m.cursor = max(m.cursor-m.pageSize(), 0)
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.PageDown):
			m.cursor = min(m.cursor+m.pageSize(), max(len(m.visible)-1, 0))
			m.ensureVisible()
		}
	}
	return m, nil
}

func (m ProcessModel) updateSearch(msg tea.KeyMsg) (ProcessModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.searchQuery = m.searchInput.Value()
		m.showSearch = false
		m.searchInput.Blur()
		m.rebuildVisible()
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

func (m ProcessModel) updateConfirmKill(msg tea.KeyMsg) (ProcessModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirmKill = false
		if p := m.selectedProcess(); p != nil {
			return m, m.killProcess(p.PID)
		}
	default:
		m.confirmKill = false
	}
	return m, nil
}

func (m ProcessModel) View() string {
	var b strings.Builder

	sortLabel := [...]string{"CPU", "MEM", "PID", "Name"}
	title := fmt.Sprintf("  Processes %s  sort:%s",
		scrollInfo(m.cursor, len(m.visible)),
		AccentStyle.Render(sortLabel[m.sortField]))
	b.WriteString(HeaderStyle.Render(title))
	b.WriteString("\n")

	if m.serial == "" {
		b.WriteString(DimStyle.Render("  No device selected") + "\n")
		return b.String()
	}

	if m.memInfo != nil {
		totalMB := m.memInfo.Total / 1024
		freeMB := m.memInfo.Free / 1024
		availMB := m.memInfo.Available / 1024
		fmt.Fprintf(&b, "  %s Total: %dMB  Free: %dMB  Available: %dMB\n",
			AccentStyle.Render("Memory"),
			totalMB, freeMB, availMB)
	}

	if m.loading {
		b.WriteString(DimStyle.Render("  Refreshing...") + "\n")
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render("  Error: "+m.err.Error()) + "\n")
	}

	if m.searchQuery != "" {
		b.WriteString("  " + DimStyle.Render("Filter: "+m.searchQuery) + "\n")
	}

	header := fmt.Sprintf("  %-8s %-12s %7s %7s  %s",
		TableHeaderStyle.Render("PID"),
		TableHeaderStyle.Render("USER"),
		TableHeaderStyle.Render("CPU%"),
		TableHeaderStyle.Render("MEM%"),
		TableHeaderStyle.Render("NAME"))
	b.WriteString(header + "\n")

	viewHeight := safeViewHeight(m.height, 12, 15)

	end := min(m.scroll+viewHeight, len(m.visible))
	for i := m.scroll; i < end; i++ {
		idx := m.visible[i]
		p := m.processes[idx]
		prefix := "  "
		if i == m.cursor {
			prefix = CursorStyle.Render("▸ ")
		}

		style := NormalStyle
		if i == m.cursor {
			style = SelectedStyle
		} else if p.CPU > 50 || p.MEM > 50 {
			style = ErrorStyle
		} else if p.CPU > 20 || p.MEM > 20 {
			style = WarningStyle
		}

		line := fmt.Sprintf("%-8d %-12s %6.1f %6.1f  %s",
			p.PID, p.User, p.CPU, p.MEM, p.Name)
		b.WriteString(prefix + style.Render(line) + "\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n")
	}

	if m.confirmKill {
		if p := m.selectedProcess(); p != nil {
			b.WriteString("\n" + DialogStyle.Render(
				fmt.Sprintf("Kill process %d (%s)? [y/N]", p.PID, p.Name)) + "\n")
		}
	}

	if m.showSearch {
		b.WriteString("\n  " + m.searchInput.View() + "\n")
	}
	if m.showKillName {
		b.WriteString("\n  " + DialogStyle.Render("Kill by name: "+m.killNameInput.View()) + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("/", "filter"),
		keyHint("s", "sort"),
		keyHint("x", "kill pid"),
		keyHint("X", "kill name"),
		keyHint("r", "refresh"),
		keyHint("g/G", "top/bottom"),
	))

	return b.String()
}

func (m *ProcessModel) sortProcesses() {
	sort.Slice(m.processes, func(i, j int) bool {
		switch m.sortField {
		case SortByCPU:
			return m.processes[i].CPU > m.processes[j].CPU
		case SortByMEM:
			return m.processes[i].MEM > m.processes[j].MEM
		case SortByPID:
			return m.processes[i].PID < m.processes[j].PID
		case SortByName:
			return m.processes[i].Name < m.processes[j].Name
		default:
			return m.processes[i].CPU > m.processes[j].CPU
		}
	})
}

func (m *ProcessModel) rebuildVisible() {
	m.visible = nil
	q := strings.ToLower(m.searchQuery)
	for i, p := range m.processes {
		if q != "" && !strings.Contains(strings.ToLower(p.Name), q) &&
			!strings.Contains(strings.ToLower(p.User), q) {
			continue
		}
		m.visible = append(m.visible, i)
	}
	if m.cursor >= len(m.visible) {
		m.cursor = max(len(m.visible)-1, 0)
	}
}

func (m *ProcessModel) ensureVisible() {
	viewHeight := safeViewHeight(m.height, 12, 15)
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+viewHeight {
		m.scroll = m.cursor - viewHeight + 1
	}
}

func (m ProcessModel) pageSize() int {
	return safeViewHeight(m.height, 12, 10)
}

func (m ProcessModel) selectedProcess() *adb.ProcessInfo {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		idx := m.visible[m.cursor]
		return &m.processes[idx]
	}
	return nil
}

func (m ProcessModel) SetDevice(serial string) (ProcessModel, tea.Cmd) {
	m.serial = serial
	m.processes = nil
	m.visible = nil
	m.memInfo = nil
	m.cursor = 0
	m.scroll = 0
	m.statusMsg = ""
	if serial == "" {
		return m, nil
	}
	m.loading = true
	return m, m.fetchProcesses()
}

func (m ProcessModel) SetSize(w, h int) ProcessModel {
	m.width = w
	m.height = h
	return m
}

func (m ProcessModel) fetchProcesses() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		procs, err := client.ListProcesses(ctx, serial)
		if err != nil {
			return processListMsg{err: err}
		}
		memInfo, _ := client.GetMemInfo(ctx, serial)
		return processListMsg{procs: procs, memInfo: memInfo}
	}
}

func (m ProcessModel) updateKillName(msg tea.KeyMsg) (ProcessModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := m.killNameInput.Value()
		m.showKillName = false
		m.killNameInput.Reset()
		m.killNameInput.Blur()
		if name != "" {
			return m, m.killProcessByName(name)
		}
		return m, nil
	case tea.KeyEsc:
		m.showKillName = false
		m.killNameInput.Reset()
		m.killNameInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.killNameInput, cmd = m.killNameInput.Update(msg)
	return m, cmd
}

func (m ProcessModel) killProcessByName(name string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.KillProcessByName(ctx, serial, name)
		return processKillMsg{pid: 0, err: err}
	}
}

func (m ProcessModel) killProcess(pid int) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.KillProcess(ctx, serial, pid)
		return processKillMsg{pid: pid, err: err}
	}
}

func (m ProcessModel) scheduleRefresh() tea.Cmd {
	d := time.Duration(m.refreshSecs) * time.Second
	return tea.Tick(d, func(time.Time) tea.Msg {
		return processTickMsg{}
	})
}
