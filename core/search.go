package core

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// SearchResult is one match within a scene's body text.
type SearchResult struct {
	Path    string
	Title   string
	LineNum int    // 1-indexed line within the body (body only; frontmatter excluded)
	Line    string // matching line, trimmed and capped at 120 chars
}

// SearchScenes performs a case-insensitive full-text search of all .md files
// under root. The exports/ and .git/ directories are skipped. Results are
// returned in filesystem walk order (stable within a run, not sorted by relevance).
func SearchScenes(root, query string) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}
	q := strings.ToLower(query)
	var results []SearchResult

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			name := d.Name()
			if name == "exports" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		results = append(results, searchFile(path, q)...)
		return nil
	})
	return results, err
}

// searchFile returns all body lines in path that contain q (case-insensitive).
func searchFile(path, q string) []SearchResult {
	sc, err := LoadScene(path)
	if err != nil {
		return nil
	}
	title := sc.Meta.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	var results []SearchResult
	for i, line := range strings.Split(sc.Body, "\n") {
		if strings.Contains(strings.ToLower(line), q) {
			text := strings.TrimSpace(line)
			if len([]rune(text)) > 120 {
				text = string([]rune(text)[:120]) + "…"
			}
			results = append(results, SearchResult{
				Path:    path,
				Title:   title,
				LineNum: i + 1,
				Line:    text,
			})
		}
	}
	return results
}
