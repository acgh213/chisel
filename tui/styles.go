package tui

import "github.com/charmbracelet/lipgloss"

// Peach theme — the only theme in v0.1.
// Warm, dark, inviting. No theme engine.
var (
	ColorBg        = lipgloss.Color("#1a1a2e") // dark purple-blue
	ColorFg        = lipgloss.Color("#e8d5c4") // warm cream
	ColorAccent    = lipgloss.Color("#c4a882") // peach/gold
	ColorMuted     = lipgloss.Color("#8a7e72") // warm gray
	ColorBorder    = lipgloss.Color("#3a3a4e") // subtle border
	ColorHighlight = lipgloss.Color("#2a2a3e") // selected item bg
	ColorDim       = lipgloss.Color("#5a5a6e") // dimmed text
	ColorGreen     = lipgloss.Color("#8ab882") // word count, saved indicator
	ColorRed       = lipgloss.Color("#c48882") // modified indicator, errors
)

// Base app style — full terminal background.
var AppStyle = lipgloss.NewStyle().
	Background(ColorBg).
	Foreground(ColorFg)

// Pane styles.
var (
	BinderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	EditorStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorHighlight).
			Foreground(ColorMuted).
			Padding(0, 1)
)

// Focused pane gets accent border.
func FocusedStyle(base lipgloss.Style) lipgloss.Style {
	return base.BorderForeground(ColorAccent)
}

// History browser styles.
var (
	HistoryStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1)

	HistoryHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Bold(true)

	DiffAddStyle  = lipgloss.NewStyle().Foreground(ColorGreen)
	DiffDelStyle  = lipgloss.NewStyle().Foreground(ColorRed)
	DiffMetaStyle = lipgloss.NewStyle().Foreground(ColorDim)
)

// Tree node styles.
var (
	TreeSelectedStyle = lipgloss.NewStyle().
				Background(ColorHighlight).
				Foreground(ColorAccent).
				Bold(true)

	TreeFolderStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	TreeFileStyle = lipgloss.NewStyle().
			Foreground(ColorFg)

	TreeModifiedStyle = lipgloss.NewStyle().
				Foreground(ColorRed).
				Italic(true)
)
