package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// heatCell is one day on the heatmap grid.
type heatCell struct {
	date  string // "2006-01-02"
	count int    // commits on that day
}

// heatmapModel is the 52-week writing calendar heatmap (issue #36). Data comes
// from core.DailyActivity, built on the first open. A detail overlay shows the
// commit list for the selected day.
type heatmapModel struct {
	cells      [7][52]heatCell          // row=weekday (0=Mon), col=week (0=52w ago)
	byDay      map[string][]core.Revision // date → commits, for detail overlay
	cursorRow  int
	cursorCol  int

	// Detail overlay.
	detailMode  bool
	detailDay   string            // date key of the day being inspected
	detailList  []core.Revision   // commits for that day
	detailCursor int

	width  int
	height int
}

// Day and month names.
var dayHeaders = [7]string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
var monthNames = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// Heatmap color styles — 5 levels based on commit count.
var (
	heatNone = lipgloss.NewStyle().Foreground(ColorDim).Faint(true)
	heatLvl1 = lipgloss.NewStyle().Foreground(ColorDim)
	heatLvl2 = lipgloss.NewStyle().Foreground(ColorMuted)
	heatLvl3 = lipgloss.NewStyle().Foreground(ColorGreen)
	heatLvl4 = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)

	heatCursorStyle = lipgloss.NewStyle().Foreground(ColorHighlight).Reverse(true)
	heatDayStyle    = lipgloss.NewStyle().Foreground(ColorMuted).Width(2).Align(lipgloss.Right)
	heatMonthStyle  = lipgloss.NewStyle().Foreground(ColorMuted)
	heatDetailStyle = lipgloss.NewStyle().Foreground(ColorMuted)
)

// open builds the heatmap grid from git history.
func (h *heatmapModel) open(gb *core.GitBackend) error {
	byDay, err := core.DailyActivity(gb)
	if err != nil {
		return err
	}
	h.byDay = byDay

	// Build the 52-week grid. Today's column is the last one (col 51).
	today := coreFloorUTC(time.Now())
	monday := weekStart(today)
	startDate := monday.AddDate(0, 0, -51*7) // 52 weeks back from this Monday

	for col := 0; col < 52; col++ {
		for row := 0; row < 7; row++ {
			date := startDate.AddDate(0, 0, col*7+row)
			key := date.Format("2006-01-02")
			commits := byDay[key]
			h.cells[row][col] = heatCell{date: key, count: len(commits)}
		}
	}

	// Start cursor on today's cell.
	todayKey := today.Format("2006-01-02")
	h.cursorCol = 51 // today's week column
	h.cursorRow = int(today.Weekday()) - 1
	if h.cursorRow < 0 {
		h.cursorRow = 6 // Sunday
	}
	_ = todayKey
	h.detailMode = false
	return nil
}

// SetSize records the heatmap's outer dimensions.
func (h *heatmapModel) SetSize(w, _ int) {
	h.width = w
}

// selected returns the date key under the cursor, or "".
func (h heatmapModel) selected() string {
	if h.cursorCol < 0 || h.cursorCol >= 52 || h.cursorRow < 0 || h.cursorRow >= 7 {
		return ""
	}
	return h.cells[h.cursorRow][h.cursorCol].date
}

// update handles keys for the heatmap and its detail overlay.
func (h heatmapModel) update(msg tea.KeyMsg) (heatmapModel, viewAction) {
	if h.detailMode {
		return h.updateDetail(msg)
	}

	switch msg.String() {
	case "left", "h":
		if h.cursorCol > 0 {
			h.cursorCol--
		}
	case "right", "l":
		if h.cursorCol < 51 {
			h.cursorCol++
		}
	case "up", "k":
		if h.cursorRow > 0 {
			h.cursorRow--
		}
	case "down", "j":
		if h.cursorRow < 6 {
			h.cursorRow++
		}
	case "enter":
		key := h.selected()
		if key == "" {
			return h, viewActionNone
		}
		commits := h.byDay[key]
		if len(commits) == 0 {
			return h, viewActionNone
		}
		h.detailMode = true
		h.detailDay = key
		h.detailList = commits
		h.detailCursor = 0
	}
	return h, viewActionNone
}

// updateDetail handles keys while the day-detail overlay is active.
func (h heatmapModel) updateDetail(msg tea.KeyMsg) (heatmapModel, viewAction) {
	switch msg.String() {
	case "esc":
		h.detailMode = false
	case "up", "k":
		if h.detailCursor > 0 {
			h.detailCursor--
		}
	case "down", "j":
		if h.detailCursor < len(h.detailList)-1 {
			h.detailCursor++
		}
	case "enter":
		// In detail view, Enter closes back to the grid (scene opening
		// from commit messages requires path resolution — future work).
		h.detailMode = false
	}
	return h, viewActionNone
}

// view renders the heatmap grid or the detail overlay.
func (h heatmapModel) view() string {
	if h.detailMode {
		return h.viewDetail()
	}

	// Header.
	header := ViewHeaderStyle.Render(truncate(
		fmt.Sprintf("Writing Heatmap — last 52 weeks"), h.width))

	// Month labels row.
	monthRow := h.renderMonthRow()

	// Day grid.
	var gridRows []string
	for row := 0; row < 7; row++ {
		dayLabel := heatDayStyle.Render(dayHeaders[row])
		var cells []string
		for col := 0; col < 52; col++ {
			cell := h.cells[row][col]
			ch := h.cellChar(cell.count)
			styled := h.cellStyle(cell.count).Render(ch)
			if row == h.cursorRow && col == h.cursorCol {
				styled = heatCursorStyle.Render(ch)
			}
			cells = append(cells, styled)
		}
		gridRows = append(gridRows, dayLabel+strings.Join(cells, ""))
	}

	// Legend.
	legend := h.renderLegend()

	// Selected day info.
	sel := h.selected()
	info := ""
	if sel != "" && h.byDay[sel] != nil {
		count := len(h.byDay[sel])
		info = fmt.Sprintf("  %s — %d commit", sel, count)
		if count > 1 {
			info += "s"
		}
	}
	if info != "" {
		info = heatDetailStyle.Render(info)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header, "", monthRow, strings.Join(gridRows, "\n"), "", legend, info)
}

// viewDetail renders the day-detail overlay listing all commits.
func (h heatmapModel) viewDetail() string {
	t, _ := time.Parse("2006-01-02", h.detailDay)
	dateStr := t.Format("Mon, Jan 02, 2006")
	header := ViewHeaderStyle.Render(truncate(
		fmt.Sprintf("%s — %d commits", dateStr, len(h.detailList)), h.width))

	var lines []string
	for i, rev := range h.detailList {
		marker := "  "
		if i == h.detailCursor {
			marker = "▶ "
		}
		timeStr := rev.Timestamp.Format("15:04")
		msg := rev.Message
		line := fmt.Sprintf("%s%s  %s  %s", marker, timeStr, core.ShortHash(rev.Hash), msg)
		style := heatDetailStyle
		if i == h.detailCursor {
			style = lipgloss.NewStyle().Foreground(ColorGreen)
		}
		lines = append(lines, style.Render(truncate(line, h.width-4)))
	}

	footer := heatDetailStyle.Render("↑/↓ Navigate  Enter/Esc=Back to grid")

	return lipgloss.JoinVertical(lipgloss.Left,
		header, "", strings.Join(lines, "\n"), "", footer)
}

// renderMonthRow builds the month-label line above the grid.
func (h heatmapModel) renderMonthRow() string {
	// Determine which month each column falls in, emit labels at transitions.
	today := coreFloorUTC(time.Now())
	monday := weekStart(today)
	startDate := monday.AddDate(0, 0, -51*7)

	label := strings.Repeat(" ", 2) // indent for day labels
	lastMonth := -1
	for col := 0; col < 52; col++ {
		date := startDate.AddDate(0, 0, col*7) // Monday of each week
		month := int(date.Month()) - 1
		if month != lastMonth {
			label += monthNames[month]
			lastMonth = month
		} else {
			label += strings.Repeat(" ", len(monthNames[month]))
		}
	}
	return heatMonthStyle.Render(label)
}

// renderLegend draws the color scale below the grid.
func (h heatmapModel) renderLegend() string {
	items := []struct {
		label string
		style lipgloss.Style
	}{
		{"0", heatNone},
		{"1", heatLvl1},
		{"2-3", heatLvl2},
		{"4-5", heatLvl3},
		{"6+", heatLvl4},
	}
	var parts []string
	for _, it := range items {
		parts = append(parts, it.style.Render("█")+" "+it.label)
	}
	return heatDetailStyle.Render("  " + strings.Join(parts, "  "))
}

// cellChar returns the display character for a given commit count.
func (h heatmapModel) cellChar(count int) string {
	if count == 0 {
		return "·"
	}
	return "█"
}

// cellStyle returns the lipgloss style for a given commit count.
func (h heatmapModel) cellStyle(count int) lipgloss.Style {
	switch {
	case count == 0:
		return heatNone
	case count == 1:
		return heatLvl1
	case count <= 3:
		return heatLvl2
	case count <= 5:
		return heatLvl3
	default:
		return heatLvl4
	}
}

// weekStart returns the Monday of the week containing t.
func weekStart(t time.Time) time.Time {
	wd := t.Weekday()
	if wd == time.Sunday {
		return t.AddDate(0, 0, -6)
	}
	return t.AddDate(0, 0, -int(wd)+1)
}

// coreFloorUTC is a copy of core.floorUTC so the TUI can use it without
// exporting the internal helper.
func coreFloorUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
