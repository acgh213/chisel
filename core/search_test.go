package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchScenesBasic(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "ch1.md"),
		"---\ntitle: Chapter One\n---\nThe dragon roared at midnight.\nShe ran through the forest.\n")
	writeFile(t, filepath.Join(dir, "ch2.md"),
		"---\ntitle: Chapter Two\n---\nThe dragon slept by the fire.\n")

	results, err := SearchScenes(dir, "dragon")
	if err != nil {
		t.Fatalf("SearchScenes: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	// Titles must come from frontmatter.
	titles := map[string]bool{}
	for _, r := range results {
		titles[r.Title] = true
	}
	if !titles["Chapter One"] || !titles["Chapter Two"] {
		t.Fatalf("unexpected titles: %v", titles)
	}
}

func TestSearchScenesCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "s.md"), "---\ntitle: Scene\n---\nDragon flew high.\n")

	r1, _ := SearchScenes(dir, "dragon")
	r2, _ := SearchScenes(dir, "DRAGON")
	r3, _ := SearchScenes(dir, "Dragon")
	if len(r1) != 1 || len(r2) != 1 || len(r3) != 1 {
		t.Fatalf("case-insensitive: got %d/%d/%d, want 1/1/1", len(r1), len(r2), len(r3))
	}
}

func TestSearchScenesNoMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "s.md"), "---\ntitle: Scene\n---\nNothing relevant here.\n")

	results, _ := SearchScenes(dir, "unicorn")
	if len(results) != 0 {
		t.Fatalf("want 0 results, got %d", len(results))
	}
}

func TestSearchScenesEmptyQuery(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "s.md"), "---\ntitle: Scene\n---\nSome text.\n")

	results, err := SearchScenes(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Fatalf("want nil for empty query, got %v", results)
	}
}

func TestSearchScenesSkipsExports(t *testing.T) {
	dir := t.TempDir()
	exportsDir := filepath.Join(dir, "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(exportsDir, "manuscript.md"), "The dragon was here.\n")

	results, err := SearchScenes(dir, "dragon")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("want 0 (exports/ skipped), got %d", len(results))
	}
}

func TestSearchScenesLineNumbers(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "s.md"),
		"---\ntitle: Scene\n---\nLine one.\nLine two with dragon.\nLine three.\n")

	results, _ := SearchScenes(dir, "dragon")
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].LineNum != 2 {
		t.Fatalf("want LineNum=2 (body line), got %d", results[0].LineNum)
	}
}

func TestSearchScenesFrontmatterNotSearched(t *testing.T) {
	dir := t.TempDir()
	// "dragon" only in frontmatter (title), not in body.
	writeFile(t, filepath.Join(dir, "s.md"),
		"---\ntitle: Dragon Tale\n---\nNothing here.\n")

	results, _ := SearchScenes(dir, "dragon")
	if len(results) != 0 {
		t.Fatalf("want 0 (frontmatter excluded from search), got %d", len(results))
	}
}

// writeFile is a test helper — writes content to path, fatals on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
