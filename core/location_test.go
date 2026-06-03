package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocation_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "the-dark-forest.md")

	l := &Location{
		Path: path,
		Meta: LocationMeta{
			Name:        "The Dark Forest",
			Type:        "forest",
			Description: "A dense forest where light never reaches the floor.",
			Tags:        []string{"dangerous", "cursed"},
		},
		Body: "# The Dark Forest\n\nAncient trees block out all sunlight.\n",
	}
	if err := l.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadLocation(path)
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	if loaded.Meta.Name != "The Dark Forest" {
		t.Errorf("Name = %q", loaded.Meta.Name)
	}
	if loaded.Meta.Type != "forest" {
		t.Errorf("Type = %q", loaded.Meta.Type)
	}
	if loaded.Meta.Description != "A dense forest where light never reaches the floor." {
		t.Errorf("Description = %q", loaded.Meta.Description)
	}
	if len(loaded.Meta.Tags) != 2 || loaded.Meta.Tags[0] != "dangerous" {
		t.Errorf("Tags = %v", loaded.Meta.Tags)
	}
	if loaded.Body != l.Body {
		t.Errorf("Body = %q, want %q", loaded.Body, l.Body)
	}
}

func TestLocation_PlainFileLoads(t *testing.T) {
	dir := t.TempDir()
	path := writeScene(t, dir, "plain.md", "Just some notes.\n")

	l, err := LoadLocation(path)
	if err != nil {
		t.Fatalf("LoadLocation plain: %v", err)
	}
	if l.Meta.Name != "" {
		t.Errorf("expected empty meta, got Name=%q", l.Meta.Name)
	}
	if l.Body == "" {
		t.Error("Body should not be empty for plain file")
	}
}

func TestLocation_DisplayName(t *testing.T) {
	l := Location{Path: "/project/locations/ha-ren-city.md", Meta: LocationMeta{Name: "Ha'ren City"}}
	if l.DisplayName() != "Ha'ren City" {
		t.Errorf("DisplayName with meta = %q", l.DisplayName())
	}

	l2 := Location{Path: "/project/locations/ha-ren-city.md"}
	if l2.DisplayName() != "ha-ren-city" {
		t.Errorf("DisplayName without meta = %q", l2.DisplayName())
	}
}

func TestListLocations(t *testing.T) {
	root := t.TempDir()
	locsDir := filepath.Join(root, "locations")
	if err := os.Mkdir(locsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"zara-keep.md", "azure-sea.md"} {
		l := &Location{
			Path: filepath.Join(locsDir, name),
			Meta: LocationMeta{Name: name[:len(name)-3]},
		}
		if err := l.Save(); err != nil {
			t.Fatal(err)
		}
	}

	locs, err := ListLocations(root)
	if err != nil {
		t.Fatalf("ListLocations: %v", err)
	}
	if len(locs) != 2 {
		t.Fatalf("expected 2 locations, got %d", len(locs))
	}
	// Sorted: azure-sea < zara-keep.
	if locs[0].DisplayName() != "azure-sea" {
		t.Errorf("first loc = %q, want azure-sea", locs[0].DisplayName())
	}
}

func TestListLocations_MissingDir(t *testing.T) {
	root := t.TempDir()
	locs, err := ListLocations(root)
	if err != nil {
		t.Errorf("expected nil error for missing locations dir, got %v", err)
	}
	if len(locs) != 0 {
		t.Errorf("expected empty list for missing dir, got %d", len(locs))
	}
}

func TestLocation_RicherFieldsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "the-citadel.md")

	l := &Location{
		Path: path,
		Meta: LocationMeta{
			Name:         "The Citadel",
			Type:         "fortress",
			Atmosphere:   "cold, stone, echoing silence",
			Significance: "seat of the Council's power",
			Tags:         []string{"political"},
		},
		Body: "Notes.\n",
	}
	if err := l.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := LoadLocation(path)
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	if loaded.Meta.Atmosphere != "cold, stone, echoing silence" {
		t.Errorf("Atmosphere = %q", loaded.Meta.Atmosphere)
	}
	if loaded.Meta.Significance != "seat of the Council's power" {
		t.Errorf("Significance = %q", loaded.Meta.Significance)
	}
}

func TestLocationMeta_IsEmptyWithRicherFields(t *testing.T) {
	if (LocationMeta{Atmosphere: "misty"}).IsEmpty() {
		t.Error("LocationMeta with Atmosphere set should not be IsEmpty()")
	}
	if (LocationMeta{Significance: "important"}).IsEmpty() {
		t.Error("LocationMeta with Significance set should not be IsEmpty()")
	}
}
