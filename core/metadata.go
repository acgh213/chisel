package core

import (
	"bytes"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

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
	WordCount  int        `yaml:"word_count,omitempty"`
	Created    *time.Time `yaml:"created,omitempty"`
	Modified   *time.Time `yaml:"modified,omitempty"`
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
		m.Modified == nil
}

// parseFrontmatter splits raw file content into metadata and the prose body.
//
// It is deliberately forgiving: a file with no leading frontmatter block — or
// one whose YAML fails to parse — returns zero metadata and the *entire* content
// as the body. A bad header never fails the load; the scene just opens as plain
// prose. The body is preserved byte-for-byte (including its leading/trailing
// newlines) so editing round-trips cleanly.
func parseFrontmatter(raw string) (Metadata, string) {
	// Must open with a delimiter line.
	if !strings.HasPrefix(raw, frontmatterDelim+"\n") && !strings.HasPrefix(raw, frontmatterDelim+"\r\n") {
		return Metadata{}, raw
	}

	// Keep newlines on each line so the body can be reassembled exactly.
	lines := strings.SplitAfter(raw, "\n")

	// Find the closing delimiter line (after the opening one).
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r\n") == frontmatterDelim {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		// Unterminated block — treat the whole file as body.
		return Metadata{}, raw
	}

	yamlBlock := strings.Join(lines[1:closeIdx], "")
	body := strings.Join(lines[closeIdx+1:], "")

	var meta Metadata
	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		// Malformed frontmatter — degrade to plain body rather than erroring.
		return Metadata{}, raw
	}
	return meta, body
}

// serializeScene renders metadata + body into the on-disk representation. When
// the metadata is empty it returns just the body (no frontmatter), so plain
// files never sprout an empty header.
func serializeScene(meta Metadata, body string) (string, error) {
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

// readMetadata reads only the metadata for a scene file, best-effort. Errors
// (missing file, bad YAML) yield zero metadata. Used to annotate the binder
// without loading whole scenes into memory.
func readMetadata(path string) Metadata {
	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}
	}
	meta, _ := parseFrontmatter(string(data))
	return meta
}
