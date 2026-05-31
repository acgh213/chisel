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
	if sc.Content != content {
		t.Errorf("loaded content = %q, want %q", sc.Content, content)
	}

	sc.Content = "# Edited\n\nNew prose.\n"
	if err := sc.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := LoadScene(path)
	if err != nil {
		t.Fatalf("LoadScene after save: %v", err)
	}
	if reloaded.Content != sc.Content {
		t.Errorf("reloaded content = %q, want %q", reloaded.Content, sc.Content)
	}
}

func TestCreateScene(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "new.md")

	sc, err := CreateScene(path)
	if err != nil {
		t.Fatalf("CreateScene: %v", err)
	}
	if sc.Content != newSceneTemplate {
		t.Errorf("new scene content = %q, want template %q", sc.Content, newSceneTemplate)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist on disk: %v", err)
	}
}
