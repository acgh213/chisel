package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitBackendRoundTrip exercises the full snapshot → log → diff → restore
// flow against a real (temporary) git repo, including init-on-first-use and the
// clean-tree no-op.
func TestGitBackendRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.md")

	v1 := "# Scene\n\nFirst version.\n"
	if err := os.WriteFile(path, []byte(v1), scenePerm); err != nil {
		t.Fatal(err)
	}

	// Init-on-first-use.
	gb, err := OpenGitBackend(dir)
	if err != nil {
		t.Fatalf("OpenGitBackend: %v", err)
	}

	if err := gb.Snapshot(path, "scene: scene — 3 words"); err != nil {
		t.Fatalf("first snapshot: %v", err)
	}

	// Re-snapshot with no change → no-op, no new revision.
	if err := gb.Snapshot(path, "scene: scene — 3 words"); err != nil {
		t.Fatalf("no-op snapshot should not error: %v", err)
	}

	// Change the file and snapshot again.
	v2 := "# Scene\n\nSecond version, expanded a little.\n"
	if err := os.WriteFile(path, []byte(v2), scenePerm); err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(path, "scene: scene — 5 words"); err != nil {
		t.Fatalf("second snapshot: %v", err)
	}

	// Log: exactly two revisions, newest first.
	revs, err := gb.Log(path)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(revs) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(revs))
	}
	if !strings.Contains(revs[0].Message, "5 words") {
		t.Errorf("newest revision message = %q, want it to mention 5 words", revs[0].Message)
	}
	if !strings.Contains(revs[1].Message, "3 words") {
		t.Errorf("oldest revision message = %q, want it to mention 3 words", revs[1].Message)
	}
	if revs[0].Short() == "" || len(revs[0].Short()) > 7 {
		t.Errorf("Short() = %q, want non-empty <= 7 chars", revs[0].Short())
	}

	// Diff between the two revisions mentions the new text.
	diff, err := gb.Diff(path, revs[1].Hash, revs[0].Hash)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, "Second version") {
		t.Errorf("diff did not mention the new content:\n%s", diff)
	}

	// Restore the older revision's content.
	restored, err := gb.Restore(path, revs[1].Hash)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored != v1 {
		t.Errorf("restored content = %q, want %q", restored, v1)
	}
}
