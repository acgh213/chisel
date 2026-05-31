package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTree(t *testing.T) {
	// Create a temp directory structure.
	tmp := t.TempDir()

	// Create folders.
	os.MkdirAll(filepath.Join(tmp, "ch01"), 0755)
	os.MkdirAll(filepath.Join(tmp, "ch02"), 0755)
	os.MkdirAll(filepath.Join(tmp, "research"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".hidden"), 0755) // should be skipped

	// Create .md files.
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# README"), 0644)
	os.WriteFile(filepath.Join(tmp, "ch01", "arrival.md"), []byte("# Arrival"), 0644)
	os.WriteFile(filepath.Join(tmp, "ch01", "the-garden.md"), []byte("# Garden"), 0644)
	os.WriteFile(filepath.Join(tmp, "ch02", "escape.md"), []byte("# Escape"), 0644)
	os.WriteFile(filepath.Join(tmp, "research", "notes.md"), []byte("# Notes"), 0644)
	// Non-.md files should be skipped.
	os.WriteFile(filepath.Join(tmp, "notes.txt"), []byte("not visible"), 0644)
	// Hidden files should be skipped.
	os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte("hidden"), 0644)

	nodes, err := NewProject(tmp).BuildTree()
	if err != nil {
		t.Fatalf("BuildTree failed: %v", err)
	}

	// We expect: ch01/, ch02/, research/, README (the .md stripped)
	// Hidden dirs and non-.md files should be absent.
	if len(nodes) != 4 {
		t.Errorf("expected 4 root nodes, got %d", len(nodes))
		for _, n := range nodes {
			t.Logf("  node: name=%q isDir=%v", n.Name, n.IsDir)
		}
	}

	// Verify ordering (case-insensitive alphabetical, dirs then files).
	expected := []struct {
		name  string
		isDir bool
	}{
		{"ch01", true},
		{"ch02", true},
		{"research", true},
		{"README", false},
	}
	for i, exp := range expected {
		if i >= len(nodes) {
			break
		}
		if nodes[i].Name != exp.name || nodes[i].IsDir != exp.isDir {
			t.Errorf("node[%d]: expected %q (isDir=%v), got %q (isDir=%v)",
				i, exp.name, exp.isDir, nodes[i].Name, nodes[i].IsDir)
		}
	}

	// Verify ch01 has two children (arrival, the-garden).
	ch01 := nodes[0]
	if len(ch01.Children) != 2 {
		t.Errorf("ch01: expected 2 children, got %d", len(ch01.Children))
	}
}

func TestFlatten(t *testing.T) {
	// Build a simple tree.
	root := &FileNode{Name: "ch01", IsDir: true, Expanded: true, Depth: 0, Children: []*FileNode{
		{Name: "arrival", IsDir: false, Depth: 1},
		{Name: "garden", IsDir: false, Depth: 1},
	}}
	leaf := &FileNode{Name: "notes", IsDir: false, Depth: 0}

	var flat []*FileNode
	Flatten([]*FileNode{root, leaf}, &flat)

	if len(flat) != 4 {
		t.Errorf("expected 4 flat nodes, got %d", len(flat))
	}

	// Verify depth values are preserved.
	depths := []int{0, 1, 1, 0}
	for i, n := range flat {
		if n.Depth != depths[i] {
			t.Errorf("node[%d] %q: expected depth %d, got %d", i, n.Name, depths[i], n.Depth)
		}
	}
}

func TestFlattenCollapsed(t *testing.T) {
	// A collapsed folder must not contribute its children to the flat list.
	root := &FileNode{Name: "ch01", IsDir: true, Expanded: false, Depth: 0, Children: []*FileNode{
		{Name: "arrival", IsDir: false, Depth: 1},
	}}

	var flat []*FileNode
	Flatten([]*FileNode{root}, &flat)

	if len(flat) != 1 {
		t.Errorf("collapsed folder: expected 1 flat node, got %d", len(flat))
	}
}
