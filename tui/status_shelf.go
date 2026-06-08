package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const bottomShelfRows = 2

func (m Model) renderBottomShelf() string {
	if m.prompt.active() {
		return lipgloss.JoinVertical(lipgloss.Left,
			m.prompt.view(m.width),
			renderShelfLine(HintBarStyle, m.width, promptHint(m.prompt.mode)),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		renderShelfLine(StatusBarStyle, m.width, m.stateRowText()),
		renderShelfLine(HintBarStyle, m.width, m.hintRowText()),
	)
}

func renderShelfLine(style lipgloss.Style, width int, text string) string {
	contentW := width - style.GetHorizontalFrameSize()
	if contentW < 1 {
		contentW = 1
	}
	return style.Width(width).MaxHeight(1).Render(truncate(text, contentW))
}

func (m Model) stateRowText() string {
	var parts []string
	if m.statusMsg != "" {
		parts = append(parts, m.statusMsg)
	}
	if ctx := m.stateContextText(); ctx != "" {
		parts = append(parts, ctx)
	}
	if m.sprintActive {
		remaining := time.Until(m.sprintEnd)
		gained := m.sessionWords - m.sprintWordStart
		parts = append(parts, fmt.Sprintf("Sprint %s +%d", formatDuration(remaining), gained))
	} else if m.sessionWords > 0 {
		if m.config.DailyGoal > 0 {
			parts = append(parts, fmt.Sprintf("Today +%d/%d", m.sessionWords, m.config.DailyGoal))
		} else {
			parts = append(parts, fmt.Sprintf("Today +%d", m.sessionWords))
		}
	}
	if m.streak.Current > 0 {
		parts = append(parts, fmt.Sprintf("🔥 %d days", m.streak.Current))
	}
	if m.config.ProjectTarget > 0 && m.projectWords > 0 {
		pct := m.projectWords * 100 / m.config.ProjectTarget
		parts = append(parts, fmt.Sprintf("📊 %s/%s (%d%%)", formatNum(m.projectWords), formatNum(m.config.ProjectTarget), pct))
	}
	return strings.Join(parts, "  |  ")
}

func (m Model) stateContextText() string {
	switch {
	case m.showHistory:
		return m.history.stateTitle()
	case m.viewMode == viewCorkboard:
		return m.corkboard.stateTitle()
	case m.viewMode == viewOutliner:
		return m.outliner.stateTitle()
	case m.viewMode == viewTimeline:
		return m.timeline.stateTitle()
	case m.viewMode == viewHeatmap:
		return m.heatmap.stateTitle()
	case m.viewMode == viewStats:
		return m.stats.stateTitle()
	case m.editor.FilePath() != "":
		mod := ""
		if m.editor.IsModified() {
			mod = " *"
		}
		return fmt.Sprintf("%s - %d words%s", filepath.Base(m.editor.FilePath()), m.wordCount, mod)
	default:
		return ""
	}
}

func (m Model) hintRowText() string {
	if m.sprintActive {
		return "F7 stop sprint  ? help"
	}

	switch {
	case m.showHistory:
		if m.history.mode == historyDiff {
			return "History: up/down scroll  Esc back  r restore  ? help"
		}
		return "History: up/down navigate  Enter diff  r restore  Esc close  ? help"
	case m.viewMode == viewCorkboard:
		return "Corkboard: arrows navigate  Enter open  F3 outliner  F4 timeline  Esc back  ? help"
	case m.viewMode == viewOutliner:
		return "Outliner: up/down navigate  left/right collapse/expand  Enter open  F2 corkboard  F4 timeline  Esc back  ? help"
	case m.viewMode == viewTimeline:
		return "Timeline: up/down navigate  Enter open  F2 corkboard  F3 outliner  Esc back  ? help"
	case m.viewMode == viewHeatmap:
		return "Heatmap: arrows navigate  Enter detail  Esc back  ? help"
	case m.viewMode == viewStats:
		return "Stats: j/k navigate  Esc close  ? help"
	case m.focus == PaneBinder:
		return "Binder: Tab switch  n new  N folder  r rename  d delete  ? help"
	default:
		return "Editor: Tab switch  ^S save  ^N new  ^E export  ^Enter link"
	}
}

func promptHint(mode promptMode) string {
	if mode == promptDelete {
		return "y confirm  Esc cancel"
	}
	return "Enter confirm  Esc cancel"
}

// formatNum formats an integer with comma separators (e.g. 23412 → "23,412").
func formatNum(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	if s != "" {
		parts = append([]string{s}, parts...)
	}
	return strings.Join(parts, ",")
}
