package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// quickNoteAction is the outcome of a key press inside the popup.
type quickNoteAction int

const (
	quickNoteNone      quickNoteAction = iota
	quickNoteConfirmed                 // Enter pressed with non-empty text
	quickNoteCancelled                 // Esc pressed (or Enter on empty input)
)

// quickNoteModel is a floating single-line input that captures a fleeting
// thought to notes/scratch.md without disrupting the current view state.
// It is activated by the backtick key (`) from any state and is checked
// before all other key dispatch so it always gets first priority.
type quickNoteModel struct {
	isActive bool
	input    textinput.Model
}

func newQuickNote() quickNoteModel {
	ti := textinput.New()
	ti.Placeholder = "capture a thought…"
	ti.CharLimit = 500
	ti.Width = 50
	return quickNoteModel{input: ti}
}

// open activates the popup and focuses the input. Returns the cursor-blink Cmd.
func (q *quickNoteModel) open() tea.Cmd {
	q.isActive = true
	q.input.SetValue("")
	return q.input.Focus()
}

// close deactivates the popup and clears the input.
func (q *quickNoteModel) close() {
	q.isActive = false
	q.input.Blur()
	q.input.SetValue("")
}

// active reports whether the popup is currently shown.
func (q quickNoteModel) active() bool {
	return q.isActive
}

// value returns the current text (use after Confirmed).
func (q quickNoteModel) value() string {
	return strings.TrimSpace(q.input.Value())
}

// update handles a key press inside the popup.
func (q quickNoteModel) update(msg tea.KeyMsg) (quickNoteModel, quickNoteAction, tea.Cmd) {
	switch msg.String() {
	case "esc":
		q.close()
		return q, quickNoteCancelled, nil
	case "enter":
		if strings.TrimSpace(q.input.Value()) != "" {
			return q, quickNoteConfirmed, nil
		}
		q.close()
		return q, quickNoteCancelled, nil
	default:
		var cmd tea.Cmd
		q.input, cmd = q.input.Update(msg)
		return q, quickNoteNone, cmd
	}
}

// view renders the popup centered within a terminal canvas of w×h. The rest
// of the canvas is filled with whitespace so the popup appears modal.
func (q quickNoteModel) view(w, h int) string {
	header := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Render("Quick Note")

	hint := lipgloss.NewStyle().
		Foreground(ColorDim).
		Italic(true).
		Render("Enter=save  Esc=cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		header,
		q.input.View(),
		"",
		hint,
	)

	// Inner width: input width + a little breathing room for the label.
	// Padding(0,2) adds 2 cols each side; border adds 1 each side = 6 total.
	const innerW = 54
	popup := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorAccent).
		Padding(0, 2).
		Width(innerW).
		Render(inner)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, popup)
}
