package tui

import (
	"os"
	"path/filepath"
	"testing"
)

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
	if !node.IsDir {
		t.Fatal("expected first node to be a directory")
	}
	node.Expanded = true
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
