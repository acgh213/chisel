package core

// ToggleBookmark adds path to bookmarks if not present, removes it if present.
// Returns true if added, false if removed. Deduplicates.
func ToggleBookmark(cfg *ChiselConfig, path string) bool {
	for i, b := range cfg.Bookmarks {
		if b == path {
			cfg.Bookmarks = append(cfg.Bookmarks[:i], cfg.Bookmarks[i+1:]...)
			return false
		}
	}
	cfg.Bookmarks = append(cfg.Bookmarks, path)
	return true
}

// IsBookmarked checks if path is in bookmarks.
func IsBookmarked(cfg *ChiselConfig, path string) bool {
	for _, b := range cfg.Bookmarks {
		if b == path {
			return true
		}
	}
	return false
}

// BookmarkPaths returns the bookmarks list (copy).
func BookmarkPaths(cfg *ChiselConfig) []string {
	if len(cfg.Bookmarks) == 0 {
		return nil
	}
	out := make([]string, len(cfg.Bookmarks))
	copy(out, cfg.Bookmarks)
	return out
}
