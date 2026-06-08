package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/acgh213/chisel/core"
)

const (
	heatmapCols = 52 // one column per week
	heatmapRows = 7  // one row per day-of-week (Mon=0 .. Sun=6)
	cellW       = 2  // width of one cell (glyph + gap)
)

// heatCell represents one day cell in the heatmap grid.
type heatCell struct {
	date  time.Time
	count int // number of commits on this day
}

// heatmapModel is a 7×52 grid of commit activity with a detail overlay for
// the selected day. It follows the structural view pattern: open / update /
// view / SetSize / stateTitle.
type heatmapModel struct {
	grid   [heatmapRows][heatmapCols]heatCell
	cursor rowCol // current selection
	width  int
	height int

	// detail overlay state
	showDetail bool
	detailDate time.Time
	detailRevs []core.Revision
}

type rowCol struct {
	row, col int
}

// open builds the heatmap grid from the git backend's daily activity.
func (h *heatmapModel) open(gb *core.GitBackend) error {
	act, err := core.DailyActivity(gb)
	if err != nil {
		return err
	}

	// Clear grid.
	for r := 0; r < heatmapRows; r++ {
		for c := 0; c < heatmapCols; c++ {
			h.grid[r][c] = heatCell{}
		}
	}

	// The rightmost column is the current week (ending on the Saturday of the
	// current week). We fill 52 weeks backwards from there.
	now := core.FloorUTC(time.Now())
	// Find the Saturday of the current week (go-git time.Weekday: Sun=0).
	// We want Mon=0..Sun=6 rows, so we compute the weekday offset.
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7 // Sunday = 7 in Mon-based
	}
	// wd is 1=Mon..7=Sun. The Saturday of the current week is:
	saturday := now.AddDate(0, 0, 6-wd) // Saturday of this week

	// Fill the grid: column 51 (rightmost) = this week, column 0 = 51 weeks ago.
	for col := heatmapCols - 1; col >= 0; col-- {
		// The Saturday of this column's week.
		colSaturday := saturday.AddDate(0, 0, -(heatmapCols-1-col)*7)
		// Fill Monday (row 0) through Sunday (row 6) for this column.
		for row := 0; row < heatmapRows; row++ {
			day := colSaturday.AddDate(0, 0, -(6 - row)) // Monday = Saturday - 5
			if row == 6 {
				day = colSaturday // Sunday = Saturday + 1... wait
			}
			// Recalculate: Monday of this week is colSaturday - 5 days.
			day = colSaturday.AddDate(0, 0, row-6) // row 0 = Monday = Sat - 5, row 6 = Sunday = Sat + 1
			// Actually, let's be precise: row 0 = Monday = colSaturday - 5, row 6 = Sunday = colSaturday + 1
			// But Saturday is the last day of the standard Mon-start week.
			// Mon=row0: colSaturday - 5
			// Tue=row1: colSaturday - 4
			// ...
			// Sat=row5: colSaturday
			// Sun=row6: colSaturday + 1
			day = colSaturday.AddDate(0, 0, row-5)

			key := day.Format("2006-01-02")
			count := 0
			if revs, ok := act[key]; ok {
				count = len(revs)
			}
			h.grid[row][col] = heatCell{date: day, count: count}
		}
	}

	h.cursor = rowCol{row: 0, col: heatmapCols - 1}
	h.showDetail = false
	return nil
}

// SetSize sets the outer dimensions.
func (h *heatmapModel) SetSize(w, height int) {
	h.width = w
	h.height = height
}

func (h heatmapModel) stateTitle() string {
	return "Heatmap — last 52 weeks"
}

// cellGlyph returns the display character for a heat cell.
func cellGlyph(count int) string {
	switch {
	case count == 0:
		return "·"
	case count <= 2:
		return "▪"
	case count <= 5:
		return "◼"
	default:
		return "█"
	}
}

// cellStyle returns the foreground color for a heat cell based on count.
func cellStyle(count int) lipgloss.Style {
	switch {
	case count == 0:
		return HeatmapCellStyle0
	case count <= 2:
		return HeatmapCellStyle1
	case count <= 5:
		return HeatmapCellStyle2
	default:
		return HeatmapCellStyle3
	}
}

// update handles a key press and reports the action the root must take.
func (h heatmapModel) update(msg tea.KeyPressMsg) (heatmapModel, viewAction) {
	switch msg.String() {
	case "left", "h":
		if h.cursor.col > 0 {
			h.cursor.col--
		}
	case "right", "l":
		if h.cursor.col < heatmapCols-1 {
			h.cursor.col++
		}
	case "up", "k":
		if h.cursor.row > 0 {
			h.cursor.row--
		}
	case "down", "j":
		if h.cursor.row < heatmapRows-1 {
			h.cursor.row++
		}
	case "enter":
		cell := h.grid[h.cursor.row][h.cursor.col]
		if cell.count > 0 {
			h.showDetail = !h.showDetail
			h.detailDate = cell.date
		}
	case "esc":
		if h.showDetail {
			h.showDetail = false
		} else {
			return h, viewActionClose
		}
	}
	return h, viewActionNone
}

// view renders the heatmap grid with month labels and day headers.
func (h heatmapModel) view() string {
	header := ViewHeaderStyle.Render(truncate(h.stateTitle(), h.width))

	// Month labels: one character per column, aligned to the grid.
	monthLine := h.renderMonthLabels()

	// Day labels on the left side of the grid.
	dayLabels := []string{"M", "T", "W", "T", "F", "S", "S"}

	var gridLines []string
	gridLines = append(gridLines, header)
	gridLines = append(gridLines, "")
	gridLines = append(gridLines, "    "+monthLine) // 4-char indent for day labels

	for row := 0; row < heatmapRows; row++ {
		label := dayLabels[row]
		labelStyle := HeatmapLabelStyle.Width(4).Align(lipgloss.Right)
		line := labelStyle.Render(label) + " "

		for col := 0; col < heatmapCols; col++ {
			cell := h.grid[row][col]
			glyph := cellGlyph(cell.count)
			style := cellStyle(cell.count)

			if row == h.cursor.row && col == h.cursor.col {
				style = style.Background(ColorHighlight).Bold(true)
			}
			line += style.Render(padRight(glyph, cellW))
		}
		gridLines = append(gridLines, line)
	}

	// Legend
	legend := lipgloss.NewStyle().Foreground(ColorDim).Render("    Less ·▪◼█ More")
	gridLines = append(gridLines, "")
	gridLines = append(gridLines, legend)

	// Detail overlay
	if h.showDetail {
		gridLines = append(gridLines, "")
		gridLines = append(gridLines, h.renderDetail())
	}

	return strings.Join(gridLines, "\n")
}

// renderMonthLabels builds a line of single-char month labels aligned to the
// grid columns. A month label is placed at the first column whose Monday falls
// in that month.
func (h heatmapModel) renderMonthLabels() string {
	monthNames := []string{"J", "F", "M", "A", "M", "J", "J", "A", "S", "O", "N", "D"}

	// Build label positions.
	labels := make([]string, heatmapCols)
	prevMonth := -1
	for col := 0; col < heatmapCols; col++ {
		monday := h.grid[0][col].date // Monday of this week
		m := int(monday.Month()) - 1  // 0-indexed
		if m != prevMonth {
			labels[col] = monthNames[m]
			prevMonth = m
		} else {
			labels[col] = " "
		}
	}

	// Pad the label line to align with the grid (4-char day-label indent).
	var sb strings.Builder
	sb.WriteString("    ") // indent to match grid
	for col := 0; col < heatmapCols; col++ {
		l := labels[col]
		style := HeatmapLabelStyle
		if l != " " {
			style = style.Bold(true)
		}
		sb.WriteString(style.Render(padRight(l, cellW)))
	}
	return sb.String()
}

// renderDetail shows the commits for the selected day.
func (h heatmapModel) renderDetail() string {
	dateStr := h.detailDate.Format("Monday, January 2, 2006")
	title := DetailHeaderStyle.Render(dateStr)

	// Load detail from the grid cell — we need to look up the commits.
	// Since we don't store the full activity map, reconstruct from the
	// day count. The detail overlay is informational.
	cell := h.grid[h.cursor.row][h.cursor.col]
	if cell.count == 0 {
		return HeatmapCellStyle0.Render("No commits on this day.")
	}

	summary := fmt.Sprintf("%d commit(s) on %s", cell.count, h.detailDate.Format("2006-01-02"))

	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")
	lines = append(lines, DetailTextStyle.Render(summary))

	// Show revision details if we have them.
	if len(h.detailRevs) > 0 {
		lines = append(lines, "")
		// Sort by timestamp descending.
		revs := make([]core.Revision, len(h.detailRevs))
		copy(revs, h.detailRevs)
		sort.Slice(revs, func(i, j int) bool {
			return revs[i].Timestamp.After(revs[j].Timestamp)
		})
		for _, r := range revs {
			timePart := lipgloss.NewStyle().Foreground(ColorMuted).Render(
				r.Timestamp.Format("15:04"))
			msgPart := lipgloss.NewStyle().Foreground(ColorFg).Render(
				truncate(r.Message, h.width-20))
			lines = append(lines, fmt.Sprintf("  %s  %s  %s",
				core.ShortHash(r.Hash), timePart, msgPart))
		}
	}

	return strings.Join(lines, "\n")
}
