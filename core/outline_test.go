package core

import (
	"os"
	"path/filepath"
	"testing"
)

func writeScene(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return p
}

func TestReadSceneInfo(t *testing.T) {
	dir := t.TempDir()

	// A scene with full frontmatter: title overrides name, word count comes
	// from the body (not any stored word_count).
	p := writeScene(t, dir, "alpha.md", "---\ntitle: The Opening\nstatus: revised\nsynopsis: A start.\nword_target: 1000\ndraft_order: 2\nword_count: 999\n---\none two three four\n")
	info := ReadSceneInfo(p)

	if info.Name != "alpha" {
		t.Errorf("Name = %q, want alpha", info.Name)
	}
	if info.Title != "The Opening" {
		t.Errorf("Title = %q, want The Opening", info.Title)
	}
	if info.Status != StatusRevised {
		t.Errorf("Status = %q, want revised", info.Status)
	}
	if info.Synopsis != "A start." {
		t.Errorf("Synopsis = %q", info.Synopsis)
	}
	if info.WordTarget != 1000 {
		t.Errorf("WordTarget = %d, want 1000", info.WordTarget)
	}
	if info.DraftOrder != 2 {
		t.Errorf("DraftOrder = %d, want 2", info.DraftOrder)
	}
	if info.WordCount != 4 { // counted from the body, not the stale word_count: 999
		t.Errorf("WordCount = %d, want 4 (from body)", info.WordCount)
	}
}

func TestReadSceneInfoPlainFile(t *testing.T) {
	dir := t.TempDir()
	p := writeScene(t, dir, "plain.md", "# Heading\n\njust some prose here\n")
	info := ReadSceneInfo(p)

	if info.Title != "plain" { // no frontmatter title -> falls back to name
		t.Errorf("Title = %q, want plain", info.Title)
	}
	if info.Status != "" {
		t.Errorf("Status = %q, want empty", info.Status)
	}
	// "# Heading" + "just some prose here" = 6 whitespace-delimited tokens
	// (the lone "#" counts, since WordCount splits on whitespace only).
	if info.WordCount != 6 {
		t.Errorf("WordCount = %d, want 6", info.WordCount)
	}
}

func TestFolderScenesOrdering(t *testing.T) {
	dir := t.TempDir()

	// Two ordered scenes (out of order on disk) plus two unordered ones whose
	// names are reverse-alphabetical, to prove both tiers sort correctly.
	writeScene(t, dir, "z-ordered.md", "---\ndraft_order: 1\n---\nbody\n")
	writeScene(t, dir, "a-ordered.md", "---\ndraft_order: 2\n---\nbody\n")
	writeScene(t, dir, "zeta.md", "plain\n")
	writeScene(t, dir, "alpha.md", "plain\n")
	// Noise that must be ignored.
	writeScene(t, dir, ".hidden.md", "plain\n")
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	scenes, err := FolderScenes(dir)
	if err != nil {
		t.Fatalf("FolderScenes: %v", err)
	}

	var got []string
	for _, s := range scenes {
		got = append(got, s.Name)
	}
	want := []string{"z-ordered", "a-ordered", "alpha", "zeta"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}
