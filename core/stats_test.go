package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeStreakEmpty(t *testing.T) {
	s := ComputeStreak(nil)
	if s.Current != 0 || s.Longest != 0 {
		t.Errorf("empty: got %+v, want {0 0}", s)
	}
}

func TestComputeStreakSingleToday(t *testing.T) {
	today := FloorUTC(time.Now())
	s := ComputeStreak([]time.Time{today})
	if s.Current != 1 || s.Longest != 1 {
		t.Errorf("single today: got %+v, want {1 1}", s)
	}
}

func TestComputeStreakSingleYesterday(t *testing.T) {
	yesterday := FloorUTC(time.Now()).AddDate(0, 0, -1)
	s := ComputeStreak([]time.Time{yesterday})
	if s.Current != 1 || s.Longest != 1 {
		t.Errorf("single yesterday: got %+v, want {1 1}", s)
	}
}

func TestComputeStreakSingleOldDay(t *testing.T) {
	old := FloorUTC(time.Now()).AddDate(0, 0, -5)
	s := ComputeStreak([]time.Time{old})
	if s.Current != 0 || s.Longest != 1 {
		t.Errorf("single old day: got %+v, want {0 1}", s)
	}
}

func TestComputeStreakConsecutive(t *testing.T) {
	now := FloorUTC(time.Now())
	days := []time.Time{
		now.AddDate(0, 0, -3),
		now.AddDate(0, 0, -2),
		now.AddDate(0, 0, -1),
		now,
	}
	s := ComputeStreak(days)
	if s.Current != 4 || s.Longest != 4 {
		t.Errorf("4 consecutive: got %+v, want {4 4}", s)
	}
}

func TestComputeStreakGap(t *testing.T) {
	now := FloorUTC(time.Now())
	// Two streaks: [today, yesterday] and [4 days ago, 5 days ago].
	days := []time.Time{
		now.AddDate(0, 0, -5),
		now.AddDate(0, 0, -4),
		now.AddDate(0, 0, -1),
		now,
	}
	s := ComputeStreak(days)
	if s.Current != 2 || s.Longest != 2 {
		t.Errorf("gap: got %+v, want {2 2}", s)
	}
}

func TestComputeStreakLongestBiggerThanCurrent(t *testing.T) {
	now := FloorUTC(time.Now())
	// A long old streak + a short current streak.
	days := []time.Time{
		now.AddDate(0, 0, -10),
		now.AddDate(0, 0, -9),
		now.AddDate(0, 0, -8),
		now.AddDate(0, 0, -7),
		now.AddDate(0, 0, -6),
		now.AddDate(0, 0, -1),
		now,
	}
	s := ComputeStreak(days)
	if s.Current != 2 || s.Longest != 5 {
		t.Errorf("longest>current: got %+v, want {2 5}", s)
	}
}

func TestFloorUTC(t *testing.T) {
	// A time with non-zero hour/minute/second should be truncated.
	input := time.Date(2025, 6, 15, 14, 30, 45, 999, time.Local)
	got := FloorUTC(input)
	want := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("FloorUTC = %v, want %v", got, want)
	}
}

func TestActiveDaysFromGit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.md")

	if err := os.WriteFile(path, []byte("# Scene\n\nHello.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gb, err := OpenGitBackend(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(path, "scene: scene — 2 words"); err != nil {
		t.Fatal(err)
	}

	days, err := ActiveDays(gb)
	if err != nil {
		t.Fatalf("ActiveDays: %v", err)
	}
	if len(days) != 1 {
		t.Fatalf("expected 1 active day, got %d", len(days))
	}
	if !FloorUTC(time.Now()).Equal(days[0]) {
		t.Errorf("active day = %v, want today %v", days[0], FloorUTC(time.Now()))
	}
}

func TestDailyActivityFromGit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.md")

	if err := os.WriteFile(path, []byte("# Scene\n\nHello.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gb, err := OpenGitBackend(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(path, "scene: scene — 2 words"); err != nil {
		t.Fatal(err)
	}

	act, err := DailyActivity(gb)
	if err != nil {
		t.Fatalf("DailyActivity: %v", err)
	}
	today := FloorUTC(time.Now()).Format("2006-01-02")
	if len(act[today]) != 1 {
		t.Errorf("expected 1 commit today, got %d", len(act[today]))
	}
}

func TestDailyWordCountsFromGit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.md")

	v1 := "# Scene\n\nHello world.\n"
	if err := os.WriteFile(path, []byte(v1), 0644); err != nil {
		t.Fatal(err)
	}
	gb, err := OpenGitBackend(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(path, "scene: scene — 2 words"); err != nil {
		t.Fatal(err)
	}

	v2 := "# Scene\n\nHello world, expanded with more words.\n"
	if err := os.WriteFile(path, []byte(v2), 0644); err != nil {
		t.Fatal(err)
	}
	if err := gb.Snapshot(path, "scene: scene — 7 words"); err != nil {
		t.Fatal(err)
	}

	counts, err := DailyWordCounts(gb)
	if err != nil {
		t.Fatalf("DailyWordCounts: %v", err)
	}
	if len(counts) != 1 {
		t.Fatalf("expected 1 day, got %d", len(counts))
	}
	if counts[0].Total != 9 { // 2 + 7
		t.Errorf("total = %d, want 9", counts[0].Total)
	}
}
