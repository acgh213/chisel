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
	rpContentEmpty  rpContent = iota // no characters or locations exist yet
	rpContentWorld                   // combined cast + locations index
	rpContentEntity                  // a specific character or location detail
	rpContentError                   // entity file found but could not be loaded
)

// entityDetail holds the display fields for one character or location. Using a
// shared struct means buildEntityLines renders both without branching on type.
type entityDetail struct {
	header      string // panel section header: "Characters" / "Locations"
	name        string
	typeLabel   string // role (characters) or type (locations)
	description string
	tags        []string
	body        string
}

func entityFromCharacter(c *core.Character) entityDetail {
	return entityDetail{
		header:      "Characters",
		name:        c.DisplayName(),
		typeLabel:   c.Meta.Role,
		description: c.Meta.Description,
		tags:        c.Meta.Tags,
		body:        c.Body,
	}
}

func entityFromLocation(l *core.Location) entityDetail {
	return entityDetail{
		header:      "Locations",
		name:        l.DisplayName(),
		typeLabel:   l.Meta.Type,
		description: l.Meta.Description,
		tags:        l.Meta.Tags,
		body:        l.Body,
	}
}

// rightPanelModel is the read-only right-hand inspector panel. It is
// binder-driven: it reflects the binder's current selection rather than
// maintaining its own cursor. No focus, no key handling.
type rightPanelModel struct {
	content    rpContent
	entity     entityDetail
	cast       []core.Character // cached character list
	locs       []core.Location  // cached location list
	worldStale bool             // true when either list needs reloading from disk
	errMsg     string           // set when rpContentError
	root       string
	width      int
	height     int
}

// newRightPanel creates a panel for the given project root.
func newRightPanel(root string) rightPanelModel {
	return rightPanelModel{root: root, content: rpContentEmpty, worldStale: true}
}

// SetSize updates the panel dimensions.
func (r *rightPanelModel) SetSize(w, h int) {
	r.width = w
	r.height = h
}

// markWorldDirty signals that both the cast and location lists should be
// reloaded on the next sync. Call after any CRUD operation that may have
// added, removed, or renamed a character or location file.
func (r *rightPanelModel) markWorldDirty() {
	r.worldStale = true
}

// SyncToSelection updates panel content to match the binder's current selection.
// selectedFile is the path of the currently selected .md file, or "" for a
// directory or empty selection.
func (r *rightPanelModel) SyncToSelection(selectedFile string) {
	charsDir := filepath.Clean(core.CharactersDir(r.root))
	locsDir := filepath.Clean(core.LocationsDir(r.root))

	if selectedFile != "" && isUnderDir(selectedFile, charsDir) {
		c, err := core.LoadCharacter(selectedFile)
		if err == nil {
			r.entity = entityFromCharacter(c)
			r.content = rpContentEntity
			return
		}
		r.errMsg = fmt.Sprintf("Could not read\n%s\n\nCheck permissions\nor YAML formatting.",
			filepath.Base(selectedFile))
		r.content = rpContentError
		return
	}

	if selectedFile != "" && isUnderDir(selectedFile, locsDir) {
		l, err := core.LoadLocation(selectedFile)
		if err == nil {
			r.entity = entityFromLocation(l)
			r.content = rpContentEntity
			return
		}
		r.errMsg = fmt.Sprintf("Could not read\n%s\n\nCheck permissions\nor YAML formatting.",
			filepath.Base(selectedFile))
		r.content = rpContentError
		return
	}

	// Not a character or location file — show the world index, reloading only
	// when stale (i.e., after a CRUD operation or first display).
	if r.worldStale || (r.cast == nil && r.locs == nil) {
		r.cast, _ = core.ListCharacters(r.root)
		r.locs, _ = core.ListLocations(r.root)
		r.worldStale = false
	}
	if len(r.cast) == 0 && len(r.locs) == 0 {
		r.content = rpContentEmpty
	} else {
		r.content = rpContentWorld
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
	wrap := func(s string, style lipgloss.Style) []string {
		return strings.Split(style.Width(contentW).Render(s), "\n")
	}
	div := RightPanelDivStyle.Render(strings.Repeat("─", contentW))

	switch r.content {
	case rpContentEntity:
		return r.buildEntityLines(contentW, wrap, div)
	case rpContentWorld:
		return r.buildWorldLines(contentW, div)
	case rpContentError:
		return r.buildErrorLines(contentW, div)
	default: // rpContentEmpty
		return r.buildEmptyLines(contentW, div)
	}
}

// buildEntityLines renders a character or location detail view using the
// unified entityDetail struct, so both types share identical layout logic.
func (r rightPanelModel) buildEntityLines(contentW int, wrap func(string, lipgloss.Style) []string, div string) []string {
	e := r.entity
	var lines []string

	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render(e.name))
	if e.typeLabel != "" {
		lines = append(lines, RightPanelRoleStyle.Width(contentW).Render(e.typeLabel))
	}
	lines = append(lines, div)

	if e.description != "" {
		lines = append(lines, wrap(e.description, RightPanelFieldStyle)...)
		lines = append(lines, "")
	}
	if len(e.tags) > 0 {
		lines = append(lines, RightPanelHintStyle.Render("tags: "+strings.Join(e.tags, ", ")))
		lines = append(lines, "")
	}

	body := strings.TrimSpace(e.body)
	if body != "" {
		lines = append(lines, div)
		lines = append(lines, RightPanelHintStyle.Render("notes"))
		for _, l := range strings.Split(body, "\n") {
			lines = append(lines, wrap(l, RightPanelFieldStyle)...)
		}
	}
	return lines
}

// buildWorldLines renders the combined characters + locations index shown when
// no entity file is selected in the binder.
func (r rightPanelModel) buildWorldLines(contentW int, div string) []string {
	var lines []string
	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render("World"))
	lines = append(lines, div)

	if len(r.cast) > 0 {
		lines = append(lines, RightPanelHintStyle.Render("characters"))
		for _, c := range r.cast {
			lines = append(lines, RightPanelFieldStyle.Width(contentW).Render(c.DisplayName()))
			if c.Meta.Role != "" {
				lines = append(lines, RightPanelRoleStyle.Width(contentW).Render("  "+c.Meta.Role))
			}
		}
		lines = append(lines, "")
	}

	if len(r.locs) > 0 {
		lines = append(lines, RightPanelHintStyle.Render("locations"))
		for _, l := range r.locs {
			lines = append(lines, RightPanelFieldStyle.Width(contentW).Render(l.DisplayName()))
			if l.Meta.Type != "" {
				lines = append(lines, RightPanelRoleStyle.Width(contentW).Render("  "+l.Meta.Type))
			}
		}
		lines = append(lines, "")
	}

	lines = append(lines, RightPanelHintStyle.Width(contentW).Render("select an entity\nin the binder"))
	return lines
}

func (r rightPanelModel) buildEmptyLines(contentW int, div string) []string {
	var lines []string
	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render("World"))
	lines = append(lines, div)
	lines = append(lines, "")
	lines = append(lines, RightPanelHintStyle.Width(contentW).Render("No characters or\nlocations yet."))
	lines = append(lines, "")
	lines = append(lines, RightPanelHintStyle.Width(contentW).Render("Press N to create\ncharacters/ or\nlocations/ folders,\nthen n to add files."))
	return lines
}

func (r rightPanelModel) buildErrorLines(contentW int, div string) []string {
	var lines []string
	lines = append(lines, RightPanelHeaderStyle.Width(contentW).Render("World"))
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
