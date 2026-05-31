package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// outlinerModel is Scrivener's outliner: a collapsible tree of the whole project
// with status and word-count/target columns. It builds its own tree snapshot
// (folders expanded) so expand/collapse here is independent of the binder.
type outlinerModel struct {
	root  string
	nodes []*core.FileNode
	flat  []*core.FileNode
	info  map[string]core.SceneInfo // path -> info, files only

	cursor int
	offset int

	width  int
	height int
}

// open builds the outline for the project rooted at root, with all folders
// expanded and per-scene info loaded. A scan error is returned so the root model
// can surface it without switching views.
func (o *outlinerModel) open(root string) error {
	nodes, err := core.NewProject(root).BuildTree()
	if err != nil {
		return err
	}
	expandAll(nodes)

	o.root = root
	o.nodes = nodes
	o.info = make(map[string]core.SceneInfo)
	o.cursor = 0
	o.offset = 0
	o.rebuildFlat()

	for _, n := range o.flat {
		if !n.IsDir {
			o.info[n.Path] = core.ReadSceneInfo(n.Path)
		}
	}
	return nil
}

// expandAll marks every directory node open so the outline shows in full.
func expandAll(nodes []*core.FileNode) {
	for _, n := range nodes {
		if n.IsDir {
			n.Expanded = true
			expandAll(n.Children)
		}
	}
}

func (o *outlinerModel) rebuildFlat() {
	o.flat = nil
	core.Flatten(o.nodes, &o.flat)
}

// SetSize sets the outliner's outer dimensions.
func (o *outlinerModel) SetSize(w, h int) {
	o.width = w
	o.height = h
}

// selected returns the path of the node under the cursor if it's a scene, or "".
func (o outlinerModel) selected() string {
	if o.cursor < 0 || o.cursor >= len(o.flat) {
		return ""
	}
	n := o.flat[o.cursor]
	if n.IsDir {
		return ""
	}
	return n.Path
}

// update handles a key press and reports any action the root model must take.
func (o outlinerModel) update(msg tea.KeyMsg) (outlinerModel, viewAction) {
	switch msg.String() {
	case "j", "down":
		if o.cursor < len(o.flat)-1 {
			o.cursor++
		}
	case "k", "up":
		if o.cursor > 0 {
			o.cursor--
		}
	case "l", "right":
		if n := o.current(); n != nil && n.IsDir && !n.Expanded {
			n.Expanded = true
			o.rebuildFlat()
		}
	case "h", "left":
		if n := o.current(); n != nil && n.IsDir && n.Expanded {
			n.Expanded = false
			o.rebuildFlat()
		}
	case "enter", " ":
		n := o.current()
		if n == nil {
			break
		}
		if n.IsDir {
			n.Expanded = !n.Expanded
			o.rebuildFlat()
		} else {
			return o, viewActionOpen
		}
	}
	o.scrollToCursor()
	return o, viewActionNone
}

// current returns the node under the cursor, or nil.
func (o outlinerModel) current() *core.FileNode {
	if o.cursor < 0 || o.cursor >= len(o.flat) {
		return nil
	}
	return o.flat[o.cursor]
}

// bodyRows is how many tree rows fit under the header (at least one).
func (o outlinerModel) bodyRows() int {
	rows := o.height - 1 // minus the header
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (o *outlinerModel) scrollToCursor() {
	rows := o.bodyRows()
	if o.cursor < o.offset {
		o.offset = o.cursor
	}
	if o.cursor >= o.offset+rows {
		o.offset = o.cursor - rows + 1
	}
	if o.offset < 0 {
		o.offset = 0
	}
}

// view renders the outline: a header plus the visible tree rows with columns.
func (o outlinerModel) view() string {
	header := ViewHeaderStyle.Render(truncate(
		fmt.Sprintf("Outliner — %s   (%d items)", folderDisplayName(o.root, o.root), len(o.flat)), o.width))

	if len(o.flat) == 0 {
		empty := lipgloss.NewStyle().Foreground(ColorDim).Render("(empty project)")
		return lipgloss.JoinVertical(lipgloss.Left, header, "", empty)
	}

	rows := o.bodyRows()
	start := o.offset
	end := start + rows
	if end > len(o.flat) {
		end = len(o.flat)
	}

	lines := []string{header}
	for i := start; i < end; i++ {
		lines = append(lines, o.rowLine(o.flat[i], i == o.cursor))
	}
	return strings.Join(lines, "\n")
}

// rowLine renders one outline row: an indented name on the left and, for scenes,
// a right-aligned status-glyph + word-count/target column.
func (o outlinerModel) rowLine(node *core.FileNode, selected bool) string {
	indent := strings.Repeat("  ", node.Depth)

	prefix := "  "
	if node.IsDir {
		if node.Expanded {
			prefix = "▾ "
		} else {
			prefix = "▸ "
		}
	}
	left := indent + prefix + node.Name

	// Right column (scenes only).
	right := ""
	met := false
	if !node.IsDir {
		info := o.info[node.Path]
		g := statusGlyph(info.Status)
		if g != "" {
			right = g + " "
		}
		right += wordsLabel(info)
		met = info.WordTarget > 0 && info.WordCount >= info.WordTarget
	}

	rw := len([]rune(right))
	leftMax := o.width - rw
	if rw > 0 {
		leftMax-- // gap between the columns
	}
	if leftMax < 1 {
		leftMax = 1
	}
	left = padRight(truncate(left, leftMax), leftMax)

	// A selected row is rendered as one highlighted band.
	if selected {
		line := left
		if rw > 0 {
			line += " " + right
		}
		return TreeSelectedStyle.Render(line)
	}

	leftStyle := TreeFileStyle
	if node.IsDir {
		leftStyle = TreeFolderStyle
	}
	out := leftStyle.Render(left)
	if rw > 0 {
		rightStyle := CardMetaStyle
		if met {
			rightStyle = MetTargetStyle
		}
		out += " " + rightStyle.Render(right)
	}
	return out
}

// padRight pads s with spaces to w runes (no-op if already >= w).
func padRight(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}
