package core

import (
	"sort"
	"time"
)

// Streak holds the current and longest consecutive-day writing streaks.
type Streak struct {
	Current int // consecutive days ending at today (or yesterday, if today has no commits yet)
	Longest int // best run ever
}

// ActiveDays returns the set of calendar days (UTC) that have at least one
// commit in the repository, sorted oldest-first.
func ActiveDays(gb *GitBackend) ([]time.Time, error) {
	revs, err := gb.AllLog()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var days []time.Time
	for _, r := range revs {
		y, m, d := r.Timestamp.UTC().Date()
		key := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		if seen[key] {
			continue
		}
		seen[key] = true
		days = append(days, time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
	}

	// Sort oldest-first for streak computation.
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days, nil
}

// ComputeStreak derives current and longest streaks from a sorted list of
// active writing days (oldest-first). A "current" streak is consecutive days
// ending at today; if today has no commit yet, it ends at yesterday. A gap of
// more than one day resets the current streak.
func ComputeStreak(days []time.Time) Streak {
	if len(days) == 0 {
		return Streak{}
	}

	today := floorUTC(time.Now())
	yesterday := today.AddDate(0, 0, -1)

	// Build a set for O(1) lookup.
	set := make(map[string]bool, len(days))
	for _, d := range days {
		set[d.Format("2006-01-02")] = true
	}

	// Current streak: walk backward from today (or yesterday) until a gap.
	current := 0
	cursor := today
	if !set[cursor.Format("2006-01-02")] {
		cursor = yesterday
	}
	for set[cursor.Format("2006-01-02")] {
		current++
		cursor = cursor.AddDate(0, 0, -1)
	}

	// Longest streak: scan the sorted list for the max consecutive run.
	longest := 0
	run := 0
	var prev time.Time
	for i, d := range days {
		if i == 0 {
			run = 1
		} else if d.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > longest {
			longest = run
		}
		prev = d
	}

	// The longest streak should be at least as long as the current one
	// (current streak may have started before the earliest active day).
	if current > longest {
		longest = current
	}

	return Streak{Current: current, Longest: longest}
}

// floorUTC returns the current time truncated to midnight UTC.
func floorUTC(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
