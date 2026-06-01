package tui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acgh213/chisel/core"
)

// BinderModel is a custom file tree for navigating folders and .md files.
// The tree data (core.FileNode) comes from core; the binder owns only the
// view state: cursor, scroll offset, focus, and size.
type BinderModel struct {
	root   string
	nodes  []*core.FileNode // root-level nodes (the tree)
	flat   []*core.FileNode // flattened visible nodes for rendering
	cursor int              // index in flat list
	offset int              // scroll offset
	focus  bool
	width  int
	height int
}

// NewBinder creates a binder model rooted at the given directory.
func NewBinder(root string) BinderModel {
	return BinderModel{
		root: root,
	}
}

// Init initializes the binder model.
func (m BinderModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input for the binder.
func (m BinderModel) Update(msg tea.Msg) (BinderModel, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		if m.cursor < len(m.flat)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter", " ":
		if m.cursor >= 0 && m.cursor < len(m.flat) {
			node := m.flat[m.cursor]
			if node.IsDir {
				node.Expanded = !node.Expanded
				m.rebuildFlat()
			}
		}
	case "l", "right":
		// Expand selected folder or no-op.
		if m.cursor >= 0 && m.cursor < len(m.flat) {
			node := m.flat[m.cursor]
			if node.IsDir && !node.Expanded {
				node.Expanded = true
				m.rebuildFlat()
			}
		}
	case "h", "left":
		// Collapse selected folder.
		if m.cursor >= 0 && m.cursor < len(m.flat) {
			node := m.flat[m.cursor]
			if node.IsDir && node.Expanded {
				node.Expanded = false
				m.rebuildFlat()
			}
		}
	}

	// Keep cursor in visible range.
	m.scrollToCursor()

	return m, nil
}

// View renders the binder tree.
func (m BinderModel) View() string {
	// Subtract only the border (padding is already inside lipgloss .Width()),
	// so the rendered box is exactly m.width × m.height.
	style := BinderStyle.
		Width(m.width - BinderStyle.GetHorizontalBorderSize()).
		Height(m.height - BinderStyle.GetVerticalBorderSize())
	if m.focus {
		style = FocusedStyle(style)
	}

	if len(m.flat) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(ColorDim).
			Padding(1).
			Render("(empty project)")
		return style.Render(empty)
	}

	// Render visible window of flattened nodes (box height minus border + padding).
	visibleHeight := m.height - BinderStyle.GetVerticalFrameSize()
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	start := m.offset
	end := start + visibleHeight
	if end > len(m.flat) {
		end = len(m.flat)
		start = end - visibleHeight
		if start < 0 {
			start = 0
		}
	}

	var lines []string
	for i := start; i < end; i++ {
		node := m.flat[i]
		line := m.renderNode(node, i == m.cursor)
		lines = append(lines, line)
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// renderNode renders a single tree node line.
func (m BinderModel) renderNode(node *core.FileNode, selected bool) string {
	indent := strings.Repeat("  ", node.Depth)

	var prefix string
	if node.IsDir {
		if node.Expanded {
			prefix = "▾ "
		} else {
			prefix = "▸ "
		}
	} else {
		prefix = "  "
	}

	display := indent + prefix + node.Name

	// Append a status glyph for scenes that carry one (shape-coded progression:
	// ○ draft → ◐ revised → ● done). Inherits the line's color.
	if !node.IsDir {
		if g := statusGlyph(node.Status); g != "" {
			display += "  " + g
		}
	}

	if selected && m.focus {
		return TreeSelectedStyle.Render(display)
	}
	if node.IsDir {
		return TreeFolderStyle.Render(display)
	}
	return TreeFileStyle.Render(display)
}

// statusGlyph maps a scene status to a single-character indicator, or "" for
// no/unknown status.
func statusGlyph(s core.Status) string {
	switch s {
	case core.StatusDraft:
		return "○"
	case core.StatusRevised:
		return "◐"
	case core.StatusDone:
		return "●"
	default:
		return ""
	}
}

// SelectedFile returns the path of the currently selected node if it's an .md file.
func (m BinderModel) SelectedFile() string {
	if m.cursor < 0 || m.cursor >= len(m.flat) {
		return ""
	}
	node := m.flat[m.cursor]
	if node.IsDir {
		return ""
	}
	return node.Path
}

// SelectedNode returns the currently selected node (file or directory), or nil.
func (m BinderModel) SelectedNode() *core.FileNode {
	if m.cursor < 0 || m.cursor >= len(m.flat) {
		return nil
	}
	return m.flat[m.cursor]
}

// SelectPath moves the cursor to the node at path, if visible. Used to restore
// cursor position after a CRUD operation that may have moved/created a node.
func (m *BinderModel) SelectPath(path string) {
	for i, n := range m.flat {
		if n.Path == path {
			m.cursor = i
			m.scrollToCursor()
			return
		}
	}
}

// RefreshPreservingExpanded rescans the project directory like Refresh, but
// restores the expanded/collapsed state of any folders that still exist after
// the rescan. Use this for CRUD operations so the user's open folders don't
// collapse unexpectedly.
func (m *BinderModel) RefreshPreservingExpanded() error {
	// Collect currently-expanded directory paths.
	expanded := make(map[string]bool)
	for _, n := range m.flat {
		if n.IsDir && n.Expanded {
			expanded[n.Path] = true
		}
	}

	if err := m.Refresh(); err != nil {
		return err
	}

	if len(expanded) > 0 {
		restoreExpanded(m.nodes, expanded)
		m.rebuildFlat()
	}
	return nil
}

// restoreExpanded walks nodes and re-sets Expanded on any directory whose path
// is in the expanded set.
func restoreExpanded(nodes []*core.FileNode, expanded map[string]bool) {
	for _, n := range nodes {
		if n.IsDir && expanded[n.Path] {
			n.Expanded = true
		}
		if len(n.Children) > 0 {
			restoreExpanded(n.Children, expanded)
		}
	}
}

// IsDirSelected returns true if the selected node is a directory.
func (m BinderModel) IsDirSelected() bool {
	if m.cursor < 0 || m.cursor >= len(m.flat) {
		return false
	}
	return m.flat[m.cursor].IsDir
}

// CurrentDir returns the folder the selection lives in: the selected node itself
// when it's a directory, the parent directory when it's a file, or the project
// root when nothing is selected. Used to scope the corkboard to a folder.
func (m BinderModel) CurrentDir() string {
	if m.cursor < 0 || m.cursor >= len(m.flat) {
		return m.root
	}
	node := m.flat[m.cursor]
	if node.IsDir {
		return node.Path
	}
	return filepath.Dir(node.Path)
}

// Focus sets focus state.
func (m *BinderModel) Focus(v bool) {
	m.focus = v
}

// SetSize sets the binder dimensions.
func (m *BinderModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Refresh rescans the root directory and rebuilds the tree.
func (m *BinderModel) Refresh() error {
	nodes, err := core.NewProject(m.root).BuildTree()
	if err != nil {
		return err
	}
	m.nodes = nodes
	m.rebuildFlat()
	if m.cursor >= len(m.flat) {
		m.cursor = len(m.flat) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return nil
}

// rebuildFlat rebuilds the flat list from the tree nodes.
func (m *BinderModel) rebuildFlat() {
	m.flat = nil
	core.Flatten(m.nodes, &m.flat)
}

// scrollToCursor keeps the cursor visible.
func (m *BinderModel) scrollToCursor() {
	visibleHeight := m.height - BinderStyle.GetVerticalFrameSize()
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visibleHeight {
		m.offset = m.cursor - visibleHeight + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}
