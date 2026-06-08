package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/acgh213/chisel/core"
)

// bookmarkListModel is a structural view that shows all bookmarked scenes for
// quick navigation. It follows the same pattern as corkboard/outliner/timeline.
type bookmarkListModel struct {
	root    string
	entries []bookmarkEntry
	cursor  int
	offset  int
	width   int
	height  int
}

// bookmarkEntry is a bookmarked scene's display info.
type bookmarkEntry struct {
	Path  string // absolute path
	Name  string // display name (basename without .md)
	Title string // frontmatter title, if available
}

// open loads bookmarked scenes from the config and resets the cursor.
func (b *bookmarkListModel) open(root string, bookmarks []string) {
	b.root = root
	b.entries = nil
	b.cursor = 0
	b.offset = 0
	for _, bp := range bookmarks {
		abs := bp
		if !filepath.IsAbs(bp) {
			abs = filepath.Join(root, bp)
		}
		name := strings.TrimSuffix(filepath.Base(bp), ".md")
		title := name
		info := core.ReadSceneInfo(abs)
		if info.Title != "" {
			title = info.Title
		}
		b.entries = append(b.entries, bookmarkEntry{
			Path:  abs,
			Name:  name,
			Title: title,
		})
	}
}

// SetSize sets the outer dimensions.
func (b *bookmarkListModel) SetSize(w, h int) {
	b.width = w
	b.height = h
}

func (b bookmarkListModel) stateTitle() string {
	return fmt.Sprintf("Bookmarks (%d)", len(b.entries))
}

// selected returns the path of the entry under the cursor, or "".
func (b bookmarkListModel) selected() string {
	if b.cursor < 0 || b.cursor >= len(b.entries) {
		return ""
	}
	return b.entries[b.cursor].Path
}

// update handles a key press and reports the action the root model must take.
func (b bookmarkListModel) update(msg tea.KeyPressMsg) (bookmarkListModel, viewAction) {
	switch msg.String() {
	case "j", "down":
		if b.cursor < len(b.entries)-1 {
			b.cursor++
		}
	case "k", "up":
		if b.cursor > 0 {
			b.cursor--
		}
	case "enter":
		if b.cursor >= 0 && b.cursor < len(b.entries) {
			return b, viewActionOpen
		}
	}
	b.scrollToCursor()
	return b, viewActionNone
}

// bodyRows is the number of data rows available below the header.
func (b bookmarkListModel) bodyRows() int {
	rows := b.height - 1
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (b *bookmarkListModel) scrollToCursor() {
	rows := b.bodyRows()
	if b.cursor < b.offset {
		b.offset = b.cursor
	}
	if b.cursor >= b.offset+rows {
		b.offset = b.cursor - rows + 1
	}
	if b.offset < 0 {
		b.offset = 0
	}
}

// view renders the bookmark list.
func (b bookmarkListModel) view() string {
	header := ViewHeaderStyle.Render(truncate(
		fmt.Sprintf("Bookmarks — %d bookmarked scenes", len(b.entries)),
		b.width))

	if len(b.entries) == 0 {
		hint := lipgloss.NewStyle().Foreground(ColorDim).Render(
			"No bookmarks — press Ctrl+B on a scene to bookmark it")
		return lipgloss.JoinVertical(lipgloss.Left, header, "", hint)
	}

	rows := b.bodyRows()
	lines := []string{header}

	for i := b.offset; i < len(b.entries) && i < b.offset+rows; i++ {
		lines = append(lines, b.rowLine(b.entries[i], i == b.cursor))
	}
	return strings.Join(lines, "\n")
}

// rowLine renders one bookmark row: star + name + title.
func (b bookmarkListModel) rowLine(e bookmarkEntry, selected bool) string {
	star := "★ "
	nameStr := e.Name
	suffix := ""
	if e.Title != e.Name {
		suffix = " — " + e.Title
	}

	full := star + nameStr + suffix

	if selected {
		return TreeSelectedStyle.Render(truncate(full, b.width))
	}

	titleStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	if suffix != "" {
		return star + TreeFileStyle.Render(nameStr) + titleStyle.Render(suffix)
	}
	return star + TreeFileStyle.Render(nameStr)
}
