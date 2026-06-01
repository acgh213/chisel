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

// Corkboard / outliner styles (Phase 4).
var (
	// ViewHeaderStyle is the title row shared by the structural views.
	ViewHeaderStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Bold(true)

	// CardStyle is one index card on the corkboard. The selected card swaps its
	// border to the accent color via CardSelectedStyle.
	CardStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	CardSelectedStyle = CardStyle.BorderForeground(ColorAccent)

	CardTitleStyle    = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	CardMetaStyle     = lipgloss.NewStyle().Foreground(ColorMuted)
	CardSynopsisStyle = lipgloss.NewStyle().Foreground(ColorFg)

	// MetTargetStyle marks a word count that has reached its target.
	MetTargetStyle = lipgloss.NewStyle().Foreground(ColorGreen)
)

// Right panel styles (Phase 8).
var (
	RightPanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	RightPanelHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	RightPanelRoleStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	RightPanelFieldStyle = lipgloss.NewStyle().
				Foreground(ColorFg)

	RightPanelHintStyle = lipgloss.NewStyle().
				Foreground(ColorDim).
				Italic(true)

	RightPanelDivStyle = lipgloss.NewStyle().
				Foreground(ColorBorder)
)

// PromptBarStyle is the inline text-input bar that replaces the status bar
// during binder CRUD operations (new file/folder, rename, delete confirm).
var PromptBarStyle = lipgloss.NewStyle().
	Background(ColorHighlight).
	Foreground(ColorAccent).
	Padding(0, 1)

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
)
