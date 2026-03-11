package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

type ForwardRule struct {
	Serial string
	Local  string
	Remote string
}

// forwardMessage is implemented by all messages routed to ForwardModel.
type forwardMessage interface{ forwardMsg() }

type forwardListMsg struct {
	forwards []ForwardRule
	reverses []ForwardRule
	err      error
}

type forwardActionMsg struct {
	action string
	err    error
}

func (forwardListMsg) forwardMsg()   {}
func (forwardActionMsg) forwardMsg() {}

type ForwardDialogMode int

const (
	ForwardDialogNone ForwardDialogMode = iota
	ForwardDialogAdd
	ForwardDialogAddReverse
)

type ForwardModel struct {
	client      *adb.Client
	serial      string
	forwards    []ForwardRule
	reverses    []ForwardRule
	cursor      int
	width       int
	height      int
	err         error
	statusMsg   string
	dialog      ForwardDialogMode
	localInput  textinput.Model
	remoteInput textinput.Model
	focusLocal  bool
	showReverse bool
}

func NewForwardModel(client *adb.Client) ForwardModel {
	li := textinput.New()
	li.Placeholder = "tcp:8080"
	li.CharLimit = 64

	ri := textinput.New()
	ri.Placeholder = "tcp:8080"
	ri.CharLimit = 64

	return ForwardModel{
		client:      client,
		localInput:  li,
		remoteInput: ri,
		focusLocal:  true,
	}
}

func (m ForwardModel) IsInputCaptured() bool {
	return m.dialog != ForwardDialogNone
}

func (m ForwardModel) Init() tea.Cmd {
	return nil
}

func (m ForwardModel) Update(msg tea.Msg) (ForwardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case forwardListMsg:
		m.err = msg.err
		if msg.err == nil {
			m.forwards = msg.forwards
			m.reverses = msg.reverses
			// Clamp cursor after list changes
			if list := m.activeList(); m.cursor >= len(list) {
				m.cursor = max(len(list)-1, 0)
			}
		}
		return m, nil

	case forwardActionMsg:
		if msg.err != nil {
			m.statusMsg = ErrorStyle.Render(msg.action + " failed: " + msg.err.Error())
		} else {
			m.statusMsg = SuccessStyle.Render(msg.action + " completed")
		}
		return m, tea.Batch(m.fetchForwards(), clearStatusAfter(5*time.Second))

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.dialog != ForwardDialogNone {
			return m.updateDialog(msg)
		}
		switch {
		case key.Matches(msg, DefaultKeyMap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, DefaultKeyMap.Down):
			max := len(m.activeList()) - 1
			if m.cursor < max {
				m.cursor++
			}
		case key.Matches(msg, DefaultKeyMap.Refresh):
			return m, m.fetchForwards()
		case msg.String() == "a":
			m.dialog = ForwardDialogAdd
			m.focusLocal = true
			m.localInput.Focus()
			return m, textinput.Blink
		case msg.String() == "A":
			m.dialog = ForwardDialogAddReverse
			m.focusLocal = true
			m.localInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, DefaultKeyMap.Delete):
			list := m.activeList()
			if m.cursor >= 0 && m.cursor < len(list) {
				rule := list[m.cursor]
				if m.showReverse {
					return m, m.removeReverse(rule)
				}
				return m, m.removeForward(rule)
			}
		case msg.String() == "s":
			m.showReverse = !m.showReverse
			m.cursor = 0
		}
	}
	return m, nil
}

func (m ForwardModel) updateDialog(msg tea.KeyMsg) (ForwardModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		m.focusLocal = !m.focusLocal
		if m.focusLocal {
			m.localInput.Focus()
			m.remoteInput.Blur()
		} else {
			m.localInput.Blur()
			m.remoteInput.Focus()
		}
		return m, textinput.Blink
	case tea.KeyEnter:
		local := m.localInput.Value()
		remote := m.remoteInput.Value()
		dialog := m.dialog
		m.dialog = ForwardDialogNone
		m.localInput.Reset()
		m.remoteInput.Reset()
		m.localInput.Blur()
		m.remoteInput.Blur()
		if local != "" && remote != "" {
			if dialog == ForwardDialogAddReverse {
				return m, m.addReverse(local, remote)
			}
			return m, m.addForward(local, remote)
		}
		return m, nil
	case tea.KeyEsc:
		m.dialog = ForwardDialogNone
		m.localInput.Reset()
		m.remoteInput.Reset()
		m.localInput.Blur()
		m.remoteInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	if m.focusLocal {
		m.localInput, cmd = m.localInput.Update(msg)
	} else {
		m.remoteInput, cmd = m.remoteInput.Update(msg)
	}
	return m, cmd
}

func (m ForwardModel) View() string {
	var b strings.Builder

	tabForward := InactiveTabStyle.Render("Forward")
	tabReverse := InactiveTabStyle.Render("Reverse")
	if !m.showReverse {
		tabForward = ActiveTabStyle.Render("Forward")
	} else {
		tabReverse = ActiveTabStyle.Render("Reverse")
	}
	title := HeaderStyle.Render("Port Forwarding")
	b.WriteString(title)
	b.WriteString("\n  " + tabForward + " | " + tabReverse + "\n\n")

	if m.serial == "" {
		b.WriteString(DimStyle.Render("  No device selected") + "\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(ErrorStyle.Render("  Error: "+m.err.Error()) + "\n")
	}

	list := m.activeList()
	if len(list) == 0 {
		label := "forwards"
		if m.showReverse {
			label = "reverses"
		}
		b.WriteString(DimStyle.Render(fmt.Sprintf("  No %s configured", label)) + "\n")
	} else {
		header := fmt.Sprintf("  %-30s %-30s", "Local", "Remote")
		b.WriteString(TableHeaderStyle.Render(header) + "\n")

		for i, rule := range list {
			prefix := "  "
			style := NormalStyle
			if i == m.cursor {
				prefix = CursorStyle.Render("▸ ")
				style = SelectedStyle
			}
			line := fmt.Sprintf("%-30s %-30s",
				style.Render(rule.Local),
				style.Render(rule.Remote),
			)
			b.WriteString(prefix + line + "\n")
		}
	}

	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n")
	}

	if m.dialog != ForwardDialogNone {
		label := "Add Forward"
		if m.dialog == ForwardDialogAddReverse {
			label = "Add Reverse"
		}
		content := fmt.Sprintf("%s\n  Local:  %s\n  Remote: %s",
			label, m.localInput.View(), m.remoteInput.View())
		b.WriteString("\n" + DialogStyle.Render(content) + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("a", "forward"),
		keyHint("A", "reverse"),
		keyHint("d", "remove"),
		keyHint("s", "fwd/rev"),
		keyHint("r", "refresh"),
	))

	return b.String()
}

func (m ForwardModel) activeList() []ForwardRule {
	if m.showReverse {
		return m.reverses
	}
	return m.forwards
}

func (m ForwardModel) SetDevice(serial string) (ForwardModel, tea.Cmd) {
	m.serial = serial
	m.forwards = nil
	m.reverses = nil
	m.cursor = 0
	if serial == "" {
		return m, nil
	}
	return m, m.fetchForwards()
}

func (m ForwardModel) SetSize(w, h int) ForwardModel {
	m.width = w
	m.height = h
	return m
}

func (m ForwardModel) fetchForwards() tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var forwards, reverses []ForwardRule

		if r, err := client.ExecDevice(ctx, serial, "forward", "--list"); err == nil {
			forwards = parseForwardList(r.Output)
		}
		if r, err := client.ExecDevice(ctx, serial, "reverse", "--list"); err == nil {
			reverses = parseForwardList(r.Output)
		}

		return forwardListMsg{forwards: forwards, reverses: reverses}
	}
}

func parseForwardList(output string) []ForwardRule {
	var rules []ForwardRule
	for line := range strings.SplitSeq(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			rules = append(rules, ForwardRule{
				Serial: fields[0],
				Local:  fields[1],
				Remote: fields[2],
			})
		}
	}
	return rules
}

func (m ForwardModel) addForward(local, remote string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.ExecDevice(ctx, serial, "forward", local, remote)
		return forwardActionMsg{action: "Add forward", err: err}
	}
}

func (m ForwardModel) addReverse(local, remote string) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.ExecDevice(ctx, serial, "reverse", local, remote)
		return forwardActionMsg{action: "Add reverse", err: err}
	}
}

func (m ForwardModel) removeForward(rule ForwardRule) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.ExecDevice(ctx, serial, "forward", "--remove", rule.Local)
		return forwardActionMsg{action: "Remove forward", err: err}
	}
}

func (m ForwardModel) removeReverse(rule ForwardRule) tea.Cmd {
	serial := m.serial
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.ExecDevice(ctx, serial, "reverse", "--remove", rule.Local)
		return forwardActionMsg{action: "Remove reverse", err: err}
	}
}
