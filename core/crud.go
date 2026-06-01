package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateFolder creates a new directory named name inside dir. Returns the new
// path, or an error if the directory already exists or cannot be created.
func CreateFolder(dir, name string) (string, error) {
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("'%s' already exists", name)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// RenameNode renames the file or directory at path so that its basename
// becomes newName. For .md scene files the .md extension is added automatically
// if newName does not already carry it. Returns the new full path, or an error
// if the target already exists or the rename fails.
func RenameNode(path, newName string) (string, error) {
	dir := filepath.Dir(path)
	if filepath.Ext(path) == ".md" && filepath.Ext(newName) != ".md" {
		newName += ".md"
	}
	newPath := filepath.Join(dir, newName)
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("'%s' already exists", newName)
	}
	if err := os.Rename(path, newPath); err != nil {
		return "", err
	}
	return newPath, nil
}

// DeleteNode removes path. For directories it removes the entire subtree
// (equivalent to rm -rf). The caller is responsible for confirming with the
// user before calling this — there is no undo.
func DeleteNode(path string) error {
	return os.RemoveAll(path)
}
