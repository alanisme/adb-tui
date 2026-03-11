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

type backupActionMsg struct {
	action string
	err    error
}

var backupOps = []struct {
	label       string
	desc        string
	placeholder string
}{
	{"Bugreport", "Generate device bug report", "/local/path/bugreport"},
	{"Backup", "Full device backup (APK+shared+all)", "/local/path/backup.ab"},
	{"Restore", "Restore from backup file", "/local/path/backup.ab"},
	{"Sideload", "Flash OTA zip via sideload", "/local/path/update.zip"},
}

type backupOverlay struct {
	cursor int
	step   int // 0=menu, 1=path input
	input  textinput.Model
}

func newBackupOverlay() backupOverlay {
	ti := textinput.New()
	ti.CharLimit = 256
	return backupOverlay{input: ti}
}

func (o backupOverlay) update(msg tea.KeyMsg, client *adb.Client, serial string) (backupOverlay, tea.Cmd, bool) {
	if msg.Type == tea.KeyEsc {
		if o.step > 0 {
			o.step = 0
			o.input.Reset()
			o.input.Blur()
			return o, nil, false
		}
		return o, nil, true
	}

	// Path input mode (step 1)
	if o.step == 1 {
		if msg.Type == tea.KeyEnter {
			filePath := o.input.Value()
			o.input.Reset()
			o.input.Blur()
			o.step = 0
			if filePath == "" {
				return o, nil, true
			}
			return o, cmdBackupOp(client, serial, o.cursor, filePath), true
		}
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(msg)
		return o, cmd, false
	}

	// Menu navigation (step 0)
	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		o.cursor = max(o.cursor-1, 0)
	case key.Matches(msg, DefaultKeyMap.Down):
		o.cursor = min(o.cursor+1, len(backupOps)-1)
	case key.Matches(msg, DefaultKeyMap.Enter):
		o.step = 1
		o.input.Placeholder = backupOps[o.cursor].placeholder
		o.input.Reset()
		o.input.Focus()
		return o, textinput.Blink, false
	}
	return o, nil, false
}

func (o backupOverlay) view(statusMsg string, _ int) string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render("Backup & Restore") + "\n\n")

	for i, op := range backupOps {
		prefix := "  "
		style := NormalStyle
		if i == o.cursor {
			prefix = CursorStyle.Render("▸ ")
			style = SelectedStyle
		}
		fmt.Fprintf(&b, "%s%-16s %s\n", prefix, style.Render(op.label), DimStyle.Render(op.desc))
	}

	if o.step == 1 {
		b.WriteString("\n  " + DialogStyle.Render("Path: "+o.input.View()) + "\n")
	}

	if statusMsg != "" {
		b.WriteString("\n  " + statusMsg + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("j/k", "select"),
		keyHint("⏎", "confirm"),
		keyHint("esc", "back"),
	))
	return b.String()
}

func cmdBackupOp(client *adb.Client, serial string, idx int, filePath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		var err error
		var action string
		switch idx {
		case 0:
			err = client.Bugreport(ctx, serial, filePath)
			action = "Bugreport saved to " + filePath
		case 1:
			opts := adb.BackupOptions{APK: true, Shared: true, All: true}
			err = client.Backup(ctx, serial, filePath, opts)
			action = "Backup saved to " + filePath
		case 2:
			err = client.Restore(ctx, serial, filePath)
			action = "Restore completed from " + filePath
		case 3:
			err = client.Sideload(ctx, serial, filePath)
			action = "Sideload completed"
		}
		return backupActionMsg{action: action, err: err}
	}
}
