package core

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Streak holds the current and longest consecutive-day writing streaks.
type Streak struct {
	Current int // consecutive days ending at today (or yesterday, if today has no commits yet)
	Longest int // best run ever
}

// DayCount holds the word-count snapshot for a single calendar day.
type DayCount struct {
	Date  time.Time
	Words int // total project words at end of day
	Delta int // words written that day (positive when words increased)
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

// DailyActivity groups all commits by UTC calendar day, returning a map from
// "2006-01-02" → revisions (newest first within each day). Used by the heatmap
// for color intensity and day-detail drilldown.
func DailyActivity(gb *GitBackend) (map[string][]Revision, error) {
	revs, err := gb.AllLog()
	if err != nil {
		return nil, err
	}

	byDay := make(map[string][]Revision)
	for _, r := range revs {
		y, m, d := r.Timestamp.UTC().Date()
		key := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		byDay[key] = append(byDay[key], r)
	}
	return byDay, nil
}

// DailyWordCounts reconstructs total project word counts over time by parsing
// commit messages. Each commit message carries per-scene word counts ("scene:
// name.md — N words"). The function tracks the latest word count for each file
// and sums them to produce a per-day total and delta. Days with no commits
// carry forward the previous total with zero delta.
func DailyWordCounts(gb *GitBackend) ([]DayCount, error) {
	byDay, err := DailyActivity(gb)
	if err != nil {
		return nil, err
	}
	if len(byDay) == 0 {
		return nil, nil
	}

	// Sort days oldest-first.
	var days []string
	for d := range byDay {
		days = append(days, d)
	}
	sort.Strings(days)

	fileWords := make(map[string]int) // file basename → latest known word count
	var result []DayCount
	prevTotal := 0

	for _, day := range days {
		commits := byDay[day]
		seen := make(map[string]bool)

		// Process commits within the day (already newest-first from DailyActivity).
		for _, rev := range commits {
			file, wc := parseCommitMessage(rev.Message)
			if file == "" || seen[file] {
				continue
			}
			seen[file] = true
			fileWords[file] = wc
		}

		// Sum all known file word counts for the project total.
		total := 0
		for _, wc := range fileWords {
			total += wc
		}

		t, err := time.Parse("2006-01-02", day)
		if err != nil {
			return nil, fmt.Errorf("parsing date %q: %w", day, err)
		}
		result = append(result, DayCount{
			Date:  t,
			Words: total,
			Delta: total - prevTotal,
		})
		prevTotal = total
	}

	return result, nil
}

// parseCommitMessage extracts the file basename and word count from a chisel
// commit message. Format: "scene: <basename> — <N> words". Returns ("", 0) on
// parse failure.
func parseCommitMessage(msg string) (string, int) {
	// Expected: "scene: ch01.md — 1,247 words"
	if !strings.HasPrefix(msg, "scene: ") {
		return "", 0
	}
	rest := strings.TrimPrefix(msg, "scene: ")

	// Split on " — " to separate filename from word count.
	parts := strings.SplitN(rest, " — ", 2)
	if len(parts) != 2 {
		return "", 0
	}
	file := parts[0]

	// Parse "<N> words" or "<N> words" (with comma).
	wcPart := strings.TrimSuffix(parts[1], " words")
	wcPart = strings.ReplaceAll(wcPart, ",", "")
	n, err := strconv.Atoi(wcPart)
	if err != nil {
		return "", 0
	}
	return file, n
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
