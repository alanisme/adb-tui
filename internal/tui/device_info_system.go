package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

type systemActionMsg struct {
	action string
	err    error
}

var systemOps = []struct {
	label string
	desc  string
}{
	{"Root", "Restart adbd with root permissions"},
	{"Unroot", "Restart adbd without root"},
	{"Remount", "Remount /system as read-write"},
	{"Disable Verity", "Disable dm-verity (reboot required)"},
	{"Enable Verity", "Enable dm-verity (reboot required)"},
}

type systemOverlay struct {
	cursor  int
	confirm bool
}

func (o systemOverlay) update(msg tea.KeyMsg, client *adb.Client, serial string) (systemOverlay, tea.Cmd, bool) {
	if msg.Type == tea.KeyEsc {
		if o.confirm {
			o.confirm = false
			return o, nil, false
		}
		return o, nil, true
	}

	if o.confirm {
		switch msg.String() {
		case "y", "Y":
			o.confirm = false
			return o, cmdSystemOp(client, serial, o.cursor), true
		default:
			o.confirm = false
		}
		return o, nil, false
	}

	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		o.cursor = max(o.cursor-1, 0)
	case key.Matches(msg, DefaultKeyMap.Down):
		o.cursor = min(o.cursor+1, len(systemOps)-1)
	case key.Matches(msg, DefaultKeyMap.Enter):
		o.confirm = true
	}
	return o, nil, false
}

func (o systemOverlay) view(statusMsg string, _ int) string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render("System Operations") + "\n\n")

	for i, op := range systemOps {
		prefix := "  "
		style := NormalStyle
		if i == o.cursor {
			prefix = CursorStyle.Render("▸ ")
			style = SelectedStyle
		}
		fmt.Fprintf(&b, "%s%-20s %s\n", prefix, style.Render(op.label), DimStyle.Render(op.desc))
	}

	if o.confirm {
		b.WriteString("\n" + DialogStyle.Render(
			fmt.Sprintf("Execute %s? [y/N]", systemOps[o.cursor].label)) + "\n")
	}

	if statusMsg != "" {
		b.WriteString("\n  " + statusMsg + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("j/k", "select"),
		keyHint("⏎", "execute"),
		keyHint("esc", "back"),
	))
	return b.String()
}

func cmdSystemOp(client *adb.Client, serial string, idx int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var err error
		var action string
		switch idx {
		case 0:
			err = client.RootDevice(ctx, serial)
			action = "Root enabled"
		case 1:
			err = client.UnrootDevice(ctx, serial)
			action = "Root disabled"
		case 2:
			err = client.Remount(ctx, serial)
			action = "Filesystem remounted"
		case 3:
			err = client.DisableVerity(ctx, serial)
			action = "Verity disabled (reboot required)"
		case 4:
			err = client.EnableVerity(ctx, serial)
			action = "Verity enabled (reboot required)"
		}
		return systemActionMsg{action: action, err: err}
	}
}
