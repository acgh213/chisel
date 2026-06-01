package core

import (
	"os"
	"time"
)

// scenePerm is the file mode used when writing scene files.
const scenePerm = 0o644

// newSceneBody is the starting prose for a freshly created scene.
const newSceneBody = "# Untitled\n\n"

// Scene is one markdown file in a project: optional YAML frontmatter (Meta) plus
// the prose Body. The on-disk file is serializeScene(Meta, Body); a file with no
// frontmatter loads with an empty Meta and the whole file as Body.
type Scene struct {
	Path string
	Meta Metadata
	Body string
}

// LoadScene reads a scene file from disk, splitting frontmatter from prose.
func LoadScene(path string) (*Scene, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseScene(path, string(data)), nil
}

// ParseScene parses raw file content (frontmatter + body) into a Scene at path,
// without touching disk. Used to load revision snapshots into the editor.
func ParseScene(path, content string) *Scene {
	meta, body := parseFrontmatter(content)
	return &Scene{Path: path, Meta: meta, Body: body}
}

// Save writes the scene back to its path. If the scene carries metadata, the
// derived fields (word_count, modified, and created on first save) are refreshed
// before serializing; a scene with no metadata is written as plain body so it
// stays plain. Refreshing derived metadata is independent of *why* Save was
// called (explicit save, autosave, etc.) — the trigger is the caller's concern.
func (s *Scene) Save() error {
	if !s.Meta.IsEmpty() {
		s.Meta.WordCount = WordCount(s.Body)
		now := time.Now()
		s.Meta.Modified = &now
		if s.Meta.Created == nil {
			created := now
			s.Meta.Created = &created
		}
	}

	out, err := serializeScene(s.Meta, s.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, []byte(out), scenePerm)
}

// CreateScene writes a new scene file (seeded with draft metadata so it joins
// the metadata system immediately) and returns it.
func CreateScene(path string) (*Scene, error) {
	s := &Scene{
		Path: path,
		Meta: Metadata{Status: StatusDraft},
		Body: newSceneBody,
	}
	if err := s.Save(); err != nil {
		return nil, err
	}
	return s, nil
}

// WordCount returns the number of whitespace-delimited words in content.
// Spaces, newlines, and tabs separate words.
func WordCount(content string) int {
	if content == "" {
		return 0
	}
	words := 0
	inWord := false
	for _, r := range content {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			inWord = false
		} else if !inWord {
			words++
			inWord = true
		}
	}
	return words
}
