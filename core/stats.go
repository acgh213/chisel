package core

import (
	"regexp"
	"sort"
	"strconv"
	"time"
)

// Streak holds the current and longest writing streaks in days.
type Streak struct {
	Current int
	Longest int
}

// DayCount is a single day's total word count (parsed from commit messages).
type DayCount struct {
	Date  time.Time
	Total int
}

// FloorUTC truncates t to midnight UTC (strips the time-of-day component).
func FloorUTC(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ActiveDays returns a sorted (ascending) slice of unique UTC dates on which
// at least one commit exists. Only day-level granularity matters; all times
// are floored to midnight UTC.
func ActiveDays(gb *GitBackend) ([]time.Time, error) {
	return ActiveDaysSince(gb, time.Time{})
}

// ActiveDaysSince is like ActiveDays but only considers commits since the
// given time. If since is zero, all commits are included.
func ActiveDaysSince(gb *GitBackend, since time.Time) ([]time.Time, error) {
	revs, err := gb.AllLog(since)
	if err != nil {
		return nil, err
	}

	seen := make(map[time.Time]bool)
	var days []time.Time
	for _, r := range revs {
		d := FloorUTC(r.Timestamp)
		if !seen[d] {
			seen[d] = true
			days = append(days, d)
		}
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days, nil
}

// ComputeStreak calculates the current and longest streaks from a sorted
// (ascending) list of unique active days. Streaks are measured in calendar-day
// adjacency: day N and day N+1 are consecutive regardless of clock time. If
// today or yesterday is not in the active set the current streak is 0.
func ComputeStreak(days []time.Time) Streak {
	if len(days) == 0 {
		return Streak{}
	}

	now := FloorUTC(time.Now())

	// current streak: walk backwards from the most recent day. It only counts
	// if the most recent day is today or yesterday (i.e. the streak hasn't
	// been broken).
	last := days[len(days)-1]
	if !last.Equal(now) && !last.Equal(now.AddDate(0, 0, -1)) {
		// The streak is already broken; compute longest only.
		return Streak{Current: 0, Longest: longestRun(days)}
	}

	cur := 1
	for i := len(days) - 1; i > 0; i-- {
		// Check if days[i-1] is exactly one calendar day before days[i].
		if days[i].AddDate(0, 0, -1).Equal(days[i-1]) {
			cur++
		} else {
			break
		}
	}

	l := longestRun(days)
	if cur > l {
		l = cur
	}
	return Streak{Current: cur, Longest: l}
}

// longestRun returns the length of the longest run of calendar-consecutive
// days in the sorted (ascending) slice.
func longestRun(days []time.Time) int {
	if len(days) == 0 {
		return 0
	}
	best := 1
	cur := 1
	for i := 1; i < len(days); i++ {
		if days[i].AddDate(0, 0, -1).Equal(days[i-1]) {
			cur++
		} else {
			if cur > best {
				best = cur
			}
			cur = 1
		}
	}
	if cur > best {
		best = cur
	}
	return best
}

// DailyActivity groups all commits by UTC date (newest day first). Only
// considers commits since the given time; pass zero time for all.
func DailyActivity(gb *GitBackend) (map[string][]Revision, error) {
	revs, err := gb.AllLog(time.Time{})
	if err != nil {
		return nil, err
	}

	m := make(map[string][]Revision)
	for _, r := range revs {
		key := FloorUTC(r.Timestamp).Format("2006-01-02")
		m[key] = append(m[key], r)
	}
	return m, nil
}

// sceneWordPattern matches commit messages of the form:
// "scene: <name>.md — <N> words" (the em-dash may be a regular dash).
var sceneWordPattern = regexp.MustCompile(`scene:\s+\S+\s+[\x{2013}\x{2014}\-]\s+(\d+)\s+words`)

// DailyWordCounts parses all commit messages for word counts and returns a
// slice of DayCount sorted descending by date. Each DayCount.Total is the sum
// of word counts from all commits on that day.
func DailyWordCounts(gb *GitBackend) ([]DayCount, error) {
	revs, err := gb.AllLog(time.Time{})
	if err != nil {
		return nil, err
	}

	dayMap := make(map[string]int)
	for _, r := range revs {
		matches := sceneWordPattern.FindStringSubmatch(r.Message)
		if matches == nil {
			continue
		}
		wc, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		key := FloorUTC(r.Timestamp).Format("2006-01-02")
		dayMap[key] += wc
	}

	var counts []DayCount
	for dateStr, total := range dayMap {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		counts = append(counts, DayCount{Date: d, Total: total})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Date.After(counts[j].Date)
	})
	return counts, nil
}

// ProjectWordCount returns the total word count across all scene files in the
// project (excluding exports/, characters/, locations/, notes/, hidden dirs).
func ProjectWordCount(root string) (int, error) {
	total := 0
	err := walkMarkdown(root, func(path string) error {
		info := ReadSceneInfo(path)
		total += info.WordCount
		return nil
	})
	return total, err
}

// ReadingTime returns the estimated reading time in minutes for the given word
// count, assuming 200 words per minute. Returns at least 1 minute for any
// non-zero word count.
func ReadingTime(words int) int {
	if words <= 0 {
		return 0
	}
	mins := words / 200
	if mins < 1 {
		mins = 1
	}
	return mins
}
