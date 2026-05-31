package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fileNode represents a node in the binder tree.
type fileNode struct {
	name     string
	path     string
	isDir    bool
	expanded bool
	children []*fileNode
	depth    int
}

// BinderModel is a custom file tree for navigating folders and .md files.
type BinderModel struct {
	root      string
	nodes     []*fileNode  // root-level nodes (the tree)
	flat      []*fileNode  // flattened visible nodes for rendering
	cursor    int          // index in flat list
	offset    int          // scroll offset
	focus     bool
	width     int
	height    int
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
			if node.isDir {
				node.expanded = !node.expanded
				m.rebuildFlat()
			}
		}
	case "l", "right":
		// Expand selected folder or no-op.
		if m.cursor >= 0 && m.cursor < len(m.flat) {
			node := m.flat[m.cursor]
			if node.isDir && !node.expanded {
				node.expanded = true
				m.rebuildFlat()
			}
		}
	case "h", "left":
		// Collapse selected folder.
		if m.cursor >= 0 && m.cursor < len(m.flat) {
			node := m.flat[m.cursor]
			if node.isDir && node.expanded {
				node.expanded = false
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
func (m BinderModel) renderNode(node *fileNode, selected bool) string {
	indent := strings.Repeat("  ", node.depth)

	var prefix string
	if node.isDir {
		if node.expanded {
			prefix = "▾ "
		} else {
			prefix = "▸ "
		}
	} else {
		prefix = "  "
	}

	display := indent + prefix + node.name

	if selected && m.focus {
		return TreeSelectedStyle.Render(display)
	}
	if node.isDir {
		return TreeFolderStyle.Render(display)
	}
	return TreeFileStyle.Render(display)
}

// SelectedFile returns the path of the currently selected node if it's an .md file.
func (m BinderModel) SelectedFile() string {
	if m.cursor < 0 || m.cursor >= len(m.flat) {
		return ""
	}
	node := m.flat[m.cursor]
	if node.isDir {
		return ""
	}
	return node.path
}

// IsDirSelected returns true if the selected node is a directory.
func (m BinderModel) IsDirSelected() bool {
	if m.cursor < 0 || m.cursor >= len(m.flat) {
		return false
	}
	return m.flat[m.cursor].isDir
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
	nodes, err := buildFileTree(m.root, m.root, 0)
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
	flattenNodes(m.nodes, &m.flat)
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

// flattenNodes recursively flattens visible nodes into the flat list.
func flattenNodes(nodes []*fileNode, flat *[]*fileNode) {
	for _, n := range nodes {
		*flat = append(*flat, n)
		if n.isDir && n.expanded && len(n.children) > 0 {
			flattenNodes(n.children, flat)
		}
	}
}

// buildFileTree recursively scans a directory and builds fileNode tree.
func buildFileTree(root, dir string, depth int) ([]*fileNode, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var nodes []*fileNode

	// Separate directories and .md files, sort alphabetically.
	var dirs, mdFiles []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		// Skip hidden files/directories.
		if len(name) > 0 && name[0] == '.' {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, e)
		} else if filepath.Ext(name) == ".md" {
			mdFiles = append(mdFiles, e)
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name()) < strings.ToLower(dirs[j].Name())
	})
	sort.Slice(mdFiles, func(i, j int) bool {
		return strings.ToLower(mdFiles[i].Name()) < strings.ToLower(mdFiles[j].Name())
	})

	// Add directories.
	for _, e := range dirs {
		fullPath := filepath.Join(dir, e.Name())
		name := e.Name()

		children, err := buildFileTree(root, fullPath, depth+1)
		if err != nil {
			children = nil // skip unreadable directories
		}

		node := &fileNode{
			name:     name,
			path:     fullPath,
			isDir:    true,
			expanded: false,
			children: children,
			depth:    depth,
		}
		nodes = append(nodes, node)
	}

	// Add .md files.
	for _, e := range mdFiles {
		fullPath := filepath.Join(dir, e.Name())
		name := e.Name()
		// Strip .md extension for display.
		displayName := name[:len(name)-3]

		node := &fileNode{
			name:  displayName,
			path:  fullPath,
			isDir: false,
			depth: depth,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
