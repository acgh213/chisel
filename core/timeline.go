package core

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TimelineEntry is one scene as it appears in the timeline view. TimelineDate
// is the story-internal date (distinct from the file's created/modified times)
// and is optional — scenes without it appear in an "Undated" section.
type TimelineEntry struct {
	Path         string
	Title        string
	TimelineDate *time.Time
	Status       Status
	WordCount    int
}

// BuildTimeline walks the whole project tree (excluding exports/, characters/,
// locations/, notes/, and hidden entries) and returns scenes sorted for the
// timeline view: dated entries first in ascending TimelineDate order, then
// undated entries sorted case-insensitively by title.
func BuildTimeline(root string) ([]TimelineEntry, error) {
	var entries []TimelineEntry

	err := walkMarkdown(root, func(path string) error {
		base := strings.TrimSuffix(filepath.Base(path), ".md")
		entry := TimelineEntry{Path: path, Title: base}

		if sc, err := LoadScene(path); err == nil {
			if sc.Meta.Title != "" {
				entry.Title = sc.Meta.Title
			}
			entry.Status = sc.Meta.Status
			entry.TimelineDate = sc.Meta.TimelineDate
			entry.WordCount = WordCount(sc.Body)
		}

		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sortTimeline(entries)
	return entries, nil
}

// sortTimeline orders entries: dated entries first (ascending TimelineDate),
// then undated entries alphabetically by title (case-insensitive).
func sortTimeline(entries []TimelineEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		ad, bd := a.TimelineDate != nil, b.TimelineDate != nil
		if ad != bd {
			return ad // dated before undated
		}
		if ad && bd {
			return a.TimelineDate.Before(*b.TimelineDate)
		}
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	})
}
