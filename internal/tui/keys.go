package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit         key.Binding
	QuitConfirm  key.Binding
	Tab          key.Binding
	ShiftTab     key.Binding
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Back         key.Binding
	Refresh      key.Binding
	Delete       key.Binding
	Install      key.Binding
	Filter       key.Binding
	Search       key.Binding
	Screenshot   key.Binding
	Logcat       key.Binding
	Help         key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Space        key.Binding
	Tab1         key.Binding
	Tab2         key.Binding
	Tab3         key.Binding
	Tab4         key.Binding
	Tab5         key.Binding
	Tab6         key.Binding
	Tab7         key.Binding
	Tab8         key.Binding
	Tab9         key.Binding
	Tab10        key.Binding
	Tab11        key.Binding
}

var DefaultKeyMap = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	QuitConfirm: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit (confirm)"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next view"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("S-tab", "prev view"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/↓", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Install: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "install"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Screenshot: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "screenshot"),
	),
	Logcat: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logcat"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	GotoTop: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "top"),
	),
	GotoBottom: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "bottom"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("^u", "½ page up"),
	),
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("^d", "½ page down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+b"),
		key.WithHelp("PgUp", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+f"),
		key.WithHelp("PgDn", "page down"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	Tab1: key.NewBinding(
		key.WithKeys("1"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
	),
	Tab4: key.NewBinding(
		key.WithKeys("4"),
	),
	Tab5: key.NewBinding(
		key.WithKeys("5"),
	),
	Tab6: key.NewBinding(
		key.WithKeys("6"),
	),
	Tab7: key.NewBinding(
		key.WithKeys("7"),
	),
	Tab8: key.NewBinding(
		key.WithKeys("8"),
	),
	Tab9: key.NewBinding(
		key.WithKeys("9"),
	),
	Tab10: key.NewBinding(
		key.WithKeys("0"),
	),
	Tab11: key.NewBinding(
		key.WithKeys("-"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Tab, k.Up, k.Down, k.Enter, k.Back, k.Help}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Tab, k.ShiftTab, k.Help},
		{k.Up, k.Down, k.GotoTop, k.GotoBottom},
		{k.HalfPageUp, k.HalfPageDown, k.PageUp, k.PageDown},
		{k.Enter, k.Back, k.Refresh},
		{k.Search, k.Filter, k.Delete, k.Install},
	}
}
