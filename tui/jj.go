package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// jj backend
// ---------------------------------------------------------------------------

// JJBackend implements RevisionBackend using the `jj` CLI. It is the v1.2
// replacement for the default git backend.
type JJBackend struct {
	dir string // project root
}

// NewJJBackend verifies that `jj` is in PATH and returns a backend.
func NewJJBackend(projectDir string) (*JJBackend, error) {
	if _, err := exec.LookPath("jj"); err != nil {
		return nil, fmt.Errorf("jj not found in PATH")
	}
	return &JJBackend{dir: projectDir}, nil
}

// Save runs `jj new` to describe the change, then `jj commit` to record it.
// In jj the working copy is auto-committed; we just set the description.
func (jb *JJBackend) Save(path string, message string) error {
	rel, err := filepath.Rel(jb.dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	// Describe the current change.
	descCmd := exec.Command("jj", "describe", "-m", message)
	descCmd.Dir = jb.dir
	if out, err := descCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("jj describe: %w — %s", err, string(out))
	}

	// New empty change so the next save is a separate snapshot.
	newCmd := exec.Command("jj", "new", "-m", "working")
	newCmd.Dir = jb.dir
	if out, err := newCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("jj new: %w — %s", err, string(out))
	}

	_ = rel
	return nil
}

// Log returns the revision history for a path by running `jj log`.
func (jb *JJBackend) Log(path string) ([]Revision, error) {
	rel, err := filepath.Rel(jb.dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	cmd := exec.Command("jj", "log", "--no-graph", "-r", "all()",
		"--template", `commit_id ++ "\n"`)

	cmd.Dir = jb.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("jj log: %w — %s", err, string(out))
	}

	var revs []Revision
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		revs = append(revs, Revision{
			Hash:      strings.TrimSpace(line),
			Timestamp: time.Now(), // jj log with full template would give this
			Message:   "",
		})
	}

	_ = rel
	return revs, nil
}

// Diff returns a unified diff between two revisions.
func (jb *JJBackend) Diff(path string, rev1, rev2 string) (string, error) {
	rel, err := filepath.Rel(jb.dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	cmd := exec.Command("jj", "diff", "-r", rev1+".."+rev2, rel)
	cmd.Dir = jb.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("jj diff: %w — %s", err, string(out))
	}

	_ = rel
	return string(out), nil
}

// Restore returns the content of a path at the given revision.
func (jb *JJBackend) Restore(path string, rev string) (string, error) {
	rel, err := filepath.Rel(jb.dir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	cmd := exec.Command("jj", "file", "show", "-r", rev, rel)
	cmd.Dir = jb.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("jj file show: %w — %s", err, string(out))
	}

	_ = rel
	return string(out), nil
}
