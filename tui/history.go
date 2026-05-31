package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// historyAction is what the history browser asks the root model to do after a
// key press it can't handle on its own.
type historyAction int

const (
	historyNone    historyAction = iota
	historyClose                 // leave the history browser
	historyRestore               // restore the selected revision into the editor
)

// historyMode is the browser's current view.
type historyMode int

const (
	historyList historyMode = iota // the revision list
	historyDiff                    // a single revision's diff
)

// historyModel is the revision browser for one scene: a list of snapshots, with
// a per-revision diff view. It owns only view state; all VCS work goes through
// the core.RevisionBackend.
type historyModel struct {
	backend core.RevisionBackend
	path    string // scene file being browsed
	name    string // display name for the header

	revs   []core.Revision
	cursor int
	offset int

	mode       historyMode
	diff       string
	diffOffset int

	err    string
	width  int
	height int
}

// open loads the revision list for a scene and resets the browser to the list
// view. A load error is returned so the root model can surface it without
// opening the pane.
func (h *historyModel) open(backend core.RevisionBackend, path, name string) error {
	revs, err := backend.Log(path)
	if err != nil {
		return err
	}
	h.backend = backend
	h.path = path
	h.name = name
	h.revs = revs
	h.cursor = 0
	h.offset = 0
	h.mode = historyList
	h.diff = ""
	h.diffOffset = 0
	h.err = ""
	return nil
}

// SetSize sets the browser's outer dimensions.
func (h *historyModel) SetSize(w, height int) {
	h.width = w
	h.height = height
}

// selectedHash returns the hash of the currently selected revision, or "".
func (h historyModel) selectedHash() string {
	if h.cursor < 0 || h.cursor >= len(h.revs) {
		return ""
	}
	return h.revs[h.cursor].Hash
}

// update handles a key press and reports any action the root model must take.
func (h historyModel) update(msg tea.KeyMsg) (historyModel, historyAction) {
	if h.mode == historyDiff {
		switch msg.String() {
		case "esc", "h", "left", "backspace":
			h.mode = historyList
		case "j", "down":
			h.diffOffset++
		case "k", "up":
			if h.diffOffset > 0 {
				h.diffOffset--
			}
		case "r":
			return h, historyRestore
		}
		return h, historyNone
	}

	// List mode.
	switch msg.String() {
	case "esc", "ctrl+h", "q":
		return h, historyClose
	case "j", "down":
		if h.cursor < len(h.revs)-1 {
			h.cursor++
		}
		h.scrollToCursor()
	case "k", "up":
		if h.cursor > 0 {
			h.cursor--
		}
		h.scrollToCursor()
	case "enter", "l", "right":
		h.showDiff()
	case "r":
		return h, historyRestore
	}
	return h, historyNone
}

// showDiff computes the diff the selected snapshot introduced (vs the previous
// snapshot of this scene) and switches to the diff view. The oldest snapshot has
// no predecessor, so its full content is shown instead.
func (h *historyModel) showDiff() {
	if len(h.revs) == 0 {
		return
	}
	sel := h.revs[h.cursor]
	h.diffOffset = 0
	h.mode = historyDiff

	if h.cursor+1 < len(h.revs) {
		older := h.revs[h.cursor+1]
		d, err := h.backend.Diff(h.path, older.Hash, sel.Hash)
		if err != nil {
			h.err = err.Error()
			return
		}
		h.diff = d
		return
	}

	// Oldest snapshot — show its content as the initial version.
	content, err := h.backend.Restore(h.path, sel.Hash)
	if err != nil {
		h.err = err.Error()
		return
	}
	h.diff = "(initial snapshot)\n\n" + content
}

func (h *historyModel) scrollToCursor() {
	rows := h.listRows()
	if h.cursor < h.offset {
		h.offset = h.cursor
	}
	if h.cursor >= h.offset+rows {
		h.offset = h.cursor - rows + 1
	}
	if h.offset < 0 {
		h.offset = 0
	}
}

// listRows is how many revision rows fit (box minus frame, minus the header).
func (h historyModel) listRows() int {
	rows := h.height - HistoryStyle.GetVerticalFrameSize() - 1
	if rows < 1 {
		rows = 1
	}
	return rows
}

// view renders the browser as a single bordered box.
func (h historyModel) view() string {
	style := HistoryStyle.
		Width(h.width - HistoryStyle.GetHorizontalBorderSize()).
		Height(h.height - HistoryStyle.GetVerticalBorderSize())

	innerW := h.width - HistoryStyle.GetHorizontalFrameSize()
	if innerW < 1 {
		innerW = 1
	}

	if h.err != "" {
		return style.Render(DiffDelStyle.Render("Error: " + h.err))
	}
	if h.mode == historyDiff {
		return style.Render(h.renderDiff(innerW))
	}
	return style.Render(h.renderList(innerW))
}

func (h historyModel) renderList(w int) string {
	lines := []string{
		HistoryHeaderStyle.Render(truncate(fmt.Sprintf("History — %s   (%d snapshots)", h.name, len(h.revs)), w)),
	}

	if len(h.revs) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorDim).
			Render("(no snapshots yet — save with Ctrl+S to create one)"))
		return strings.Join(lines, "\n")
	}

	rows := h.listRows()
	start := h.offset
	end := start + rows
	if end > len(h.revs) {
		end = len(h.revs)
	}
	for i := start; i < end; i++ {
		r := h.revs[i]
		line := truncate(fmt.Sprintf("%s   %s   %s", r.Short(), relTime(r.Timestamp), firstLine(r.Message)), w)
		if i == h.cursor {
			lines = append(lines, TreeSelectedStyle.Render(line))
		} else {
			lines = append(lines, TreeFileStyle.Render(line))
		}
	}
	return strings.Join(lines, "\n")
}

func (h historyModel) renderDiff(w int) string {
	header := "Diff"
	if h.cursor < len(h.revs) {
		header = fmt.Sprintf("Diff — %s   (↑/↓ scroll · Esc back · r restore)", h.revs[h.cursor].Short())
	}
	lines := []string{HistoryHeaderStyle.Render(truncate(header, w))}

	bodyRows := h.height - HistoryStyle.GetVerticalFrameSize() - 1
	if bodyRows < 1 {
		bodyRows = 1
	}

	diffLines := strings.Split(h.diff, "\n")
	start := h.diffOffset
	if start > len(diffLines)-1 {
		start = max(0, len(diffLines)-1)
	}
	end := start + bodyRows
	if end > len(diffLines) {
		end = len(diffLines)
	}
	for i := start; i < end; i++ {
		lines = append(lines, styleDiffLine(truncate(diffLines[i], w)))
	}
	return strings.Join(lines, "\n")
}

// styleDiffLine colors a unified-diff line by its leading marker.
func styleDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"),
		strings.HasPrefix(line, "@@"), strings.HasPrefix(line, "diff "):
		return DiffMetaStyle.Render(line)
	case strings.HasPrefix(line, "+"):
		return DiffAddStyle.Render(line)
	case strings.HasPrefix(line, "-"):
		return DiffDelStyle.Render(line)
	default:
		return line
	}
}

// firstLine returns the first line of a (possibly multi-line) commit message.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// truncate clips s to at most w runes, adding an ellipsis when cut.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

// relTime renders a timestamp as a short relative string.
func relTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2 15:04")
	}
}
