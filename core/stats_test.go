package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseCommitMessage(t *testing.T) {
	tests := []struct {
		msg      string
		wantFile string
		wantWC   int
	}{
		{"scene: ch01.md — 1,247 words", "ch01.md", 1247},
		{"scene: scene.md — 42 words", "scene.md", 42},
		{"scene: ch01.md — 0 words", "ch01.md", 0},
		{"not a scene commit", "", 0},
		{"scene: no separator", "", 0},
		{"", "", 0},
		{"scene: file.md — 1,000,000 words", "file.md", 1000000},
	}
	for _, tt := range tests {
		file, wc := parseCommitMessage(tt.msg)
		if file != tt.wantFile || wc != tt.wantWC {
			t.Errorf("parseCommitMessage(%q) = (%q, %d), want (%q, %d)",
				tt.msg, file, wc, tt.wantFile, tt.wantWC)
		}
	}
}

func TestDailyWordCounts_Integration(t *testing.T) {
	dir := t.TempDir()
	gb, err := OpenGitBackend(dir)
	if err != nil {
		t.Fatalf("OpenGitBackend: %v", err)
	}

	// Day 1: write scene1 with 100 words.
	f1 := filepath.Join(dir, "scene1.md")
	if err := os.WriteFile(f1, []byte(strings.Repeat("word ", 100)), 0644); err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(f1, "scene: scene1.md — 100 words"); err != nil {
		t.Fatal(err)
	}

	// Day 2: write scene2 with 50 words, scene1 unchanged (still 100).
	f2 := filepath.Join(dir, "scene2.md")
	if err := os.WriteFile(f2, []byte(strings.Repeat("word ", 50)), 0644); err != nil {
		t.Fatal(err)
	}
	// scene1 unchanged — snapshot it too so git has a commit for the day
	if err := gb.Snapshot(f1, "scene: scene1.md — 100 words"); err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(f2, "scene: scene2.md — 50 words"); err != nil {
		t.Fatal(err)
	}

	counts, err := DailyWordCounts(gb)
	if err != nil {
		t.Fatalf("DailyWordCounts: %v", err)
	}

	// Both commits happen in the same second, so they land on the same UTC day.
	// The test verifies that the last word count per file is used and summed correctly.
	if len(counts) != 1 {
		t.Fatalf("expected 1 day (same-second commits), got %d", len(counts))
	}

	// scene1 (100) + scene2 (50) = 150 total project words.
	if counts[0].Words != 150 {
		t.Errorf("total words: got %d, want 150", counts[0].Words)
	}
	if counts[0].Delta != 150 {
		t.Errorf("delta: got %d, want 150", counts[0].Delta)
	}
}

func TestComputeStreak_Empty(t *testing.T) {
	s := ComputeStreak(nil)
	if s.Current != 0 || s.Longest != 0 {
		t.Errorf("empty days: got current=%d longest=%d, want 0/0", s.Current, s.Longest)
	}
}

func TestComputeStreak_SingleDay(t *testing.T) {
	today := floorUTC(time.Now())
	s := ComputeStreak([]time.Time{today})
	if s.Current != 1 {
		t.Errorf("current streak: got %d, want 1", s.Current)
	}
	if s.Longest != 1 {
		t.Errorf("longest streak: got %d, want 1", s.Longest)
	}
}

func TestComputeStreak_ConsecutiveEndingToday(t *testing.T) {
	today := floorUTC(time.Now())
	days := []time.Time{
		today.AddDate(0, 0, -4),
		today.AddDate(0, 0, -3),
		today.AddDate(0, 0, -2),
		today.AddDate(0, 0, -1),
		today,
	}
	s := ComputeStreak(days)
	if s.Current != 5 {
		t.Errorf("current streak: got %d, want 5", s.Current)
	}
	if s.Longest != 5 {
		t.Errorf("longest streak: got %d, want 5", s.Longest)
	}
}

func TestComputeStreak_EndingYesterday(t *testing.T) {
	today := floorUTC(time.Now())
	yesterday := today.AddDate(0, 0, -1)
	days := []time.Time{
		yesterday.AddDate(0, 0, -2),
		yesterday.AddDate(0, 0, -1),
		yesterday,
	}
	s := ComputeStreak(days)
	if s.Current != 3 {
		t.Errorf("current streak (ending yesterday): got %d, want 3", s.Current)
	}
}

func TestComputeStreak_GapResetsCurrent(t *testing.T) {
	today := floorUTC(time.Now())
	// Wrote 3 days ago but not yesterday or today — gap breaks the streak.
	days := []time.Time{
		today.AddDate(0, 0, -5),
		today.AddDate(0, 0, -4),
		today.AddDate(0, 0, -3),
	}
	s := ComputeStreak(days)
	if s.Current != 0 {
		t.Errorf("current streak with gap: got %d, want 0", s.Current)
	}
	if s.Longest != 3 {
		t.Errorf("longest streak: got %d, want 3", s.Longest)
	}
}

func TestComputeStreak_LongestInPast(t *testing.T) {
	today := floorUTC(time.Now())
	days := []time.Time{
		// A 4-day run in the distant past.
		today.AddDate(0, 0, -20),
		today.AddDate(0, 0, -19),
		today.AddDate(0, 0, -18),
		today.AddDate(0, 0, -17),
		// Gap.
		// A 2-day run ending yesterday.
		today.AddDate(0, 0, -2),
		today.AddDate(0, 0, -1),
	}
	s := ComputeStreak(days)
	if s.Current != 2 {
		t.Errorf("current streak: got %d, want 2", s.Current)
	}
	if s.Longest != 4 {
		t.Errorf("longest streak: got %d, want 4", s.Longest)
	}
}

func TestActiveDays_Integration(t *testing.T) {
	// Create a temp dir, init git, make two commits on the same day.
	dir := t.TempDir()
	gb, err := OpenGitBackend(dir)
	if err != nil {
		t.Fatalf("OpenGitBackend: %v", err)
	}

	// Commit a file.
	f1 := filepath.Join(dir, "scene1.md")
	if err := os.WriteFile(f1, []byte("day one"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(f1, "first commit"); err != nil {
		t.Fatal(err)
	}

	// Commit another file — same day, so still 1 unique active day.
	f2 := filepath.Join(dir, "scene2.md")
	if err := os.WriteFile(f2, []byte("day two"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(f2, "second commit"); err != nil {
		t.Fatal(err)
	}

	days, err := ActiveDays(gb)
	if err != nil {
		t.Fatalf("ActiveDays: %v", err)
	}

	// Both commits happened today, so we expect 1 unique day.
	if len(days) != 1 {
		t.Errorf("expected 1 active day, got %d: %v", len(days), days)
	}

	// The day should be today (UTC).
	expected := floorUTC(time.Now())
	if !days[0].Equal(expected) {
		t.Errorf("expected %v, got %v", expected, days[0])
	}
}
