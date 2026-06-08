package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/acgh213/chisel/core"
)

// statsModel is a word-count bar chart view showing daily word totals parsed
// from git commit history.
type statsModel struct {
	days   []core.DayCount
	cursor int
	offset int
	width  int
	height int
}

func (s *statsModel) open(gb *core.GitBackend) error {
	counts, err := core.DailyWordCounts(gb)
	if err != nil {
		return err
	}
	s.days = counts
	s.cursor = 0
	s.offset = 0
	return nil
}

func (s statsModel) update(msg tea.KeyPressMsg) (statsModel, viewAction) {
	switch msg.String() {
	case "j", "down":
		if s.cursor < len(s.days)-1 {
			s.cursor++
		}
	case "k", "up":
		if s.cursor > 0 {
			s.cursor--
		}
	case "esc":
		return s, viewActionClose
	}
	s.scrollToCursor()
	return s, viewActionNone
}

func (s *statsModel) scrollToCursor() {
	rows := s.bodyRows()
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+rows {
		s.offset = s.cursor - rows + 1
	}
	if s.offset < 0 {
		s.offset = 0
	}
}

func (s statsModel) bodyRows() int {
	rows := s.height - 1
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (s *statsModel) SetSize(w, h int) {
	s.width = w
	s.height = h
}

func (s statsModel) stateTitle() string {
	return fmt.Sprintf("Word History — %d days", len(s.days))
}

func (s statsModel) view() string {
	header := ViewHeaderStyle.Render(truncate(s.stateTitle(), s.width))

	if len(s.days) == 0 {
		hint := lipgloss.NewStyle().Foreground(ColorDim).Render(
			"(no word-count history — save some scenes first)")
		return lipgloss.JoinVertical(lipgloss.Left, header, "", hint)
	}

	rows := s.bodyRows()
	lines := []string{header}

	for i := s.offset; i < len(s.days) && i < s.offset+rows; i++ {
		lines = append(lines, s.renderRow(i))
	}
	return strings.Join(lines, "\n")
}

func (s statsModel) renderRow(idx int) string {
	day := s.days[idx]
	selected := idx == s.cursor

	// Date column (10 chars)
	dateStr := day.Date.Format("2006-01-02")

	// Delta: compare to previous day (i+1 since days are sorted descending).
	delta := 0
	if idx+1 < len(s.days) {
		delta = day.Total - s.days[idx+1].Total
	}

	// Bar proportional to max total.
	maxTotal := 0
	for _, d := range s.days {
		if d.Total > maxTotal {
			maxTotal = d.Total
		}
	}

	// Available width for bar: total width minus date(10), total(10), delta(12), spaces.
	barW := s.width - 36
	if barW < 4 {
		barW = 4
	}

	barLen := 0
	if maxTotal > 0 {
		barLen = int(float64(day.Total) / float64(maxTotal) * float64(barW))
	}

	totalStr := fmt.Sprintf("%6d words", day.Total)
	deltaStr := fmt.Sprintf("%+d", delta)

	bar := strings.Repeat("█", barLen)
	barRemainder := strings.Repeat(" ", barW-barLen)

	barStyle := lipgloss.NewStyle().Foreground(ColorGreen)
	if delta < 0 {
		barStyle = lipgloss.NewStyle().Foreground(ColorRed)
	}

	line := fmt.Sprintf("%s  %s%s  %s  %s",
		dateStr, barStyle.Render(bar), barRemainder, totalStr,
		lipgloss.NewStyle().Foreground(ColorMuted).Render(deltaStr))

	if selected {
		return TreeSelectedStyle.Render(truncate(line, s.width))
	}
	return line
}
