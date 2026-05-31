package core

import "os"

// scenePerm is the file mode used when writing scene files.
const scenePerm = 0o644

// newSceneTemplate is the starting content for a freshly created scene.
const newSceneTemplate = "# Untitled\n\n"

// Scene is one markdown file in a project. In this phase it is just a path plus
// its raw contents; structured metadata (YAML frontmatter) arrives in a later
// phase, at which point Content splits into frontmatter + body.
type Scene struct {
	Path    string
	Content string
}

// LoadScene reads a scene file from disk.
func LoadScene(path string) (*Scene, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &Scene{Path: path, Content: string(data)}, nil
}

// Save writes the scene's content back to its path.
func (s *Scene) Save() error {
	return os.WriteFile(s.Path, []byte(s.Content), scenePerm)
}

// CreateScene writes a new scene file with the default template and returns it.
func CreateScene(path string) (*Scene, error) {
	s := &Scene{Path: path, Content: newSceneTemplate}
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
		if r == ' ' || r == '\n' || r == '\t' {
			inWord = false
		} else if !inWord {
			words++
			inWord = true
		}
	}
	return words
}
