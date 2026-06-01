package tui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// rpContent identifies what the right panel is currently displaying.
type rpContent int

const (
	rpContentChar  rpContent = iota // a specific character's detail
	rpContentCast                   // the project's full cast list
	rpContentEmpty                  // no characters exist yet
)

// rightPanelModel is the read-only right-hand inspector panel. It is
// binder-driven: it reflects whatever the binder's current selection is rather
// than maintaining its own cursor. No focus, no key handling — it is a pure
// view of the current selection and the characters/ directory.
type rightPanelModel struct {
	content rpContent
	char    *core.Character // non-nil when rpContentChar
	cast    []core.Character
	root    string
	width   int
	height  int
}

// newRightPanel creates an empty right panel for the given project root.
func newRightPanel(root string) rightPanelModel {
	return rightPanelModel{root: root, content: rpContentEmpty}
}

// SetSize updates the panel dimensions.
func (r *rightPanelModel) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// SyncToSelection updates panel content based on what the binder has selected.
// selectedFile is the path of the selected .md file, or "" when a directory
// or nothing is selected. It should be called whenever the binder cursor moves
// or after any CRUD operation that refreshes the binder.
func (r *rightPanelModel) SyncToSelection(selectedFile, root string) {
	r.root = root
	charsDir := filepath.Clean(core.CharactersDir(root))

	if selectedFile != "" && isUnderDir(selectedFile, charsDir) {
		c, err := core.LoadCharacter(selectedFile)
		if err == nil {
			r.char = c
			r.content = rpContentChar
			return
		}
	}

	// Not a character file — show the cast list.
	r.char = nil
	chars, _ := core.ListCharacters(root)
	r.cast = chars
	if len(chars) == 0 {
		r.content = rpContentEmpty
	} else {
		r.content = rpContentCast
	}
}

// view renders the right panel at its current dimensions.
func (r rightPanelModel) view() string {
	style := RightPanelStyle.
		Width(r.width - RightPanelStyle.GetHorizontalBorderSize()).
		Height(r.height - RightPanelStyle.GetVerticalBorderSize())

	contentW := r.width - RightPanelStyle.GetHorizontalFrameSize()
	if contentW < 1 {
		contentW = 1
	}
	visibleH := r.height - RightPanelStyle.GetVerticalFrameSize()
	if visibleH < 1 {
		visibleH = 1
	}

	lines := r.buildLines(contentW)
	if len(lines) > visibleH {
		lines = lines[:visibleH]
	}

	return style.Render(strings.Join(lines, "\n"))
}

// buildLines builds the panel's content as individual lines, capped to contentW.
func (r rightPanelModel) buildLines(contentW int) []string {
	wrap := func(s string, style lipgloss.Style) []string {
		rendered := style.Width(contentW).Render(s)
		return strings.Split(rendered, "\n")
	}
	div := RightPanelDivStyle.Render(strings.Repeat("─", contentW))

	switch r.content {
	case rpContentChar:
		return r.buildCharLines(contentW, wrap, div)
	case rpContentCast:
		return r.buildCastLines(contentW, wrap, div)
	default:
		return r.buildEmptyLines(contentW, wrap, div)
	}
}

func (r rightPanelModel) buildCharLines(contentW int, wrap func(string, lipgloss.Style) []string, div string) []string {
	c := r.char
	var lines []string

	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render(c.DisplayName()))
	if c.Meta.Role != "" {
		lines = append(lines, RightPanelRoleStyle.Width(contentW).Render(c.Meta.Role))
	}
	lines = append(lines, div)

	if c.Meta.Description != "" {
		lines = append(lines, wrap(c.Meta.Description, RightPanelFieldStyle)...)
		lines = append(lines, "")
	}
	if len(c.Meta.Tags) > 0 {
		lines = append(lines, RightPanelHintStyle.Render("tags: "+strings.Join(c.Meta.Tags, ", ")))
		lines = append(lines, "")
	}

	body := strings.TrimSpace(c.Body)
	if body != "" {
		lines = append(lines, div)
		lines = append(lines, RightPanelHintStyle.Render("notes"))
		for _, l := range strings.Split(body, "\n") {
			lines = append(lines, wrap(l, RightPanelFieldStyle)...)
		}
	}
	return lines
}

func (r rightPanelModel) buildCastLines(contentW int, wrap func(string, lipgloss.Style) []string, div string) []string {
	var lines []string
	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render("Characters"))
	lines = append(lines, div)
	for _, c := range r.cast {
		lines = append(lines, RightPanelFieldStyle.Width(contentW).Render(c.DisplayName()))
		if c.Meta.Role != "" {
			lines = append(lines, RightPanelRoleStyle.Width(contentW).Render("  "+c.Meta.Role))
		}
	}
	lines = append(lines, "")
	lines = append(lines, RightPanelHintStyle.Width(contentW).Render("select a character\nin the binder"))
	return lines
}

func (r rightPanelModel) buildEmptyLines(contentW int, wrap func(string, lipgloss.Style) []string, div string) []string {
	var lines []string
	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render("Characters"))
	lines = append(lines, div)
	lines = append(lines, "")
	lines = append(lines, RightPanelHintStyle.Width(contentW).Render("No characters yet."))
	lines = append(lines, "")
	lines = append(lines, RightPanelHintStyle.Width(contentW).Render("Add .md files to\ncharacters/ via\nthe binder (N)."))
	return lines
}

// isUnderDir reports whether path is inside (or equal to) dir, using cleaned
// paths to avoid false negatives from separator differences.
func isUnderDir(path, dir string) bool {
	p := filepath.Clean(path)
	d := filepath.Clean(dir)
	return p == d || strings.HasPrefix(p, d+string(filepath.Separator))
}
