package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type helpModel struct {
	isActive bool
}

func (h *helpModel) open() {
	h.isActive = true
}

func (h *helpModel) close() {
	h.isActive = false
}

func (h helpModel) active() bool {
	return h.isActive
}

func (h helpModel) update(msg tea.KeyPressMsg) helpModel {
	switch msg.String() {
	case "esc", "?":
		h.close()
	}
	return h
}

func (h helpModel) view(w, height int, bg string) string {
	innerW := 72
	if maxW := w - HelpStyle.GetHorizontalFrameSize(); maxW > 20 && innerW > maxW {
		innerW = maxW
	}

	lines := []string{
		HelpHeaderStyle.Render("Chisel Help"),
		"",
		HelpSectionStyle.Render("Global"),
		helpLine("? help", "F2 corkboard", "F3 outliner", "F4 timeline", "F5 panel", "F6 read", "F7 sprint", "F8 theme", "^F search"),
		"",
		HelpSectionStyle.Render("Binder"),
		helpLine("Tab editor", "Enter open", "Space toggle", "n new scene", "N new folder", "r rename", "d delete"),
		"",
		HelpSectionStyle.Render("Editor"),
		helpLine("Tab binder", "^S save", "^N new scene", "^E export", "^H history"),
		"",
		HelpSectionStyle.Render("Structural Views"),
		helpLine("arrows navigate", "Enter open", "F2/F3/F4 hop", "Esc back"),
		"",
		HelpSectionStyle.Render("Overlays"),
		helpLine("` quick note", "Esc close/cancel", "Enter confirm/open"),
	}

	inner := strings.Join(lines, "\n")
	popup := HelpStyle.Width(innerW).Render(inner)
	popupH := lipgloss.Height(popup)
	centeredStrip := lipgloss.Place(w, popupH, lipgloss.Center, lipgloss.Center, popup)
	stripLines := strings.Split(centeredStrip, "\n")

	bgLines := strings.Split(bg, "\n")
	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", w))
	}

	startRow := (height - popupH) / 2
	if startRow < 0 {
		startRow = 0
	}

	result := make([]string, height)
	copy(result, bgLines[:height])
	for i, line := range stripLines {
		if startRow+i < height {
			result[startRow+i] = line
		}
	}
	return strings.Join(result, "\n")
}

func helpLine(items ...string) string {
	var parts []string
	for _, item := range items {
		key, rest, ok := strings.Cut(item, " ")
		if !ok {
			parts = append(parts, HelpKeyStyle.Render(item))
			continue
		}
		parts = append(parts, HelpKeyStyle.Render(key)+" "+rest)
	}
	return strings.Join(parts, "  ")
}
