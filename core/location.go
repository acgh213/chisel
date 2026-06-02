package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LocationMeta is the YAML frontmatter for a location file.
type LocationMeta struct {
	Name         string   `yaml:"name,omitempty"`
	Type         string   `yaml:"type,omitempty"` // city, forest, building, region, …
	Description  string   `yaml:"description,omitempty"`
	Atmosphere   string   `yaml:"atmosphere,omitempty"`
	Significance string   `yaml:"significance,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
}

// IsEmpty reports whether the metadata carries no information.
func (m LocationMeta) IsEmpty() bool {
	return m.Name == "" && m.Type == "" && m.Description == "" &&
		m.Atmosphere == "" && m.Significance == "" && len(m.Tags) == 0
}

// Location is one file in the locations/ subdirectory: optional YAML
// frontmatter (Meta) plus free-form notes (Body).
type Location struct {
	Path string
	Meta LocationMeta
	Body string
}

// DisplayName returns the location's name from metadata if set, otherwise
// the file's basename without extension.
func (l Location) DisplayName() string {
	if l.Meta.Name != "" {
		return l.Meta.Name
	}
	return strings.TrimSuffix(filepath.Base(l.Path), ".md")
}

// LoadLocation reads a location file from disk, splitting frontmatter from body.
func LoadLocation(path string) (*Location, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta LocationMeta
	body, ok := parseFrontmatterInto(string(data), &meta)
	if !ok {
		body = string(data)
	}
	return &Location{Path: path, Meta: meta, Body: body}, nil
}

// Save writes the location back to its path.
func (l *Location) Save() error {
	out, err := serializeFrontmatter(l.Meta, l.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(l.Path, []byte(out), 0o644)
}

// ListLocations reads all location files from <root>/locations/ and returns
// them sorted by display name. A missing locations/ directory returns nil
// without error.
func ListLocations(root string) ([]Location, error) {
	paths, err := listEntityFiles(LocationsDir(root))
	if err != nil {
		return nil, err
	}
	var locs []Location
	for _, p := range paths {
		l, err := LoadLocation(p)
		if err != nil {
			continue // best-effort: skip unreadable files
		}
		locs = append(locs, *l)
	}
	sort.Slice(locs, func(i, j int) bool {
		return strings.ToLower(locs[i].DisplayName()) < strings.ToLower(locs[j].DisplayName())
	})
	return locs, nil
}

// LocationsDir returns the path to the locations/ subdirectory under root.
func LocationsDir(root string) string {
	return filepath.Join(root, "locations")
}
