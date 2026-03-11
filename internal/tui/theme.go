package tui

import "github.com/charmbracelet/lipgloss"

// Theme defines the color palette for the entire UI.
type Theme struct {
	Name      string
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Text      lipgloss.Color
	Dim       lipgloss.Color
	Border    lipgloss.Color
	Highlight lipgloss.Color
	Surface   lipgloss.Color

	// LogDebug and LogFatal allow per-theme log level colors.
	LogDebug lipgloss.Color
	LogFatal lipgloss.Color

	// SearchHighlightFg/Bg for search match rendering.
	SearchHighlightFg lipgloss.Color
	SearchHighlightBg lipgloss.Color
}

// Built-in themes.

// ThemeDefault — purple/cyan dark theme inspired by Tailwind.
var ThemeDefault = Theme{
	Name:              "default",
	Primary:           lipgloss.Color("#7C3AED"),
	Secondary:         lipgloss.Color("#6B7280"),
	Accent:            lipgloss.Color("#06B6D4"),
	Success:           lipgloss.Color("#10B981"),
	Warning:           lipgloss.Color("#F59E0B"),
	Error:             lipgloss.Color("#EF4444"),
	Text:              lipgloss.Color("#E5E7EB"),
	Dim:               lipgloss.Color("#6B7280"),
	Border:            lipgloss.Color("#374151"),
	Highlight:         lipgloss.Color("#8B5CF6"),
	Surface:           lipgloss.Color("#111827"),
	LogDebug:          lipgloss.Color("#3B82F6"),
	LogFatal:          lipgloss.Color("#D946EF"),
	SearchHighlightFg: lipgloss.Color("#1F2937"),
	SearchHighlightBg: lipgloss.Color("#FBBF24"),
}

// ThemeNord — clean arctic palette from nordtheme.com.
var ThemeNord = Theme{
	Name:              "nord",
	Primary:           lipgloss.Color("#88C0D0"),
	Secondary:         lipgloss.Color("#4C566A"),
	Accent:            lipgloss.Color("#81A1C1"),
	Success:           lipgloss.Color("#A3BE8C"),
	Warning:           lipgloss.Color("#EBCB8B"),
	Error:             lipgloss.Color("#BF616A"),
	Text:              lipgloss.Color("#ECEFF4"),
	Dim:               lipgloss.Color("#4C566A"),
	Border:            lipgloss.Color("#3B4252"),
	Highlight:         lipgloss.Color("#5E81AC"),
	Surface:           lipgloss.Color("#2E3440"),
	LogDebug:          lipgloss.Color("#81A1C1"),
	LogFatal:          lipgloss.Color("#B48EAD"),
	SearchHighlightFg: lipgloss.Color("#2E3440"),
	SearchHighlightBg: lipgloss.Color("#EBCB8B"),
}

// ThemeTokyoNight — warm dark theme inspired by tokyo-night.
var ThemeTokyoNight = Theme{
	Name:              "tokyonight",
	Primary:           lipgloss.Color("#7AA2F7"),
	Secondary:         lipgloss.Color("#565F89"),
	Accent:            lipgloss.Color("#7DCFFF"),
	Success:           lipgloss.Color("#9ECE6A"),
	Warning:           lipgloss.Color("#E0AF68"),
	Error:             lipgloss.Color("#F7768E"),
	Text:              lipgloss.Color("#C0CAF5"),
	Dim:               lipgloss.Color("#565F89"),
	Border:            lipgloss.Color("#3B4261"),
	Highlight:         lipgloss.Color("#BB9AF7"),
	Surface:           lipgloss.Color("#1A1B26"),
	LogDebug:          lipgloss.Color("#7AA2F7"),
	LogFatal:          lipgloss.Color("#BB9AF7"),
	SearchHighlightFg: lipgloss.Color("#1A1B26"),
	SearchHighlightBg: lipgloss.Color("#E0AF68"),
}

// ThemeCatppuccin — warm pastel dark theme (Mocha variant).
var ThemeCatppuccin = Theme{
	Name:              "catppuccin",
	Primary:           lipgloss.Color("#CBA6F7"),
	Secondary:         lipgloss.Color("#6C7086"),
	Accent:            lipgloss.Color("#89DCEB"),
	Success:           lipgloss.Color("#A6E3A1"),
	Warning:           lipgloss.Color("#F9E2AF"),
	Error:             lipgloss.Color("#F38BA8"),
	Text:              lipgloss.Color("#CDD6F4"),
	Dim:               lipgloss.Color("#6C7086"),
	Border:            lipgloss.Color("#45475A"),
	Highlight:         lipgloss.Color("#B4BEFE"),
	Surface:           lipgloss.Color("#1E1E2E"),
	LogDebug:          lipgloss.Color("#89B4FA"),
	LogFatal:          lipgloss.Color("#F5C2E7"),
	SearchHighlightFg: lipgloss.Color("#1E1E2E"),
	SearchHighlightBg: lipgloss.Color("#F9E2AF"),
}

// BuiltinThemes maps theme name to definition.
var BuiltinThemes = map[string]Theme{
	"default":    ThemeDefault,
	"nord":       ThemeNord,
	"tokyonight": ThemeTokyoNight,
	"catppuccin": ThemeCatppuccin,
}

// ThemeNames returns the sorted list of available theme names.
func ThemeNames() []string {
	return []string{"default", "nord", "tokyonight", "catppuccin"}
}

// ApplyTheme sets all global style variables from the given theme.
func ApplyTheme(t Theme) {
	ColorPrimary = t.Primary
	ColorSecondary = t.Secondary
	ColorAccent = t.Accent
	ColorSuccess = t.Success
	ColorWarning = t.Warning
	ColorError = t.Error
	ColorText = t.Text
	ColorDim = t.Dim
	ColorBorder = t.Border
	ColorHighlight = t.Highlight
	ColorSurface = t.Surface

	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	SubtitleStyle = lipgloss.NewStyle().Foreground(t.Secondary)
	StatusBarStyle = lipgloss.NewStyle().Background(t.Surface).Foreground(t.Text)
	HelpStyle = lipgloss.NewStyle().Foreground(t.Dim)
	ActiveTabStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	InactiveTabStyle = lipgloss.NewStyle().Foreground(t.Dim)
	SelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Highlight)
	NormalStyle = lipgloss.NewStyle().Foreground(t.Text)
	ErrorStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Error)
	SuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Success)
	DimStyle = lipgloss.NewStyle().Foreground(t.Dim)
	WarningStyle = lipgloss.NewStyle().Foreground(t.Warning)
	AccentStyle = lipgloss.NewStyle().Foreground(t.Accent)
	BorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Border)
	ActiveBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Primary)
	PaddedStyle = lipgloss.NewStyle().Padding(0, 1)
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Accent).PaddingLeft(2)
	TableHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Accent)
	TableCellStyle = lipgloss.NewStyle().Foreground(t.Text).Padding(0, 1)
	InputStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Primary).Padding(0, 1)
	DialogStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.Primary).Padding(1, 2)
	KeyHintKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Accent)
	KeyHintDescStyle = lipgloss.NewStyle().Foreground(t.Dim)
	ScrollPosStyle = lipgloss.NewStyle().Foreground(t.Dim)
	CursorStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	SearchHighlightStyle = lipgloss.NewStyle().Bold(true).Foreground(t.SearchHighlightFg).Background(t.SearchHighlightBg)

	// Update LogLevelStyle colors cache
	logDebugColor = t.LogDebug
	logFatalColor = t.LogFatal

	activeTheme = t
}

// activeTheme tracks the current theme for LogLevelStyle.
var activeTheme = ThemeDefault

// logDebugColor and logFatalColor are set by ApplyTheme for LogLevelStyle.
var (
	logDebugColor = ThemeDefault.LogDebug
	logFatalColor = ThemeDefault.LogFatal
)
