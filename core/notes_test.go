package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendScratch_CreatesFileAndDir(t *testing.T) {
	dir := t.TempDir()
	if err := AppendScratch(dir, "first thought"); err != nil {
		t.Fatalf("AppendScratch: %v", err)
	}

	path := filepath.Join(dir, "notes", "scratch.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("scratch.md not created: %v", err)
	}
	if !strings.Contains(string(data), "first thought") {
		t.Errorf("scratch.md does not contain entry text: %q", data)
	}
}

func TestAppendScratch_TimestampPrefix(t *testing.T) {
	dir := t.TempDir()
	if err := AppendScratch(dir, "timestamped"); err != nil {
		t.Fatalf("AppendScratch: %v", err)
	}

	path := filepath.Join(dir, "notes", "scratch.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "<!-- ") {
		t.Errorf("entry missing timestamp prefix, got: %q", line)
	}
	if !strings.Contains(line, " --> timestamped") {
		t.Errorf("entry missing text after timestamp, got: %q", line)
	}
}

func TestAppendScratch_Accumulates(t *testing.T) {
	dir := t.TempDir()
	for _, note := range []string{"alpha", "beta", "gamma"} {
		if err := AppendScratch(dir, note); err != nil {
			t.Fatalf("AppendScratch(%q): %v", note, err)
		}
	}

	path := filepath.Join(dir, "notes", "scratch.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	for i, want := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(lines[i], want) {
			t.Errorf("line %d = %q, want to contain %q", i, lines[i], want)
		}
	}
}

func TestAppendScratch_IdempotentOnReopen(t *testing.T) {
	dir := t.TempDir()
	// Two separate calls should append, not overwrite.
	if err := AppendScratch(dir, "first"); err != nil {
		t.Fatal(err)
	}
	if err := AppendScratch(dir, "second"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "notes", "scratch.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "first") || !strings.Contains(string(data), "second") {
		t.Errorf("both entries should be present, got: %q", data)
	}
}
