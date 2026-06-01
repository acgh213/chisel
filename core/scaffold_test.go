package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"My Novel", "my-novel"},
		{"SHORT STORIES", "short-stories"},
		{"hello world!", "hello-world"},
		{"  leading spaces  ", "leading-spaces"},
		{"already-slug", "already-slug"},
		{"with_underscores", "with-underscores"},
		{"multiple   spaces", "multiple-spaces"},
		{"123 Numbers", "123-numbers"},
		{"!@#$%", ""},
	}
	for _, c := range cases {
		got := Slugify(c.in)
		if got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseTemplate(t *testing.T) {
	if tmpl, ok := ParseTemplate("novel"); !ok || tmpl != TemplateNovel {
		t.Error("ParseTemplate(novel) failed")
	}
	if _, ok := ParseTemplate("unknown"); ok {
		t.Error("ParseTemplate(unknown) should return false")
	}
}

func TestScaffoldProject_Minimal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-project")
	if err := ScaffoldProject(dir, ScaffoldOptions{Name: "My Project", Template: TemplateMinimal}); err != nil {
		t.Fatalf("ScaffoldProject minimal: %v", err)
	}
	checkFileExists(t, filepath.Join(dir, "README.md"))

	// Minimal should not create scenes/ or other subdirs.
	if _, err := os.Stat(filepath.Join(dir, "scenes")); !os.IsNotExist(err) {
		t.Error("minimal template should not create scenes/")
	}
}

func TestScaffoldProject_Novel(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-novel")
	if err := ScaffoldProject(dir, ScaffoldOptions{Name: "My Novel", Template: TemplateNovel}); err != nil {
		t.Fatalf("ScaffoldProject novel: %v", err)
	}

	checkFileExists(t, filepath.Join(dir, "README.md"))
	checkDirExists(t, filepath.Join(dir, "scenes"))
	checkDirExists(t, filepath.Join(dir, "characters"))
	checkDirExists(t, filepath.Join(dir, "locations"))
	checkFileExists(t, filepath.Join(dir, "scenes", "ch01-opening.md"))
	checkFileExists(t, filepath.Join(dir, "scenes", "ch02-rising-action.md"))

	// Load-back: ch01 must parse correctly with the expected metadata.
	sc, err := LoadScene(filepath.Join(dir, "scenes", "ch01-opening.md"))
	if err != nil {
		t.Fatalf("LoadScene ch01: %v", err)
	}
	if sc.Meta.Title != "Opening" {
		t.Errorf("ch01 Title = %q, want %q", sc.Meta.Title, "Opening")
	}
	if sc.Meta.Status != StatusDraft {
		t.Errorf("ch01 Status = %q, want draft", sc.Meta.Status)
	}
	if sc.Meta.DraftOrder != 1 {
		t.Errorf("ch01 DraftOrder = %d, want 1", sc.Meta.DraftOrder)
	}
	if sc.Body == "" {
		t.Error("ch01 Body should not be empty")
	}

	// Load-back: ch02 must have draft_order 2.
	sc2, err := LoadScene(filepath.Join(dir, "scenes", "ch02-rising-action.md"))
	if err != nil {
		t.Fatalf("LoadScene ch02: %v", err)
	}
	if sc2.Meta.DraftOrder != 2 {
		t.Errorf("ch02 DraftOrder = %d, want 2", sc2.Meta.DraftOrder)
	}
}

func TestScaffoldProject_ShortStories(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "stories")
	if err := ScaffoldProject(dir, ScaffoldOptions{Name: "Stories", Template: TemplateShortStories}); err != nil {
		t.Fatalf("ScaffoldProject short-stories: %v", err)
	}

	checkFileExists(t, filepath.Join(dir, "story-01.md"))

	sc, err := LoadScene(filepath.Join(dir, "story-01.md"))
	if err != nil {
		t.Fatalf("LoadScene story-01: %v", err)
	}
	if sc.Meta.Status != StatusDraft {
		t.Errorf("story-01 Status = %q, want draft", sc.Meta.Status)
	}
	if sc.Meta.DraftOrder != 1 {
		t.Errorf("story-01 DraftOrder = %d, want 1", sc.Meta.DraftOrder)
	}
}

func TestScaffoldProject_NonEmptyTargetErrors(t *testing.T) {
	dir := t.TempDir()
	writeScene(t, dir, "existing.md", "content\n")

	if err := ScaffoldProject(dir, ScaffoldOptions{Name: "X", Template: TemplateMinimal}); err == nil {
		t.Error("expected error for non-empty target directory, got nil")
	}
}

func TestScaffoldProject_CreatesDir(t *testing.T) {
	// Target directory does not exist yet — ScaffoldProject must create it.
	dir := filepath.Join(t.TempDir(), "new-dir")
	if err := ScaffoldProject(dir, ScaffoldOptions{Name: "New", Template: TemplateMinimal}); err != nil {
		t.Fatalf("ScaffoldProject: %v", err)
	}
	checkDirExists(t, dir)
}

func checkFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected file %s: %v", path, err)
		return
	}
	if info.IsDir() {
		t.Errorf("%s is a directory, expected a file", path)
	}
}

func checkDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected directory %s: %v", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("%s is a file, expected a directory", path)
	}
}
