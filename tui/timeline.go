package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// timelineModel is a full-width structural view showing all project scenes
// sorted by story-internal date (timeline_date frontmatter field). Dated scenes
// appear first in ascending order; undated scenes follow alphabetically. A
// divider row separates the two sections when both are present.
type timelineModel struct {
	root         string
	entries      []core.TimelineEntry
	firstUndated int // index of first undated entry; -1 if all are dated
	cursor       int
	offset       int
	width        int
	height       int
}

// open loads the timeline for root and resets the cursor.
func (t *timelineModel) open(root string) error {
	entries, err := core.BuildTimeline(root)
	if err != nil {
		return err
	}
	t.root = root
	t.entries = entries
	t.cursor = 0
	t.offset = 0
	t.firstUndated = -1
	for i, e := range entries {
		if e.TimelineDate == nil {
			t.firstUndated = i
			break
		}
	}
	return nil
}

// SetSize sets the outer dimensions.
func (t *timelineModel) SetSize(w, h int) {
	t.width = w
	t.height = h
}

// selected returns the path of the entry under the cursor, or "".
func (t timelineModel) selected() string {
	if t.cursor < 0 || t.cursor >= len(t.entries) {
		return ""
	}
	return t.entries[t.cursor].Path
}

// update handles a key press and reports the action the root model must take.
// F1/Esc and F2/F3/F4 are caught by updateView before reaching here, so this
// only needs to handle navigation and Enter.
func (t timelineModel) update(msg tea.KeyMsg) (timelineModel, viewAction) {
	switch msg.String() {
	case "j", "down":
		if t.cursor < len(t.entries)-1 {
			t.cursor++
		}
	case "k", "up":
		if t.cursor > 0 {
			t.cursor--
		}
	case "enter":
		if t.cursor >= 0 && t.cursor < len(t.entries) {
			return t, viewActionOpen
		}
	}
	t.scrollToCursor()
	return t, viewActionNone
}

// bodyRows is the number of data rows available below the header.
func (t timelineModel) bodyRows() int {
	rows := t.height - 1
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (t *timelineModel) scrollToCursor() {
	rows := t.bodyRows()
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	if t.cursor >= t.offset+rows {
		t.offset = t.cursor - rows + 1
	}
	if t.offset < 0 {
		t.offset = 0
	}
}

// view renders the timeline: a header row followed by dated+undated scene rows
// with an optional divider between them.
func (t timelineModel) view() string {
	header := ViewHeaderStyle.Render(truncate(
		fmt.Sprintf("Timeline — %s   (%d scenes)", folderDisplayName(t.root, t.root), len(t.entries)),
		t.width))

	if len(t.entries) == 0 {
		hint := lipgloss.NewStyle().Foreground(ColorDim).Render(
			"(no scenes — add timeline_date: YYYY-MM-DD to scene frontmatter)")
		return lipgloss.JoinVertical(lipgloss.Left, header, "", hint)
	}

	hasDivider := t.firstUndated > 0
	rows := t.bodyRows()
	lines := []string{header}
	rowsUsed := 0

	for i := t.offset; i < len(t.entries) && rowsUsed < rows; i++ {
		// Insert a divider before the first undated entry when both sections exist.
		if hasDivider && i == t.firstUndated {
			w := t.width
			if w < 1 {
				w = 1
			}
			lines = append(lines, RightPanelDivStyle.Render(strings.Repeat("─", w)))
			rowsUsed++
			if rowsUsed >= rows {
				break
			}
		}
		lines = append(lines, t.rowLine(t.entries[i], i == t.cursor))
		rowsUsed++
	}
	return strings.Join(lines, "\n")
}

// rowLine renders one timeline row: date | glyph | title (padded) | word count
func (t timelineModel) rowLine(e core.TimelineEntry, selected bool) string {
	const dateW = 10 // "YYYY-MM-DD"
	const wcW = 12   // "12,345 words"
	// Layout: datePart(10) + "  " + glyph(1) + "  " + title(var) + " " + wc(wcW)
	const fixedCols = dateW + 2 + 1 + 2 + 1 + wcW
	titleW := t.width - fixedCols
	if titleW < 1 {
		titleW = 1
	}

	datePart := strings.Repeat(" ", dateW)
	if e.TimelineDate != nil {
		datePart = e.TimelineDate.Format("2006-01-02")
	}

	glyph := statusGlyph(e.Status)
	if glyph == "" {
		glyph = "○"
	}

	wcStr := "—"
	if e.WordCount > 0 {
		wcStr = fmt.Sprintf("%d words", e.WordCount)
	}

	titleStr := padRight(truncate(e.Title, titleW), titleW)
	wcPart := padRight(wcStr, wcW)

	if selected {
		full := datePart + "  " + glyph + "  " + titleStr + " " + wcPart
		return TreeSelectedStyle.Render(full)
	}

	dateStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	glyphStyle := lipgloss.NewStyle().Foreground(ColorAccent)
	wcStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	return dateStyle.Render(datePart) + "  " +
		glyphStyle.Render(glyph) + "  " +
		TreeFileStyle.Render(titleStr) + " " +
		wcStyle.Render(wcPart)
}
