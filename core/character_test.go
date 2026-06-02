package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCharacter_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "elara.md")

	c := &Character{
		Path: path,
		Meta: CharacterMeta{
			Name:        "Elara Nightwood",
			Role:        "protagonist",
			Description: "A young mage with silver hair.",
			Tags:        []string{"mage", "main-cast"},
		},
		Body: "# Elara Nightwood\n\nBackstory notes here.\n",
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadCharacter(path)
	if err != nil {
		t.Fatalf("LoadCharacter: %v", err)
	}
	if loaded.Meta.Name != "Elara Nightwood" {
		t.Errorf("Name = %q, want %q", loaded.Meta.Name, "Elara Nightwood")
	}
	if loaded.Meta.Role != "protagonist" {
		t.Errorf("Role = %q, want %q", loaded.Meta.Role, "protagonist")
	}
	if loaded.Meta.Description != "A young mage with silver hair." {
		t.Errorf("Description = %q", loaded.Meta.Description)
	}
	if len(loaded.Meta.Tags) != 2 || loaded.Meta.Tags[0] != "mage" {
		t.Errorf("Tags = %v", loaded.Meta.Tags)
	}
	if loaded.Body != c.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, c.Body)
	}
}

func TestCharacter_PlainFileLoads(t *testing.T) {
	dir := t.TempDir()
	path := writeScene(t, dir, "plain.md", "Just some notes with no frontmatter.\n")

	c, err := LoadCharacter(path)
	if err != nil {
		t.Fatalf("LoadCharacter plain: %v", err)
	}
	if c.Meta.Name != "" {
		t.Errorf("expected empty meta for plain file, got Name=%q", c.Meta.Name)
	}
	if c.Body == "" {
		t.Error("Body should not be empty for plain file")
	}
}

func TestCharacter_DisplayName(t *testing.T) {
	c := Character{Path: "/project/characters/kade-storm.md", Meta: CharacterMeta{Name: "Kade Storm"}}
	if c.DisplayName() != "Kade Storm" {
		t.Errorf("DisplayName with meta = %q", c.DisplayName())
	}

	c2 := Character{Path: "/project/characters/kade-storm.md"}
	if c2.DisplayName() != "kade-storm" {
		t.Errorf("DisplayName without meta = %q", c2.DisplayName())
	}
}

func TestListCharacters(t *testing.T) {
	root := t.TempDir()
	charsDir := filepath.Join(root, "characters")
	if err := os.Mkdir(charsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write two character files.
	for _, name := range []string{"zara.md", "aiden.md"} {
		c := &Character{
			Path: filepath.Join(charsDir, name),
			Meta: CharacterMeta{Name: name[:len(name)-3]},
		}
		if err := c.Save(); err != nil {
			t.Fatal(err)
		}
	}

	chars, err := ListCharacters(root)
	if err != nil {
		t.Fatalf("ListCharacters: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(chars))
	}
	// Must be sorted by display name: aiden < zara.
	if chars[0].DisplayName() != "aiden" {
		t.Errorf("first char = %q, want aiden", chars[0].DisplayName())
	}
}

func TestListCharacters_MissingDir(t *testing.T) {
	root := t.TempDir() // no characters/ subdir
	chars, err := ListCharacters(root)
	if err != nil {
		t.Errorf("expected nil error for missing characters dir, got %v", err)
	}
	if len(chars) != 0 {
		t.Errorf("expected empty list for missing dir, got %d", len(chars))
	}
}

func TestCharacter_RicherFieldsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kira.md")

	c := &Character{
		Path: path,
		Meta: CharacterMeta{
			Name:          "Kira Voss",
			Role:          "antagonist",
			Arc:           "learns that control is an illusion",
			Voice:         "clipped, formal, never contracts",
			Relationships: []string{"Elara: rival", "The Council: employer"},
			Tags:          []string{"main-cast"},
		},
		Body: "Notes.\n",
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := LoadCharacter(path)
	if err != nil {
		t.Fatalf("LoadCharacter: %v", err)
	}
	if loaded.Meta.Arc != "learns that control is an illusion" {
		t.Errorf("Arc = %q", loaded.Meta.Arc)
	}
	if loaded.Meta.Voice != "clipped, formal, never contracts" {
		t.Errorf("Voice = %q", loaded.Meta.Voice)
	}
	if len(loaded.Meta.Relationships) != 2 || loaded.Meta.Relationships[0] != "Elara: rival" {
		t.Errorf("Relationships = %v", loaded.Meta.Relationships)
	}
}

func TestCharacterMeta_IsEmptyWithRicherFields(t *testing.T) {
	if (CharacterMeta{Arc: "an arc"}).IsEmpty() {
		t.Error("CharacterMeta with Arc set should not be IsEmpty()")
	}
	if (CharacterMeta{Voice: "a voice"}).IsEmpty() {
		t.Error("CharacterMeta with Voice set should not be IsEmpty()")
	}
	if (CharacterMeta{Relationships: []string{"x: y"}}).IsEmpty() {
		t.Error("CharacterMeta with Relationships set should not be IsEmpty()")
	}
}
