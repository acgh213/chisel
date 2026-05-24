package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// scene node
// ---------------------------------------------------------------------------

// SceneNode represents a node in the binder tree — either a directory or a
// .md scene file.
type SceneNode struct {
	Name     string      // display name (filename or dir name)
	RelPath  string      // relative path from project root
	IsDir    bool
	Children []*SceneNode
	Expanded bool
	Depth    int
}

// ---------------------------------------------------------------------------
// binder model
// ---------------------------------------------------------------------------

// BinderModel manages the scene tree panel.
type BinderModel struct {
	nodes   []*SceneNode // root-level nodes
	cursor  int          // index into the linearised node list
	linear  []*SceneNode // flattened visible nodes (computed)
	entries []ManifestEntry
}

// NewBinderModel scans the scenes/ directory and builds the tree.
func NewBinderModel(projectDir string, entries []ManifestEntry) BinderModel {
	bm := BinderModel{
		entries: entries,
	}
	bm.nodes = scanScenesDir(projectDir)
	bm.relinearise()
	if len(bm.linear) > 0 {
		bm.cursor = 0
	}
	return bm
}

// scanScenesDir recursively reads the scenes/ directory tree.
func scanScenesDir(projectDir string) []*SceneNode {
	root := filepath.Join(projectDir, "scenes")
	return scanDir(root, root, 0)
}

func scanDir(root, current string, depth int) []*SceneNode {
	entries, err := os.ReadDir(current)
	if err != nil {
		return nil
	}

	// Sort: directories first, then files; alphabetical within each group.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	var nodes []*SceneNode
	for _, e := range entries {
		name := e.Name()

		// Skip hidden files/dirs and non-.md files.
		if strings.HasPrefix(name, ".") {
			continue
		}
		fullPath := filepath.Join(current, name)
		relPath, _ := filepath.Rel(filepath.Dir(root), fullPath)
		relPath = filepath.ToSlash(relPath)
		relPath = "scenes/" + relPath

		if e.IsDir() {
			children := scanDir(root, fullPath, depth+1)
			nodes = append(nodes, &SceneNode{
				Name:     name,
				RelPath:  relPath,
				IsDir:    true,
				Children: children,
				Expanded: depth < 1, // auto-expand first level
				Depth:    depth,
			})
		} else if strings.HasSuffix(name, ".md") {
			nodes = append(nodes, &SceneNode{
				Name:    name,
				RelPath: relPath,
				IsDir:   false,
				Depth:   depth,
			})
		}
	}
	return nodes
}

// ---------------------------------------------------------------------------
// linearisation
// ---------------------------------------------------------------------------

func (bm *BinderModel) relinearise() {
	bm.linear = nil
	flatten(bm.nodes, &bm.linear)
	// Clamp cursor.
	if bm.cursor >= len(bm.linear) {
		bm.cursor = len(bm.linear) - 1
	}
	if bm.cursor < 0 {
		bm.cursor = 0
	}
}

func flatten(nodes []*SceneNode, out *[]*SceneNode) {
	for _, n := range nodes {
		*out = append(*out, n)
		if n.IsDir && n.Expanded && len(n.Children) > 0 {
			flatten(n.Children, out)
		}
	}
}

// ---------------------------------------------------------------------------
// accessors
// ---------------------------------------------------------------------------

func (bm BinderModel) selectedNode() *SceneNode {
	if bm.cursor < 0 || bm.cursor >= len(bm.linear) {
		return nil
	}
	return bm.linear[bm.cursor]
}

func (bm BinderModel) lookupEntry(name string) *ManifestEntry {
	id := strings.TrimSuffix(name, ".md")
	for i := range bm.entries {
		if bm.entries[i].ID == id {
			return &bm.entries[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// update
// ---------------------------------------------------------------------------

func (bm BinderModel) Update(msg tea.Msg) (BinderModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if bm.cursor > 0 {
				bm.cursor--
			}
		case "down":
			if bm.cursor < len(bm.linear)-1 {
				bm.cursor++
			}
		case "left":
			node := bm.selectedNode()
			if node != nil && node.IsDir && node.Expanded {
				node.Expanded = false
				bm.relinearise()
			}
		case "right":
			node := bm.selectedNode()
			if node != nil && node.IsDir && !node.Expanded {
				node.Expanded = true
				bm.relinearise()
			}
		}
	}
	return bm, nil
}

// ---------------------------------------------------------------------------
// view
// ---------------------------------------------------------------------------

func (bm BinderModel) View(width int) string {
	if len(bm.linear) == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Foreground(ColorMuted).
			Padding(0, 1).
			Render("(empty)")
	}

	var sb strings.Builder
	for i, node := range bm.linear {
		line := bm.renderNode(node, i == bm.cursor)
		sb.WriteString(line)
		sb.WriteRune('\n')
	}

	return lipgloss.NewStyle().
		Width(width).
		MaxHeight(20). // will be constrained by parent
		Render(sb.String())
}

func (bm BinderModel) renderNode(node *SceneNode, selected bool) string {
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

	name := node.Name
	if !node.IsDir {
		// Strip .md extension for display.
		name = strings.TrimSuffix(name, ".md")
		// Append status indicator.
		if entry := bm.lookupEntry(node.Name); entry != nil {
			switch entry.Status {
			case "revised":
				name += " ·"
			case "done":
				name += " ✓"
			default:
				name += "  "
			}
		} else {
			name += "  "
		}
	}

	display := indent + prefix + name

	style := lipgloss.NewStyle()
	if selected {
		style = style.Background(ColorHighlight).Foreground(ColorAccent)
	}
	if node.IsDir {
		style = style.Foreground(ColorMuted)
	}

	return style.Render(display)
}
