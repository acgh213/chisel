package core

import "testing"

func TestToggleBookmark(t *testing.T) {
	cfg := &ChiselConfig{}

	// Add a bookmark.
	if got := ToggleBookmark(cfg, "scenes/ch1.md"); !got {
		t.Fatal("expected true (added)")
	}
	if len(cfg.Bookmarks) != 1 || cfg.Bookmarks[0] != "scenes/ch1.md" {
		t.Fatalf("unexpected bookmarks after add: %v", cfg.Bookmarks)
	}

	// Toggle (remove) the same bookmark.
	if got := ToggleBookmark(cfg, "scenes/ch1.md"); got {
		t.Fatal("expected false (removed)")
	}
	if len(cfg.Bookmarks) != 0 {
		t.Fatalf("expected empty bookmarks after remove, got %v", cfg.Bookmarks)
	}

	// Toggle again (re-add).
	if got := ToggleBookmark(cfg, "scenes/ch1.md"); !got {
		t.Fatal("expected true (re-added)")
	}

	// Dedup: toggling the same path again should remove (not duplicate).
	if got := ToggleBookmark(cfg, "scenes/ch1.md"); got {
		t.Fatal("expected false (dedup toggle removes)")
	}
	if len(cfg.Bookmarks) != 0 {
		t.Fatalf("expected 0 bookmarks after dedup toggle, got %d: %v", len(cfg.Bookmarks), cfg.Bookmarks)
	}

	// Dedup: toggling an already-present path removes it.
	cfg.Bookmarks = []string{"a.md"}
	if got := ToggleBookmark(cfg, "a.md"); got {
		t.Fatal("expected false (already present, toggled off)")
	}
	if len(cfg.Bookmarks) != 0 {
		t.Fatalf("expected 0 bookmarks after toggle-off, got %d: %v", len(cfg.Bookmarks), cfg.Bookmarks)
	}
}

func TestToggleBookmarkDedup(t *testing.T) {
	cfg := &ChiselConfig{Bookmarks: []string{"a.md", "b.md", "c.md"}}
	// Remove middle.
	if got := ToggleBookmark(cfg, "b.md"); got {
		t.Fatal("expected false")
	}
	if len(cfg.Bookmarks) != 2 || cfg.Bookmarks[0] != "a.md" || cfg.Bookmarks[1] != "c.md" {
		t.Fatalf("unexpected bookmarks: %v", cfg.Bookmarks)
	}
}

func TestIsBookmarked(t *testing.T) {
	cfg := &ChiselConfig{Bookmarks: []string{"a.md", "b.md"}}
	if !IsBookmarked(cfg, "a.md") {
		t.Fatal("expected a.md to be bookmarked")
	}
	if IsBookmarked(cfg, "c.md") {
		t.Fatal("expected c.md to not be bookmarked")
	}
	if IsBookmarked(&ChiselConfig{}, "a.md") {
		t.Fatal("expected false on empty bookmarks")
	}
}

func TestBookmarkPaths(t *testing.T) {
	cfg := &ChiselConfig{Bookmarks: []string{"x.md", "y.md"}}
	paths := BookmarkPaths(cfg)
	if len(paths) != 2 || paths[0] != "x.md" || paths[1] != "y.md" {
		t.Fatalf("unexpected paths: %v", paths)
	}
	// Mutation of the copy must not affect the original.
	paths[0] = "z.md"
	if cfg.Bookmarks[0] != "x.md" {
		t.Fatal("BookmarkPaths returned a reference instead of a copy")
	}
	// Empty.
	if got := BookmarkPaths(&ChiselConfig{}); got != nil {
		t.Fatalf("expected nil for empty, got %v", got)
	}
}
