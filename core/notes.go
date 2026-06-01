package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AppendScratch appends a timestamped entry to <root>/notes/scratch.md,
// creating the notes/ directory and the file if they don't already exist.
// scratch.md has no frontmatter — it's a plain append-only markdown journal.
// Each entry is one line in the format: <!-- YYYY-MM-DD HH:MM --> text
func AppendScratch(root, text string) error {
	dir := filepath.Join(root, "notes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "scratch.md")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04")
	_, err = fmt.Fprintf(f, "<!-- %s --> %s\n", ts, text)
	return err
}
