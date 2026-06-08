package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// readerColW is the width of the reading column in characters.
const readerColW = 72

// readerModel is the full-screen reading mode. It word-wraps the current
// scene's body into a centered column and allows scrolling with arrow keys.
// F6 or Esc exits back to the previous view.
type readerModel struct {
	isActive bool
	title    string
	lines    []string // word-wrapped body lines
	offset   int
	visible  int // lines shown at once (derived from terminal height)
}

// open activates the reader for the given scene title and body.
func (r *readerModel) open(title, body string, h int) {
	r.isActive = true
	r.title = title
	if r.title == "" {
		r.title = "Untitled"
	}
	r.lines = wrapLines(body, readerColW)
	r.offset = 0
	// Reserve rows: title + blank line + hint line + 1 padding = 4
	r.visible = h - 4
	if r.visible < 1 {
		r.visible = 1
	}
}

// setHeight updates the number of visible lines after a terminal resize.
func (r *readerModel) setHeight(h int) {
	r.visible = h - 4
	if r.visible < 1 {
		r.visible = 1
	}
	// Clamp scroll offset so it stays within the new bounds.
	max := len(r.lines) - r.visible
	if max < 0 {
		max = 0
	}
	if r.offset > max {
		r.offset = max
	}
}

// close deactivates the reader and frees memory.
func (r *readerModel) close() {
	r.isActive = false
	r.lines = nil
	r.offset = 0
}

// active reports whether reading mode is currently shown.
func (r readerModel) active() bool { return r.isActive }

// update handles a key press. Returns (updated model, true if exit requested).
func (r readerModel) update(msg tea.KeyPressMsg) (readerModel, bool) {
	switch msg.String() {
	case "esc", "f6":
		r.close()
		return r, true
	case "j", "down":
		max := len(r.lines) - r.visible
		if max < 0 {
			max = 0
		}
		if r.offset < max {
			r.offset++
		}
	case "k", "up":
		if r.offset > 0 {
			r.offset--
		}
	case "ctrl+d", "pgdown":
		r.offset += r.visible / 2
		max := len(r.lines) - r.visible
		if max < 0 {
			max = 0
		}
		if r.offset > max {
			r.offset = max
		}
	case "ctrl+u", "pgup":
		r.offset -= r.visible / 2
		if r.offset < 0 {
			r.offset = 0
		}
	}
	return r, false
}

// view renders the full-screen reading view (no status bar).
func (r readerModel) view(w, h int) string {
	margin := (w - readerColW) / 2
	if margin < 0 {
		margin = 0
	}
	indent := strings.Repeat(" ", margin)

	titleStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(ColorFg)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim).Italic(true)

	var sb strings.Builder

	// Title + blank line
	sb.WriteString(indent + titleStyle.Render(r.title) + "\n\n")

	// Body: show [offset, offset+visible)
	end := r.offset + r.visible
	if end > len(r.lines) {
		end = len(r.lines)
	}
	shown := 0
	for _, line := range r.lines[r.offset:end] {
		sb.WriteString(indent + bodyStyle.Render(line) + "\n")
		shown++
	}
	// Pad empty rows so the hint line stays at a stable position.
	for i := shown; i < r.visible; i++ {
		sb.WriteString("\n")
	}

	// Hint line with scroll position when the text is longer than the viewport.
	hint := "↑/↓ Scroll  F6/Esc=Exit"
	if len(r.lines) > r.visible {
		hint += fmt.Sprintf("  (%d/%d lines)", r.offset+r.visible, len(r.lines))
	}
	sb.WriteString(indent + dimStyle.Render(hint))

	return sb.String()
}

// wrapLines word-wraps text to fit within width rune-columns per line.
// Blank lines in the source are preserved as empty entries.
func wrapLines(text string, width int) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		if strings.TrimSpace(para) == "" {
			out = append(out, "")
			continue
		}
		words := strings.Fields(para)
		cur := words[0]
		for _, word := range words[1:] {
			if len(cur)+1+len(word) <= width {
				cur += " " + word
			} else {
				out = append(out, cur)
				cur = word
			}
		}
		out = append(out, cur)
	}
	return out
}
