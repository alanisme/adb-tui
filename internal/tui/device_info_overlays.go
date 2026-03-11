package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alanisme/adb-tui/internal/adb"
)

// Messages returned by overlay commands.

type batterySimMsg struct {
	action string
	err    error
}

type displayAdjustMsg struct {
	action string
	err    error
}

type notificationListMsg struct {
	items []adb.NotificationInfo
	err   error
}

type rebootMsg struct {
	mode string
	err  error
}

// --- Battery overlay ---

type batteryOverlay struct {
	cursor int
	level  int
	status int // 1-5
	plug   int // 0,1,2,4
}

func newBatteryOverlay() batteryOverlay {
	return batteryOverlay{level: 50, status: 2}
}

func (o batteryOverlay) update(msg tea.KeyMsg, client *adb.Client, serial string) (batteryOverlay, tea.Cmd, bool) {
	if msg.Type == tea.KeyEsc {
		return o, nil, true
	}
	const items = 5
	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		o.cursor = max(o.cursor-1, 0)
	case key.Matches(msg, DefaultKeyMap.Down):
		o.cursor = min(o.cursor+1, items-1)
	case msg.String() == "h", msg.Type == tea.KeyLeft:
		switch o.cursor {
		case 0:
			o.level = max(o.level-5, 0)
		case 1:
			o.status--
			if o.status < 1 {
				o.status = 5
			}
		case 2:
			switch o.plug {
			case 4:
				o.plug = 2
			case 2:
				o.plug = 1
			case 1:
				o.plug = 0
			default:
				o.plug = 4
			}
		}
	case msg.String() == "l", msg.Type == tea.KeyRight:
		switch o.cursor {
		case 0:
			o.level = min(o.level+5, 100)
		case 1:
			o.status++
			if o.status > 5 {
				o.status = 1
			}
		case 2:
			switch o.plug {
			case 0:
				o.plug = 1
			case 1:
				o.plug = 2
			case 2:
				o.plug = 4
			default:
				o.plug = 0
			}
		}
	case key.Matches(msg, DefaultKeyMap.Enter):
		var cmd tea.Cmd
		switch o.cursor {
		case 0:
			cmd = o.cmdSetLevel(client, serial)
		case 1:
			cmd = o.cmdSetStatus(client, serial)
		case 2:
			cmd = o.cmdSetPlugged(client, serial)
		case 3:
			cmd = o.cmdUnplug(client, serial)
		case 4:
			cmd = o.cmdReset(client, serial)
		}
		return o, cmd, false
	}
	return o, nil, false
}

func (o batteryOverlay) view(statusMsg string, _ int) string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render("Battery Simulation") + "\n\n")

	items := []struct {
		label string
		value string
	}{
		{"Level", fmt.Sprintf("◂ %d%% ▸", o.level)},
		{"Status", fmt.Sprintf("◂ %s ▸", adb.BatteryStatusName(o.status))},
		{"Plugged", fmt.Sprintf("◂ %s ▸", plugName(o.plug))},
		{"Unplug", "simulate battery disconnect"},
		{"Reset", "restore real battery state"},
	}

	for i, item := range items {
		prefix := "  "
		style := NormalStyle
		if i == o.cursor {
			prefix = CursorStyle.Render("▸ ")
			style = SelectedStyle
		}
		fmt.Fprintf(&b, "%s%-12s %s\n", prefix, style.Render(item.label), DimStyle.Render(item.value))
	}

	if statusMsg != "" {
		b.WriteString("\n  " + statusMsg + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("j/k", "select"),
		keyHint("h/l", "adjust"),
		keyHint("⏎", "apply"),
		keyHint("esc", "back"),
	))
	return b.String()
}

func plugName(p int) string {
	switch p {
	case 0:
		return "None"
	case 1:
		return "AC"
	case 2:
		return "USB"
	case 4:
		return "Wireless"
	default:
		return strconv.Itoa(p)
	}
}

func (o batteryOverlay) cmdSetLevel(client *adb.Client, serial string) tea.Cmd {
	level := o.level
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SetBatteryLevel(ctx, serial, level)
		return batterySimMsg{action: fmt.Sprintf("Battery level set to %d%%", level), err: err}
	}
}

func (o batteryOverlay) cmdSetStatus(client *adb.Client, serial string) tea.Cmd {
	status := o.status
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SetBatteryStatus(ctx, serial, status)
		return batterySimMsg{action: "Battery status set to " + adb.BatteryStatusName(status), err: err}
	}
}

func (o batteryOverlay) cmdSetPlugged(client *adb.Client, serial string) tea.Cmd {
	plug := o.plug
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SetBatteryPlugged(ctx, serial, plug)
		return batterySimMsg{action: "Battery plugged set to " + plugName(plug), err: err}
	}
}

func (o batteryOverlay) cmdUnplug(client *adb.Client, serial string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SimulateBatteryUnplug(ctx, serial)
		return batterySimMsg{action: "Battery unplugged", err: err}
	}
}

func (o batteryOverlay) cmdReset(client *adb.Client, serial string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.ResetBattery(ctx, serial)
		return batterySimMsg{action: "Battery reset to real state", err: err}
	}
}

// --- Display overlay ---

type displayOverlay struct {
	field   int // 0=width, 1=height, 2=density, 3=font scale
	input   textinput.Model
	width   string
	height  string
	density string
	font    string
}

func newDisplayOverlay() displayOverlay {
	ti := textinput.New()
	ti.CharLimit = 10
	return displayOverlay{input: ti}
}

func (o displayOverlay) open() displayOverlay {
	o.field = 0
	o.width = ""
	o.height = ""
	o.density = ""
	o.font = ""
	o.input.Placeholder = "width"
	o.input.Reset()
	o.input.Focus()
	return o
}

func (o displayOverlay) update(msg tea.KeyMsg, client *adb.Client, serial string) (displayOverlay, tea.Cmd, bool) {
	if msg.Type == tea.KeyEsc {
		o.input.Blur()
		return o, nil, true
	}
	switch msg.Type {
	case tea.KeyTab:
		o.field = (o.field + 1) % 4
		switch o.field {
		case 0:
			o.input.Placeholder = "width"
		case 1:
			o.input.Placeholder = "height"
		case 2:
			o.input.Placeholder = "density (dpi)"
		case 3:
			o.input.Placeholder = "font scale (e.g. 1.0)"
		}
		o.input.Reset()
		o.input.Focus()
		return o, textinput.Blink, false
	case tea.KeyEnter:
		val := o.input.Value()
		o.input.Reset()
		switch o.field {
		case 0:
			o.width = val
			if o.width != "" && o.height != "" {
				w, _ := strconv.Atoi(o.width)
				h, _ := strconv.Atoi(o.height)
				if w > 0 && h > 0 {
					return o, cmdSetDisplaySize(client, serial, w, h), false
				}
				return o, cmdDisplayError("Invalid size"), false
			}
			o.field = 1
			o.input.Placeholder = "height"
			o.input.Focus()
			return o, textinput.Blink, false
		case 1:
			o.height = val
			if o.width != "" && o.height != "" {
				w, _ := strconv.Atoi(o.width)
				h, _ := strconv.Atoi(o.height)
				if w > 0 && h > 0 {
					return o, cmdSetDisplaySize(client, serial, w, h), false
				}
				return o, cmdDisplayError("Invalid size"), false
			}
		case 2:
			if dpi, err := strconv.Atoi(val); err == nil && dpi > 0 {
				return o, cmdSetDensity(client, serial, dpi), false
			}
			return o, cmdDisplayError("Invalid density"), false
		case 3:
			if scale, err := strconv.ParseFloat(val, 64); err == nil && scale > 0 {
				return o, cmdSetFontScale(client, serial, scale), false
			}
			return o, cmdDisplayError("Invalid font scale"), false
		}
	case tea.KeyBackspace, tea.KeyDelete:
		// let input handle it
	default:
		if msg.String() == "R" {
			return o, cmdResetDisplay(client, serial), false
		}
	}
	var cmd tea.Cmd
	o.input, cmd = o.input.Update(msg)
	return o, cmd, false
}

func (o displayOverlay) view(statusMsg string, _ int) string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render("Display Settings") + "\n\n")

	fields := []string{"Width", "Height", "Density (dpi)", "Font Scale"}
	for i, label := range fields {
		marker := "  "
		if i == o.field {
			marker = CursorStyle.Render("▸ ")
		}
		fmt.Fprintf(&b, "%s%s\n", marker, AccentStyle.Render(label))
	}

	b.WriteString("\n  " + o.input.View() + "\n")

	if statusMsg != "" {
		b.WriteString("\n  " + statusMsg + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("tab", "field"),
		keyHint("⏎", "apply"),
		keyHint("R", "reset all"),
		keyHint("esc", "back"),
	))
	return b.String()
}

// Display command helpers — standalone so the displayOverlay doesn't need
// to capture state beyond the arguments.

func cmdDisplayError(text string) tea.Cmd {
	return func() tea.Msg {
		return displayAdjustMsg{action: text, err: fmt.Errorf("%s", text)}
	}
}

func cmdSetDisplaySize(client *adb.Client, serial string, w, h int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SetDisplaySize(ctx, serial, w, h)
		return displayAdjustMsg{action: fmt.Sprintf("Display size set to %dx%d", w, h), err: err}
	}
}

func cmdSetDensity(client *adb.Client, serial string, dpi int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SetDensity(ctx, serial, dpi)
		return displayAdjustMsg{action: fmt.Sprintf("Density set to %d dpi", dpi), err: err}
	}
}

func cmdSetFontScale(client *adb.Client, serial string, scale float64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SetFontScale(ctx, serial, scale)
		return displayAdjustMsg{action: fmt.Sprintf("Font scale set to %.2f", scale), err: err}
	}
}

func cmdResetDisplay(client *adb.Client, serial string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var errs []string
		if err := client.ResetDisplaySize(ctx, serial); err != nil {
			errs = append(errs, err.Error())
		}
		if err := client.ResetDensity(ctx, serial); err != nil {
			errs = append(errs, err.Error())
		}
		if err := client.SetFontScale(ctx, serial, 1.0); err != nil {
			errs = append(errs, err.Error())
		}
		if len(errs) > 0 {
			return displayAdjustMsg{action: "Reset display", err: fmt.Errorf("%s", strings.Join(errs, "; "))}
		}
		return displayAdjustMsg{action: "Display settings reset"}
	}
}

// --- Notification overlay ---

type notifOverlay struct {
	items  []adb.NotificationInfo
	scroll int
}

func (o notifOverlay) update(msg tea.KeyMsg, client *adb.Client, serial string) (notifOverlay, tea.Cmd, bool) {
	if msg.Type == tea.KeyEsc {
		return o, nil, true
	}
	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		o.scroll = max(o.scroll-1, 0)
	case key.Matches(msg, DefaultKeyMap.Down):
		o.scroll++
	case key.Matches(msg, DefaultKeyMap.Refresh):
		return o, cmdFetchNotifications(client, serial), false
	}
	return o, nil, false
}

func (o notifOverlay) view(statusMsg string, viewHeight int) string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render(fmt.Sprintf("Notifications (%d)", len(o.items))) + "\n\n")

	if len(o.items) == 0 {
		b.WriteString("  " + DimStyle.Render("No notifications") + "\n")
	} else {
		maxVisible := viewHeight - 4
		maxScroll := max(len(o.items)-maxVisible, 0)
		scroll := min(o.scroll, maxScroll)
		end := min(scroll+maxVisible, len(o.items))
		for _, n := range o.items[scroll:end] {
			pkg := DimStyle.Render(n.Package)
			title := n.Title
			if title == "" {
				title = DimStyle.Render("(no title)")
			}
			fmt.Fprintf(&b, "  %s  %s\n", SelectedStyle.Render(title), pkg)
			if n.Text != "" {
				fmt.Fprintf(&b, "    %s\n", n.Text)
			}
		}
	}

	if statusMsg != "" {
		b.WriteString("\n  " + statusMsg + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("j/k", "scroll"),
		keyHint("r", "refresh"),
		keyHint("esc", "back"),
	))
	return b.String()
}

func cmdFetchNotifications(client *adb.Client, serial string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		items, err := client.ListNotifications(ctx, serial)
		return notificationListMsg{items: items, err: err}
	}
}

// --- Reboot overlay ---

type rebootOverlay struct {
	cursor  int
	confirm bool
}

var rebootModes = []struct {
	label string
	mode  string
	desc  string
}{
	{"System", "", "Normal reboot"},
	{"Recovery", "recovery", "Boot into recovery mode"},
	{"Bootloader", "bootloader", "Boot into bootloader/fastboot"},
	{"Sideload", "sideload", "Boot into sideload mode"},
}

func (o rebootOverlay) update(msg tea.KeyMsg, client *adb.Client, serial string) (rebootOverlay, tea.Cmd, bool) {
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
			mode := rebootModes[o.cursor].mode
			return o, cmdReboot(client, serial, mode), true
		default:
			o.confirm = false
		}
		return o, nil, false
	}

	switch {
	case key.Matches(msg, DefaultKeyMap.Up):
		o.cursor = max(o.cursor-1, 0)
	case key.Matches(msg, DefaultKeyMap.Down):
		o.cursor = min(o.cursor+1, len(rebootModes)-1)
	case key.Matches(msg, DefaultKeyMap.Enter):
		o.confirm = true
	}
	return o, nil, false
}

func (o rebootOverlay) view(statusMsg string, _ int) string {
	var b strings.Builder
	b.WriteString("  " + TitleStyle.Render("Reboot Device") + "\n\n")

	for i, rm := range rebootModes {
		prefix := "  "
		style := NormalStyle
		if i == o.cursor {
			prefix = CursorStyle.Render("▸ ")
			style = SelectedStyle
		}
		fmt.Fprintf(&b, "%s%-16s %s\n", prefix, style.Render(rm.label), DimStyle.Render(rm.desc))
	}

	if o.confirm {
		b.WriteString("\n" + DialogStyle.Render(
			fmt.Sprintf("Reboot into %s? [y/N]", rebootModes[o.cursor].label)) + "\n")
	}

	if statusMsg != "" {
		b.WriteString("\n  " + statusMsg + "\n")
	}

	b.WriteString("\n" + helpBar(
		keyHint("j/k", "select"),
		keyHint("⏎", "reboot"),
		keyHint("esc", "back"),
	))
	return b.String()
}

func cmdReboot(client *adb.Client, serial, mode string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := client.Reboot(ctx, serial, mode)
		return rebootMsg{mode: mode, err: err}
	}
}
