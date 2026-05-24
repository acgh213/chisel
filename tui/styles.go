package tui

import "github.com/charmbracelet/lipgloss"

// Theme colour tokens. Every component references these — no hardcoded hex
// values in component code. Theme switching means swapping the token values.
var (
	// Peach theme (default)
	ColorBg         = lipgloss.Color("#1a1a1a")
	ColorFg         = lipgloss.Color("#e8d5c4")
	ColorAccent     = lipgloss.Color("#e8935a")
	ColorMuted      = lipgloss.Color("#6b5e53")
	ColorHighlight  = lipgloss.Color("#2a2520")
	ColorBorder     = lipgloss.Color("#3a3530")
	ColorStatusDraft   = lipgloss.Color("#8a7a6a")
	ColorStatusRevised = lipgloss.Color("#c9a87c")
	ColorStatusDone    = lipgloss.Color("#7a9a6a")
	ColorError      = lipgloss.Color("#c95a5a")
	ColorSuccess    = lipgloss.Color("#7a9a6a")
)

// Common styles used across components.
var (
	StyleStatusBar = lipgloss.NewStyle().
			Background(ColorHighlight).
			Foreground(ColorMuted).
			Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorMuted)
)
