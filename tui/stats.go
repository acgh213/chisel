package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// statsModel is the word-count history bar chart (issue #21). Each row shows
// one day: date, a proportional bar of daily words written, and the total
// project word count at the end of that day.
type statsModel struct {
	counts []core.DayCount

	cursor int // selected row index
	offset int // first visible row

	width  int
	height int
}

// Bar chart constants.
const (
	statsBarW     = 20 // max bar width in characters
	statsDateW    = 6  // "Mon DD"
	statsWordsW   = 7  // "12,345"
	statsDeltaW   = 8  // "(+1,234)"
)

// Stats view styles.
var (
	statsBarFill = lipgloss.NewStyle().Foreground(ColorGreen)
	statsBarRest = lipgloss.NewStyle().Foreground(ColorDim).Faint(true)
	statsDateFmt = lipgloss.NewStyle().Foreground(ColorMuted).Width(statsDateW)
	statsWordFmt = lipgloss.NewStyle().Foreground(ColorGreen).Width(statsWordsW).Align(lipgloss.Right)
	statsDeltaFmt = lipgloss.NewStyle().Foreground(ColorDim).Width(statsDeltaW).Align(lipgloss.Right)
	statsZeroFmt  = lipgloss.NewStyle().Foreground(ColorDim).Faint(true).Width(statsWordsW).Align(lipgloss.Right)
	statsCursor   = lipgloss.NewStyle().Foreground(ColorHighlight).Reverse(true)
	statsSumFmt   = lipgloss.NewStyle().Foreground(ColorMuted)
)

// open loads the word count history from git.
func (s *statsModel) open(gb *core.GitBackend) error {
	counts, err := core.DailyWordCounts(gb)
	if err != nil {
		return err
	}
	s.counts = counts
	s.cursor = 0
	s.offset = 0
	return nil
}

// SetSize records the view dimensions.
func (s *statsModel) SetSize(w, h int) {
	s.width = w
	s.height = h
}

// selected returns the date key for the cursor row.
func (s statsModel) selected() string {
	idx := s.offset + s.cursor
	if idx < 0 || idx >= len(s.counts) {
		return ""
	}
	return s.counts[idx].Date.Format("2006-01-02")
}

// update handles navigation keys.
func (s statsModel) update(msg tea.KeyMsg) (statsModel, viewAction) {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.offset+s.cursor < len(s.counts)-1 {
			s.cursor++
		}
	}
	s.scrollToCursor()
	return s, viewActionNone
}

// scrollToCursor keeps the cursor row within the visible window.
func (s *statsModel) scrollToCursor() {
	visible := s.visibleRows()
	if visible < 1 {
		visible = 1
	}
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+visible {
		s.offset = s.cursor - visible + 1
	}
	if s.offset < 0 {
		s.offset = 0
	}
}

// visibleRows returns how many data rows fit (accounting for header + footer).
func (s statsModel) visibleRows() int {
	rows := s.height - 4 // header + footer + spacing
	if rows < 1 {
		rows = 1
	}
	return rows
}

// view renders the stats chart.
func (s statsModel) view() string {
	header := ViewHeaderStyle.Render(truncate(
		"Project Statistics — Word Count History", s.width))

	if len(s.counts) == 0 {
		empty := lipgloss.NewStyle().Foreground(ColorDim).
			Render("(no git history yet — start writing and saving to see stats)")
		return lipgloss.JoinVertical(lipgloss.Left, header, "", empty)
	}

	// Render visible rows.
	visible := s.visibleRows()
	end := s.offset + visible
	if end > len(s.counts) {
		end = len(s.counts)
	}

	var rows []string
	// Find the max delta for scaling the bar.
	maxDelta := 0
	for _, dc := range s.counts {
		if dc.Delta > maxDelta {
			maxDelta = dc.Delta
		}
	}

	for i := s.offset; i < end; i++ {
		dc := s.counts[i]
		row := s.renderRow(dc, i, maxDelta)
		if i-s.offset == s.cursor-s.offset {
			row = statsCursor.Render(row)
		}
		rows = append(rows, row)
	}

	// Summary footer.
	footer := s.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left,
		header, "", strings.Join(rows, "\n"), "", footer)
}

// renderRow builds one day's row: date, bar, total words, delta.
func (s statsModel) renderRow(dc core.DayCount, idx, maxDelta int) string {
	dateStr := dc.Date.Format("Jan 02")
	date := statsDateFmt.Render(dateStr)

	// Bar: █ for filled portion, ░ for rest.
	var bar string
	if maxDelta > 0 && dc.Delta > 0 {
		filled := (dc.Delta * statsBarW) / maxDelta
		if filled < 1 {
			filled = 1
		}
		bar = statsBarFill.Render(strings.Repeat("█", filled))
		rest := statsBarW - filled
		if rest > 0 {
			bar += statsBarRest.Render(strings.Repeat("░", rest))
		}
	} else {
		bar = statsBarRest.Render(strings.Repeat("░", statsBarW))
	}

	// Word counts.
	var words, delta string
	if dc.Delta > 0 {
		words = statsWordFmt.Render(formatNum(dc.Words))
		delta = statsDeltaFmt.Render(fmt.Sprintf("(+%s)", formatNum(dc.Delta)))
	} else {
		words = statsZeroFmt.Render(formatNum(dc.Words))
		delta = statsDeltaFmt.Render("")
	}

	return date + " " + bar + " " + words + " " + delta
}

// renderFooter builds the summary line with averages and best day.
func (s statsModel) renderFooter() string {
	if len(s.counts) == 0 {
		return ""
	}

	// Compute stats.
	totalDelta := 0
	bestDelta := 0
	bestDate := ""
	activeDays := 0
	for _, dc := range s.counts {
		totalDelta += dc.Delta
		if dc.Delta > bestDelta {
			bestDelta = dc.Delta
			bestDate = dc.Date.Format("Jan 02")
		}
		if dc.Delta > 0 {
			activeDays++
		}
	}
	avg := 0
	if activeDays > 0 {
		avg = totalDelta / activeDays
	}

	totalWords := 0
	if len(s.counts) > 0 {
		totalWords = s.counts[len(s.counts)-1].Words
	}

	parts := []string{
		fmt.Sprintf("Total: %s words", formatNum(totalWords)),
		fmt.Sprintf("Avg: %s/day", formatNum(avg)),
	}
	if bestDate != "" {
		parts = append(parts, fmt.Sprintf("Best: %s (+%s)", bestDate, formatNum(bestDelta)))
	}
	return statsSumFmt.Render(strings.Join(parts, "  │  "))
}

// formatNum formats an integer with comma separators (e.g., 1234567 → "1,234,567").
func formatNum(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var parts []string
	for n > 0 {
		parts = append([]string{fmt.Sprintf("%d", n%1000)}, parts...)
		n /= 1000
	}
	// Fix leading zeros in intermediate groups.
	for i := 0; i < len(parts)-1; i++ {
		parts[i] = fmt.Sprintf("%03s", parts[i])
	}
	// Strip leading zeros from the first group.
	parts[0] = strings.TrimLeft(parts[0], "0")
	if parts[0] == "" {
		parts[0] = "0"
	}
	return sign + strings.Join(parts, ",")
}
