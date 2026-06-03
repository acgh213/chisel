package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// searchAction is the outcome the root model must act on after a key press.
type searchAction int

const (
	searchNone  searchAction = iota
	searchOpen               // user pressed Enter on a result — root should open selectedPath()
	searchClose              // popup closed (Esc with no results)
)

type searchInputMode int

const (
	searchInputting searchInputMode = iota // textinput is focused
	searchBrowsing                         // result list is focused
)

// searchModel is a floating overlay for full-text scene search. Ctrl+F opens it
// from any state. The user types a query and presses Enter to run the search;
// results appear in a scrollable list below the input. A second Enter opens the
// selected scene. Esc from the result list returns to the input; Esc from the
// input closes the popup entirely.
type searchModel struct {
	isActive bool
	mode     searchInputMode
	input    textinput.Model
	results  []core.SearchResult // nil = not yet searched; empty = no matches
	cursor   int
	offset   int
	root     string
}

// searchPopupW is the inner content width of the search popup.
const searchPopupW = 72

// searchMaxVisible is the number of result rows shown at once.
const searchMaxVisible = 10

func newSearch(root string) searchModel {
	ti := textinput.New()
	ti.Placeholder = "search all scenes…"
	ti.Width = searchPopupW - 4 // subtract popup padding
	return searchModel{input: ti, root: root}
}

// open activates the popup and focuses the text input.
func (s *searchModel) open() tea.Cmd {
	s.isActive = true
	s.mode = searchInputting
	s.results = nil
	s.cursor = 0
	s.offset = 0
	s.input.SetValue("")
	return s.input.Focus()
}

// close deactivates the popup and clears all state.
func (s *searchModel) close() {
	s.isActive = false
	s.input.Blur()
	s.input.SetValue("")
	s.results = nil
	s.cursor = 0
	s.offset = 0
}

// active reports whether the popup is currently shown.
func (s searchModel) active() bool { return s.isActive }

// selectedPath returns the path of the result under the cursor. Call this
// before close() when handling searchOpen — results are cleared on close.
func (s searchModel) selectedPath() string {
	if s.cursor >= 0 && s.cursor < len(s.results) {
		return s.results[s.cursor].Path
	}
	return ""
}

// update handles a key press and returns the updated model, the action for the
// root model, and any Cmd to batch.
func (s searchModel) update(msg tea.KeyMsg) (searchModel, searchAction, tea.Cmd) {
	switch s.mode {
	case searchInputting:
		switch msg.String() {
		case "esc":
			s.close()
			return s, searchClose, nil
		case "enter":
			q := strings.TrimSpace(s.input.Value())
			if q == "" {
				return s, searchNone, nil
			}
			results, err := core.SearchScenes(s.root, q)
			if err != nil {
				// Treat search errors as no-results; the root model will show nothing.
				s.results = []core.SearchResult{}
				return s, searchNone, nil
			}
			s.results = results
			s.cursor = 0
			s.offset = 0
			if len(results) > 0 {
				s.mode = searchBrowsing
				s.input.Blur()
			}
			return s, searchNone, nil
		default:
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, searchNone, cmd
		}

	case searchBrowsing:
		switch msg.String() {
		case "esc":
			// Return to input so the user can refine the query.
			s.mode = searchInputting
			s.results = nil
			s.cursor = 0
			s.offset = 0
			return s, searchNone, s.input.Focus()
		case "j", "down":
			if s.cursor < len(s.results)-1 {
				s.cursor++
				s.scrollToCursor()
			}
		case "k", "up":
			if s.cursor > 0 {
				s.cursor--
				s.scrollToCursor()
			}
		case "enter":
			if s.cursor >= 0 && s.cursor < len(s.results) {
				// Do NOT close here — root reads selectedPath() then calls close().
				return s, searchOpen, nil
			}
		}
	}
	return s, searchNone, nil
}

func (s *searchModel) scrollToCursor() {
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+searchMaxVisible {
		s.offset = s.cursor - searchMaxVisible + 1
	}
}

// view renders the popup overlaid on bg. w and h are the terminal dimensions.
func (s searchModel) view(w, h int, bg string) string {
	header := lipgloss.NewStyle().
		Foreground(ColorAccent).Bold(true).
		Render("Search")

	var contentLines []string
	contentLines = append(contentLines, header)
	contentLines = append(contentLines, s.input.View())
	contentLines = append(contentLines, "")

	switch s.mode {
	case searchBrowsing:
		if len(s.results) == 0 {
			contentLines = append(contentLines,
				lipgloss.NewStyle().Foreground(ColorDim).Italic(true).Render("no matches found"))
		} else {
			for i := s.offset; i < len(s.results) && i < s.offset+searchMaxVisible; i++ {
				contentLines = append(contentLines, s.resultLine(s.results[i], i == s.cursor))
			}
			noun := "match"
			if len(s.results) != 1 {
				noun = "matches"
			}
			contentLines = append(contentLines, "")
			contentLines = append(contentLines,
				lipgloss.NewStyle().Foreground(ColorDim).Render(
					fmt.Sprintf("%d %s", len(s.results), noun)))
		}
		contentLines = append(contentLines, "")
		contentLines = append(contentLines,
			lipgloss.NewStyle().Foreground(ColorDim).Italic(true).
				Render("↑/↓ Navigate  Enter=Open  Esc=Refine"))

	case searchInputting:
		if s.results != nil && len(s.results) == 0 {
			contentLines = append(contentLines,
				lipgloss.NewStyle().Foreground(ColorDim).Render("no matches found"))
		}
		contentLines = append(contentLines, "")
		contentLines = append(contentLines,
			lipgloss.NewStyle().Foreground(ColorDim).Italic(true).
				Render("Enter=Search  Esc=Close"))
	}

	inner := strings.Join(contentLines, "\n")

	popup := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorAccent).
		Padding(0, 2).
		Width(searchPopupW).
		Render(inner)

	popupH := lipgloss.Height(popup)
	centeredStrip := lipgloss.Place(w, popupH, lipgloss.Center, lipgloss.Top, popup)
	stripLines := strings.Split(centeredStrip, "\n")

	bgLines := strings.Split(bg, "\n")
	for len(bgLines) < h {
		bgLines = append(bgLines, strings.Repeat(" ", w))
	}

	// Center the popup vertically.
	startRow := (h - popupH) / 2
	if startRow < 0 {
		startRow = 0
	}

	result := make([]string, h)
	copy(result, bgLines[:h])
	for i, line := range stripLines {
		if startRow+i < h {
			result[startRow+i] = line
		}
	}

	return strings.Join(result, "\n")
}

// resultLine renders one search result row.
func (s searchModel) resultLine(r core.SearchResult, selected bool) string {
	const metaW = 24
	lineW := searchPopupW - metaW - 2
	if lineW < 1 {
		lineW = 1
	}

	meta := fmt.Sprintf("%s:%d", r.Title, r.LineNum)
	lineText := truncate(r.Line, lineW)

	if selected {
		full := padRight(meta, metaW) + "  " + lineText
		return TreeSelectedStyle.Render(truncate(full, searchPopupW))
	}

	metaStyle := lipgloss.NewStyle().Foreground(ColorAccent)
	textStyle := lipgloss.NewStyle().Foreground(ColorFg)
	return metaStyle.Render(padRight(truncate(meta, metaW), metaW)) + "  " + textStyle.Render(lineText)
}
