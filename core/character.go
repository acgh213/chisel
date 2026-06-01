package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CharacterMeta is the YAML frontmatter for a character file. Fields are
// deliberately minimal here; richer data (arc notes, relationships, voice)
// is added in a later phase.
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
	var meta CharacterMeta
	body, ok := parseFrontmatterInto(string(data), &meta)
	if !ok {
		body = string(data)
	}
	return &Character{Path: path, Meta: meta, Body: body}, nil
}

// Save writes the character back to its path.
func (c *Character) Save() error {
	out, err := serializeFrontmatter(c.Meta, c.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(c.Path, []byte(out), 0o644)
}

// ListCharacters reads all character files from <root>/characters/ and returns
// them sorted by display name. A missing characters/ directory returns nil
// without error — it simply means no characters exist yet.
func ListCharacters(root string) ([]Character, error) {
	paths, err := listEntityFiles(CharactersDir(root))
	if err != nil {
		return nil, err
	}
	var chars []Character
	for _, p := range paths {
		c, err := LoadCharacter(p)
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

// listEntityFiles returns the paths of all non-hidden .md files directly inside
// dir. A missing directory returns nil without error. Used by ListCharacters,
// ListLocations, and any future world-building entity list.
func listEntityFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".md" || (len(name) > 0 && name[0] == '.') {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	return paths, nil
}
