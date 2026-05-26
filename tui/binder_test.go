package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildFileTree(t *testing.T) {
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

	nodes, err := buildFileTree(tmp, tmp, 0)
	if err != nil {
		t.Fatalf("buildFileTree failed: %v", err)
	}

	// We expect: ch01/, ch02/, research/, README (the .md stripped)
	// Hidden dirs and non-.md files should be absent.
	if len(nodes) != 4 {
		t.Errorf("expected 4 root nodes, got %d", len(nodes))
		for _, n := range nodes {
			t.Logf("  node: name=%q isDir=%v", n.name, n.isDir)
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
		if nodes[i].name != exp.name || nodes[i].isDir != exp.isDir {
			t.Errorf("node[%d]: expected %q (isDir=%v), got %q (isDir=%v)",
				i, exp.name, exp.isDir, nodes[i].name, nodes[i].isDir)
		}
	}

	// Verify ch01 has two children (arrival, the-garden).
	ch01 := nodes[0]
	if len(ch01.children) != 2 {
		t.Errorf("ch01: expected 2 children, got %d", len(ch01.children))
	}
}

func TestBinderNewAndRefresh(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "scene.md"), []byte("# Test"), 0644)
	os.MkdirAll(filepath.Join(tmp, "acts"), 0755)
	os.WriteFile(filepath.Join(tmp, "acts", "act1.md"), []byte("# Act 1"), 0644)

	b := NewBinder(tmp)
	if err := b.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	if len(b.flat) != 2 { // acts/ (collapsed) + scene
		t.Errorf("expected 2 flat nodes (collapsed dir + file), got %d", len(b.flat))
	}

	// Expand acts by setting it via cursor.
	b.cursor = 0 // acts/
	node := b.flat[0]
	if !node.isDir {
		t.Fatal("expected first node to be a directory")
	}
	node.expanded = true
	b.rebuildFlat()

	if len(b.flat) != 3 { // acts/, act1, scene
		t.Errorf("expected 3 flat nodes after expand, got %d", len(b.flat))
	}

	// Verify SelectedFile returns path for .md files.
	b.cursor = 1 // act1
	path := b.SelectedFile()
	if path == "" {
		t.Error("expected SelectedFile to return path for act1")
	}

	// Verify SelectedFile returns empty for directories.
	b.cursor = 0 // acts/
	path = b.SelectedFile()
	if path != "" {
		t.Error("expected SelectedFile to return empty for directory")
	}
}

func TestFlattenNodes(t *testing.T) {
	// Build a simple tree.
	root := &fileNode{name: "ch01", isDir: true, expanded: true, depth: 0, children: []*fileNode{
		{name: "arrival", isDir: false, depth: 1},
		{name: "garden", isDir: false, depth: 1},
	}}
	leaf := &fileNode{name: "notes", isDir: false, depth: 0}

	var flat []*fileNode
	flattenNodes([]*fileNode{root, leaf}, &flat)

	if len(flat) != 4 {
		t.Errorf("expected 4 flat nodes, got %d", len(flat))
	}

	// Verify depth values are preserved.
	depths := []int{0, 1, 1, 0}
	for i, n := range flat {
		if n.depth != depths[i] {
			t.Errorf("node[%d] %q: expected depth %d, got %d", i, n.name, depths[i], n.depth)
		}
	}
}
