package core

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// LinkMatch holds the position and anchor text of one [[...]] link in a body.
type LinkMatch struct {
	Start  int    // byte offset of the opening [[
	End    int    // byte offset just past the closing ]]
	Anchor string // the text between [[ and ]]
}

// linkRe matches [[...]] with at least one character between the brackets.
var linkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// FindLinks returns every [[...]] pattern in text, recording byte offsets
// and the anchor text inside the brackets.
func FindLinks(text string) []LinkMatch {
	matches := linkRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]LinkMatch, 0, len(matches))
	for _, m := range matches {
		// m[0], m[1] = full match start/end; m[2], m[3] = capture group start/end
		out = append(out, LinkMatch{
			Start:  m[0],
			End:    m[1],
			Anchor: text[m[2]:m[3]],
		})
	}
	return out
}

// LinkAtCursor checks whether the cursor at the given 0-indexed (line, col)
// falls inside a [[...]] link in text. If so it returns the anchor text and
// true; otherwise it returns "", false.
func LinkAtCursor(text string, line, col int) (string, bool) {
	offset, ok := lineColToOffset(text, line, col)
	if !ok {
		return "", false
	}
	links := FindLinks(text)
	for _, lk := range links {
		if offset >= lk.Start && offset < lk.End {
			return lk.Anchor, true
		}
	}
	return "", false
}

// lineColToOffset converts a 0-indexed (line, col) position to a byte offset
// in text. The col parameter is a rune index within the line (matching the
// textarea's Column() semantics). Returns the byte offset and true on success,
// or 0, false if the position is out of range.
func lineColToOffset(text string, line, col int) (int, bool) {
	offset := 0
	curLine := 0
	runesSeen := 0
	for i := 0; i <= len(text); i++ {
		if curLine == line {
			// We are on the target line; count rune columns.
			lineByteStart := offset
			j := lineByteStart
			for runesSeen = 0; runesSeen < col; runesSeen++ {
				if j >= len(text) || text[j] == '\n' {
					return 0, false
				}
				_, size := utf8.DecodeRuneInString(text[j:])
				j += size
			}
			if j > len(text) {
				return 0, false
			}
			return j, true
		}
		if i < len(text) {
			if text[i] == '\n' {
				offset = i + 1
				curLine++
			}
		}
	}
	// Cursor on a trailing empty line.
	if curLine == line && col == 0 {
		return offset, true
	}
	return 0, false
}

// ResolveLink resolves a [[link]] anchor to an absolute file path under root.
// It searches scenes, characters, and locations, matching by:
//  1. Exact filename (without .md) or name/title — case-insensitive
//  2. Partial match (anchor is a case-insensitive substring of filename or title)
//
// Returns the first exact match found; failing that, the first partial match.
// Returns an error if nothing matches.
func ResolveLink(root, anchor string) (string, error) {
	if anchor == "" {
		return "", fmt.Errorf("empty link anchor")
	}
	aLower := strings.ToLower(anchor)

	type candidate struct {
		path    string
		name    string // display name for matching
		exact   bool   // true if this is an exact (not partial) match
	}

	var exactMatches []candidate
	var partialMatches []candidate

	// --- Scenes ---
	_ = walkMarkdown(root, func(path string) error {
		name := strings.TrimSuffix(filepath.Base(path), ".md")
		sc := readMetadata(path)
		title := sc.Title

		// Check filename match.
		if strings.EqualFold(name, aLower) {
			exactMatches = append(exactMatches, candidate{path: path, name: name, exact: true})
			return nil
		}
		// Check title match.
		if title != "" && strings.EqualFold(title, aLower) {
			exactMatches = append(exactMatches, candidate{path: path, name: title, exact: true})
			return nil
		}
		// Partial match on filename or title.
		if strings.Contains(strings.ToLower(name), aLower) ||
			(title != "" && strings.Contains(strings.ToLower(title), aLower)) {
			partialMatches = append(partialMatches, candidate{path: path, name: name})
		}
		return nil
	})

	// --- Characters ---
	chars, _ := ListCharacters(root)
	for _, c := range chars {
		name := c.DisplayName()
		if strings.EqualFold(name, aLower) {
			exactMatches = append(exactMatches, candidate{path: c.Path, name: name, exact: true})
		} else if strings.Contains(strings.ToLower(name), aLower) {
			partialMatches = append(partialMatches, candidate{path: c.Path, name: name})
		}
	}

	// --- Locations ---
	locs, _ := ListLocations(root)
	for _, l := range locs {
		name := l.DisplayName()
		if strings.EqualFold(name, aLower) {
			exactMatches = append(exactMatches, candidate{path: l.Path, name: name, exact: true})
		} else if strings.Contains(strings.ToLower(name), aLower) {
			partialMatches = append(partialMatches, candidate{path: l.Path, name: name})
		}
	}

	// Prefer exact matches; fall back to partial.
	if len(exactMatches) > 0 {
		return exactMatches[0].path, nil
	}
	if len(partialMatches) > 0 {
		return partialMatches[0].path, nil
	}
	return "", fmt.Errorf("link not found: %s", anchor)
}
