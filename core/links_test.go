package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindLinks(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		count int
		first string // anchor of first match
	}{
		{"no links", "Hello world", 0, ""},
		{"single", "See [[chapter-one]] for details.", 1, "chapter-one"},
		{"multiple", "Link [[a]] and [[b]] here.", 2, "a"},
		{"empty anchor", "Bad: [[]] link.", 0, ""},
		{"adjacent", "[[x]][[y]]", 2, "x"},
		{"nested brackets", "[[a]b]", 0, ""},
		{"with spaces", "Go to [[my scene]] now.", 1, "my scene"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			links := FindLinks(tc.text)
			if len(links) != tc.count {
				t.Fatalf("got %d links, want %d", len(links), tc.count)
			}
			if tc.count > 0 && links[0].Anchor != tc.first {
				t.Fatalf("first anchor = %q, want %q", links[0].Anchor, tc.first)
			}
		})
	}
}

func TestFindLinksPositions(t *testing.T) {
	text := "before [[my-link]] after"
	links := FindLinks(text)
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1", len(links))
	}
	lk := links[0]
	// "before " = 7 bytes, then "[[my-link]]" = 11 bytes
	if lk.Start != 7 {
		t.Errorf("Start = %d, want 7", lk.Start)
	}
	if lk.End != 18 {
		t.Errorf("End = %d, want 18", lk.End)
	}
	if text[lk.Start:lk.End] != "[[my-link]]" {
		t.Errorf("sliced text = %q, want %q", text[lk.Start:lk.End], "[[my-link]]")
	}
}

func TestLinkAtCursor(t *testing.T) {
	text := "line0\nSee [[my-link]] here\nline2"

	// Cursor inside [[my-link]]: line 1, col 7 (the 'm' of my-link)
	if _, ok := LinkAtCursor(text, 1, 7); !ok {
		t.Error("expected cursor at (1,7) to be inside link")
	}

	// Cursor on the opening [[: line 1, col 5
	if _, ok := LinkAtCursor(text, 1, 5); !ok {
		t.Error("expected cursor at (1,5) to be inside link")
	}

	// Cursor on the second closing ]: line 1, col 14 (the last ] of ]])
	if _, ok := LinkAtCursor(text, 1, 14); !ok {
		t.Error("expected cursor at (1,14) to be inside link")
	}

	// Cursor after the link: line 1, col 15 (space after ]])
	if anchor, ok := LinkAtCursor(text, 1, 15); ok {
		t.Errorf("expected cursor at (1,15) to be outside link, got anchor=%q", anchor)
	}

	// Cursor before the link: line 1, col 3
	if anchor, ok := LinkAtCursor(text, 1, 3); ok {
		t.Errorf("expected cursor at (1,3) to be outside link, got anchor=%q", anchor)
	}

	// Different line entirely
	if anchor, ok := LinkAtCursor(text, 0, 0); ok {
		t.Errorf("expected cursor at (0,0) to be outside link, got anchor=%q", anchor)
	}
}

func TestLinkAtCursorEdgeCases(t *testing.T) {
	// Empty text
	if anchor, ok := LinkAtCursor("", 0, 0); ok {
		t.Errorf("empty text: expected false, got anchor=%q", anchor)
	}

	// Out-of-bounds line
	if anchor, ok := LinkAtCursor("abc", 5, 0); ok {
		t.Errorf("out-of-bounds line: expected false, got anchor=%q", anchor)
	}

	// Out-of-bounds column
	if anchor, ok := LinkAtCursor("abc", 0, 10); ok {
		t.Errorf("out-of-bounds col: expected false, got anchor=%q", anchor)
	}

	// Multiple links: cursor in second
	text := "[[first]] and [[second]]"
	if anchor, ok := LinkAtCursor(text, 0, 17); !ok || anchor != "second" {
		t.Errorf("second link: got anchor=%q ok=%v, want %q true", anchor, ok, "second")
	}
}

func TestResolveLink(t *testing.T) {
	// Create a temp project with scenes, characters, and locations.
	dir := t.TempDir()

	// Scenes
	writeFile(t, filepath.Join(dir, "hello-world.md"), "---\ntitle: Hello World\n---\nbody")
	writeFile(t, filepath.Join(dir, "chapter-one.md"), "---\ntitle: The Beginning\n---\nbody")
	writeFile(t, filepath.Join(dir, "epilogue.md"), "---\ntitle: The End\n---\nbody")
	writeFile(t, filepath.Join(dir, "partial-match-scene.md"), "---\ntitle: Partial Scene\n---\nbody")

	// Characters
	charDir := filepath.Join(dir, "characters")
	os.MkdirAll(charDir, 0o755)
	writeFile(t, filepath.Join(charDir, "alice.md"), "---\nname: Alice\n---\nbody")
	writeFile(t, filepath.Join(charDir, "bob-smith.md"), "---\nname: Bob Smith\n---\nbody")

	// Locations
	locDir := filepath.Join(dir, "locations")
	os.MkdirAll(locDir, 0o755)
	writeFile(t, filepath.Join(locDir, "enchanted-forest.md"), "---\nname: Enchanted Forest\n---\nbody")

	// --- Exact filename match ---
	p, err := ResolveLink(dir, "hello-world")
	if err != nil {
		t.Fatalf("exact filename: %v", err)
	}
	if p != filepath.Join(dir, "hello-world.md") {
		t.Errorf("exact filename: got %s", p)
	}

	// --- Exact title match ---
	p, err = ResolveLink(dir, "The Beginning")
	if err != nil {
		t.Fatalf("exact title: %v", err)
	}
	if p != filepath.Join(dir, "chapter-one.md") {
		t.Errorf("exact title: got %s", p)
	}

	// --- Case-insensitive filename ---
	p, err = ResolveLink(dir, "HELLO-WORLD")
	if err != nil {
		t.Fatalf("case insensitive filename: %v", err)
	}
	if p != filepath.Join(dir, "hello-world.md") {
		t.Errorf("case insensitive filename: got %s", p)
	}

	// --- Case-insensitive title ---
	p, err = ResolveLink(dir, "the end")
	if err != nil {
		t.Fatalf("case insensitive title: %v", err)
	}
	if p != filepath.Join(dir, "epilogue.md") {
		t.Errorf("case insensitive title: got %s", p)
	}

	// --- Partial match on filename ---
	p, err = ResolveLink(dir, "partial")
	if err != nil {
		t.Fatalf("partial filename: %v", err)
	}
	if p != filepath.Join(dir, "partial-match-scene.md") {
		t.Errorf("partial filename: got %s", p)
	}

	// --- Partial match on title ---
	p, err = ResolveLink(dir, "beginning")
	if err != nil {
		t.Fatalf("partial title: %v", err)
	}
	if p != filepath.Join(dir, "chapter-one.md") {
		t.Errorf("partial title: got %s", p)
	}

	// --- Character exact match ---
	p, err = ResolveLink(dir, "Alice")
	if err != nil {
		t.Fatalf("character exact: %v", err)
	}
	if p != filepath.Join(charDir, "alice.md") {
		t.Errorf("character exact: got %s", p)
	}

	// --- Character case-insensitive ---
	p, err = ResolveLink(dir, "alice")
	if err != nil {
		t.Fatalf("character case insensitive: %v", err)
	}
	if p != filepath.Join(charDir, "alice.md") {
		t.Errorf("character case insensitive: got %s", p)
	}

	// --- Character partial match ---
	p, err = ResolveLink(dir, "Bob")
	if err != nil {
		t.Fatalf("character partial: %v", err)
	}
	if p != filepath.Join(charDir, "bob-smith.md") {
		t.Errorf("character partial: got %s", p)
	}

	// --- Location exact match ---
	p, err = ResolveLink(dir, "Enchanted Forest")
	if err != nil {
		t.Fatalf("location exact: %v", err)
	}
	if p != filepath.Join(locDir, "enchanted-forest.md") {
		t.Errorf("location exact: got %s", p)
	}

	// --- Location case-insensitive ---
	p, err = ResolveLink(dir, "enchanted forest")
	if err != nil {
		t.Fatalf("location case insensitive: %v", err)
	}
	if p != filepath.Join(locDir, "enchanted-forest.md") {
		t.Errorf("location case insensitive: got %s", p)
	}

	// --- Not found ---
	_, err = ResolveLink(dir, "does-not-exist")
	if err == nil {
		t.Fatal("expected error for non-existent link")
	}

	// --- Empty anchor ---
	_, err = ResolveLink(dir, "")
	if err == nil {
		t.Fatal("expected error for empty anchor")
	}
}


