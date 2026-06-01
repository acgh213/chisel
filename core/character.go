package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// CharacterMeta is the YAML frontmatter for a character file. Fields are
// deliberately minimal for Phase 8; richer data (relationships, arc notes,
// voice) is added in a later phase.
type CharacterMeta struct {
	Name        string   `yaml:"name,omitempty"`
	Role        string   `yaml:"role,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

// IsEmpty reports whether the metadata carries no information.
func (m CharacterMeta) IsEmpty() bool {
	return m.Name == "" && m.Role == "" && m.Description == "" && len(m.Tags) == 0
}

// Character is one file in the characters/ subdirectory: optional YAML
// frontmatter (Meta) plus free-form notes (Body).
type Character struct {
	Path string
	Meta CharacterMeta
	Body string
}

// DisplayName returns the character's name from metadata if set, otherwise
// the file's basename without extension.
func (c Character) DisplayName() string {
	if c.Meta.Name != "" {
		return c.Meta.Name
	}
	return strings.TrimSuffix(filepath.Base(c.Path), ".md")
}

// LoadCharacter reads a character file from disk, splitting frontmatter from body.
func LoadCharacter(path string) (*Character, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	meta, body := parseCharacterFrontmatter(string(data))
	return &Character{Path: path, Meta: meta, Body: body}, nil
}

// Save writes the character back to its path.
func (c *Character) Save() error {
	out, err := serializeCharacter(c.Meta, c.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(c.Path, []byte(out), 0o644)
}

// ListCharacters reads all character files from <root>/characters/ and returns
// them sorted by display name. A missing characters/ directory returns nil
// without error — it simply means no characters exist yet.
func ListCharacters(root string) ([]Character, error) {
	dir := filepath.Join(root, "characters")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var chars []Character
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".md" || (len(name) > 0 && name[0] == '.') {
			continue
		}
		c, err := LoadCharacter(filepath.Join(dir, name))
		if err != nil {
			continue // best-effort: skip unreadable files
		}
		chars = append(chars, *c)
	}
	sort.Slice(chars, func(i, j int) bool {
		return strings.ToLower(chars[i].DisplayName()) < strings.ToLower(chars[j].DisplayName())
	})
	return chars, nil
}

// CharactersDir returns the path to the characters/ subdirectory under root.
func CharactersDir(root string) string {
	return filepath.Join(root, "characters")
}

// parseCharacterFrontmatter uses the shared splitFrontmatter splitter so the
// CRLF-aware delimiter scanning lives in exactly one place.
func parseCharacterFrontmatter(raw string) (CharacterMeta, string) {
	yamlBlock, body, ok := splitFrontmatter(raw)
	if !ok {
		return CharacterMeta{}, raw
	}
	var meta CharacterMeta
	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		return CharacterMeta{}, raw
	}
	return meta, body
}

func serializeCharacter(meta CharacterMeta, body string) (string, error) {
	return serializeFrontmatter(meta, body)
}
