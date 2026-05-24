package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ManifestEntry represents one scene's metadata line in manifest.jsonl.
type ManifestEntry struct {
	ID            string   `json:"id"`
	File          string   `json:"file"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	WordCount     int      `json:"word_count"`
	POV           string   `json:"pov"`
	DraftOrder    int      `json:"draft_order"`
	Tags          []string `json:"tags"`
	Created       string   `json:"created"`
	Modified      string   `json:"modified"`
	ResearchRefs  []string `json:"research_refs"`
	Notes         string   `json:"notes"`
}

// LoadManifest reads all entries from manifest.jsonl in the project directory.
// An empty (or missing) file returns an empty slice — it is not an error.
func LoadManifest(projectDir string) ([]ManifestEntry, error) {
	path := filepath.Join(projectDir, "manifest.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening manifest: %w", err)
	}
	defer f.Close()

	var entries []ManifestEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry ManifestEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return entries, fmt.Errorf("parsing manifest line: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("scanning manifest: %w", err)
	}

	return entries, nil
}

// SaveManifest overwrites manifest.jsonl with the given entries, one JSON
// object per line.
func SaveManifest(projectDir string, entries []ManifestEntry) error {
	path := filepath.Join(projectDir, "manifest.jsonl")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating manifest: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	for _, entry := range entries {
		if err := enc.Encode(entry); err != nil {
			return fmt.Errorf("writing manifest entry: %w", err)
		}
	}

	return nil
}

// AppendEntry appends a single entry to manifest.jsonl without rewriting the
// file. This is the normal write path — the manifest is append-only.
func AppendEntry(projectDir string, entry ManifestEntry) error {
	path := filepath.Join(projectDir, "manifest.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening manifest for append: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(entry); err != nil {
		return fmt.Errorf("appending manifest entry: %w", err)
	}

	return nil
}
