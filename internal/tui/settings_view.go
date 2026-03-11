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

// settingsMessage is implemented by all messages routed to SettingsModel.
type settingsMessage interface{ settingsMsg() }

type settingsListMsg struct {
	settings []settingEntry
	err      error
}

type settingsActionMsg struct {
	action string
	err    error
}

func (settingsListMsg) settingsMsg()  {}
func (settingsActionMsg) settingsMsg() {}

type settingEntry struct {
	Key   string
	Value string
}

type SettingsDialog int

const (
	SettingsDialogNone SettingsDialog = iota
	SettingsDialogEdit
	SettingsDialogAdd
	SettingsDialogDelete
)

type SettingsModel struct {
	client      *adb.Client
	serial      string
	namespace   adb.SettingNamespace
	settings    []settingEntry
	visible     []int
	cursor      int
	scroll      int
	width       int
	height      int
	err         error
	loading     bool
	dialog      SettingsDialog
	searchInput textinput.Model
	showSearch  bool
	searchQuery string
	editInput   textinput.Model
	addKeyInput textinput.Model
	addValInput textinput.Model
	addFocusVal bool
	statusMsg   string
}

var settingsNamespaces = []adb.SettingNamespace{
	adb.NamespaceSystem,
	adb.NamespaceSecure,
	adb.NamespaceGlobal,
}

func NewSettingsModel(client *adb.Client) SettingsModel {
	si := textinput.New()
	si.Placeholder = "filter settings..."
	si.CharLimit = 128

	ei := textinput.New()
	ei.Placeholder = "new value"
	ei.CharLimit = 256

	aki := textinput.New()
	aki.Placeholder = "setting key"
	aki.CharLimit = 256

	avi := textinput.New()
	avi.Placeholder = "setting value"
	avi.CharLimit = 256

	return SettingsModel{
		client:      client,
		namespace:   adb.NamespaceSystem,
		searchInput: si,
		editInput:   ei,
		addKeyInput: aki,
		addValInput: avi,
	}
}

func (m SettingsModel) IsInputCaptured() bool {
	return m.showSearch || m.dialog != SettingsDialogNone
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case settingsListMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.settings = msg.settings
			m.rebuildVisible()
		}
		return m, nil

	case settingsActionMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action + " completed")
		}
		m.loading = true
		return m, tea.Batch(m.fetchSettings(), clearStatusAfter(5*time.Second))

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.showSearch {
			return m.updateSearch(msg)
		}
		if m.dialog == SettingsDialogEdit {
			return m.updateEdit(msg)
		}
		if m.dialog == SettingsDialogAdd {
			return m.updateAdd(msg)
		}
		if m.dialog == SettingsDialogDelete {
			return m.updateDelete(msg)
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
		case key.Matches(msg, DefaultKeyMap.Enter):
			if s := m.selectedSetting(); s != nil {
				m.dialog = SettingsDialogEdit
				m.editInput.SetValue(s.Value)
				m.editInput.Focus()
				return m, textinput.Blink
			}
		case key.Matches(msg, DefaultKeyMap.Search):
			m.showSearch = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.Refresh):
			m.loading = true
			return m, m.fetchSettings()
		case key.Matches(msg, DefaultKeyMap.Delete):
			if m.selectedSetting() != nil {
				m.dialog = SettingsDialogDelete
			}
		case msg.String() == "a":
			m.dialog = SettingsDialogAdd
			m.addKeyInput.Reset()
			m.addValInput.Reset()
			m.addFocusVal = false
			m.addKeyInput.Focus()
			return m, textinput.Blink
		case msg.String() == "n":
			idx := 0
			for i, ns := range settingsNamespaces {
				if ns == m.namespace {
					idx = i
					break
				}
			}
			m.namespace = settingsNamespaces[(idx+1)%len(settingsNamespaces)]
			m.cursor = 0
			m.scroll = 0
			m.searchQuery = ""
			m.searchInput.Reset()
			if m.serial != "" {
				m.loading = true
				return m, m.fetchSettings()
			}
		case key.Matches(msg, DefaultKeyMap.GotoTop):
			m.cursor = 0
			m.ensureVisible()
		case key.Matches(msg, DefaultKeyMap.GotoBottom):
			m.cursor = max(len(m.visible)-1, 0)
			m.ensureVisible()
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

func (m SettingsModel) updateSearch(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
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

func (m SettingsModel) updateEdit(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := m.editInput.Value()
		m.dialog = SettingsDialogNone
		m.editInput.Blur()
		if s := m.selectedSetting(); s != nil && val != s.Value {
			return m, m.putSetting(s.Key, val)
		}
		return m, nil
	case tea.KeyEsc:
		m.dialog = SettingsDialogNone
		m.editInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}

func (m SettingsModel) updateAdd(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if !m.addFocusVal {
			m.addFocusVal = true
			m.addKeyInput.Blur()
			m.addValInput.Focus()
			return m, textinput.Blink
		}
		k := m.addKeyInput.Value()
		v := m.addValInput.Value()
		m.dialog = SettingsDialogNone
		m.addKeyInput.Blur()
		m.addValInput.Blur()
		if k != "" {
			return m, m.putSetting(k, v)
		}
		return m, nil
	case tea.KeyEsc:
		m.dialog = SettingsDialogNone
		m.addKeyInput.Blur()
		m.addValInput.Blur()
		return m, nil
	case tea.KeyTab:
		if !m.addFocusVal {
			m.addFocusVal = true
			m.addKeyInput.Blur()
			m.addValInput.Focus()
		} else {
			m.addFocusVal = false
			m.addValInput.Blur()
			m.addKeyInput.Focus()
		}
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	if m.addFocusVal {
		m.addValInput, cmd = m.addValInput.Update(msg)
	} else {
		m.addKeyInput, cmd = m.addKeyInput.Update(msg)
	}
	return m, cmd
}

func (m SettingsModel) updateDelete(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.dialog = SettingsDialogNone
		if s := m.selectedSetting(); s != nil {
			return m, m.deleteSetting(s.Key)
		}
	default:
		m.dialog = SettingsDialogNone
	}
	return m, nil
}

func (m SettingsModel) View() string {
	var b strings.Builder

	var tabs []string
	for _, ns := range settingsNamespaces {
		name := string(ns)
		if len(name) > 0 {
			name = strings.ToUpper(name[:1]) + name[1:]
		}
		label := fmt.Sprintf(" %s ", name)
		if ns == m.namespace {
			tabs = append(tabs, ActiveTabStyle.Render(label))
		} else {
			tabs = append(tabs, InactiveTabStyle.Render(label))
		}
	}
	b.WriteString("  " + strings.Join(tabs, DimStyle.Render("|")) + "\n")

	title := HeaderStyle.Render(fmt.Sprintf("Settings [%s] %s (%d)",
		m.namespace, scrollInfo(m.cursor, len(m.visible)), len(m.settings)))
	b.WriteString(title)
	b.WriteString("\n")

	if m.serial == "" {
		b.WriteString(DimStyle.Render("  No device selected") + "\n")
		return b.String()
	}

	if m.loading {
		b.WriteString(DimStyle.Render("  Loading...") + "\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render("  Error: "+m.err.Error()) + "\n")
	}

	if m.searchQuery != "" {
		b.WriteString("  " + DimStyle.Render("Filter: "+m.searchQuery) + "\n")
	}

	viewHeight := safeViewHeight(m.height, 12, 15)

	end := min(m.scroll+viewHeight, len(m.visible))
	for i := m.scroll; i < end; i++ {
		idx := m.visible[i]
		s := m.settings[idx]
		prefix := "  "
		style := NormalStyle
		if i == m.cursor {
			prefix = CursorStyle.Render("▸ ")
			style = SelectedStyle
		}
		val := s.Value
		if len(val) > m.width-40 && m.width > 50 {
			val = val[:m.width-43] + "..."
		}
		b.WriteString(prefix + style.Render(s.Key) + DimStyle.Render("=") + NormalStyle.Render(val) + "\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n")
	}

	if m.dialog == SettingsDialogEdit {
		if s := m.selectedSetting(); s != nil {
			b.WriteString("\n" + DialogStyle.Render(
				fmt.Sprintf("Edit [%s]:\n%s", s.Key, m.editInput.View())) + "\n")
		}
	}

	if m.dialog == SettingsDialogAdd {
		b.WriteString("\n" + DialogStyle.Render(
			fmt.Sprintf("Add Setting:\nKey:   %s\nValue: %s",
				m.addKeyInput.View(), m.addValInput.View())) + "\n")
	}

	if m.dialog == SettingsDialogDelete {
		if s := m.selectedSetting(); s != nil {
			b.WriteString("\n" + DialogStyle.Render(
				fmt.Sprintf("Delete setting %q? [y/N]", s.Key)) + "\n")
		}
	}

	if m.showSearch {
		b.WriteString("\n  " + m.searchInput.View() + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("n", "namespace"),
		keyHint("/", "filter"),
		keyHint("⏎", "edit"),
		keyHint("a", "add"),
		keyHint("d", "delete"),
		keyHint("r", "refresh"),
	))

	return b.String()
}

func (m *SettingsModel) rebuildVisible() {
	m.visible = nil
	q := strings.ToLower(m.searchQuery)
	for i, s := range m.settings {
		if q != "" && !strings.Contains(strings.ToLower(s.Key), q) &&
			!strings.Contains(strings.ToLower(s.Value), q) {
			continue
		}
		m.visible = append(m.visible, i)
	}
	if m.cursor >= len(m.visible) {
		m.cursor = max(len(m.visible)-1, 0)
	}
}

func (m *SettingsModel) ensureVisible() {
	viewHeight := safeViewHeight(m.height, 12, 15)
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+viewHeight {
		m.scroll = m.cursor - viewHeight + 1
	}
}

func (m SettingsModel) pageSize() int {
	return safeViewHeight(m.height, 12, 10)
}

func (m SettingsModel) selectedSetting() *settingEntry {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		idx := m.visible[m.cursor]
		return &m.settings[idx]
	}
	return nil
}

func (m SettingsModel) SetDevice(serial string) (SettingsModel, tea.Cmd) {
	m.serial = serial
	m.settings = nil
	m.visible = nil
	m.cursor = 0
	m.scroll = 0
	m.dialog = SettingsDialogNone
	m.statusMsg = ""
	if serial == "" {
		return m, nil
	}
	m.loading = true
	return m, m.fetchSettings()
}

func (m SettingsModel) SetSize(w, h int) SettingsModel {
	m.width = w
	m.height = h
	return m
}

func (m SettingsModel) fetchSettings() tea.Cmd {
	serial := m.serial
	client := m.client
	ns := m.namespace
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		raw, err := client.ListSettings(ctx, serial, ns)
		if err != nil {
			return settingsListMsg{err: err}
		}
		entries := make([]settingEntry, 0, len(raw))
		for k, v := range raw {
			entries = append(entries, settingEntry{Key: k, Value: v})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Key < entries[j].Key
		})
		return settingsListMsg{settings: entries}
	}
}

func (m SettingsModel) putSetting(k, v string) tea.Cmd {
	serial := m.serial
	client := m.client
	ns := m.namespace
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.PutSetting(ctx, serial, ns, k, v)
		return settingsActionMsg{action: "Put " + k, err: err}
	}
}

func (m SettingsModel) deleteSetting(k string) tea.Cmd {
	serial := m.serial
	client := m.client
	ns := m.namespace
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.DeleteSetting(ctx, serial, ns, k)
		return settingsActionMsg{action: "Delete " + k, err: err}
	}
}
