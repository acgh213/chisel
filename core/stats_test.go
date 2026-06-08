package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
