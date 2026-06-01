package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildTimeline_DatedBeforeUndated(t *testing.T) {
	dir := t.TempDir()

	later := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("undated.md", "---\ntitle: No Date\n---\nbody\n")
	write("later.md", "---\ntitle: Later\ntimeline_date: 2026-04-15T00:00:00Z\n---\nbody\n")
	write("earlier.md", "---\ntitle: Earlier\ntimeline_date: 2026-03-01T00:00:00Z\n---\nbody\n")

	entries, err := BuildTimeline(dir)
	if err != nil {
		t.Fatalf("BuildTimeline: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	// Dated entries come first, in ascending date order.
	if entries[0].Title != "Earlier" {
		t.Errorf("entries[0] = %q, want Earlier", entries[0].Title)
	}
	if !entries[0].TimelineDate.Equal(earlier) {
		t.Errorf("entries[0].TimelineDate = %v, want %v", entries[0].TimelineDate, earlier)
	}
	if entries[1].Title != "Later" {
		t.Errorf("entries[1] = %q, want Later", entries[1].Title)
	}
	if !entries[1].TimelineDate.Equal(later) {
		t.Errorf("entries[1].TimelineDate = %v, want %v", entries[1].TimelineDate, later)
	}

	// Undated entry comes last.
	if entries[2].Title != "No Date" {
		t.Errorf("entries[2] = %q, want No Date", entries[2].Title)
	}
	if entries[2].TimelineDate != nil {
		t.Errorf("entries[2].TimelineDate = %v, want nil", entries[2].TimelineDate)
	}
}

func TestBuildTimeline_UndatedAlphabetical(t *testing.T) {
	dir := t.TempDir()

	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("c.md", "---\ntitle: Charlie\n---\nbody\n")
	write("a.md", "---\ntitle: Alpha\n---\nbody\n")
	write("b.md", "---\ntitle: Bravo\n---\nbody\n")

	entries, err := BuildTimeline(dir)
	if err != nil {
		t.Fatalf("BuildTimeline: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	titles := []string{entries[0].Title, entries[1].Title, entries[2].Title}
	want := []string{"Alpha", "Bravo", "Charlie"}
	for i := range want {
		if titles[i] != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, titles[i], want[i])
		}
	}
}

func TestBuildTimeline_ExcludesExports(t *testing.T) {
	dir := t.TempDir()

	exportsDir := filepath.Join(dir, "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "scene.md"), []byte("body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exportsDir, "manuscript.md"), []byte("compiled\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := BuildTimeline(dir)
	if err != nil {
		t.Fatalf("BuildTimeline: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (exports/ must be excluded)", len(entries))
	}
	if filepath.Base(entries[0].Path) != "scene.md" {
		t.Errorf("entry = %q, want scene.md", filepath.Base(entries[0].Path))
	}
}

func TestTimelineDateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.md")

	td := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	sc := Scene{
		Path: path,
		Meta: Metadata{
			Title:        "Test",
			TimelineDate: &td,
		},
		Body: "some prose\n",
	}
	if err := sc.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadScene(path)
	if err != nil {
		t.Fatalf("LoadScene: %v", err)
	}
	if loaded.Meta.TimelineDate == nil {
		t.Fatal("TimelineDate is nil after round-trip")
	}
	if !loaded.Meta.TimelineDate.Equal(td) {
		t.Errorf("TimelineDate = %v, want %v", loaded.Meta.TimelineDate, td)
	}
}
