package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// rpContent identifies what the right panel is currently displaying.
type rpContent int

const (
	rpContentEmpty rpContent = iota // no characters/ dir, or empty
	rpContentCast                   // project cast list (non-character file selected)
	rpContentChar                   // a specific character's detail
	rpContentError                  // character file found but could not be loaded
)

// rightPanelModel is the read-only right-hand inspector panel. It is
// binder-driven: it reflects the binder's current selection rather than
// maintaining its own cursor. No focus, no key handling.
type rightPanelModel struct {
	content   rpContent
	char      *core.Character  // non-nil when rpContentChar
	cast      []core.Character // cached cast list
	castStale bool             // true when cast needs reloading from disk
	errMsg    string           // set when rpContentError
	root      string
	width     int
	height    int
}

// newRightPanel creates a panel for the given project root.
func newRightPanel(root string) rightPanelModel {
	return rightPanelModel{root: root, content: rpContentEmpty, castStale: true}
}

// SetSize updates the panel dimensions.
func (r *rightPanelModel) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// markCastDirty signals that the cast list should be reloaded on the next sync.
// Call this after any CRUD operation that may have added, removed, or renamed
// a character file.
func (r *rightPanelModel) markCastDirty() {
	r.castStale = true
}

// SyncToSelection updates panel content to match the binder's current selection.
// selectedFile is the path of the currently selected .md file, or "" for a
// directory or empty selection.
func (r *rightPanelModel) SyncToSelection(selectedFile string) {
	charsDir := filepath.Clean(core.CharactersDir(r.root))

	if selectedFile != "" && isUnderDir(selectedFile, charsDir) {
		c, err := core.LoadCharacter(selectedFile)
		if err == nil {
			r.char = c
			r.content = rpContentChar
			return
		}
		// File is under characters/ but unreadable — show an error hint rather
		// than silently falling back to the cast list, which would look like the
		// character doesn't exist.
		r.char = nil
		r.errMsg = fmt.Sprintf("Could not read\n%s\n\nCheck permissions\nor YAML formatting.",
			filepath.Base(selectedFile))
		r.content = rpContentError
		return
	}

	// Not a character file — show the cast list, reloading only when stale.
	r.char = nil
	if r.castStale || r.cast == nil {
		chars, _ := core.ListCharacters(r.root)
		r.cast = chars
		r.castStale = false
	}
	if len(r.cast) == 0 {
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

// buildLines builds the panel's content as individual rendered lines.
func (r rightPanelModel) buildLines(contentW int) []string {
	// wrap is used only for multi-line prose (character detail); cast/empty/error
	// views use direct Width renders which already word-wrap.
	wrap := func(s string, style lipgloss.Style) []string {
		return strings.Split(style.Width(contentW).Render(s), "\n")
	}
	div := RightPanelDivStyle.Render(strings.Repeat("─", contentW))

	switch r.content {
	case rpContentChar:
		return r.buildCharLines(contentW, wrap, div)
	case rpContentCast:
		return r.buildCastLines(contentW, div)
	case rpContentError:
		return r.buildErrorLines(contentW, div)
	default: // rpContentEmpty
		return r.buildEmptyLines(contentW, div)
	}
}

func (r rightPanelModel) buildCharLines(contentW int, wrap func(string, lipgloss.Style) []string, div string) []string {
	c := r.char
	if c == nil {
		// Guard against zero-value struct; should not happen in normal use.
		return r.buildEmptyLines(contentW, div)
	}
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

func (r rightPanelModel) buildCastLines(contentW int, div string) []string {
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

func (r rightPanelModel) buildEmptyLines(contentW int, div string) []string {
	var lines []string
	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render("Characters"))
	lines = append(lines, div)
	lines = append(lines, "")
	lines = append(lines, RightPanelHintStyle.Width(contentW).Render("No characters yet."))
	lines = append(lines, "")
	// Two-step hint: create the folder first (N), then add files (n).
	lines = append(lines, RightPanelHintStyle.Width(contentW).Render("Press N to create a\ncharacters/ folder,\nthen n to add files."))
	return lines
}

func (r rightPanelModel) buildErrorLines(contentW int, div string) []string {
	var lines []string
	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render("Characters"))
	lines = append(lines, div)
	lines = append(lines, "")
	lines = append(lines, RightPanelHintStyle.Width(contentW).Render(r.errMsg))
	return lines
}

// isUnderDir reports whether path is inside (or equal to) dir, using cleaned
// paths to avoid false negatives from separator differences.
func isUnderDir(path, dir string) bool {
	p := filepath.Clean(path)
	d := filepath.Clean(dir)
	return p == d || strings.HasPrefix(p, d+string(filepath.Separator))
}

