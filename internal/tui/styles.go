package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// clearStatusMsg is sent after a delay to clear statusMsg in any view.
type clearStatusMsg struct{}

// downloadsDir returns the system Downloads directory, cross-platform.
// Falls back to home directory, then current directory.
func downloadsDir() string {
	var home string
	if h, err := os.UserHomeDir(); err == nil {
		home = h
	}

	switch runtime.GOOS {
	case "windows":
		// %USERPROFILE%\Downloads
		if home != "" {
			return filepath.Join(home, "Downloads")
		}
	case "darwin":
		if home != "" {
			return filepath.Join(home, "Downloads")
		}
	default: // linux, freebsd, etc.
		// Respect XDG if set
		if xdg := os.Getenv("XDG_DOWNLOAD_DIR"); xdg != "" {
			return xdg
		}
		if home != "" {
			return filepath.Join(home, "Downloads")
		}
	}

	// Last resort: current directory
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// clearStatusAfter returns a tea.Cmd that sends clearStatusMsg after d.
func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

var (
	ColorPrimary   = lipgloss.Color("#7C3AED")
	ColorSecondary = lipgloss.Color("#6B7280")
	ColorAccent    = lipgloss.Color("#06B6D4")
	ColorSuccess   = lipgloss.Color("#10B981")
	ColorWarning   = lipgloss.Color("#F59E0B")
	ColorError     = lipgloss.Color("#EF4444")
	ColorText      = lipgloss.Color("#E5E7EB")
	ColorDim       = lipgloss.Color("#6B7280")
	ColorBorder    = lipgloss.Color("#374151")
	ColorHighlight = lipgloss.Color("#8B5CF6")
	ColorSurface   = lipgloss.Color("#111827")
)

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorSurface).
			Foreground(ColorText)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorDim)

	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight)

	NormalStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	AccentStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	ActiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)

	PaddedStyle = lipgloss.NewStyle().
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			PaddingLeft(2)

	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	KeyHintKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	KeyHintDescStyle = lipgloss.NewStyle().
				Foreground(ColorDim)

	ScrollPosStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	CursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	SearchHighlightStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#1F2937")).
				Background(lipgloss.Color("#FBBF24"))
)

func DeviceStateStyle(state string) lipgloss.Style {
	switch state {
	case "device":
		return SuccessStyle
	case "offline":
		return ErrorStyle
	case "unauthorized":
		return WarningStyle
	default:
		return DimStyle
	}
}

func LogLevelStyle(level string) lipgloss.Style {
	switch level {
	case "V":
		return DimStyle
	case "D":
		return lipgloss.NewStyle().Foreground(logDebugColor)
	case "I":
		return lipgloss.NewStyle().Foreground(ColorSuccess)
	case "W":
		return lipgloss.NewStyle().Foreground(ColorWarning)
	case "E":
		return lipgloss.NewStyle().Foreground(ColorError)
	case "F":
		return lipgloss.NewStyle().Foreground(logFatalColor)
	default:
		return NormalStyle
	}
}

// safeViewHeight computes view height by subtracting overhead from total,
// with a sane minimum that never exceeds the actual terminal height.
func safeViewHeight(totalHeight, overhead, minRows int) int {
	h := totalHeight - overhead
	if h < minRows {
		h = minRows
	}
	// Never exceed total height to avoid rendering beyond terminal
	if totalHeight > 0 && h > totalHeight {
		h = totalHeight
	}
	return h
}

func keyHint(k, desc string) string {
	return KeyHintKeyStyle.Render(k) + KeyHintDescStyle.Render(":"+desc)
}

func helpBar(hints ...string) string {
	return "  " + DimStyle.Render("─") + " " + lipgloss.JoinHorizontal(lipgloss.Top, joinHints(hints)...)
}

func joinHints(hints []string) []string {
	result := make([]string, 0, len(hints)*2-1)
	for i, h := range hints {
		if i > 0 {
			result = append(result, DimStyle.Render("  "))
		}
		result = append(result, h)
	}
	return result
}

// highlightMatches highlights all case-insensitive occurrences of query in text.
func highlightMatches(text, query string) string {
	if query == "" {
		return text
	}
	lower := strings.ToLower(text)
	q := strings.ToLower(query)
	var b strings.Builder
	i := 0
	for {
		idx := strings.Index(lower[i:], q)
		if idx < 0 {
			b.WriteString(text[i:])
			break
		}
		b.WriteString(text[i : i+idx])
		b.WriteString(SearchHighlightStyle.Render(text[i+idx : i+idx+len(query)]))
		i += idx + len(query)
	}
	return b.String()
}

func renderBar(value, maxVal float64, width int) string {
	if maxVal <= 0 {
		maxVal = 100
	}
	pct := value / maxVal
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(pct * float64(width))
	empty := width - filled

	var style func(string) string
	if pct > 0.8 {
		style = func(s string) string { return ErrorStyle.Render(s) }
	} else if pct > 0.5 {
		style = func(s string) string { return WarningStyle.Render(s) }
	} else {
		style = func(s string) string { return SuccessStyle.Render(s) }
	}

	bar := style(strings.Repeat("█", filled)) + DimStyle.Render(strings.Repeat("░", empty))
	return fmt.Sprintf("[%s] %.1f%%", bar, value)
}

// fitHeight truncates or pads content to exactly n lines.
// Prevents terminal scrolling when a view renders too many lines.
func fitHeight(content string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	// Remove trailing empty line from Split if content ends with \n
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) > n {
		lines = lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n"
}

func scrollInfo(cursor, total int) string {
	if total == 0 {
		return ScrollPosStyle.Render("[empty]")
	}
	return ScrollPosStyle.Render(fmt.Sprintf("[%d/%d]", cursor+1, total))
}
