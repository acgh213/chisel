package core

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// maxFrontmatterLine bounds a single frontmatter line when scanning lazily, so a
// pathological file can't blow up memory. Frontmatter lines are short in practice.
const maxFrontmatterLine = 1 << 20 // 1 MiB

// frontmatterDelim is the line that opens and closes a YAML frontmatter block.
const frontmatterDelim = "---"

// Status is a scene's revision status.
type Status string

const (
	StatusDraft   Status = "draft"
	StatusRevised Status = "revised"
	StatusDone    Status = "done"
)

// Metadata is a scene's YAML frontmatter. Every field is optional; zero values
// mean "unset" and are omitted on serialization, so a scene only carries the
// metadata it actually uses. Field declaration order is the on-disk key order
// (yaml.v3 marshals struct fields in order), which keeps round-trips stable.
type Metadata struct {
	Title      string     `yaml:"title,omitempty"`
	Status     Status     `yaml:"status,omitempty"`
	Synopsis   string     `yaml:"synopsis,omitempty"`
	Tags       []string   `yaml:"tags,omitempty"`
	DraftOrder int        `yaml:"draft_order,omitempty"`
	WordTarget int        `yaml:"word_target,omitempty"`
	POV        string     `yaml:"pov,omitempty"`
	WordCount    int        `yaml:"word_count,omitempty"`
	Created      *time.Time `yaml:"created,omitempty"`
	Modified     *time.Time `yaml:"modified,omitempty"`
	TimelineDate *time.Time `yaml:"timeline_date,omitempty"`
}

// IsEmpty reports whether the metadata carries no information. An empty scene is
// written as body-only (no frontmatter block), so plain markdown stays plain.
func (m Metadata) IsEmpty() bool {
	return m.Title == "" &&
		m.Status == "" &&
		m.Synopsis == "" &&
		len(m.Tags) == 0 &&
		m.DraftOrder == 0 &&
		m.WordTarget == 0 &&
		m.POV == "" &&
		m.WordCount == 0 &&
		m.Created == nil &&
		m.Modified == nil &&
		m.TimelineDate == nil
}

// parseFrontmatterInto splits raw content using splitFrontmatter and
// unmarshals the YAML block into dst (which must be a pointer to a struct
// with yaml struct tags). Returns the prose body and ok=true on success;
// returns raw and ok=false on any parse or unmarshal failure, preserving
// the forgiving-load invariant shared by all frontmatter types.
func parseFrontmatterInto(raw string, dst interface{}) (body string, ok bool) {
	yamlBlock, b, found := splitFrontmatter(raw)
	if !found {
		return raw, false
	}
	if err := yaml.Unmarshal([]byte(yamlBlock), dst); err != nil {
		return raw, false
	}
	return b, true
}

// splitFrontmatter splits raw content at the YAML frontmatter delimiters. It
// returns the YAML block (without the --- lines), the prose body, and ok=true.
// When raw has no valid frontmatter block (no leading delimiter, no closing
// delimiter, or CRLF/LF issues) it returns "", raw, false so callers can degrade
// gracefully rather than duplicating this CRLF-aware scanning.
func splitFrontmatter(raw string) (yamlBlock, body string, ok bool) {
	if !strings.HasPrefix(raw, frontmatterDelim+"\n") && !strings.HasPrefix(raw, frontmatterDelim+"\r\n") {
		return "", raw, false
	}
	// SplitAfter keeps the newline on each element so the body can be
	// reassembled byte-for-byte.
	lines := strings.SplitAfter(raw, "\n")
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r\n") == frontmatterDelim {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		return "", raw, false
	}
	return strings.Join(lines[1:closeIdx], ""), strings.Join(lines[closeIdx+1:], ""), true
}

// parseFrontmatter splits raw file content into metadata and the prose body.
//
// It is deliberately forgiving: a file with no leading frontmatter block — or
// one whose YAML fails to parse — returns zero metadata and the *entire* content
// as the body. A bad header never fails the load; the scene just opens as plain
// prose. The body is preserved byte-for-byte (including its leading/trailing
// newlines) so editing round-trips cleanly.
func parseFrontmatter(raw string) (Metadata, string) {
	yamlBlock, body, ok := splitFrontmatter(raw)
	if !ok {
		return Metadata{}, raw
	}
	var meta Metadata
	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		return Metadata{}, raw
	}
	return meta, body
}

// emptyChecker is satisfied by any metadata type that can report whether it
// carries no information. Used by serializeFrontmatter to decide whether to
// emit a frontmatter block.
type emptyChecker interface {
	IsEmpty() bool
}

// serializeFrontmatter is the single serialization path for all frontmatter
// types (scene metadata, character metadata, …). When meta.IsEmpty() it
// returns just the body so plain files never sprout an empty header.
func serializeFrontmatter(meta emptyChecker, body string) (string, error) {
	if meta.IsEmpty() {
		return body, nil
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(meta); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return frontmatterDelim + "\n" + buf.String() + frontmatterDelim + "\n" + body, nil
}

// serializeScene renders scene metadata + body into the on-disk representation.
func serializeScene(meta Metadata, body string) (string, error) {
	return serializeFrontmatter(meta, body)
}

// readMetadata reads only the metadata for a scene file, best-effort. Errors
// (missing file, bad YAML) yield zero metadata. Used to annotate the binder.
//
// It reads only through the closing frontmatter delimiter rather than the whole
// file, so annotating the tree never loads prose bodies into memory — BuildTree
// stays cheap even for projects with many large scenes. The result matches what
// parseFrontmatter would return for the same file's metadata.
func readMetadata(path string) Metadata {
	f, err := os.Open(path)
	if err != nil {
		return Metadata{}
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxFrontmatterLine)

	// The file must open with a delimiter line, else there's no frontmatter.
	if !sc.Scan() || strings.TrimRight(sc.Text(), "\r") != frontmatterDelim {
		return Metadata{}
	}

	var block strings.Builder
	for sc.Scan() {
		if strings.TrimRight(sc.Text(), "\r") == frontmatterDelim {
			// Closing delimiter reached — parse what we collected.
			var meta Metadata
			if err := yaml.Unmarshal([]byte(block.String()), &meta); err != nil {
				return Metadata{}
			}
			return meta
		}
		block.WriteString(sc.Text())
		block.WriteByte('\n')
	}

	// No closing delimiter (or a scan error) — treat as no metadata, matching
	// parseFrontmatter's handling of an unterminated block.
	return Metadata{}
}
