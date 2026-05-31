package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWordCount(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  spaced   out  words ", 3},
		{"line one\nline two", 4},
		{"tabs\tand\nnewlines here", 4},
	}
	for _, c := range cases {
		if got := WordCount(c.in); got != c.want {
			t.Errorf("WordCount(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestSceneRoundTrip covers a plain file with no frontmatter: it loads as
// body-only and stays plain across an edit + save (no frontmatter injected).
func TestSceneRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "scene.md")
	content := "# A Scene\n\nSome prose.\n"

	if err := os.WriteFile(path, []byte(content), scenePerm); err != nil {
		t.Fatal(err)
	}

	sc, err := LoadScene(path)
	if err != nil {
		t.Fatalf("LoadScene: %v", err)
	}
	if !sc.Meta.IsEmpty() {
		t.Errorf("plain file should load with empty metadata, got %+v", sc.Meta)
	}
	if sc.Body != content {
		t.Errorf("loaded body = %q, want %q", sc.Body, content)
	}

	sc.Body = "# Edited\n\nNew prose.\n"
	if err := sc.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// A plain file must remain plain — no frontmatter block on disk.
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != sc.Body {
		t.Errorf("plain file should stay plain on save: on disk = %q, want %q", onDisk, sc.Body)
	}

	reloaded, err := LoadScene(path)
	if err != nil {
		t.Fatalf("LoadScene after save: %v", err)
	}
	if reloaded.Body != sc.Body {
		t.Errorf("reloaded body = %q, want %q", reloaded.Body, sc.Body)
	}
}

func TestCreateScene(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "new.md")

	sc, err := CreateScene(path)
	if err != nil {
		t.Fatalf("CreateScene: %v", err)
	}
	if sc.Body != newSceneBody {
		t.Errorf("new scene body = %q, want %q", sc.Body, newSceneBody)
	}
	// New scenes are seeded with draft status and join the metadata system.
	if sc.Meta.Status != StatusDraft {
		t.Errorf("new scene status = %q, want %q", sc.Meta.Status, StatusDraft)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist on disk: %v", err)
	}

	// Reloading should recover the draft status and the body.
	reloaded, err := LoadScene(path)
	if err != nil {
		t.Fatalf("LoadScene: %v", err)
	}
	if reloaded.Meta.Status != StatusDraft {
		t.Errorf("reloaded status = %q, want %q", reloaded.Meta.Status, StatusDraft)
	}
	if reloaded.Body != newSceneBody {
		t.Errorf("reloaded body = %q, want %q", reloaded.Body, newSceneBody)
	}
}
