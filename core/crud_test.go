package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateFolder(t *testing.T) {
	root := t.TempDir()

	path, err := CreateFolder(root, "chapter-1")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if path != filepath.Join(root, "chapter-1") {
		t.Errorf("path = %q, want %q", path, filepath.Join(root, "chapter-1"))
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		t.Errorf("expected a directory at %s", path)
	}

	// Creating again must fail.
	if _, err := CreateFolder(root, "chapter-1"); err == nil {
		t.Error("expected error creating duplicate folder, got nil")
	}
}

func TestRenameNode_File(t *testing.T) {
	root := t.TempDir()
	original := writeScene(t, root, "old-name.md", "body\n")

	newPath, err := RenameNode(original, "new-name")
	if err != nil {
		t.Fatalf("RenameNode: %v", err)
	}
	want := filepath.Join(root, "new-name.md")
	if newPath != want {
		t.Errorf("newPath = %q, want %q", newPath, want)
	}
	if _, err := os.Stat(original); !os.IsNotExist(err) {
		t.Errorf("original path should be gone: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new path not found: %v", err)
	}
}

func TestRenameNode_PreservesExtension(t *testing.T) {
	root := t.TempDir()
	original := writeScene(t, root, "scene.md", "body\n")

	// Passing new name with explicit .md — should not double the extension.
	newPath, err := RenameNode(original, "renamed.md")
	if err != nil {
		t.Fatalf("RenameNode: %v", err)
	}
	if filepath.Base(newPath) != "renamed.md" {
		t.Errorf("expected 'renamed.md', got %q", filepath.Base(newPath))
	}
}

func TestRenameNode_Directory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "old-dir")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	newPath, err := RenameNode(dir, "new-dir")
	if err != nil {
		t.Fatalf("RenameNode dir: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new dir not found: %v", err)
	}
}

func TestRenameNode_ConflictError(t *testing.T) {
	root := t.TempDir()
	writeScene(t, root, "a.md", "")
	writeScene(t, root, "b.md", "")

	if _, err := RenameNode(filepath.Join(root, "a.md"), "b"); err == nil {
		t.Error("expected conflict error, got nil")
	}
}

func TestDeleteNode_File(t *testing.T) {
	root := t.TempDir()
	path := writeScene(t, root, "to-delete.md", "body\n")

	if err := DeleteNode(path); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should be gone after delete")
	}
}

func TestDeleteNode_Directory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "chapter")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeScene(t, dir, "scene.md", "body\n")

	if err := DeleteNode(dir); err != nil {
		t.Fatalf("DeleteNode dir: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("directory should be gone after delete")
	}
}
