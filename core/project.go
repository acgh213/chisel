// Package core holds chisel's data model — projects, scenes, file I/O — with
// zero dependency on any TUI/GUI library. Everything here is plain Go so the
// same logic can back the terminal UI today and a graphical UI later.
//
// HARD RULE: nothing in this package may import charmbracelet/* (or any UI
// toolkit). If a presentation type needs to cross this boundary, the design is
// wrong — keep core pure data.
package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileNode is one node in the project file tree. Plain data only.
//
// Expanded is folder open/closed state. It lives here (rather than in the TUI)
// because it is a simple bool with no UI dependency, and keeping it on the node
// lets tree-flattening stay in core.
type FileNode struct {
	Name     string // display name (.md extension stripped for files)
	Path     string // absolute/real path on disk
	IsDir    bool
	Expanded bool
	Children []*FileNode
	Depth    int

	// Status is the scene's frontmatter status, for files only. Empty for
	// directories and for files without a status. Populated by BuildTree.
	Status Status
}

// Project is a writing project rooted at a directory. The filesystem is the
// data model: folders and .md files are the structure.
type Project struct {
	Root string
}

// NewProject returns a project rooted at the given directory.
func NewProject(root string) Project {
	return Project{Root: root}
}

// BuildTree scans the project root and returns the top-level file nodes,
// recursively. Hidden entries (leading dot) and non-.md files are skipped.
func (p Project) BuildTree() ([]*FileNode, error) {
	return buildFileTree(p.Root, 0)
}

// Flatten appends the visible nodes (expanded folders' children included)
// depth-first into out.
func Flatten(nodes []*FileNode, out *[]*FileNode) {
	for _, n := range nodes {
		*out = append(*out, n)
		if n.IsDir && n.Expanded && len(n.Children) > 0 {
			Flatten(n.Children, out)
		}
	}
}

// buildFileTree recursively scans dir and builds the FileNode tree. Directories
// come before files; both are sorted case-insensitively.
func buildFileTree(dir string, depth int) ([]*FileNode, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

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

	var nodes []*FileNode

	for _, e := range dirs {
		fullPath := filepath.Join(dir, e.Name())
		children, err := buildFileTree(fullPath, depth+1)
		if err != nil {
			children = nil // skip unreadable directories
		}
		nodes = append(nodes, &FileNode{
			Name:     e.Name(),
			Path:     fullPath,
			IsDir:    true,
			Expanded: false,
			Children: children,
			Depth:    depth,
		})
	}

	for _, e := range mdFiles {
		fullPath := filepath.Join(dir, e.Name())
		name := e.Name()
		displayName := name[:len(name)-len(".md")] // strip .md for display
		nodes = append(nodes, &FileNode{
			Name:   displayName,
			Path:   fullPath,
			IsDir:  false,
			Depth:  depth,
			Status: readMetadata(fullPath).Status, // best-effort; empty if none
		})
	}

	return nodes, nil
}
