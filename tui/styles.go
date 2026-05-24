package tui

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// theme
// ---------------------------------------------------------------------------

// Theme holds all colour tokens for a named theme.
type Theme struct {
	Name             string
	Bg               lipgloss.Color
	Fg               lipgloss.Color
	Accent           lipgloss.Color
	Muted            lipgloss.Color
	Highlight        lipgloss.Color
	Border           lipgloss.Color
	StatusDraft      lipgloss.Color
	StatusRevised    lipgloss.Color
	StatusDone       lipgloss.Color
	Error            lipgloss.Color
	Success          lipgloss.Color
}

var themes = map[string]Theme{
	"peach": {
		Name:          "peach",
		Bg:            lipgloss.Color("#1a1a1a"),
		Fg:            lipgloss.Color("#e8d5c4"),
		Accent:        lipgloss.Color("#e8935a"),
		Muted:         lipgloss.Color("#6b5e53"),
		Highlight:     lipgloss.Color("#2a2520"),
		Border:        lipgloss.Color("#3a3530"),
		StatusDraft:   lipgloss.Color("#8a7a6a"),
		StatusRevised: lipgloss.Color("#c9a87c"),
		StatusDone:    lipgloss.Color("#7a9a6a"),
		Error:         lipgloss.Color("#c95a5a"),
		Success:       lipgloss.Color("#7a9a6a"),
	},
	"dark": {
		Name:          "dark",
		Bg:            lipgloss.Color("#0d0d0d"),
		Fg:            lipgloss.Color("#cccccc"),
		Accent:        lipgloss.Color("#5a8ec9"),
		Muted:         lipgloss.Color("#555555"),
		Highlight:     lipgloss.Color("#1a1a1a"),
		Border:        lipgloss.Color("#333333"),
		StatusDraft:   lipgloss.Color("#666666"),
		StatusRevised: lipgloss.Color("#8899aa"),
		StatusDone:    lipgloss.Color("#6a9a6a"),
		Error:         lipgloss.Color("#c95a5a"),
		Success:       lipgloss.Color("#6a9a6a"),
	},
	"light": {
		Name:          "light",
		Bg:            lipgloss.Color("#f5f0e8"),
		Fg:            lipgloss.Color("#2a2520"),
		Accent:        lipgloss.Color("#c95a2a"),
		Muted:         lipgloss.Color("#8a8578"),
		Highlight:     lipgloss.Color("#e8e0d0"),
		Border:        lipgloss.Color("#d0c8b8"),
		StatusDraft:   lipgloss.Color("#9a9588"),
		StatusRevised: lipgloss.Color("#b8904a"),
		StatusDone:    lipgloss.Color("#5a8a4a"),
		Error:         lipgloss.Color("#c94040"),
		Success:       lipgloss.Color("#5a8a4a"),
	},
	"forest": {
		Name:          "forest",
		Bg:            lipgloss.Color("#1a221a"),
		Fg:            lipgloss.Color("#c8d8c0"),
		Accent:        lipgloss.Color("#7aaa5a"),
		Muted:         lipgloss.Color("#5a6a50"),
		Highlight:     lipgloss.Color("#222a20"),
		Border:        lipgloss.Color("#3a4a30"),
		StatusDraft:   lipgloss.Color("#6a7a60"),
		StatusRevised: lipgloss.Color("#9aba70"),
		StatusDone:    lipgloss.Color("#6aaa60"),
		Error:         lipgloss.Color("#c95a5a"),
		Success:       lipgloss.Color("#6aaa60"),
	},
	"ocean": {
		Name:          "ocean",
		Bg:            lipgloss.Color("#0d1a2a"),
		Fg:            lipgloss.Color("#c0d8e8"),
		Accent:        lipgloss.Color("#5aaac9"),
		Muted:         lipgloss.Color("#4a6a7a"),
		Highlight:     lipgloss.Color("#14202e"),
		Border:        lipgloss.Color("#2a4a5a"),
		StatusDraft:   lipgloss.Color("#5a7a8a"),
		StatusRevised: lipgloss.Color("#7aaaC0"),
		StatusDone:    lipgloss.Color("#5aaa90"),
		Error:         lipgloss.Color("#c95a5a"),
		Success:       lipgloss.Color("#5aaa90"),
	},
}

// CurrentTheme is the active theme. Set via ApplyTheme.
var CurrentTheme = themes["peach"]

// Convenience colour variables that point at the current theme's tokens.
var (
	ColorBg             lipgloss.Color
	ColorFg             lipgloss.Color
	ColorAccent         lipgloss.Color
	ColorMuted          lipgloss.Color
	ColorHighlight      lipgloss.Color
	ColorBorder         lipgloss.Color
	ColorStatusDraft    lipgloss.Color
	ColorStatusRevised  lipgloss.Color
	ColorStatusDone     lipgloss.Color
	ColorError          lipgloss.Color
	ColorSuccess        lipgloss.Color
)

func init() {
	ApplyTheme("peach")
}

// ApplyTheme switches all colour tokens to the named theme. Unknown names
// fall back to peach.
func ApplyTheme(name string) {
	t, ok := themes[name]
	if !ok {
		t = themes["peach"]
	}
	CurrentTheme = t

	ColorBg = t.Bg
	ColorFg = t.Fg
	ColorAccent = t.Accent
	ColorMuted = t.Muted
	ColorHighlight = t.Highlight
	ColorBorder = t.Border
	ColorStatusDraft = t.StatusDraft
	ColorStatusRevised = t.StatusRevised
	ColorStatusDone = t.StatusDone
	ColorError = t.Error
	ColorSuccess = t.Success
}

// ThemeNames returns the list of available theme names.
func ThemeNames() []string {
	return []string{"peach", "dark", "light", "forest", "ocean"}
}

// Common styles used across components.
var (
	StyleStatusBar = lipgloss.NewStyle().
			Background(ColorHighlight).
			Foreground(ColorMuted).
			Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorMuted)
)
