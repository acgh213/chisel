package tui

import "github.com/charmbracelet/lipgloss"

// Color variables — reassigned by ApplyTheme(). Default values are the peach theme.
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

// Style variables — bare declarations. rebuildStyles() is the single source of
// truth for all initialization; it is called from init() at package load and
// again on every Ctrl+T theme change.
var (
	BinderStyle        lipgloss.Style
	EditorStyle        lipgloss.Style
	StatusBarStyle     lipgloss.Style
	HistoryStyle       lipgloss.Style
	HistoryHeaderStyle lipgloss.Style
	DiffAddStyle       lipgloss.Style
	DiffDelStyle       lipgloss.Style
	DiffMetaStyle      lipgloss.Style
	ViewHeaderStyle    lipgloss.Style
	CardStyle          lipgloss.Style
	CardSelectedStyle  lipgloss.Style
	CardTitleStyle     lipgloss.Style
	CardMetaStyle      lipgloss.Style
	CardSynopsisStyle  lipgloss.Style
	MetTargetStyle     lipgloss.Style

	RightPanelStyle       lipgloss.Style
	RightPanelHeaderStyle lipgloss.Style
	RightPanelRoleStyle   lipgloss.Style
	RightPanelFieldStyle  lipgloss.Style
	RightPanelHintStyle   lipgloss.Style
	RightPanelDivStyle    lipgloss.Style

	PromptBarStyle    lipgloss.Style
	TreeSelectedStyle lipgloss.Style
	TreeFolderStyle   lipgloss.Style
	TreeFileStyle     lipgloss.Style
)

func init() { rebuildStyles() }

// FocusedStyle returns base with its border swapped to the accent color.
// It reads ColorAccent at call time so it always reflects the current theme.
func FocusedStyle(base lipgloss.Style) lipgloss.Style {
	return base.BorderForeground(ColorAccent)
}

// themeTokens holds the nine color slots that define a chisel theme.
type themeTokens struct {
	bg, fg, accent, muted, border, highlight, dim, green, red lipgloss.Color
}

var themes = map[string]themeTokens{
	"peach": {
		bg:        "#1a1a2e",
		fg:        "#e8d5c4",
		accent:    "#c4a882",
		muted:     "#8a7e72",
		border:    "#3a3a4e",
		highlight: "#2a2a3e",
		dim:       "#5a5a6e",
		green:     "#8ab882",
		red:       "#c48882",
	},
	"forest": {
		bg:        "#181e16",
		fg:        "#d4e8c0",
		accent:    "#88c070",
		muted:     "#789060",
		border:    "#2e3e2a",
		highlight: "#202e1c",
		dim:       "#4a5a44",
		green:     "#a0d880",
		red:       "#d08080",
	},
	"ocean": {
		bg:        "#0e1c2a",
		fg:        "#c4d8e8",
		accent:    "#70b0e0",
		muted:     "#6080a0",
		border:    "#1e3040",
		highlight: "#142030",
		dim:       "#3a5070",
		green:     "#70c0a0",
		red:       "#d07080",
	},
	"midnight": {
		bg:        "#0c0c1a",
		fg:        "#d0d0e8",
		accent:    "#9090d0",
		muted:     "#606080",
		border:    "#1a1a2e",
		highlight: "#101020",
		dim:       "#383858",
		green:     "#7098a8",
		red:       "#c07080",
	},
}

var themeOrder = []string{"peach", "forest", "ocean", "midnight"}

// ApplyTheme reassigns the Color* globals from the named theme token set and
// rebuilds all Style* vars. Unknown names fall back to peach.
func ApplyTheme(name string) {
	t, ok := themes[name]
	if !ok {
		t = themes["peach"]
	}
	ColorBg = t.bg
	ColorFg = t.fg
	ColorAccent = t.accent
	ColorMuted = t.muted
	ColorBorder = t.border
	ColorHighlight = t.highlight
	ColorDim = t.dim
	ColorGreen = t.green
	ColorRed = t.red
	rebuildStyles()
}

// NextTheme returns the theme name that follows current in the cycle order.
func NextTheme(current string) string {
	for i, name := range themeOrder {
		if name == current {
			return themeOrder[(i+1)%len(themeOrder)]
		}
	}
	return themeOrder[0]
}

// rebuildStyles is the single source of truth for all Style* initialization.
// It must be called after any Color* reassignment (theme change or init).
// CardSelectedStyle is assigned after CardStyle — order inside this function matters.
func rebuildStyles() {
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

	HistoryStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorAccent).
		Padding(0, 1)

	HistoryHeaderStyle = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Bold(true)

	DiffAddStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	DiffDelStyle = lipgloss.NewStyle().Foreground(ColorRed)
	DiffMetaStyle = lipgloss.NewStyle().Foreground(ColorDim)

	ViewHeaderStyle = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Bold(true)

	CardStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	CardSelectedStyle = CardStyle.BorderForeground(ColorAccent)

	CardTitleStyle = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	CardMetaStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	CardSynopsisStyle = lipgloss.NewStyle().Foreground(ColorFg)

	MetTargetStyle = lipgloss.NewStyle().Foreground(ColorGreen)

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

	PromptBarStyle = lipgloss.NewStyle().
		Background(ColorHighlight).
		Foreground(ColorAccent).
		Padding(0, 1)

	TreeSelectedStyle = lipgloss.NewStyle().
		Background(ColorHighlight).
		Foreground(ColorAccent).
		Bold(true)

	TreeFolderStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	TreeFileStyle = lipgloss.NewStyle().
		Foreground(ColorFg)
}
